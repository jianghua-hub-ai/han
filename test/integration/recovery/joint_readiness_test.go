//go:build integration

package recovery_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// s44NewFixture owns one isolated PostgreSQL schema plus the real migrations.
// Unlike s417NewFixture it intentionally does not append to the S4.17 retained
// schema registry: this task only owns the three S4.4 test files, so writing a
// different task's evidence registry would violate the file ownership window.
// The schema is still never dropped; close only releases the pool.
func s44NewFixture(t *testing.T) *s417Fixture {
	t.Helper()
	dsn := os.Getenv("HAN_TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Fatal("HAN_TEST_POSTGRES_DSN is required for S4.4; no Skip")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	schema := "han_s44_" + s417RandomHex(t)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		cancel()
		t.Fatal("invalid HAN_TEST_POSTGRES_DSN value")
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	config.MaxConns = 8
	config.MinConns = 0
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		cancel()
		t.Fatal("open PostgreSQL pool failed")
	}
	fixture := &s417Fixture{t: t, ctx: ctx, cancel: cancel, schema: schema, pool: pool}
	if _, err := pool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		fixture.close()
		t.Fatalf("create isolated schema failed: %v", err)
	}
	t.Cleanup(fixture.close)
	fixture.migrate()
	fixture.now = s417Canonical(time.Now())
	return fixture
}

type s44WorldOptions struct {
	RecoveryJournal  func(*softwareJournal) recovery.ContextRecoveryJournal
	ArticleSnapshots func(*softwareJournal) recovery.VerifiedJournalReader
}

// s44TamperedJournal is a read-only observation wrapper. It never fabricates a
// passing proof: every mutation is fed into the real runtime admission checks.
type s44TamperedJournal struct {
	*softwareJournal
	mutate func(*recovery.VerifiedJournalSnapshot) error
}

func (j *s44TamperedJournal) ReadVerifiedSnapshot(ctx context.Context, namespace string) (recovery.VerifiedJournalSnapshot, error) {
	snapshot, err := j.softwareJournal.ReadVerifiedSnapshot(ctx, namespace)
	if err != nil {
		return snapshot, err
	}
	if j.mutate != nil {
		if err := j.mutate(&snapshot); err != nil {
			return snapshot, err
		}
	}
	return snapshot, nil
}

func s44NewWorld(t *testing.T) *s417World {
	t.Helper()
	return s44NewWorldWith(t, s44WorldOptions{})
}

