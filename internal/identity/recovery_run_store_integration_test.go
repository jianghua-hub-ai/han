//go:build integration

package identity

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"han/internal/recovery"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRecoveryRunStoreMainlineUTCAndJournalVerification(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	store, pool := newMigratedRecoveryRunStore(t, ctx)
	started := time.Date(2026, 10, 22, 1, 2, 3, 123456000, time.UTC)
	run := recovery.RecoveryRun{
		RunID: "run-01", BackupRef: "backup-01", JournalKeyID: "recovery",
		JournalKeyVersion: "v1", JournalSecretFingerprint: "hmac-sha256:synthetic",
		JournalStage: recovery.StageBegun, StartedAt: started, Outcome: recovery.RecoveryOutcomeRunning,
	}
	if err := store.Create(ctx, run); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	loaded, err := store.Load(ctx, run.RunID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.BackupRef != run.BackupRef || !loaded.StartedAt.Equal(started) {
		t.Fatalf("loaded = %#v", loaded)
	}
	assertUTCMicrosecond(t, "started_at", loaded.StartedAt)
	seedS49LegacyTrustProof(t, ctx, pool, run.RunID, started)
	changed := started.Add(time.Minute)
	loaded, err = store.Transition(ctx, run.RunID, recovery.RecoveryOutcomeRunning,
		recovery.RecoveryOutcomeIdentityReady, recovery.RecoveryRunChange{ChangedAt: changed})
	if err != nil {
		t.Fatalf("running -> identity_ready: %v", err)
	}
	verifyS49LegacyConsumption(t, ctx, store, run, changed)
}

func verifyS49LegacyConsumption(t *testing.T, ctx context.Context, store *RecoveryRunStore, run recovery.RecoveryRun, changed time.Time) {
	t.Helper()
	loaded, err := store.Transition(ctx, run.RunID, recovery.RecoveryOutcomeIdentityReady,
		recovery.RecoveryOutcomeArticleDeletionReady, recovery.RecoveryRunChange{
			ArticleEvidence: &recovery.ReadyReplayStatusV1{
				EvidenceVersion: recovery.ReplayEvidenceVersionV1, RunID: run.RunID,
				BackupRef: run.BackupRef, Namespace: recovery.ArticleDeletionNamespaceV1,
				EvidenceDigest: s49Hash, EvidenceRef: "evidence-01",
				CompletedAt: changed.Format("2006-01-02T15:04:05.000000Z"),
			},
			ArticleEvidenceCompletedAt: changed, ChangedAt: changed,
			ArticlePublication: s49LegacyPublication(run, "evidence-01", changed),
		})
	if err != nil {
		t.Fatalf("Transition() error = %v", err)
	}
	if loaded.Outcome != recovery.RecoveryOutcomeArticleDeletionReady {
		t.Fatalf("outcome = %q", loaded.Outcome)
	}
	changed = changed.Add(time.Minute)
	if _, err := store.Transition(ctx, run.RunID, recovery.RecoveryOutcomeArticleDeletionReady,
		recovery.RecoveryOutcomeReadyToConsume, recovery.RecoveryRunChange{ChangedAt: changed},
	); err != nil {
		t.Fatalf("article_deletion_ready -> ready_to_consume: %v", err)
	}
	consumedAt := changed.Add(time.Minute)
	if _, err := store.Transition(ctx, run.RunID, recovery.RecoveryOutcomeReadyToConsume,
		recovery.RecoveryOutcomeJournalConsumed, recovery.RecoveryRunChange{
			JournalConsumedAt: &consumedAt, ChangedAt: consumedAt,
		},
	); err != nil {
		t.Fatalf("ready_to_consume -> journal_consumed: %v", err)
	}
	completedAt := consumedAt.Add(time.Minute)
	if _, err := store.Transition(ctx, run.RunID, recovery.RecoveryOutcomeJournalConsumed,
		recovery.RecoveryOutcomeSucceeded, recovery.RecoveryRunChange{ChangedAt: completedAt},
	); err != nil {
		t.Fatalf("journal_consumed -> succeeded: %v", err)
	}
	verifyS49LegacyRunTimes(t, ctx, store, run.RunID)
}

func verifyS49LegacyRunTimes(t *testing.T, ctx context.Context, store *RecoveryRunStore, runID string) {
	t.Helper()
	loaded, err := store.Load(ctx, runID)
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]time.Time{
		"started_at":                    loaded.StartedAt,
		"journal_consumed_at":           *loaded.JournalConsumedAt,
		"article_evidence_completed_at": loaded.ArticleEvidenceCompletedAt,
		"article_replay_verified_at":    loaded.ArticleReplayVerifiedAt,
		"completed_at":                  loaded.CompletedAt,
	} {
		assertUTCMicrosecond(t, name, value)
	}
}

