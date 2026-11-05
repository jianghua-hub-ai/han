//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"han/internal/identity"
	"han/internal/migration"
	"han/internal/recovery"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type s414PG struct {
	ctx    context.Context
	config *pgxpool.Config
	pool   *pgxpool.Pool
	store  *identity.RecoveryRunStore
	schema string
}

func s414PGHarness(t *testing.T) *s414PG {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)
	config, err := parseDatabasePoolConfig(testenv.RequirePostgresDSN(t), func(string) string { return "" })
	if err != nil || config.ConnConfig.Host != "127.0.0.1" || config.ConnConfig.Port != 51934 || config.ConnConfig.Database != "han_b13d_restore" {
		t.Fatal("explicit authorized RestorePG required")
	}
	nonce := make([]byte, 12)
	if _, err = rand.Read(nonce); err != nil {
		t.Fatal("schema nonce")
	}
	schema := "han_s414_" + hex.EncodeToString(nonce)
	config.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	config.MaxConns = 3
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open RestorePG")
	}
	t.Cleanup(pool.Close)
	if _, err = pool.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal("create retained schema")
	}
	h := &s414PG{ctx: ctx, config: config, pool: pool, schema: schema}
	h.migrate(t)
	h.store, err = identity.NewRecoveryRunStore(pool)
	if err != nil {
		t.Fatal("Store constructor")
	}
	t.Logf("retained_schema=%s", schema)
	return h
}

func (h *s414PG) migrate(t *testing.T) {
	t.Helper()
	// Explicit allowlist, no glob and no legacy cleanup harness.
	names := []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql", "0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql", "0009_article_replay_attempt.sql", "0010_recovery_runtime_metadata.sql"}
	files := fstest.MapFS{}
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatal("read approved migration")
		}
		files["approved/"+name] = &fstest.MapFile{Data: raw}
	}
	runner, err := migration.NewRunner(h.pool, files, "approved")
	if err != nil {
		t.Fatal("migration constructor")
	}
	if err = runner.Run(h.ctx); err != nil {
		t.Fatal("approved migration failed")
	}
}

func (h *s414PG) seed(t *testing.T, f *s414Fixture) {
	t.Helper()
	f.driver.dsn = testenv.RequirePostgresDSN(t)
	f.world = newS414World(f.source.plan)
	f.world.actualStore = h.store
	i := f.world.installation
	_, err := h.pool.Exec(h.ctx, `INSERT INTO recovery_installation(singleton,metadata_version,deployment_id,installation_id,target_digest,commitment_ref,commitment_digest,recovery_journal_id,article_journal_id,recovery_epoch,article_epoch,recovery_initial_sequence,article_initial_sequence,recovery_initial_hash,article_initial_hash,revision,installed_at) VALUES(true,$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,0,0,'','',1,$11)`, i.Version, i.DeploymentID, i.InstallationID, i.TargetDigest, i.CommitmentRef, i.CommitmentDigest, i.RecoveryInitial.JournalID, i.ArticleInitial.JournalID, int64(i.RecoveryInitial.Epoch), int64(i.ArticleInitial.Epoch), i.InstalledAt)
	if err != nil {
		t.Fatal("seed approved target installation")
	}
	// Software-only independent source witness, established before tool execution.
	_, err = h.pool.Exec(h.ctx, `CREATE TABLE s414_source_witness(singleton boolean PRIMARY KEY,source_installation text NOT NULL,schema_digest text NOT NULL,artifact_digest text NOT NULL,receipt bytea NOT NULL)`)
	if err != nil {
		t.Fatal("create software witness")
	}
	p := f.source.plan
	_, err = h.pool.Exec(h.ctx, `INSERT INTO s414_source_witness VALUES(true,$1,$2,$3,$4)`, p.SourceInstallationID, p.SourceSchemaDigest, p.ArtifactDigest, s414ReceiptJSON(p))
	if err != nil {
		t.Fatal("seed independent software witness")
	}
	f.target.read = h.readRestored
}

func (h *s414PG) readRestored(ctx context.Context, p recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	// A new pool/physical connection, independent from Runtime's Store and child.
	pool, err := pgxpool.NewWithConfig(ctx, h.config.Copy())
	if err != nil {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionIncomplete
	}
	defer pool.Close()
	store, err := identity.NewRecoveryRunStore(pool)
	if err != nil {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionIncomplete
	}
	snapshot, err := store.ReadAdmissionSnapshot(ctx)
	if err != nil {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionIncomplete
	}
	i := snapshot.Installation
	if i.DeploymentID != p.DeploymentID || i.InstallationID != p.InstallationID || i.TargetDigest != p.TargetDigest {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionBinding
	}
	var source, schema, artifact string
	var raw []byte
	if pool.QueryRow(ctx, `SELECT source_installation,schema_digest,artifact_digest,receipt FROM s414_source_witness WHERE singleton`).Scan(&source, &schema, &artifact, &raw) != nil {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionIncomplete
	}
	defer clear(raw)
	if source != p.SourceInstallationID || schema != p.SourceSchemaDigest || artifact != p.ArtifactDigest {
		return recovery.RestoreReceipt{}, recovery.ErrAdmissionBinding
	}
	return decodeRestoreReceipt(raw, p)
}