func s44NewWorldWith(t *testing.T, opts s44WorldOptions) *s417World {
	t.Helper()
	fixture := s44NewFixture(t)
	now := fixture.now
	installation := recovery.InstallationBinding{
		Version:         recovery.InstallationVersion,
		DeploymentID:    "s44-deployment",
		InstallationID:  "s44-installation",
		TargetDigest:    s417Digest("s44-target"),
		CommitmentRef:   "recovery/v1/installation/s44-deployment",
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "s44-recovery-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		ArticleInitial:  recovery.AnchorSnapshot{JournalID: "s44-article-journal", Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 1},
		InstalledAt:     now,
	}
	manifest := recovery.InstallationManifest{
		Version:              recovery.InstallationVersion,
		Kind:                 "genesis",
		DeploymentID:         installation.DeploymentID,
		InstallationID:       installation.InstallationID,
		TargetDigest:         installation.TargetDigest,
		SourceApprovalRef:    "s44-approved-source",
		SourceApprovalDigest: s417Digest("s44-approval"),
		Recovery:             installation.RecoveryInitial,
		Article:              installation.ArticleInitial,
		IssuedAt:             installation.InstalledAt,
	}
	installation.CommitmentDigest = s417ManifestDigest(manifest)
	fixture.insertInstallation(installation)

	store, err := identity.NewRecoveryRunStore(fixture.pool)
	if err != nil {
		t.Fatalf("real recovery run store failed: %v", err)
	}
	clock := func() time.Time { return fixture.now }
	journal := s417NewSoftwareJournal(installation.RecoveryInitial.JournalID, recovery.RecoveryNamespaceV1, installation.RecoveryInitial.Epoch, clock)
	articleSnapshots := s417NewSoftwareJournal(installation.ArticleInitial.JournalID, recovery.ArticleDeletionNamespaceV1, installation.ArticleInitial.Epoch, clock)
	exclusive := &softwareExclusive{}
	lifecycle := &softwareLifecycle{}
	commitments := &softwareCommitments{
		deploymentID: installation.DeploymentID,
		commitment: recovery.InstallationCommitment{
			Manifest: manifest, ManifestDigest: installation.CommitmentDigest, Phase: "bound",
		},
		revision: "s44-commitment-1",
	}
	sources := &softwareSourceVerifier{expected: manifest.SourceApprovalDigest}
	closures, err := article.NewPostgresReplayEvidenceStore(fixture.pool)
	if err != nil {
		t.Fatalf("real published closure store failed: %v", err)
	}
	genesis, err := recovery.NewInstallationReader(commitments, sources)
	if err != nil {
		t.Fatalf("real installation reader failed: %v", err)
	}
	var journalPort recovery.ContextRecoveryJournal = journal
	if opts.RecoveryJournal != nil {
		journalPort = opts.RecoveryJournal(journal)
	}
	var articlePort recovery.VerifiedJournalReader = articleSnapshots
	if opts.ArticleSnapshots != nil {
		articlePort = opts.ArticleSnapshots(articleSnapshots)
	}
	runtime, err := recovery.NewRuntime(fixture.ctx, recovery.RuntimeDependencies{
		Store:            store,
		Journal:          journalPort,
		ArticleSnapshots: articlePort,
		Closures:         closures,
		Genesis:          genesis,
		Exclusive:        exclusive,
		Lifecycle:        lifecycle,
	})
	if err != nil {
		t.Fatalf("real recovery runtime factory failed: %v", err)
	}
	return &s417World{
		f: fixture, store: store, journal: journal, articleSnapshots: articleSnapshots,
		exclusive: exclusive, lifecycle: lifecycle, commitments: commitments, sources: sources,
		closures: closures, installation: installation, manifest: manifest, runtime: runtime, now: now,
	}
}

func s44RunNormal(t *testing.T, w *s417World) {
	t.Helper()
	if err := w.runtime.RunNormal(w.f.ctx); err != nil {
		t.Fatalf("real runtime RunNormal failed: %v", err)
	}
	observation, err := w.runtime.Verify(w.f.ctx)
	if err != nil {
		t.Fatalf("real runtime Verify failed: %v", err)
	}
	if observation.State != "public_ready" || observation.Reason != "verified" {
		t.Fatalf("real runtime observation=%+v; want public_ready/verified", observation)
	}
}