func TestRecoveryRunStoreFileRecoveryJournalVerificationPOSIX(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("FileRecoveryJournal is unsupported on Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	store, _ := newMigratedRecoveryRunStore(t, ctx)
	now := time.Date(2026, 10, 22, 4, 0, 0, 123456000, time.UTC)
	keyring, err := recovery.NewKeyring([]recovery.Key{{
		ID: "recovery", Version: "v1", State: recovery.KeyStateActive,
		Material: []byte("synthetic-integration-key"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	anchor := newStoreTestAnchor()
	journal, err := recovery.NewFileRecoveryJournal(ctx, recovery.JournalOptions{ReadMaxBytes: 8 << 20, ReadMaxRecords: 100000,
		Path:     filepath.Join(t.TempDir(), "recovery.journal"),
		FileMode: 0o600, Now: func() time.Time { return now },
		JournalID: "identity-store-integration", Epoch: 1,
		Anchor: anchor, Bootstrap: true,
	}, keyring)
	if err != nil {
		t.Fatalf("NewFileRecoveryJournal() error = %v", err)
	}
	begun, err := journal.Begin(ctx,
		"run-file-journal", "backup-file-journal", []byte("synthetic-integration-secret"),
	)
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}
	run := recovery.RecoveryRun{
		RunID: begun.RunID, BackupRef: begun.BackupRef,
		JournalKeyID: begun.KeyID, JournalKeyVersion: begun.KeyVersion,
		JournalSecretFingerprint: begun.SecretFingerprint,
		JournalStage:             begun.Stage, StartedAt: begun.RecordedAt,
		Outcome: recovery.RecoveryOutcomeRunning,
	}
	if err := store.Create(ctx, run); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	loaded, err := store.Load(ctx, run.RunID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if _, err := journal.VerifyAgainst(ctx, loaded); err != nil {
		t.Fatalf("VerifyAgainst(Store.Load()) error = %v", err)
	}
}

func TestRecoveryRunStoreRejectsDirectIdentityReadyDuplicateAndTriggerBypass(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	store, pool := newMigratedRecoveryRunStore(t, ctx)
	started := time.Date(2026, 10, 22, 5, 0, 0, 0, time.UTC)
	run := recovery.RecoveryRun{
		RunID: "run-direct", BackupRef: "backup-direct", JournalKeyID: "recovery",
		JournalKeyVersion: "v1", JournalSecretFingerprint: "hmac-sha256:synthetic",
		JournalStage: recovery.StageBegun, StartedAt: started,
		Outcome: recovery.RecoveryOutcomeIdentityReady,
	}
	if err := store.Create(ctx, run); !errors.Is(err, recovery.ErrInvalidRecoveryRun) {
		t.Fatalf("Create(identity_ready) error = %v", err)
	}
	run.RunID = "run-duplicate"
	run.Outcome = recovery.RecoveryOutcomeRunning
	if err := store.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, run); !errors.Is(err, recovery.ErrRunAlreadyExists) {
		t.Fatalf("Create(duplicate) error = %v", err)
	}
	_, err := pool.Exec(ctx, insertRecoveryRunSQL,
		"run-trigger", "backup-trigger", "recovery", "v1",
		"hmac-sha256:synthetic", recovery.StageBegun, started,
		recovery.RecoveryOutcomeIdentityReady,
	)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55000" {
		t.Fatalf("direct INSERT error = %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE recovery_runs SET outcome = 'article_deletion_ready' WHERE run_id = $1`,
		run.RunID,
	); !errors.As(err, &pgErr) || pgErr.Code != "55000" {
		t.Fatalf("trigger transition error = %v", err)
	}
}

func TestRecoveryRunStoreConcurrentTransitionCAS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	store, pool := newMigratedRecoveryRunStore(t, ctx)
	started := time.Date(2026, 10, 22, 6, 0, 0, 0, time.UTC)
	run := recovery.RecoveryRun{
		RunID: "run-cas", BackupRef: "backup-cas", JournalKeyID: "recovery",
		JournalKeyVersion: "v1", JournalSecretFingerprint: "hmac-sha256:synthetic",
		JournalStage: recovery.StageBegun, StartedAt: started,
		Outcome: recovery.RecoveryOutcomeRunning,
	}
	if err := store.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	seedS49LegacyTrustProof(t, ctx, pool, run.RunID, started)
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, err := store.Transition(
				ctx, run.RunID, recovery.RecoveryOutcomeRunning,
				recovery.RecoveryOutcomeIdentityReady,
				recovery.RecoveryRunChange{ChangedAt: started.Add(time.Minute)},
			)
			results <- err
		}()
	}
	close(start)
	workers.Wait()
	close(results)
	var successes, conflicts int
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, recovery.ErrRecoveryRunConflict):
			conflicts++
		default:
			t.Fatalf("Transition() error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes = %d, conflicts = %d", successes, conflicts)
	}
}

