package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"han/internal/recovery"
)

func TestRecoveryInstallationStoreBinding(t *testing.T) {
	t.Run("IndependentApprovalAndPreparedBound", func(t *testing.T) {
		t.Run("MissingSameStoreRejected", func(t *testing.T) {
			if _, err := NewRecoveryInstallationExecutor(nil, nil, recovery.InstallationDependencies{}, nil); !errors.Is(err, recovery.ErrAdmissionBinding) {
				t.Fatalf("missing same Store accepted: %v", err)
			}
		})
	})
	t.Run("InstallationExecutorAndBoundReentry", func(t *testing.T) {
		t.Run("NoAmbientWriteAuthority", func(t *testing.T) {
			s := &RecoveryRunStore{}
			if err := s.BindInstallation(context.Background(), recovery.PhasePermit{}, recovery.InstallationBinding{}); !errors.Is(err, recovery.ErrAdmissionBinding) {
				t.Fatalf("no current permit: %v", err)
			}
			if err := s.BindLegacyPublication(context.Background(), recovery.PhasePermit{}, "run", recovery.PublicationBinding{}); !errors.Is(err, recovery.ErrAdmissionBinding) {
				t.Fatalf("no imported source: %v", err)
			}
		})
	})
}

var s413Time = time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)

func s413Digest(s string) string { return recovery.SHA256Digest([]byte(s)) }

type s413ExecutionForwarder struct{ target recovery.InstallationExecutor }

func (f *s413ExecutionForwarder) Install(c context.Context, p recovery.PhasePermit, m recovery.InstallationManifest) error {
	if f.target == nil {
		return recovery.ErrAdmissionClosed
	}
	return f.target.Install(c, p, m)
}

type s413PGWorld struct {
	m                                                         recovery.InstallationManifest
	approved                                                  recovery.InstallationManifest
	commit                                                    recovery.InstallationCommitment
	commitToken                                               string
	anchors                                                   map[string]recovery.AnchorSnapshot
	tokens                                                    map[string]string
	journal, article                                          recovery.VerifiedJournalSnapshot
	runtime                                                   recovery.Runtime
	store                                                     *RecoveryRunStore
	closures                                                  recovery.PublishedClosureReader
	serial, writes, media, coordinates, starts, stops, closes int
	fail                                                      string
	approvalError                                             error
	legacy                                                    func(context.Context, recovery.PhasePermit, recovery.InstallationManifest) ([]recovery.PublicationBinding, error)
	coordinateHook                                            func(context.Context, recovery.PhasePermit) error
	mediaHook                                                 func(context.Context, recovery.PhasePermit) error
}

