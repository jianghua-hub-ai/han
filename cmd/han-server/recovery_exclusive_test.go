package main

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"han/internal/recovery"
)

func TestRecoveryDeploymentExclusive(t *testing.T) {
	t.Run("ExclusiveAndRealChildJoin", testS414Exclusive)
}

func testS414Exclusive(t *testing.T) {
	t.Run("MissingFacility", s414ExclusiveCase1)
	t.Run("InvalidPathAndContext", s414ExclusiveCase2)
	for _, pair := range []string{"normal/recovery", "normal/requeue", "recovery/requeue"} {
		t.Run("Pair_"+pair, func(t *testing.T) {
			facility := &s414Facility{}
			e := &restoreDeploymentExclusive{path: filepath.Join(s414Directory(t), "outside-runtime.lock"), acquire: facility.acquire}
			first, err := e.Acquire(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if _, err = e.Acquire(context.Background()); !errors.Is(err, recovery.ErrAdmissionBusy) {
				t.Fatal("double deployment ownership", err)
			}
			if err = first.Check(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err = first.Close(); err != nil {
				t.Fatal(err)
			}
			if err = first.Close(); err != nil {
				t.Fatal(err)
			}
			if err = first.Check(context.Background()); err == nil {
				t.Fatal("closed lease usable")
			}
			next, err := e.Acquire(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			next.Close()
			if facility.closes.Load() != 2 {
				t.Fatal("Close not once per owner")
			}
		})
	}
	t.Run("LostAndCancelled", s414ExclusiveCase3)
	t.Run("AcquireErrorRetainsReturnedOwner", s414ExclusiveCase4)
	t.Run("ConcurrentCloseActuallyJoins", s414ExclusiveCase5)
	t.Run("RuntimeStopJoinsChild", testS414ChildJoin)
}

type s414Facility struct {
	mu     sync.Mutex
	held   bool
	closes atomic.Int32
}

func (f *s414Facility) acquire(ctx context.Context, p string) (recovery.ExclusiveLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.held {
		return nil, recovery.ErrAdmissionBusy
	}
	f.held = true
	return &s414FacilityLease{facility: f}, nil
}

type s414FacilityLease struct{ facility *s414Facility }

func (l *s414FacilityLease) Check(ctx context.Context) error {
	l.facility.mu.Lock()
	defer l.facility.mu.Unlock()
	if !l.facility.held {
		return recovery.ErrAdmissionClosed
	}
	return ctx.Err()
}
func (l *s414FacilityLease) Close() error {
	l.facility.mu.Lock()
	defer l.facility.mu.Unlock()
	l.facility.held = false
	l.facility.closes.Add(1)
	return nil
}

type s414BlockingLease struct {
	entered, release chan struct{}
	closes           atomic.Int32
}

func (l *s414BlockingLease) Check(ctx context.Context) error { return ctx.Err() }
func (l *s414BlockingLease) Close() error                    { l.closes.Add(1); close(l.entered); <-l.release; return nil }

func s414ExclusiveCase1(t *testing.T) {
	e := &restoreDeploymentExclusive{}
	if _, err := e.Acquire(context.Background()); !errors.Is(err, recovery.ErrAdmissionUnsupported) {
		t.Fatal("missing actual facility accepted")
	}
}

func s414ExclusiveCase2(t *testing.T) {
	calls := 0
	e := &restoreDeploymentExclusive{acquire: func(context.Context, string) (recovery.ExclusiveLease, error) { calls++; return &s414Lease{}, nil }}
	for _, p := range []string{"", "relative", " /unsafe "} {
		e.path = p
		if _, err := e.Acquire(context.Background()); err == nil {
			t.Fatal("path accepted")
		}
	}
	if _, err := e.Acquire(nil); err == nil {
		t.Fatal("nil ctx")
	}
	if calls != 0 {
		t.Fatal("invalid acquisition")
	}
}

func s414ExclusiveCase3(t *testing.T) {
	l := &s414Lease{}
	e := &restoreDeploymentExclusive{path: filepath.Join(s414Directory(t), "existing.lock"), acquire: func(context.Context, string) (recovery.ExclusiveLease, error) { return l, nil }}
	owned, err := e.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	l.lost.Store(true)
	if owned.Check(context.Background()) == nil {
		t.Fatal("lost facility accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(owned.Check(ctx), context.Canceled) {
		t.Fatal("cancellation lost")
	}
	owned.Close()
}

func s414ExclusiveCase4(t *testing.T) {
	l := &s414Lease{}
	e := &restoreDeploymentExclusive{path: filepath.Join(s414Directory(t), "existing.lock"), acquire: func(context.Context, string) (recovery.ExclusiveLease, error) {
		return l, recovery.ErrAdmissionIncomplete
	}}
	owned, err := e.Acquire(context.Background())
	if err == nil || owned == nil || l.closes.Load() != 0 {
		t.Fatal("returned owner discarded")
	}
	owned.Close()
	if l.closes.Load() != 1 {
		t.Fatal("owned handle leaked")
	}
}

func s414ExclusiveCase5(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	l := &s414BlockingLease{entered: entered, release: release}
	e := &restoreDeploymentExclusive{path: filepath.Join(s414Directory(t), "existing.lock"), acquire: func(context.Context, string) (recovery.ExclusiveLease, error) { return l, nil }}
	owned, err := e.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	done1, done2 := make(chan error, 1), make(chan error, 1)
	go func() { done1 <- owned.Close() }()
	<-entered
	go func() { done2 <- owned.Close() }()
	select {
	case <-done1:
		t.Fatal("early Close")
	default:
	}
	select {
	case <-done2:
		t.Fatal("repeat Close did not join")
	default:
	}
	close(release)
	if <-done1 != nil || <-done2 != nil || l.closes.Load() != 1 {
		t.Fatal("actual Close not joined once")
	}
}