// s44NewRecoveryWorld mirrors s417NewRecoveryWorld (same real identity/recovery
// wiring) but owns an S4.4 schema that is not written into the S4.17 registry.
// The recovery journal and article deletion journal are softwareJournal test
// doubles; the PostgreSQL run store, closure store, coordinator and finalizer
// are the real implementations.
func s44NewRecoveryWorld(t *testing.T, withMember bool, hook func(context.Context, recovery.PhasePermit, recovery.RunBinding) error, worldHook func(*s417RecoveryWorld) func(context.Context, recovery.PhasePermit, recovery.RunBinding) error) *s417RecoveryWorld {
	t.Helper()
	fixture := s44NewFixture(t)
	now := fixture.now
	deletionJournal := s417NewSoftwareDeletionJournal(func() time.Time { return fixture.now })
	deletionJournal.journalID = "s44-recovery-article-journal"
	installation := recovery.InstallationBinding{
		Version:         recovery.InstallationVersion,
		DeploymentID:    "s44-recovery-deployment",
		InstallationID:  "s44-recovery-installation",
		TargetDigest:    s417Digest("s44-recovery-target"),
		CommitmentRef:   "recovery/v1/installation/s44-recovery-deployment",
		RecoveryInitial: recovery.AnchorSnapshot{JournalID: "s44-recovery-run-journal", Namespace: recovery.RecoveryNamespaceV1, Epoch: 1},
		ArticleInitial:  recovery.AnchorSnapshot{JournalID: deletionJournal.journalID, Namespace: recovery.ArticleDeletionNamespaceV1, Epoch: 1},
		InstalledAt:     now,
	}
	manifest := recovery.InstallationManifest{
		Version:              recovery.InstallationVersion,
		Kind:                 "genesis",
		DeploymentID:         installation.DeploymentID,
		InstallationID:       installation.InstallationID,
		TargetDigest:         installation.TargetDigest,
		SourceApprovalRef:    "s44-recovery-approved-source",
		SourceApprovalDigest: s417Digest("s44-recovery-approval"),
		Recovery:             installation.RecoveryInitial,
		Article:              installation.ArticleInitial,
		IssuedAt:             installation.InstalledAt,
	}
	installation.CommitmentDigest = s417ManifestDigest(manifest)
	fixture.insertInstallation(installation)

	store, err := identity.NewRecoveryRunStore(fixture.pool)
	if err != nil {
		t.Fatalf("real recovery run store failed: %v", err)
	}
	clock := func() time.Time { return fixture.now }
	journal := s417NewSoftwareJournal(installation.RecoveryInitial.JournalID, recovery.RecoveryNamespaceV1, installation.RecoveryInitial.Epoch, clock)
	closures, err := article.NewPostgresReplayEvidenceStore(fixture.pool)
	if err != nil {
		t.Fatalf("real published closure store failed: %v", err)
	}
	world := &s417RecoveryWorld{
		f: fixture, store: store, journal: journal, deletionJournal: deletionJournal,
		commitments: &softwareCommitments{
			deploymentID: installation.DeploymentID,
			commitment: recovery.InstallationCommitment{
				Manifest: manifest, ManifestDigest: installation.CommitmentDigest, Phase: "bound",
			},
			revision: "s44-recovery-commitment-1",
		},
		sources:      &softwareSourceVerifier{expected: manifest.SourceApprovalDigest},
		closures:     closures,
		installation: installation, manifest: manifest, now: now,
		secret:    []byte("s44-recovery-secret"),
		backupRef: "s44-recovery-backup-" + s417NewUUID()[:8],
	}
	world.s417WireFinalizer(t)
	if worldHook != nil {
		hook = worldHook(world)
	}
	world.s417WireRuntime(t, hook)
	world.s417Seed(t, withMember)
	return world
}

func s44AssertClosedProofUnavailable(t *testing.T, observation recovery.AdmissionObservation, err error) {
	t.Helper()
	if observation.State != "closed" || observation.Reason != "proof_unavailable" {
		t.Fatalf("observation=%+v; want closed/proof_unavailable", observation)
	}
	if err == nil {
		t.Fatal("Verify returned nil error; want fail-closed error")
	}
}

