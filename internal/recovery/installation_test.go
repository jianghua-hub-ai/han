package recovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

func TestRecoveryInstallationBinding(t *testing.T) {
	t.Run("IndependentApprovalAndPreparedBound", func(t *testing.T) {
		t.Run("FreshPublishesBothAnchorsAndDatabase", s413Fresh)
		t.Run("BoundOriginEligibility", s413BoundOrigins)
		t.Run("IndependentApprovalFailureHasNoWrites", func(t *testing.T) {
			x := s413New(t)
			x.w.sourceErr = ErrAdmissionIncomplete
			if x.call(t) == nil || x.writes != 0 || x.media != 0 {
				t.Fatal("unapproved source changed installation")
			}
		})
		t.Run("MissingSourcesRejectedWithoutIO", func(t *testing.T) {
			if _, err := NewInstallationExecutor(InstallationDependencies{}); !errors.Is(err, ErrAdmissionBinding) {
				t.Fatalf("missing independent sources accepted: %v", err)
			}
		})
	})
	t.Run("InterruptedSameManifestOnly", s413Interruptions)
	t.Run("InstallationExecutorAndBoundReentry", func(t *testing.T) {
		t.Run("ConfigurationPreflight", s413ConfigurationPreflight)
		t.Run("ApprovalMissingIsNotFresh", s413ApprovalMissingIsNotFresh)
		t.Run("BoundOnlyCoordinates", s413Bound)
		t.Run("FlatV1StrictFields", s413Codec)
		t.Run("RejectCallerPermit", func(t *testing.T) {
			e := &installationExecutor{}
			if err := e.Install(context.Background(), PhasePermit{}, InstallationManifest{}); !errors.Is(err, ErrAdmissionBinding) {
				t.Fatalf("caller permit must be rejected before I/O: %v", err)
			}
		})
	})
}

func s413Codec(t *testing.T) {
	w := newInstallationFixture(t, false)
	raw, err := EncodeInstallationManifest(w.m)
	if err != nil {
		t.Fatal(err)
	}
	if SHA256Digest(raw) != installationOracleDigest(w.m) || bytes.Contains(raw, []byte("committed")) || bytes.Contains(raw, []byte("pending")) {
		t.Fatal("original flat v1 canonical changed")
	}
	m, err := DecodeInstallationManifest(raw)
	if err != nil || m != w.m {
		t.Fatalf("roundtrip: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"version", "kind", "deployment_id", "installation_id", "target_digest", "source_approval_ref", "source_approval_digest", "recovery", "article", "issued_at"} {
		t.Run("Missing/"+name, func(t *testing.T) {
			var v map[string]json.RawMessage
			_ = json.Unmarshal(raw, &v)
			delete(v, name)
			b, _ := json.Marshal(v)
			if _, err := DecodeInstallationManifest(b); err == nil {
				t.Fatal("missing field accepted")
			}
		})
		t.Run("Null/"+name, func(t *testing.T) {
			var v map[string]json.RawMessage
			_ = json.Unmarshal(raw, &v)
			v[name] = json.RawMessage("null")
			b, _ := json.Marshal(v)
			if _, err := DecodeInstallationManifest(b); err == nil {
				t.Fatal("null accepted")
			}
		})
	}
	for name, b := range map[string][]byte{"duplicate": append([]byte(`{"version":"han-recovery-install/v1",`), raw[1:]...), "extra": append([]byte(`{"extra":1,`), raw[1:]...), "trailing": append(append([]byte{}, raw...), []byte("{}")...), "invalidUTF8": {0xff}, "oversize": []byte(strings.Repeat(" ", 65537)), "nestedDuplicate": bytes.Replace(raw, []byte(`"epoch":1`), []byte(`"epoch":1,"epoch":1`), 1), "nestedExtra": bytes.Replace(raw, []byte(`"head_hash":""`), []byte(`"head_hash":"","committed":{}`), 1)} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeInstallationManifest(b); err == nil {
				t.Fatal("malformed original manifest accepted")
			}
		})
	}
	if err := quick.Check(func(n uint32) bool {
		m := w.m
		m.Recovery.Epoch = uint64(n) + 1
		b, e := EncodeInstallationManifest(m)
		if e != nil {
			return false
		}
		v, e := DecodeInstallationManifest(b)
		return e == nil && v == m
	}, &quick.Config{MaxCount: 100}); err != nil {
		t.Fatal(err)
	}
}