func TestRecoveryRunStoreScanRejectsNonCanonicalTimes(t *testing.T) {
	tests := []struct {
		name      string
		startedAt time.Time
	}{
		{
			name: "non UTC location",
			startedAt: time.Date(
				2026, 10, 22, 7, 0, 0, 123456000,
				time.FixedZone("non-utc", 3600),
			),
		},
		{
			name:      "sub microsecond precision",
			startedAt: time.Date(2026, 10, 22, 7, 0, 0, 123456789, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := scanRecoveryRun(rowScannerFunc(func(dest ...any) error {
				*(dest[0].(*string)) = "run-noncanonical"
				*(dest[1].(*string)) = "backup-noncanonical"
				*(dest[2].(*string)) = "recovery"
				*(dest[3].(*string)) = "v1"
				*(dest[4].(*string)) = "hmac-sha256:synthetic"
				*(dest[5].(*recovery.Stage)) = recovery.StageBegun
				*(dest[13].(*time.Time)) = test.startedAt
				*(dest[15].(*recovery.RecoveryOutcome)) = recovery.RecoveryOutcomeRunning
				return nil
			}))
			if !errors.Is(err, recovery.ErrInvalidRecoveryRun) {
				t.Fatalf("scanRecoveryRun() error = %v", err)
			}
		})
	}
}

func TestRecoveryRunStoreCreateErrorMappingRequiresPrimaryKeyConstraint(t *testing.T) {
	primary := &pgconn.PgError{
		Code: "23505", ConstraintName: "recovery_runs_pkey",
	}
	if err := mapRecoveryRunCreateError(primary); !errors.Is(
		err, recovery.ErrRunAlreadyExists,
	) {
		t.Fatalf("mapRecoveryRunCreateError(primary) = %v", err)
	}
	other := &pgconn.PgError{
		Code: "23505", ConstraintName: "other_unique_constraint",
	}
	err := mapRecoveryRunCreateError(other)
	if errors.Is(err, recovery.ErrRunAlreadyExists) {
		t.Fatalf("mapRecoveryRunCreateError(other) = %v", err)
	}
	var preserved *pgconn.PgError
	if !errors.As(err, &preserved) || preserved != other {
		t.Fatalf("mapRecoveryRunCreateError(other) lost root cause: %v", err)
	}
}

