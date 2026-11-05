//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package recovery

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestPOSIXFileLockUsesHandleAndIgnoresLegacyLockPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recovery.journal")
	if err := os.WriteFile(path+".lock", []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := acquireJournalFileLock(context.Background(), path, 0o600)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	busyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if _, err := acquireJournalFileLock(busyCtx, path, 0o600); !errors.Is(err, ErrJournalBusy) {
		t.Fatalf("second lock error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := acquireJournalFileLock(context.Background(), path, 0o600)
	if err != nil {
		t.Fatalf("lock after close: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPOSIXFileLockRejectsUnsafeLockObjects(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, string)
	}{
		{name: "symlink", setup: func(t *testing.T, path string) {
			target := path + ".target"
			if err := os.WriteFile(target, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "directory", setup: func(t *testing.T, path string) {
			if err := os.Mkdir(path, 0o700); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "non regular", setup: func(t *testing.T, path string) {
			if err := unix.Mkfifo(path, 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "wide permissions", setup: func(t *testing.T, path string) {
			if err := os.WriteFile(path, nil, 0o640); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, 0o640); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "recovery.journal.lock")
			test.setup(t, path)
			if _, err := acquireJournalFileLock(
				context.Background(), path[:len(path)-len(".lock")], 0o600,
			); !errors.Is(err, ErrJournalBusy) {
				t.Fatalf("acquireJournalFileLock() error = %v", err)
			}
		})
	}
}

func TestPOSIXFileLockRejectsPathReplacementWithDifferentInode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recovery.journal.lock")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	opened, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	if err := os.Rename(path, path+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateOpenedJournalLock(path, opened); !errors.Is(err, ErrJournalBusy) {
		t.Fatalf("validateOpenedJournalLock() error = %v", err)
	}
}

func TestWithLockJoinsOperationUnlockAndCloseErrors(t *testing.T) {
	operationErr := errors.New("operation failed")
	unlockErr := errors.New("unlock failed")
	closeErr := errors.New("close failed")
	journal := &FileRecoveryJournal{
		acquireLock: func(context.Context, string, fs.FileMode) (*journalFileLock, error) {
			return &journalFileLock{
				check:  checkedContext,
				unlock: func() error { return unlockErr },
				close:  func() error { return closeErr },
			}, nil
		},
	}
	err := journal.withLock(context.Background(), func(ctx context.Context) error { return operationErr })
	for _, want := range []error{operationErr, unlockErr, closeErr} {
		if !errors.Is(err, want) {
			t.Fatalf("withLock() error = %v, missing %v", err, want)
		}
	}
}

func TestWithLockInstanceSeamsRemainIsolatedInParallel(t *testing.T) {
	firstErr := errors.New("first release failed")
	secondErr := errors.New("second release failed")
	newJournal := func(releaseErr error) *FileRecoveryJournal {
		return &FileRecoveryJournal{
			acquireLock: func(context.Context, string, fs.FileMode) (*journalFileLock, error) {
				return &journalFileLock{
					check:  checkedContext,
					unlock: func() error { return releaseErr },
					close:  func() error { return nil },
				}, nil
			},
		}
	}
	first := newJournal(firstErr)
	second := newJournal(secondErr)
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, journal := range []*FileRecoveryJournal{first, second} {
		workers.Add(1)
		go func(journal *FileRecoveryJournal) {
			defer workers.Done()
			<-start
			results <- journal.withLock(context.Background(), func(ctx context.Context) error { return nil })
		}(journal)
	}
	close(start)
	workers.Wait()
	close(results)
	seen := map[error]int{}
	for err := range results {
		switch {
		case errors.Is(err, firstErr) && !errors.Is(err, secondErr):
			seen[firstErr]++
		case errors.Is(err, secondErr) && !errors.Is(err, firstErr):
			seen[secondErr]++
		default:
			t.Fatalf("withLock() error crossed instance seams: %v", err)
		}
	}
	if seen[firstErr] != 1 || seen[secondErr] != 1 {
		t.Fatalf("release errors = %#v", seen)
	}
}

// Native parser + real pending artifact replacement; the lease loss itself is
// synthetic. This test is not evidence of native execution on a Windows host.
func TestJournalPendingArtifactHeldLockContinuity(t *testing.T) {
	for _, lose := range []bool{false, true} {
		t.Run(map[bool]string{false: "Positive", true: "LossAfterArtifactRename"}[lose], func(t *testing.T) {
			j, p, ctx, records := newBoundaryWriter(t)
			next := headForRecords(records)
			p.forceSnapshot(AnchorSnapshot{JournalID: "journal", Namespace: RecoveryNamespaceV1, Epoch: 1, Pending: &AnchorPending{NewHead: next, OperationID: journalOperationID("journal", next)}})
			if e := os.Chmod(p.dir, 0o700); e != nil {
				t.Fatal(e)
			}
			raw, e := encodeDiskRecords(records)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.WriteFile(p.path, nil, 0o600); e != nil {
				t.Fatal(e)
			}
			stage := filepath.Join(p.dir, ".journal.tmp-native")
			if e = os.WriteFile(stage, raw, 0o600); e != nil {
				t.Fatal(e)
			}
			if lose {
				p.after = "ArtifactRename"
			}
			observed, e := lockedTestRecords(ctx, j, false)
			if p.closes != 1 {
				t.Fatalf("release count=%d", p.closes)
			}
			if lose {
				if !errors.Is(e, p.cause) || !errors.Is(e, ErrRecoveryWriteUnknown) || p.syncs != 0 || p.calls != 0 {
					t.Fatalf("repair continued after loss: CAS=%d sync=%d err=%v", p.calls, p.syncs, e)
				}
				actual, readErr := os.ReadFile(p.path)
				if readErr != nil || string(actual) != string(raw) {
					t.Fatal("completed rename evidence was not retained")
				}
			} else if e != nil || headForRecords(observed) != next || p.calls != 1 || p.syncs != 1 {
				t.Fatalf("positive pending repair: CAS=%d sync=%d err=%v", p.calls, p.syncs, e)
			}
		})
	}
}