type s413World struct {
	w                                            *installationFixture
	anchors                                      map[string]AnchorSnapshot
	tokens                                       map[string]string
	commit                                       InstallationCommitment
	commitToken                                  string
	serial, writes, bindings, media, coordinates int
	bound                                        bool
	fail                                         string
}

func s413New(t *testing.T) *s413World {
	w := newInstallationFixture(t, false)
	x := &s413World{w: w, anchors: map[string]AnchorSnapshot{}, tokens: map[string]string{}}
	x.runtime(t)
	return x
}

func (x *s413World) runtime(t *testing.T) {
	t.Helper()
	e, err := NewInstallationExecutor(InstallationDependencies{Commitments: x, Sources: x.w, Store: x, Writer: x, Anchor: x,
		Recovery: installationJournalPort{x.w.f, x.w}, Article: x, Media: x.prepare, Coordinate: x.coordinate})
	if err != nil {
		t.Fatal(err)
	}
	d := x.w.r.(*admissionRuntime).deps
	d.Store, d.InstallationWriter, d.Commitments, d.ArticleAnchorReader, d.ArticleSnapshots, d.Installation = x, x, x, x, x, e
	r, err := NewRuntime(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	x.w.r = r
	t.Cleanup(func() {
		if err := r.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
}

func (x *s413World) LoadInstallation(context.Context, string) (InstallationCommitment, string, error) {
	if x.commitToken == "" {
		return InstallationCommitment{}, "", ErrAnchorMissing
	}
	return x.commit, x.commitToken, nil
}
func (x *s413World) CompareAndSwapInstallation(_ context.Context, _ string, expected string, next InstallationCommitment) (InstallationCommitment, string, error) {
	if expected != x.commitToken {
		return InstallationCommitment{}, "", ErrAnchorConflict
	}
	x.serial++
	x.writes++
	x.commit = next
	x.commitToken = fmt.Sprint("commit-", x.serial)
	if x.fail == next.Phase {
		x.fail = ""
		return InstallationCommitment{}, "", ErrRecoveryWriteUnknown
	}
	return next, x.commitToken, nil
}
func (x *s413World) Load(_ context.Context, ns string) (AnchorSnapshot, string, error) {
	a, ok := x.anchors[ns]
	if !ok {
		return AnchorSnapshot{}, "", ErrAnchorMissing
	}
	return a, x.tokens[ns], nil
}
func (x *s413World) CompareAndSwap(_ context.Context, ns, expected string, next AnchorSnapshot) (AnchorSnapshot, string, error) {
	if expected != x.tokens[ns] {
		return AnchorSnapshot{}, "", ErrAnchorConflict
	}
	x.serial++
	x.writes++
	x.anchors[ns] = next
	x.tokens[ns] = fmt.Sprint("anchor-", x.serial)
	if ns == RecoveryNamespaceV1 {
		x.w.f.recovery.Revision = x.tokens[ns]
	} else {
		x.w.f.article.Revision = x.tokens[ns]
	}
	if x.fail == ns {
		x.fail = ""
		return AnchorSnapshot{}, "", ErrRecoveryWriteUnknown
	}
	return next, x.tokens[ns], nil
}
func (x *s413World) ReadAdmissionSnapshot(ctx context.Context) (AdmissionSnapshot, error) {
	if !x.bound {
		return AdmissionSnapshot{}, ErrAdmissionIncomplete
	}
	return x.w.f.ReadAdmissionSnapshot(ctx)
}
func (x *s413World) ReadAdmissionRevision(ctx context.Context) (StoreRevision, error) {
	s, e := x.ReadAdmissionSnapshot(ctx)
	return s.Revision, e
}
func (x *s413World) BindInstallation(ctx context.Context, p PhasePermit, b InstallationBinding) error {
	if err := x.w.r.CheckPhase(ctx, p, phaseInstall); err != nil {
		return err
	}
	if x.bound && x.w.f.snapshot.Installation != b {
		return ErrAdmissionBinding
	}
	if !x.bound {
		x.bindings++
		x.bound = true
		x.w.f.snapshot.Installation = b
	}
	if x.fail == "db" {
		x.fail = ""
		return ErrRecoveryWriteUnknown
	}
	return nil
}
func (x *s413World) BindLegacyPublication(context.Context, PhasePermit, string, PublicationBinding) error {
	return errors.New("unexpected legacy binding")
}
func (x *s413World) ReadVerifiedSnapshot(ctx context.Context, ns string) (VerifiedJournalSnapshot, error) {
	return x.w.f.ReadVerifiedSnapshot(ctx, ns)
}
func (x *s413World) prepare(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	x.media++
	return x.w.r.CheckPhase(ctx, p, phaseInstall)
}
func (x *s413World) coordinate(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	x.coordinates++
	a, token, err := x.w.r.CheckArticleCoordination(ctx, p, m.InstallationID, m.Article.JournalID, m.Article.Epoch)
	if err != nil {
		return err
	}
	if a != x.anchors[ArticleDeletionNamespaceV1] || token != x.tokens[ArticleDeletionNamespaceV1] {
		return ErrAdmissionDrift
	}
	return nil
}
func (x *s413World) call(t *testing.T) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return x.w.r.Install(ctx, x.w.m)
}
func s413Fresh(t *testing.T) {
	x := s413New(t)
	if err := x.call(t); err != nil {
		t.Fatalf("fresh installation did not complete: %v; phase=%s writes=%d db=%d media=%d coordinate=%d", err, x.commit.Phase, x.writes, x.bindings, x.media, x.coordinates)
	}
	if x.commit.Phase != "bound" || x.bindings != 1 || len(x.anchors) != 2 || x.coordinates != 1 || x.media != 1 {
		t.Fatalf("incomplete installation: commit=%s anchors=%d db=%d media=%d coordinate=%d", x.commit.Phase, len(x.anchors), x.bindings, x.media, x.coordinates)
	}
	if x.w.f.starts != 0 {
		t.Fatal("Install opened ordinary lifecycle")
	}
}
func s413Bound(t *testing.T) {
	x := s413New(t)
	x.commit = x.w.commitment
	x.commitToken = "bound-original"
	x.bound = true
	x.anchors[x.w.m.Recovery.Namespace] = x.w.m.Recovery
	x.tokens[x.w.m.Recovery.Namespace] = x.w.f.recovery.Revision
	x.anchors[x.w.m.Article.Namespace] = x.w.m.Article
	x.tokens[x.w.m.Article.Namespace] = x.w.f.article.Revision
	before := x.w.f.snapshot
	if err := x.call(t); err != nil {
		t.Fatalf("bound reentry did not coordinate: %v", err)
	}
	if x.writes != 0 || x.bindings != 0 || x.media != 0 || x.coordinates != 1 || !reflect.DeepEqual(before, x.w.f.snapshot) {
		t.Fatal("bound reentry reinstalled or changed Store")
	}
}
func s413Interruptions(t *testing.T) {
	for _, point := range []string{"prepared", RecoveryNamespaceV1, ArticleDeletionNamespaceV1, "db", "bound"} {
		t.Run(point, func(t *testing.T) {
			x := s413New(t)
			x.fail = point
			if err := x.call(t); err == nil {
				t.Fatal("injected unknown was silently accepted")
			}
			if err := x.w.r.Stop(context.Background()); err != nil {
				t.Fatal(err)
			}
			prior := x.writes
			x.runtime(t)
			if err := x.call(t); err != nil {
				t.Fatalf("same manifest interruption did not resume: %v", err)
			}
			if x.commit.Phase != "bound" || x.bindings != 1 || len(x.anchors) != 2 || x.writes < prior {
				t.Fatal("resume reset or duplicated installation")
			}
		})
	}
}

func s413BoundOrigins(t *testing.T) {
	cases := []struct {
		name, kind string
		change     func(*installationFixture, *InstallationBinding)
		want       error
	}{
		{"Genesis", "genesis", nil, nil},
		{"AnchoredMigration", "anchored_migration", nil, nil},
		{"UnknownKind", "other", nil, ErrAdmissionBinding},
		{"Prepared", "anchored_migration", func(w *installationFixture, _ *InstallationBinding) { w.commitment.Phase = "prepared" }, ErrAdmissionBinding},
		{"RecoveryHistory", "anchored_migration", s413OriginWithHistory, ErrAdmissionBinding},
		{"Unapproved", "anchored_migration", func(w *installationFixture, _ *InstallationBinding) { w.sourceErr = ErrAdmissionIncomplete }, ErrAdmissionIncomplete},
		{"WrongApproval", "anchored_migration", func(w *installationFixture, _ *InstallationBinding) { w.approved.TargetDigest = fixtureHash("other") }, ErrAdmissionBinding},
		{"WrongDigest", "anchored_migration", func(w *installationFixture, _ *InstallationBinding) {
			w.commitment.ManifestDigest = fixtureHash("other")
		}, ErrAdmissionBinding},
		{"WrongBinding", "anchored_migration", func(_ *installationFixture, b *InstallationBinding) { b.TargetDigest = fixtureHash("other") }, ErrAdmissionBinding},
		{"WrongInitialHead", "anchored_migration", func(_ *installationFixture, b *InstallationBinding) { b.ArticleInitial.Committed = JournalHead{} }, ErrAdmissionBinding},
		{"RevisionDrift", "anchored_migration", func(w *installationFixture, _ *InstallationBinding) {
			w.hook = func(event string) {
				if event == "approval" {
					w.commitRevision = "changed-revision"
				}
			}
		}, ErrAdmissionDrift},
		{"CommitmentDrift", "anchored_migration", func(w *installationFixture, _ *InstallationBinding) {
			w.hook = func(event string) {
				if event == "approval" {
					w.commitment.Phase = "prepared"
				}
			}
		}, ErrAdmissionDrift},
		{"MissingRevision", "anchored_migration", func(w *installationFixture, _ *InstallationBinding) { w.commitRevision = "" }, ErrAdmissionBinding},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, binding := s413ReaderOrigin(t, tc.kind)
			if tc.change != nil {
				tc.change(w, &binding)
			}
			s413CheckOrigin(t, w, binding, tc.want)
		})
	}
}