func TestRecoveryRestoreTargetReadbackPostgres(t *testing.T) {
	t.Run("StableArtifactHandleAndApprovedTarget", func(t *testing.T) {
		h := s414PGHarness(t)
		f := newS414Fixture(t, "backup")
		h.seed(t, f)
		r := f.runtime(t)
		if err := s414Enter(t, f, r); !errors.Is(err, s414Boundary) {
			t.Fatal("driver/real Store did not reach bind boundary", err)
		}
		if f.target.calls != 2 || len(f.observations) != 2 || f.uses[0] != f.uses[1] {
			t.Fatal("independent target/verify/handle ledger")
		}
		for _, o := range f.observations {
			t.Logf("child_pid=%d waited=%t pipes_joined=%t", o.PID, o.Waited, o.PipesJoined)
		}
	})
	t.Run("IndependentTargetAndNativeReceiptReadback", testS414PGWitness)
	t.Run("BeginAckAndRestoreUnknownNoReplay", testS414PGUnknown)
	t.Run("ExclusiveAndRealChildJoin", testS414PGTransactionJoin)
}

func testS414PGWitness(t *testing.T) {
	for _, field := range []string{"source_installation", "schema_digest", "artifact_digest", "receipt"} {
		t.Run(field, func(t *testing.T) {
			h := s414PGHarness(t)
			f := newS414Fixture(t, "backup")
			h.seed(t, f)
			sql := "UPDATE s414_source_witness SET " + pgx.Identifier{field}.Sanitize() + "=$1"
			var value any = "wrong-source"
			if field == "receipt" {
				value = []byte("null")
			}
			if _, err := h.pool.Exec(h.ctx, sql, value); err != nil {
				t.Fatal("mutate fixture witness")
			}
			r := f.runtime(t)
			if err := s414Enter(t, f, r); !errors.Is(err, recovery.ErrRecoveryWriteUnknown) {
				t.Fatal("exit0 accepted missing source", err)
			}
			if f.world.binds != 0 {
				t.Fatal("invalid source bound")
			}
		})
	}
}

func testS414PGUnknown(t *testing.T) {
	h := s414PGHarness(t)
	f := newS414Fixture(t, "backup")
	h.seed(t, f)
	r := f.runtime(t)
	if err := s414Enter(t, f, r); !errors.Is(err, s414Boundary) {
		t.Fatal(err)
	}
	if err := r.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.driver.Close()
	if _, err := h.pool.Exec(h.ctx, "UPDATE s414_source_witness SET receipt=convert_to('null','UTF8') WHERE singleton"); err != nil {
		t.Fatal("simulate absent native witness")
	}
	next := newS414Fixture(t, "backup")
	next.source.plan = f.source.plan
	next.world = f.world
	next.world.lease = &s414Lease{}
	next.world.binds = 0
	next.target.read = h.readRestored
	rr := next.runtime(t)
	if err := rr.Resume(s414Context(t), recovery.RunBinding{RunID: f.source.plan.RunID, BackupRef: f.source.plan.BackupRef}); err == nil {
		t.Fatal("unknown resume accepted")
	}
	snapshot, err := h.store.ReadAdmissionSnapshot(h.ctx)
	if err != nil || len(snapshot.Runs) != 0 || next.world.begins != 1 || next.world.binds != 0 || len(next.observations) != 1 {
		t.Fatal("unknown replayed Replace/Begin/bind")
	}
}

func testS414PGTransactionJoin(t *testing.T) {
	h := s414PGHarness(t)
	f := newS414Fixture(t, "backup")
	h.seed(t, f)
	entered, release := make(chan struct{}), make(chan struct{})
	cancelled := make(chan struct{})
	f.target.read = s414HeldTarget(h, entered, release, cancelled)
	r := f.runtime(t)
	action := make(chan error, 1)
	go func() { action <- s414Enter(t, f, r) }()
	select {
	case <-entered:
	case e := <-action:
		close(release)
		t.Fatal(e)
	case <-time.After(30 * time.Second):
		close(release)
		t.Fatal("target read did not start")
	}
	stop := make(chan error, 1)
	go func() { stop <- r.Stop(context.Background()) }()
	<-cancelled
	if f.world.lease.closes.Load() != 0 {
		t.Fatal("lock returned with active target tx")
	}
	select {
	case <-action:
		t.Fatal("target tx not joined")
	default:
	}
	select {
	case <-stop:
		t.Fatal("Stop abandoned target transaction")
	default:
	}
	close(release)
	if err := <-action; err == nil {
		t.Fatal("cancelled target became proof")
	}
	if err := <-stop; err != nil {
		t.Fatal(err)
	}
	if h.pool.Stat().AcquiredConns() != 0 || f.world.lease.closes.Load() != 1 {
		t.Fatal("target connection/lease not returned")
	}
}

func s414HeldTarget(h *s414PG, entered, release, cancelled chan struct{}) func(context.Context, recovery.RestorePlan) (recovery.RestoreReceipt, error) {
	return func(ctx context.Context, p recovery.RestorePlan) (recovery.RestoreReceipt, error) {
		tx, err := h.pool.BeginTx(h.ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
		if err != nil {
			return recovery.RestoreReceipt{}, recovery.ErrAdmissionIncomplete
		}
		var one int
		if err = tx.QueryRow(h.ctx, "SELECT 1").Scan(&one); err != nil {
			tx.Rollback(h.ctx)
			return recovery.RestoreReceipt{}, recovery.ErrAdmissionIncomplete
		}
		close(entered)
		<-ctx.Done()
		close(cancelled)
		<-release
		err = tx.Rollback(h.ctx)
		return recovery.RestoreReceipt{}, errors.Join(err, ctx.Err())
	}
}