type rowScannerFunc func(...any) error

func (scan rowScannerFunc) Scan(dest ...any) error { return scan(dest...) }

func newMigratedRecoveryRunStore(
	t *testing.T,
	ctx context.Context,
) (*RecoveryRunStore, *pgxpool.Pool) {
	t.Helper()
	h := s49PGHarness(t, 10)
	h.install()
	return h.store, h.pool
}

func seedS49LegacyTrustProof(t *testing.T, ctx context.Context, pool *pgxpool.Pool, runID string, at time.Time) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO users(id,username,role,status,credential_mode,is_initial_admin,created_at,updated_at)
	 VALUES($1,'synthetic-initial','admin','active','recovery_bootstrap',true,$2,$2)`, s49User, at)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO audit_events(id,occurred_at,actor_kind,actor_system,action,target_type,target_id,outcome,trace_id)
	 VALUES($1,$2,'system','synthetic','recovery','run',$3,'succeeded','synthetic-trace')`, s49Audit, at, runID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `UPDATE recovery_runs SET initial_admin_user_id=$2,initial_admin_bootstrap_generation=2,
	 initial_admin_changed_generation=3,initial_admin_change_audit_id=$3,initial_admin_changed_at=$4 WHERE run_id=$1`, runID, s49User, s49Audit, at)
	if err != nil {
		t.Fatal(err)
	}
}

func s49LegacyPublication(run recovery.RecoveryRun, ref string, at time.Time) *recovery.PublicationBinding {
	return &recovery.PublicationBinding{Version: "article-replay-publication/v1", RunID: run.RunID, BackupRef: run.BackupRef,
		Ref: "publication-legacy-test", Digest: s49Hash, EvidenceRef: ref, EvidenceDigest: s49Hash, JournalID: "article", Epoch: 1, EvidenceCompletedAt: at, PublishedAt: at}
}

func assertUTCMicrosecond(t *testing.T, name string, value time.Time) {
	t.Helper()
	if value.Location() != time.UTC || value.Nanosecond()%1000 != 0 {
		t.Fatalf("%s = %s (%s)", name, value, value.Location())
	}
}

type storeTestAnchor struct {
	mu          sync.Mutex
	snapshot    recovery.AnchorSnapshot
	revision    string
	missing     bool
	revisionSeq int
}

func newStoreTestAnchor() *storeTestAnchor { return &storeTestAnchor{missing: true} }

func (anchor *storeTestAnchor) Load(
	context.Context,
	string,
) (recovery.AnchorSnapshot, string, error) {
	anchor.mu.Lock()
	defer anchor.mu.Unlock()
	if anchor.missing {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAnchorMissing
	}
	return anchor.snapshot, anchor.revision, nil
}

func (anchor *storeTestAnchor) CompareAndSwap(
	_ context.Context,
	_ string,
	expected string,
	next recovery.AnchorSnapshot,
) (recovery.AnchorSnapshot, string, error) {
	anchor.mu.Lock()
	defer anchor.mu.Unlock()
	if (!anchor.missing && expected != anchor.revision) ||
		(anchor.missing && expected != "") {
		return recovery.AnchorSnapshot{}, "", recovery.ErrAnchorConflict
	}
	anchor.missing = false
	anchor.revisionSeq++
	anchor.revision = string(rune('a' + anchor.revisionSeq))
	anchor.snapshot = next
	return next, anchor.revision, nil
}

func newRecoveryRunTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	dsn := testenv.RequirePostgresDSN(t)
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("open PostgreSQL recovery test database")
	}
	t.Cleanup(admin.Close)
	requirePostgres18(t, ctx, admin)
	schema := "recovery_" + randomHex(t, 8)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+quoted+" CASCADE"); err != nil {
			t.Errorf("drop schema: %v", err)
		}
	})
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("configure PostgreSQL recovery test database")
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open schema-scoped recovery test database")
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal("connect schema-scoped recovery test database")
	}
	return pool
}