func s413CheckOrigin(t *testing.T, w *installationFixture, binding InstallationBinding, want error) {
	t.Helper()
	reader, err := NewInstallationReader(w, w)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	proof, err := reader.VerifyGenesis(ctx, binding)
	if !errors.Is(err, want) {
		t.Fatalf("bound origin %s: got %v want %v", w.m.Kind, err, want)
	}
	if err == nil && (proof.Binding != binding || proof.RecoveryInitial != binding.RecoveryInitial || proof.ArticleInitial != binding.ArticleInitial || proof.CommitmentDigest != binding.CommitmentDigest || proof.CommitmentRevision != w.commitRevision) {
		t.Fatal("origin projection lost exact approved binding")
	}
}

func s413ReaderOrigin(t *testing.T, kind string) (*installationFixture, InstallationBinding) {
	t.Helper()
	w := newInstallationFixture(t, false)
	w.m.Kind = kind
	if kind != "genesis" {
		w.m.Article.Committed = JournalHead{Sequence: 1, Hash: fixtureHash("independently-approved-old-article-chain")}
	}
	w.approved = w.m
	w.commitment = InstallationCommitment{Manifest: w.m, ManifestDigest: installationOracleDigest(w.m), Phase: "bound"}
	b := InstallationBinding{Version: w.m.Version, DeploymentID: w.m.DeploymentID, InstallationID: w.m.InstallationID, TargetDigest: w.m.TargetDigest, CommitmentRef: "recovery/v1/installation/" + w.m.DeploymentID, CommitmentDigest: w.commitment.ManifestDigest, RecoveryInitial: w.m.Recovery, ArticleInitial: w.m.Article, InstalledAt: w.m.IssuedAt}
	return w, b
}