func s413WorldFor(store *RecoveryRunStore) *s413PGWorld {
	r := recovery.AnchorSnapshot{Namespace: recovery.RecoveryNamespaceV1, JournalID: "recovery-journal", Epoch: 1}
	a := recovery.AnchorSnapshot{Namespace: recovery.ArticleDeletionNamespaceV1, JournalID: "article-journal", Epoch: 2}
	m := recovery.InstallationManifest{Version: recovery.InstallationVersion, Kind: "genesis", DeploymentID: "s413-deployment", InstallationID: "s413-installation", TargetDigest: s413Digest("synthetic-db-installation-source"), SourceApprovalRef: "independent-installation-checkpoint", SourceApprovalDigest: s413Digest("independent-approved-source"), Recovery: r, Article: a, IssuedAt: s413Time}
	x := &s413PGWorld{m: m, approved: m, store: store, anchors: map[string]recovery.AnchorSnapshot{}, tokens: map[string]string{}}
	x.journal = recovery.VerifiedJournalSnapshot{Namespace: r.Namespace, JournalID: r.JournalID, Epoch: r.Epoch, Entries: []recovery.JournalEntry{}, Prefixes: []recovery.JournalHead{{}}}
	x.article = recovery.VerifiedJournalSnapshot{Namespace: a.Namespace, JournalID: a.JournalID, Epoch: a.Epoch, Entries: []recovery.JournalEntry{}, Prefixes: []recovery.JournalHead{{}}}
	x.closures = x
	return x
}
func (x *s413PGWorld) newRuntime(t *testing.T) {
	t.Helper()
	f := &s413ExecutionForwarder{}
	r, err := recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{Store: x.store, Journal: x, ArticleSnapshots: x, Closures: x.closures, Genesis: x, Exclusive: x, Lifecycle: x, Installation: f, Commitments: x, InstallationSources: x, ArticleAnchorReader: x, InstallationWriter: x.store})
	if err != nil {
		t.Fatal(err)
	}
	d := recovery.InstallationDependencies{Commitments: x, Sources: x, Writer: x.store, Store: x.store, Anchor: x, Recovery: x, Article: x, Media: x.prepare, Coordinate: x.coordinate, Legacy: x.legacy}
	f.target, err = NewRecoveryInstallationExecutor(x.store, r, d, x.closures)
	if err != nil {
		t.Fatal(err)
	}
	x.runtime = r
	t.Cleanup(func() {
		if err := r.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
}
func (x *s413PGWorld) LoadInstallation(context.Context, string) (recovery.InstallationCommitment, string, error) {
	if x.commitToken == "" {
		return recovery.InstallationCommitment{}, "", recovery.ErrAnchorMissing
	}
	return x.commit, x.commitToken, nil
}
func (x *s413PGWorld) CompareAndSwapInstallation(_ context.Context, _ string, old string, next recovery.InstallationCommitment) (recovery.InstallationCommitment, string, error) {
	if old != x.commitToken {
		return recovery.InstallationCommitment{}, "", recovery.ErrAnchorConflict
	}
	x.serial++
	x.writes++
	x.commit = next
	x.commitToken = fmt.Sprint("commit-", x.serial)
	if x.fail == next.Phase {
		x.fail = ""
		return recovery.InstallationCommitment{}, "", recovery.ErrRecoveryWriteUnknown
	}
	return x.commit, x.commitToken, nil
}
func (x *s413PGWorld) VerifyInstallationSource(_ context.Context, m recovery.InstallationManifest) error {
	if m != x.approved {
		return recovery.ErrAdmissionBinding
	}
	return x.approvalError
}
func (x *s413PGWorld) Load(_ context.Context, ns string) (recovery.AnchorSnapshot, string, error) {
	a, ok := x.anchors[ns]
	if !ok {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAnchorMissing
	}
	return a, x.tokens[ns], nil
}
func (x *s413PGWorld) CompareAndSwap(_ context.Context, ns, old string, next recovery.AnchorSnapshot) (recovery.AnchorSnapshot, string, error) {
	if old != x.tokens[ns] {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAnchorConflict
	}
	x.serial++
	x.writes++
	x.anchors[ns] = next
	x.tokens[ns] = fmt.Sprint("anchor-", x.serial)
	if ns == recovery.RecoveryNamespaceV1 {
		x.journal.Revision = x.tokens[ns]
	} else {
		x.article.Revision = x.tokens[ns]
	}
	if x.fail == ns {
		x.fail = ""
		return recovery.AnchorSnapshot{}, "", recovery.ErrRecoveryWriteUnknown
	}
	return next, x.tokens[ns], nil
}
func (x *s413PGWorld) ReadVerifiedSnapshot(_ context.Context, ns string) (recovery.VerifiedJournalSnapshot, error) {
	if ns == recovery.RecoveryNamespaceV1 {
		return x.journal, nil
	}
	if ns == recovery.ArticleDeletionNamespaceV1 {
		return x.article, nil
	}
	return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
}
func (x *s413PGWorld) prepare(ctx context.Context, p recovery.PhasePermit, m recovery.InstallationManifest) error {
	x.media++
	if err := x.runtime.CheckPhase(ctx, p, "install"); err != nil {
		return err
	}
	if m != x.approved {
		return recovery.ErrAdmissionBinding
	}
	if x.mediaHook != nil {
		return x.mediaHook(ctx, p)
	}
	return nil
}
func (x *s413PGWorld) coordinate(ctx context.Context, p recovery.PhasePermit, m recovery.InstallationManifest) error {
	x.coordinates++
	a, v, err := x.runtime.CheckArticleCoordination(ctx, p, m.InstallationID, m.Article.JournalID, m.Article.Epoch)
	if err != nil {
		return err
	}
	if a != x.anchors[m.Article.Namespace] || v != x.tokens[m.Article.Namespace] {
		return recovery.ErrAdmissionDrift
	}
	if x.coordinateHook != nil {
		return x.coordinateHook(ctx, p)
	}
	return nil
}
func (x *s413PGWorld) VerifyGenesis(ctx context.Context, b recovery.InstallationBinding) (recovery.GenesisProof, error) {
	v, err := recovery.NewInstallationReader(x, x)
	if err != nil {
		return recovery.GenesisProof{}, err
	}
	return v.VerifyGenesis(ctx, b)
}
func (x *s413PGWorld) ReadPublishedClosure(context.Context, recovery.ClosureRequest) (recovery.PublishedClosure, error) {
	return recovery.PublishedClosure{}, recovery.ErrAdmissionIncomplete
}
func (x *s413PGWorld) Acquire(context.Context) (recovery.ExclusiveLease, error) {
	return &s413SyntheticLease{world: x}, nil
}

type s413SyntheticLease struct {
	world  *s413PGWorld
	closed bool
}

func (l *s413SyntheticLease) Check(ctx context.Context) error {
	if l.closed {
		return recovery.ErrAdmissionClosed
	}
	return ctx.Err()
}
func (l *s413SyntheticLease) Close() error {
	if !l.closed {
		l.closed = true
		l.world.closes++
	}
	return nil
}
func (x *s413PGWorld) ReconcileAndStart(ctx context.Context, p recovery.PhasePermit) error {
	x.starts++
	return x.runtime.CheckPhase(ctx, p, recovery.PhaseStartupReconcile)
}
func (x *s413PGWorld) StopAndJoin(context.Context) error { x.stops++; return nil }
func (x *s413PGWorld) Begin(context.Context, string, string, []byte) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, errors.New("unexpected Begin")
}
func (x *s413PGWorld) VerifyBeginSecret(context.Context, recovery.RunBinding, []byte) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, errors.New("unexpected secret verification")
}
func (x *s413PGWorld) MarkConsumed(context.Context, string) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, errors.New("unexpected Consume")
}
func (x *s413PGWorld) Read(context.Context, string) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, errors.New("unexpected repairing read")
}
func (x *s413PGWorld) VerifyAgainst(context.Context, recovery.RecoveryRun) (recovery.JournalEntry, error) {
	return recovery.JournalEntry{}, errors.New("unexpected repair")
}
func s413HMAC() string { return "hmac-sha256:" + strings.Repeat("a", 64) }