func TestJointReadiness_NamespaceAndRecovery_FailClosed(t *testing.T) {
	t.Run("PositiveGenesisAdmission", func(t *testing.T) {
		w := s44NewWorld(t)
		s44RunNormal(t, w)
		if got := w.f.countRuns(); got != 0 {
			t.Fatalf("recovery_runs=%d; want 0 for the empty-genesis joint baseline", got)
		}
		recoverySnapshot, err := w.journal.ReadVerifiedSnapshot(w.f.ctx, recovery.RecoveryNamespaceV1)
		if err != nil {
			t.Fatalf("read recovery namespace failed: %v", err)
		}
		articleSnapshot, err := w.articleSnapshots.ReadVerifiedSnapshot(w.f.ctx, recovery.ArticleDeletionNamespaceV1)
		if err != nil {
			t.Fatalf("read article namespace failed: %v", err)
		}
		if recoverySnapshot.Head.Sequence != 0 || articleSnapshot.Head.Sequence != 0 {
			t.Fatalf("joint baseline heads recovery=%d article=%d; want 0/0", recoverySnapshot.Head.Sequence, articleSnapshot.Head.Sequence)
		}
		if _, err := w.journal.ReadVerifiedSnapshot(w.f.ctx, recovery.ArticleDeletionNamespaceV1); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("recovery journal accepted article namespace: %v", err)
		}
		if _, err := w.articleSnapshots.ReadVerifiedSnapshot(w.f.ctx, recovery.RecoveryNamespaceV1); !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("article journal accepted recovery namespace: %v", err)
		}
		t.Logf("joint readiness positive: state=public_ready recovery_head=0 article_head=0 runs=0 namespaces=isolated")
	})

	t.Run("RecoveryNamespaceMismatchFailsClosed", func(t *testing.T) {
		w := s44NewWorldWith(t, s44WorldOptions{
			RecoveryJournal: func(j *softwareJournal) recovery.ContextRecoveryJournal {
				return &s44TamperedJournal{softwareJournal: j, mutate: func(s *recovery.VerifiedJournalSnapshot) error {
					s.Namespace = recovery.ArticleDeletionNamespaceV1
					return nil
				}}
			},
		})
		observation, err := w.runtime.Verify(w.f.ctx)
		s44AssertClosedProofUnavailable(t, observation, err)
		if !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("recovery namespace mismatch err=%v; want ErrAdmissionBinding", err)
		}
		t.Logf("recovery namespace mismatch rejected: state=%s reason=%s binding=true", observation.State, observation.Reason)
	})

	t.Run("ArticleNamespaceMismatchFailsClosed", func(t *testing.T) {
		w := s44NewWorldWith(t, s44WorldOptions{
			ArticleSnapshots: func(j *softwareJournal) recovery.VerifiedJournalReader {
				return &s44TamperedJournal{softwareJournal: j, mutate: func(s *recovery.VerifiedJournalSnapshot) error {
					s.Namespace = recovery.RecoveryNamespaceV1
					return nil
				}}
			},
		})
		observation, err := w.runtime.Verify(w.f.ctx)
		s44AssertClosedProofUnavailable(t, observation, err)
		if !errors.Is(err, recovery.ErrAdmissionBinding) {
			t.Fatalf("article namespace mismatch err=%v; want ErrAdmissionBinding", err)
		}
		t.Logf("article namespace mismatch rejected: state=%s reason=%s binding=true", observation.State, observation.Reason)
	})

	t.Run("ArticleHeadDriftFailsClosed", func(t *testing.T) {
		w := s44NewWorld(t)
		s44RunNormal(t, w)
		if _, err := w.articleSnapshots.Begin(w.f.ctx, "s44-article-drift-run", "s44-article-drift-backup", []byte("s44-article-drift-secret")); err != nil {
			t.Fatalf("advance article journal head failed: %v", err)
		}
		observation, err := w.runtime.Verify(w.f.ctx)
		s44AssertClosedProofUnavailable(t, observation, err)
		if !errors.Is(err, recovery.ErrAdmissionDrift) {
			t.Fatalf("article head drift err=%v; want ErrAdmissionDrift", err)
		}
		observation, err = w.runtime.Verify(w.f.ctx)
		s44AssertClosedProofUnavailable(t, observation, err)
		t.Logf("article head drift rejected and runtime stayed closed: state=%s reason=%s drift=true", observation.State, observation.Reason)
	})

	t.Run("CorruptRecoveryEntryFailsClosed", func(t *testing.T) {
		var tampered *s44TamperedJournal
		w := s44NewWorldWith(t, s44WorldOptions{
			RecoveryJournal: func(j *softwareJournal) recovery.ContextRecoveryJournal {
				tampered = &s44TamperedJournal{softwareJournal: j}
				return tampered
			},
		})
		tampered.mutate = func(s *recovery.VerifiedJournalSnapshot) error {
			s44InjectRecoveryEntry(s, func(e *recovery.JournalEntry) {
				e.RecordHMAC = "hmac-sha256:not-a-valid-record-hmac"
			})
			return nil
		}
		observation, err := w.runtime.Verify(w.f.ctx)
		s44AssertClosedProofUnavailable(t, observation, err)
		if !errors.Is(err, recovery.ErrAdmissionIncomplete) {
			t.Fatalf("corrupt recovery entry err=%v; want ErrAdmissionIncomplete", err)
		}
		t.Logf("corrupt recovery entry rejected: state=%s reason=%s incomplete=true", observation.State, observation.Reason)
	})

	t.Run("KeptDeletedCanonicalReview", func(t *testing.T) {
		world := s44NewRecoveryWorld(t, true, nil, func(w *s417RecoveryWorld) func(context.Context, recovery.PhasePermit, recovery.RunBinding) error {
			return func(liveCtx context.Context, p recovery.PhasePermit, b recovery.RunBinding) error {
				return w.s417BootstrapIdentity(liveCtx, p, b, true)
			}
		})
		liveRunID := "s44-run-kept-deleted-" + s417NewUUID()[:8]
		world.s417SeedFinalizedDeletion(t)
		trace, err := platform.NewTraceID()
		if err != nil {
			t.Fatalf("generate kept_deleted trace ID failed: %v", err)
		}
		tracedCtx, err := platform.ContextWithTraceID(world.f.ctx, trace)
		if err != nil {
			t.Fatalf("attach kept_deleted trace ID failed: %v", err)
		}
		ctx, cancel := context.WithTimeout(tracedCtx, 120*time.Second)
		defer cancel()
		err = world.runtime.Enter(ctx, recovery.RecoveryInput{
			RunBinding: recovery.RunBinding{RunID: liveRunID, BackupRef: world.backupRef},
			Secret:     append([]byte(nil), world.secret...),
		})
		if err != nil {
			for cause := err; cause != nil; cause = errors.Unwrap(cause) {
				t.Logf("kept_deleted Enter cause chain: %T: %v", cause, cause)
			}
			t.Fatalf("runtime.Enter with live kept_deleted review failed: %v", err)
		}
		liveRun, err := world.store.Load(ctx, liveRunID)
		if err != nil {
			t.Fatalf("kept_deleted run Load failed: %v", err)
		}
		if liveRun.Outcome != recovery.RecoveryOutcomeSucceeded {
			t.Fatalf("kept_deleted run outcome=%s; want succeeded", liveRun.Outcome)
		}

		// Assertions 1 and 2: the published recovery run must have consumed the
		// recovery journal and recorded the consumption timestamp.
		var journalStage string
		var journalConsumed bool
		if err := world.f.pool.QueryRow(ctx,
			"SELECT journal_stage, journal_consumed_at IS NOT NULL FROM recovery_runs WHERE run_id=$1",
			liveRunID).Scan(&journalStage, &journalConsumed); err != nil {
			t.Fatalf("read recovery_runs journal state failed: %v", err)
		}
		if journalStage != string(recovery.StageConsumed) || !journalConsumed {
			t.Fatalf("journal_stage=%q journal_consumed_at_present=%v; want consumed/true", journalStage, journalConsumed)
		}

		// Assertions 3 and 4: exactly one review row for this run/user and the
		// kept_deleted decision must not reset credentials.
		var reviews, resetReviews int
		if err := world.f.pool.QueryRow(ctx,
			`SELECT count(*), count(*) FILTER (WHERE credential_reset_at IS NOT NULL)
			 FROM recovery_account_reviews WHERE recovery_run_id=$1 AND user_id=$2`,
			liveRunID, world.memberID).Scan(&reviews, &resetReviews); err != nil {
			t.Fatalf("read recovery_account_reviews failed: %v", err)
		}
		if reviews != 1 || resetReviews != 0 {
			t.Fatalf("recovery_account_reviews rows=%d credential_reset=%d; want 1/0", reviews, resetReviews)
		}

		var status, decision, evidence string
		if err := world.f.pool.QueryRow(ctx, "SELECT status FROM users WHERE id=$1", world.memberID).Scan(&status); err != nil {
			t.Fatalf("read kept_deleted target status failed: %v", err)
		}
		if err := world.f.pool.QueryRow(ctx,
			"SELECT decision, COALESCE(evidence_ref,'') FROM recovery_account_reviews WHERE recovery_run_id=$1 AND user_id=$2",
			liveRunID, world.memberID).Scan(&decision, &evidence); err != nil {
			t.Fatalf("read kept_deleted review decision failed: %v", err)
		}
		var canonicalAudits int
		if err := world.f.pool.QueryRow(ctx,
			"SELECT count(*) FROM audit_events WHERE action='identity.user.delete.canonical' AND target_id=$1 AND metadata->>'mode'='recovery_kept_deleted'",
			world.memberID).Scan(&canonicalAudits); err != nil {
			t.Fatalf("read canonical delete audit failed: %v", err)
		}
		if status != "deleted" || decision != "kept_deleted" || evidence != "s417-recovery-review-evidence" || canonicalAudits != 1 {
			t.Fatalf("kept_deleted review mismatch: status=%q decision=%q evidence=%q canonical_audit=%d",
				status, decision, evidence, canonicalAudits)
		}
		t.Logf("kept_deleted canonical review: run=%s outcome=%s journal_stage=%s journal_consumed_at_present=%v member_status=%s decision=%s reviews=%d credential_reset=%d canonical_audit=%d journals=softwareJournal(test double)",
			liveRunID, liveRun.Outcome, journalStage, journalConsumed, status, decision, reviews, resetReviews, canonicalAudits)
	})
	t.Run("KeptDeletedZeroPermit_NoStateWritten", func(t *testing.T) {
		world := s44NewRecoveryWorld(t, true, nil, nil)
		err := world.maintenance.ResolveRestrictedAccountReview(world.f.ctx, recovery.PhasePermit{}, recovery.RunBinding{}, world.memberID, "kept_deleted", "s44-zero-permit-evidence")
		if err == nil {
			t.Fatal("zero-permit kept_deleted review was accepted")
		}
		if code := platform.PublicCode(err); code != platform.CodePlatformInternalError {
			t.Fatalf("zero-permit public code=%v; want %s", code, platform.CodePlatformInternalError)
		}
		var status string
		if err := world.f.pool.QueryRow(world.f.ctx, "SELECT status FROM users WHERE id=$1", world.memberID).Scan(&status); err != nil {
			t.Fatalf("read zero-permit target status failed: %v", err)
		}
		var reviews int
		if err := world.f.pool.QueryRow(world.f.ctx, "SELECT count(*) FROM recovery_account_reviews WHERE user_id=$1", world.memberID).Scan(&reviews); err != nil {
			t.Fatalf("count zero-permit reviews failed: %v", err)
		}
		var audits int
		if err := world.f.pool.QueryRow(world.f.ctx, "SELECT count(*) FROM audit_events WHERE target_id=$1", world.memberID).Scan(&audits); err != nil {
			t.Fatalf("count zero-permit audit events failed: %v", err)
		}
		if status != "active" || reviews != 0 || audits != 0 {
			t.Fatalf("zero-permit rejection wrote state: status=%q reviews=%d audits=%d; want active/0/0", status, reviews, audits)
		}
		if calls := world.journal.consumedMarkCount(); calls != 0 {
			t.Fatalf("zero-permit branch called MarkConsumed %d times; want 0 (no journal consumption)", calls)
		}
		t.Logf("zero-permit kept_deleted rejected fail-closed: asserted code=%s status=%s reviews=0 audits=0 consumed_marks=%d", platform.PublicCode(err), status, world.journal.consumedMarkCount())
	})
}

func s44InjectRecoveryEntry(s *recovery.VerifiedJournalSnapshot, mutate func(*recovery.JournalEntry)) {
	entry := recovery.JournalEntry{
		Namespace:         recovery.RecoveryNamespaceV1,
		Sequence:          1,
		RunID:             "s44-injected-run",
		BackupRef:         "s44-injected-backup",
		KeyID:             "s44-injected-key",
		KeyVersion:        "v1",
		SecretFingerprint: s417HMAC([]byte("s44-injected-key-material"), []byte("s44-injected-secret")),
		Stage:             recovery.StageBegun,
		RecordedAt:        s417Canonical(time.Now()),
	}
	if mutate != nil {
		mutate(&entry)
	}
	head := recovery.JournalHead{Sequence: entry.Sequence, Hash: s417ChainHash("", entry)}
	if entry.RecordHMAC == "" {
		entry.RecordHMAC = s417HMAC([]byte("s44-injected-key-material"), []byte(head.Hash+"|"+entry.SecretFingerprint))
	}
	s.Entries = []recovery.JournalEntry{entry}
	s.Head = head
	s.Prefixes = []recovery.JournalHead{{Sequence: 0, Hash: ""}, head}
	s.Revision = "s44-injected/r1/s1"
}