func s413OriginWithHistory(w *installationFixture, b *InstallationBinding) {
	w.m.Recovery.Committed = JournalHead{Sequence: 1, Hash: fixtureHash("prior-recovery-begin")}
	w.approved = w.m
	w.commitment.Manifest = w.m
	w.commitment.ManifestDigest = installationOracleDigest(w.m)
	b.RecoveryInitial = w.m.Recovery
	b.CommitmentDigest = w.commitment.ManifestDigest
}

// Configuration negatives enter through actual NewRuntime/Install and verify
// branch-specific dependencies without installation side effects.
func s413ConfigurationPreflight(t *testing.T) {
	for _, mode := range []string{"FreshWrongWriter", "PreparedWrongWriter", "FreshMissingMedia", "BoundWithoutWriterOrMedia", "BoundWithUnusedDifferentWriter"} {
		t.Run(mode, func(t *testing.T) { s413ConfigurationCase(t, mode) })
	}
}

func s413ConfigurationCase(t *testing.T, mode string) {
	x, other := s413New(t), s413New(t)
	bound := strings.HasPrefix(mode, "Bound")
	if mode == "PreparedWrongWriter" {
		x.commit = InstallationCommitment{Manifest: x.w.m, ManifestDigest: installationOracleDigest(x.w.m), Phase: "prepared"}
		x.commitToken = "independently-observed-prepared"
	}
	if bound {
		s413PreflightBound(x)
	}
	writer := InstallationRepository(other)
	media := x.prepare
	if mode == "FreshMissingMedia" {
		writer, media = x, nil
	}
	if bound {
		media = nil
	}
	if mode == "BoundWithoutWriterOrMedia" {
		writer = nil
	}
	s413PreflightRuntime(t, x, writer, media, bound)
	before, token := x.commit, x.commitToken
	result := x.call(t)
	stopErr := x.w.r.Stop(context.Background())
	if bound && result != nil {
		t.Errorf("bound-only unused dependency rejected: %v", result)
	}
	if !bound && !errors.Is(result, ErrAdmissionBinding) {
		t.Errorf("invalid fresh/prepared configuration: %v", result)
	}
	wantAnchors, wantCoordinates := 0, 0
	if bound {
		wantAnchors, wantCoordinates = 2, 1
	}
	if x.writes != 0 || x.media != 0 || x.bindings != 0 || other.bindings != 0 || len(x.anchors) != wantAnchors || x.coordinates != wantCoordinates || x.commit != before || x.commitToken != token {
		t.Error("configuration rejection/bound-only action changed installation sources")
	}
	if stopErr != nil {
		t.Errorf("actual Stop did not join: %v", stopErr)
	}
	t.Logf("configuration=%s CAS=%d media=%d ownDB=%d otherDB=%d anchors=%d coordinate=%d StopJoined=%t", mode, x.writes, x.media, x.bindings, other.bindings, len(x.anchors), x.coordinates, stopErr == nil)
}

func s413PreflightBound(x *s413World) {
	x.commit = x.w.commitment
	x.commitToken = "independently-observed-bound"
	x.bound = true
	x.anchors[x.w.m.Recovery.Namespace] = x.w.m.Recovery
	x.tokens[x.w.m.Recovery.Namespace] = x.w.f.recovery.Revision
	x.anchors[x.w.m.Article.Namespace] = x.w.m.Article
	x.tokens[x.w.m.Article.Namespace] = x.w.f.article.Revision
}

func s413PreflightRuntime(t *testing.T, x *s413World, writer InstallationRepository, media func(context.Context, PhasePermit, InstallationManifest) error, bound bool) {
	t.Helper()
	d := x.w.r.(*admissionRuntime).deps
	e, err := NewInstallationExecutor(InstallationDependencies{Commitments: x, Sources: x.w, Store: x, Writer: writer, Anchor: x, Recovery: installationJournalPort{x.w.f, x.w}, Article: x, Media: media, Coordinate: x.coordinate})
	if err != nil {
		t.Fatal(err)
	}
	d.Installation = e
	if bound {
		d.InstallationWriter = nil
	}
	r, err := NewRuntime(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	x.w.r = r
	t.Cleanup(func() {
		if err := r.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
}

// Inject a source read failure only after public Runtime issues its real grant;
// the forwarding observer neither creates nor changes the permit or context.
func s413ApprovalMissingIsNotFresh(t *testing.T) {
	x := s413New(t)
	marker := errors.New("author synthetic approved source became unreadable")
	injected := false
	d := x.w.r.(*admissionRuntime).deps
	d.Installation = &s413InstallObserver{target: d.Installation, start: func(ctx context.Context, p PhasePermit) error {
		if err := x.w.r.CheckPhase(ctx, p, "install"); err != nil {
			return err
		}
		injected = true
		x.w.sourceErr = errors.Join(ErrAdmissionIncomplete, ErrAnchorMissing, marker)
		return nil
	}}
	r, err := NewRuntime(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	x.w.r = r
	t.Cleanup(func() {
		if err := r.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	result := x.call(t)
	stopErr := x.w.r.Stop(context.Background())
	if !injected || !errors.Is(result, marker) || !errors.Is(result, ErrAdmissionIncomplete) {
		t.Errorf("actual post-grant source failure/cause was not observed: %v", result)
	}
	if x.writes != 0 || x.media != 0 || x.bindings != 0 || len(x.anchors) != 0 || x.commitToken != "" {
		t.Error("independent source read failure was treated as missing commitment and wrote metadata")
	}
	if stopErr != nil {
		t.Errorf("actual Stop did not join: %v", stopErr)
	}
	t.Logf("sourceFailureInjected=%t CAS=%d media=%d DBbind=%d anchors=%d StopJoined=%t", injected, x.writes, x.media, x.bindings, len(x.anchors), stopErr == nil)
}

type s413InstallObserver struct {
	target InstallationExecutor
	start  func(context.Context, PhasePermit) error
}

func (o *s413InstallObserver) Install(ctx context.Context, p PhasePermit, m InstallationManifest) error {
	if err := o.start(ctx, p); err != nil {
		return err
	}
	return o.target.Install(ctx, p, m)
}
