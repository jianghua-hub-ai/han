//go:build integration

package article

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/recovery"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Every case retains its schema, synthetic media, inventory, checkpoint and SQL images.
func newReplayPublicationPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("HAN_TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Fatal("explicit isolated Article DSN is required; no fallback")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("parse explicit isolated Article DSN")
	}
	if cfg.ConnConfig.Host != "127.0.0.1" {
		t.Fatal("Article publication PG must be loopback-only")
	}
	expectedHost, expectedPort, expectedDB, expectedUser := cfg.ConnConfig.Host, cfg.ConnConfig.Port, cfg.ConnConfig.Database, cfg.ConnConfig.User
	verify := replayPublicationConnectionVerifier(t, expectedHost, expectedPort, expectedDB, expectedUser)
	cfg.AfterConnect = verify
	admin, err := pgxpool.NewWithConfig(ctx, cfg.Copy())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "han_b13d_a48_" + randomIntegrationHex(t, 8)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = quoted
	cfg.MaxConns = 8
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		if err := verify(ctx, c); err != nil {
			return err
		}
		_, err := c.Exec(ctx, "SET search_path = "+quoted)
		return err
	}
	cfg.BeforeAcquire = func(ctx context.Context, c *pgx.Conn) bool {
		_, err := c.Exec(ctx, "SET search_path = "+quoted)
		return err == nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	applyReplayPublicationMigrations(t, ctx, pool, schema)
	return pool
}

type publicationPGTransactions struct {
	pool       *pgxpool.Pool
	mu         sync.Mutex
	ackError   error
	ackOnce    bool
	events     []string
	readHook   func()
	statements atomic.Int64
}
type publicationPGTx struct {
	pgx.Tx
	owner       *publicationPGTransactions
	publication bool
	readonly    bool
}

func (p *publicationPGTransactions) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	tx, err := p.pool.BeginTx(ctx, o)
	if err != nil {
		return nil, err
	}
	var schema string
	if err := tx.QueryRow(ctx, "SELECT current_schema()").Scan(&schema); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path = "+pgx.Identifier{schema}.Sanitize()); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	return &publicationPGTx{Tx: tx, owner: p, readonly: o.AccessMode == pgx.ReadOnly}, nil
}
func (p *publicationPGTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	p.owner.statements.Add(1)
	if strings.Contains(sql, "INSERT INTO article_replay_publications") {
		p.publication = true
	}
	return p.Tx.Exec(ctx, sql, args...)
}
func (p *publicationPGTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if p.readonly {
		p.owner.mu.Lock()
		if strings.Contains(sql, "article_replay_publications") {
			p.owner.events = append(p.owner.events, "ReadPublication")
		}
		if strings.Contains(sql, "deletion_replay_runs") {
			p.owner.events = append(p.owner.events, "ReadCurrent")
		}
		if strings.Contains(sql, "deletion_replay_evidence") {
			p.owner.events = append(p.owner.events, "ReadEvidence")
		}
		hook := p.owner.readHook
		p.owner.mu.Unlock()
		if hook != nil && strings.Contains(sql, "article_replay_publications") {
			hook()
		}
	}
	return p.Tx.QueryRow(ctx, sql, args...)
}
func (p *publicationPGTx) Commit(ctx context.Context) error {
	err := p.Tx.Commit(ctx)
	if err != nil || !p.publication || p.readonly {
		return err
	}
	p.owner.mu.Lock()
	defer p.owner.mu.Unlock()
	p.owner.events = append(p.owner.events, "ActualCommitNil")
	if p.owner.ackError != nil && !p.owner.ackOnce {
		p.owner.ackOnce = true
		p.owner.events = append(p.owner.events, "SyntheticACKError")
		return p.owner.ackError
	}
	return nil
}

type publicationPGTransactor struct{ pool *pgxpool.Pool }

func (p publicationPGTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	return (&publicationPGTransactions{pool: p.pool}).BeginTx(ctx, pgx.TxOptions{})
}

type publicationPhysicalReplay struct {
	actual    *DeletionFinalizer
	execution sync.Mutex
	calls     atomic.Int64
	mu        sync.Mutex
	result    DeletionReplayAllResult
	trace     string
}

func (p *publicationPhysicalReplay) ReplayAllTombstones(ctx context.Context, records []ArticleDeletionTombstone, trace string) (DeletionReplayAllResult, error) {
	// Explicit synthetic external worker lease; the Finalizer and every SQL outcome are real.
	p.execution.Lock()
	defer p.execution.Unlock()
	p.calls.Add(1)
	current, present := platform.TraceIDFromContext(ctx)
	if !present || current != trace {
		return DeletionReplayAllResult{}, errors.New("replay Trace context was lost")
	}
	r, err := p.actual.ReplayAllTombstones(ctx, records, trace)
	p.mu.Lock()
	p.result = r
	p.trace = trace
	p.mu.Unlock()
	return r, err
}

type publicationPGFixture struct {
	t                 *testing.T
	ctx               context.Context
	pool              *pgxpool.Pool
	observer          *pgxpool.Pool
	store             *PostgresReplayEvidenceStore
	provider          *DeletionReplayProvider
	source            *ProductionDeletionReplaySource
	phase             *publicationReplayPhase
	txs               *publicationPGTransactions
	physical          *publicationPhysicalReplay
	facts             *publicationFactGate
	ids               *publicationIDs
	clock             *publicationClock
	run, backup, path string
}

func newPublicationPGFixture(t *testing.T) *publicationPGFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	return newPublicationPGFixtureOnPool(t, newReplayPublicationPool(t, ctx), ctx)
}
func newPublicationPGFixtureOnPool(t *testing.T, pool *pgxpool.Pool, parent context.Context, presence ...int) *publicationPGFixture {
	t.Helper()
	f := &publicationPGFixture{t: t, pool: pool, run: "synthetic-run", backup: "backup-integration", ids: &publicationIDs{}, clock: &publicationClock{}}
	observer, err := pgxpool.NewWithConfig(parent, pool.Config().Copy())
	pubMust(t, err)
	f.observer = observer
	t.Cleanup(observer.Close)
	f.phase = startPublicationReplay(t, parent, f.run, f.backup)
	f.ctx = f.phase.ctx
	present := 2
	if len(presence) == 1 {
		present = presence[0]
	}
	pubMust(t, publicationSeedTargets(f.ctx, pool, present))
	dir, err := os.MkdirTemp("", "han-b13d-a48-media-")
	pubMust(t, err)
	f.path = filepath.Join(dir, "inventory.json")
	writeInventory(t, f.path, inventoryManifestWithCutoff(t, f.backup, "2026-10-20T00:00:00.000000Z"))
	reader, err := NewFileBackupInventoryReader(f.path)
	pubMust(t, err)
	facts, err := NewPostgresReplayFactReader(pool)
	pubMust(t, err)
	f.facts = &publicationFactGate{actual: facts}
	finalizer, err := NewDeletionFinalizer(DeletionFinalizerDependencies{Transactions: publicationPGTransactor{pool}, Repository: Repository{},
		Journal: f.phase.journal, Auditor: audit.Store{}, AuditIDs: &publicationIDs{}, Clock: integrationClock{}, WorkerID: "synthetic-a48-replay", LeaseDuration: time.Minute, Identity: pgA47Identity{}})
	pubMust(t, err)
	f.physical = &publicationPhysicalReplay{actual: finalizer}
	f.source, err = NewProductionDeletionReplaySource(f.phase.journal, reader, f.physical, f.facts, f.facts, integrationClock{}.Now)
	pubMust(t, err)
	f.store, err = NewPostgresReplayEvidenceStore(pool)
	pubMust(t, err)
	f.txs = &publicationPGTransactions{pool: pool}
	f.store.publicationTransactions = f.txs
	_, err = NewReplayPublicationPorts(f.store, ReplayPublicationDependencies{Source: f.source, Runtime: f.phase.runtime, InstallationID: "synthetic-installation", PublicationIDs: f.ids, Clock: f.clock})
	pubMust(t, err)
	f.provider, err = NewDeletionReplayProvider(DeletionReplayDependencies{Store: f.store, Source: f.source, EvidenceIDs: &publicationIDs{}, Tokens: &publicationIDs{}, Clock: integrationClock{}})
	pubMust(t, err)
	return f
}

func pubMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func publicationSeedTargets(ctx context.Context, pool *pgxpool.Pool, present int) error {
	tx, err := (publicationPGTransactor{pool}).Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(ctx, `INSERT INTO users(id,username,email,phone,permanent_password_digest,role,status,credential_mode,created_at,updated_at)
 VALUES($1,'a48-author','a48@example.invalid','+15550000000','synthetic-digest','member','active','permanent',$2,$2)`, deletionAuthorID, anchorRequest(1).DeletedAt.Add(-time.Hour))
	if err != nil {
		return err
	}
	for kind, value := range map[string]string{"username": "a48-author", "email": "a48@example.invalid", "phone": "+15550000000"} {
		if _, err := tx.Exec(ctx, `INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at) VALUES($1,$2,$3,'active',$4)`, deletionAuthorID, kind, value, anchorRequest(1).DeletedAt.Add(-time.Hour)); err != nil {
			return err
		}
	}
	author := deletionAuthorID
	for n := 1; n <= 3; n++ {
		r := anchorRequest(n)
		_, err = tx.Exec(ctx, `INSERT INTO deletion_operations(id,article_id,article_version,delete_kind,actor_user_id,actor_role,status,tombstone_sequence,journal_key_id,journal_key_version,started_at)
 VALUES($1,$2,$3,'ordinary',$4,'member','journaled',$5,'synthetic-key','v1',$6)`, r.DeletionOperationID, r.ArticleID, r.ArticleVersion, deletionAuthorID, n, r.DeletedAt)
		if err != nil {
			return err
		}
		if n <= present {
			row := Article{ID: r.ArticleID, AuthorID: &author, Title: "synthetic publication", Body: "retained synthetic article", PublicationState: PublicationPublished, ViolationState: ViolationClear,
				DeletionState: DeletionPending, DeletionOperationID: &r.DeletionOperationID, Version: r.ArticleVersion + 1, CreatedAt: r.DeletedAt.Add(-time.Hour), UpdatedAt: r.DeletedAt}
			if err := (Repository{}).InsertArticle(ctx, tx, row); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func (f *publicationPGFixture) replay() recovery.ReadyReplayStatusV1 {
	f.t.Helper()
	raw, err := f.provider.ReplayAndVerify(f.ctx, f.run, f.backup)
	pubMust(f.t, err)
	decoded, err := recovery.DecodeReplayStatusV1(raw)
	pubMust(f.t, err)
	if decoded.Ready == nil {
		f.t.Fatalf("real replay unavailable: %s", raw)
	}
	return *decoded.Ready
}
func (f *publicationPGFixture) binding(ready recovery.ReadyReplayStatusV1) recovery.PublicationBinding {
	f.t.Helper()
	binding, err := f.store.ReadCurrentPublication(f.ctx, f.run, f.backup, ready.EvidenceRef, ready.EvidenceDigest)
	pubMust(f.t, err)
	return binding
}
func publicationRequest(b recovery.PublicationBinding) recovery.ClosureRequest {
	return recovery.ClosureRequest{RunID: b.RunID, BackupRef: b.BackupRef, EvidenceRef: b.EvidenceRef, EvidenceDigest: b.EvidenceDigest, PublicationRef: b.Ref, PublicationDigest: b.Digest}
}
func publicationCount(t *testing.T, pool *pgxpool.Pool, table string) int {
	t.Helper()
	if table != "articles" && table != "deletion_replay_evidence" && table != "article_replay_publications" && table != "audit_events" {
		t.Fatal("count table outside fixed allowlist")
	}
	var count int
	pubMust(t, pool.QueryRow(context.Background(), "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()).Scan(&count))
	return count
}

func TestReplayPublicationPostgres(t *testing.T) {
	t.Run("PublicationCurrentEvidenceAtomic", publicationPGAtomic)
	t.Run("CompositeFKImmutableCanonical", publicationPGConstraints)
	t.Run("ExactHistoricalClosureAllowsRequiredReads", publicationPGHistory)
	t.Run("HeadAdvanceAndReadDrift", publicationPGAdvance)
	t.Run("LegacyReassertionVersusTrustedImport", publicationPGLegacy)
}

func publicationPGAtomic(t *testing.T) {
	for _, present := range []int{0, 3} {
		t.Run(fmt.Sprintf("CountsAndReadback/Present%dAbsent%d", present, 3-present), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			t.Cleanup(cancel)
			f := newPublicationPGFixtureOnPool(t, newReplayPublicationPool(t, ctx), ctx, present)
			ready := f.replay()
			if ready.TombstonesReplayed != uint64(present) || ready.AlreadyAbsent != uint64(3-present) || publicationCount(t, f.observer, "articles") != 0 {
				t.Fatal("real all-pending/all-absent partition changed")
			}
			_ = f.binding(ready)
		})
	}
	t.Run("PhysicalDeleteFirstCountAndLateWinner", publicationPGFirstCount)
	for _, stage := range []string{"EvidenceInsert", "CurrentUpdate", "PublicationInsert"} {
		t.Run(stage+"RollbackAndSameOwnerRetry", func(t *testing.T) { publicationPGRollback(t, stage) })
	}
	t.Run("ActualCommitThenACKErrorReadsTripleBeforeRetry", publicationPGAck)
	for _, operation := range []string{"Publish", "ReadCurrent"} {
		t.Run("RefreshOutsideRunAndRecheck/"+operation, func(t *testing.T) { publicationPGRefreshRecheck(t, operation) })
	}
	t.Run("InterruptedReplayAndPublish/ActualPartialCommitThenRetry", publicationPGPartialReplay)
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("TraceContext/ExistingTrace_%v", existing), func(t *testing.T) {
			publicationPGTrace(t, existing)
		})
	}
	t.Run("NextGenerationFactsAToBToAReusesEvidenceNotReceipt", publicationPGNextGeneration)
}

type publicationCompletionACK struct {
	pool  *pgxpool.Pool
	fired atomic.Bool
}
type publicationCompletionTx struct {
	pgx.Tx
	owner      *publicationCompletionACK
	completion bool
}

func (p *publicationCompletionACK) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := (publicationPGTransactor{p.pool}).Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &publicationCompletionTx{Tx: tx, owner: p}, nil
}
func (p *publicationCompletionTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if strings.Contains(sql, "UPDATE deletion_operations") && strings.Contains(sql, "status = 'completed'") {
		p.completion = true
	}
	return p.Tx.Exec(ctx, sql, args...)
}
func (p *publicationCompletionTx) Commit(ctx context.Context) error {
	err := p.Tx.Commit(ctx)
	if err == nil && p.completion && p.owner.fired.CompareAndSwap(false, true) {
		return errors.New("synthetic physical completion ACK loss after actual Commit")
	}
	return err
}
func publicationPGPartialReplay(t *testing.T) {
	f := newPublicationPGFixture(t)
	adapter := &publicationCompletionACK{pool: f.pool}
	f.physical.actual.dependencies.Transactions = adapter
	raw, err := f.provider.ReplayAndVerify(f.ctx, f.run, f.backup)
	pubMust(t, err)
	first, err := recovery.DecodeReplayStatusV1(raw)
	pubMust(t, err)
	if first.Ready != nil || !adapter.fired.Load() || publicationCount(t, f.observer, "deletion_replay_evidence") != 0 || publicationCount(t, f.observer, "article_replay_publications") != 0 {
		t.Fatal("partial actual replay bound uncommitted outcome")
	}
	if publicationCount(t, f.observer, "articles") != 1 {
		t.Fatal("first committed physical DELETE did not happen")
	}
	ready := f.replay()
	if ready.TombstonesReplayed != 1 || ready.AlreadyAbsent != 2 || publicationCount(t, f.observer, "articles") != 0 || publicationCount(t, f.observer, "audit_events") != 3 {
		t.Fatal("retry accumulated/fabricated first incomplete outcomes")
	}
	_ = f.binding(ready)
}

func publicationPrepareRequest(t *testing.T, f *publicationPGFixture) (ReplayPublishRequest, *publicationIDs, *replayClockStub) {
	t.Helper()
	input, err := f.source.ReplayOldBackup(f.ctx, f.backup)
	pubMust(t, err)
	fresh, err := f.source.Snapshot(f.ctx, f.backup)
	pubMust(t, err)
	a, err := f.store.Begin(f.ctx, f.run, f.backup, AttemptSnapshot{}, fresh, "retained-synthetic-candidate")
	pubMust(t, err)
	ids := &publicationIDs{}
	clock := &replayClockStub{now: integrationClock{}.Now()}
	return ReplayPublishRequest{RunID: f.run, BackupRef: f.backup, Attempt: a.Attempt, Input: input, Refresh: func(c context.Context) (DeletionReplayFacts, error) { return f.source.Snapshot(c, f.backup) }, EvidenceIDs: ids, Clock: clock}, ids, clock
}

func publicationImage(t *testing.T, pool *pgxpool.Pool) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, table := range []string{"articles", "deletion_operations", "audit_events", "users", "identity_identifiers", "deletion_replay_runs", "deletion_replay_evidence", "article_replay_publications", "recovery_runs"} {
		var image string
		sql := `SELECT COALESCE(jsonb_agg(x ORDER BY x::text),'[]'::jsonb)::text FROM (SELECT jsonb_build_object('row',to_jsonb(r)-ARRAY['permanent_password_digest','temporary_password_digest','attempt_token','journal_secret_fingerprint','body'],'xmin',r.xmin::text,'ctid',r.ctid::text) x FROM ` + pgx.Identifier{table}.Sanitize() + ` r) images`
		pubMust(t, pool.QueryRow(context.Background(), sql).Scan(&image))
		result[table] = image
	}
	return result
}

func publicationPGRollback(t *testing.T, stage string) {
	f := newPublicationPGFixture(t)
	request, ids, clock := publicationPrepareRequest(t, f)
	table, operation := "deletion_replay_evidence", "INSERT"
	if stage == "CurrentUpdate" {
		table, operation = "deletion_replay_runs", "UPDATE"
	}
	if stage == "PublicationInsert" {
		table = "article_replay_publications"
	}
	_, err := f.pool.Exec(f.ctx, `CREATE TABLE a48_fault(enabled boolean NOT NULL);INSERT INTO a48_fault VALUES(true);
 CREATE FUNCTION a48_refuse() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN IF (SELECT enabled FROM a48_fault) THEN RAISE EXCEPTION 'synthetic transaction witness' USING ERRCODE='23514';END IF;RETURN NEW;END;$$;
 CREATE TRIGGER a48_publication_witness BEFORE `+operation+` ON `+pgx.Identifier{table}.Sanitize()+` FOR EACH ROW EXECUTE FUNCTION a48_refuse()`)
	pubMust(t, err)
	before, events := assertPublicationRollback(t, f, request)
	_, err = f.pool.Exec(f.ctx, "UPDATE a48_fault SET enabled=false")
	pubMust(t, err)
	events = append(events, "Publish")
	e, err := f.store.Publish(f.ctx, request)
	pubMust(t, err)
	ready, err := readyReplayStatus(e)
	pubMust(t, err)
	for range 2 {
		events = append(events, "ReadCurrent")
		_, current, err := f.store.ReadCurrent(f.ctx, f.run, f.backup, request.Refresh)
		pubMust(t, err)
		if current == nil || *current != e {
			t.Fatal("same-owner retry current lost first evidence")
		}
	}
	if !reflect.DeepEqual(events, []string{"Publish", "ReadCurrent", "Publish", "ReadCurrent", "ReadCurrent"}) {
		t.Fatal("original current-update event order changed")
	}
	image, _ := json.Marshal(before)
	t.Logf("rollback stage=%s exact retained before image=%s events=%v", stage, image, events)
	_ = f.binding(ready)
	if ids.calls.Load() != 2 || clock.calls != 2 || f.physical.calls.Load() != 1 {
		t.Fatal("same-owner retry lost original metadata/replay oracle")
	}
	_, err = f.store.Publish(f.ctx, request)
	pubMust(t, err)
	if ids.calls.Load() != 2 || clock.calls != 2 || publicationCount(t, f.observer, "deletion_replay_evidence") != 1 || publicationCount(t, f.observer, "article_replay_publications") != 1 {
		t.Fatal("Ready reuse consumed metadata or duplicated rows")
	}
	assertPublicationRunUnlocked(t, f)
}

func publicationPGAck(t *testing.T) {
	f := newPublicationPGFixture(t)
	sentinel := errors.New("synthetic ACK loss after actual Commit nil")
	f.txs.ackError = sentinel
	ready := f.replay()
	if ready.TombstonesReplayed != 2 || ready.AlreadyAbsent != 1 || f.physical.calls.Load() != 1 || f.provider.dependencies.Tokens.(*publicationIDs).calls.Load() != 1 {
		t.Fatal("ACK loss caused replay or another candidate")
	}
	var ref, digest, pubref, pubdigest string
	pubMust(t, f.observer.QueryRow(f.ctx, `SELECT e.evidence_ref,e.evidence_digest,p.publication_ref,p.publication_digest FROM deletion_replay_runs r JOIN deletion_replay_evidence e ON(e.evidence_ref=r.current_evidence_ref AND e.run_id=r.run_id AND e.namespace=r.namespace) JOIN article_replay_publications p ON(p.run_id=r.run_id AND p.namespace=r.namespace AND p.attempt_generation=r.attempt_generation AND p.attempt_token=r.attempt_token AND p.evidence_ref=e.evidence_ref AND p.evidence_digest=e.evidence_digest) WHERE r.run_id=$1 AND r.status='ready'`, f.run).Scan(&ref, &digest, &pubref, &pubdigest))
	if ref != ready.EvidenceRef || digest != ready.EvidenceDigest || pubref == "" || pubdigest == "" {
		t.Fatal("independent observer did not see exact triple")
	}
	f.txs.mu.Lock()
	events := append([]string(nil), f.txs.events...)
	f.txs.mu.Unlock()
	if len(events) < 5 || events[0] != "ActualCommitNil" || events[1] != "SyntheticACKError" || events[2] != "ReadCurrent" || events[3] != "ReadEvidence" || events[4] != "ReadPublication" {
		t.Fatalf("ACK/readback order=%v", events)
	}
	if f.ids.calls.Load() != 1 || f.clock.calls.Load() != 1 {
		t.Fatal("ACK readback created another receipt")
	}
	t.Logf("actual commit/readback events=%v exact independent triple=%s/%s/%s/%s", events, ref, digest, pubref, pubdigest)
}

func publicationPGConstraints(t *testing.T) {
	t.Run("CompositeFKAndImmutableCatalog", publicationPGCatalog)
}

func publicationPGHistory(t *testing.T) {
	t.Run("OrdinaryContextRequiredReadsAndZeroFreshWrites", publicationPGRequiredReads)
	for _, field := range []string{"Run", "Backup", "EvidenceRef", "EvidenceDigest", "PublicationRef", "PublicationDigest"} {
		t.Run("ExactMismatch"+field, func(t *testing.T) {
			f := newPublicationPGFixture(t)
			b := f.binding(f.replay())
			q := publicationRequest(b)
			switch field {
			case "Run":
				q.RunID = "wrong"
			case "Backup":
				q.BackupRef = "wrong"
			case "EvidenceRef":
				q.EvidenceRef = "wrong"
			case "EvidenceDigest":
				q.EvidenceDigest = "wrong"
			case "PublicationRef":
				q.PublicationRef = "wrong"
			case "PublicationDigest":
				q.PublicationDigest = "wrong"
			}
			if _, err := f.store.ReadPublishedClosure(context.Background(), q); err == nil {
				t.Fatal("historical exact mismatch accepted")
			}
		})
	}
	t.Run("ReadRevisionDriftClosed", func(t *testing.T) {
		f := newPublicationPGFixture(t)
		b := f.binding(f.replay())
		a := f.phase.journal.options.Anchor.(*anchorCoreCAS)
		var once sync.Once
		f.txs.readHook = func() { once.Do(func() { a.mu.Lock(); a.revision++; a.mu.Unlock() }) }
		_, err := f.store.ReadPublishedClosure(context.Background(), publicationRequest(b))
		if !errors.Is(err, recovery.ErrAdmissionDrift) {
			t.Fatalf("read drift not closed: %v", err)
		}
	})
}

type publicationNormalWorld struct {
	*anchorReplayWorld
	storeReads, recoveryReads, articleReads atomic.Int64
	consumed                                recovery.JournalEntry
}

func (w *publicationNormalWorld) ReadAdmissionSnapshot(ctx context.Context) (recovery.AdmissionSnapshot, error) {
	w.storeReads.Add(1)
	return w.anchorReplayWorld.ReadAdmissionSnapshot(ctx)
}
func (w *publicationNormalWorld) ReadAdmissionRevision(ctx context.Context) (recovery.StoreRevision, error) {
	s, err := w.ReadAdmissionSnapshot(ctx)
	return s.Revision, err
}
func (w *publicationNormalWorld) ReadVerifiedSnapshot(ctx context.Context, ns string) (recovery.VerifiedJournalSnapshot, error) {
	if ns == ArticleDeletionNamespace {
		w.articleReads.Add(1)
		return w.journal.ReadVerifiedSnapshot(ctx, ns)
	}
	if ns != recovery.RecoveryNamespaceV1 {
		return recovery.VerifiedJournalSnapshot{}, recovery.ErrAdmissionBinding
	}
	w.recoveryReads.Add(1)
	first := recovery.JournalHead{Sequence: 1, Hash: recovery.SHA256Digest([]byte("synthetic-committed-Begin"))}
	last := recovery.JournalHead{Sequence: 2, Hash: recovery.SHA256Digest([]byte("synthetic-committed-Consumed"))}
	return recovery.VerifiedJournalSnapshot{Namespace: ns, JournalID: w.installation.RecoveryInitial.JournalID, Epoch: w.installation.RecoveryInitial.Epoch,
		Head: last, Revision: "synthetic-recovery-consumed-r2", Entries: []recovery.JournalEntry{w.begin, w.consumed}, Prefixes: []recovery.JournalHead{{}, first, last}}, ctx.Err()
}

type publicationNormalFixture struct {
	store    *PostgresReplayEvidenceStore
	provider *DeletionReplayProvider
	runtime  recovery.Runtime
	journal  *anchoredArticleJournal
	world    *publicationNormalWorld
}

// Sequential component rebuild: the prior Runtime/action/handle has fully ended.
// External recovery Store/Journal are explicit synthetic facilities, not S4.13 SQL.
func publicationNormalRebuild(t *testing.T, f *publicationPGFixture, b recovery.PublicationBinding, ready recovery.ReadyReplayStatusV1, missing bool) *publicationNormalFixture {
	t.Helper()
	f.phase.close(t)
	base := &anchorRuntimeWorld{installation: f.phase.world.installation, join: make(chan struct{})}
	w := &publicationNormalWorld{anchorReplayWorld: newAnchorReplayWorld(base)}
	w.begin.RunID, w.begin.BackupRef = f.run, f.backup
	w.row.RunID, w.row.BackupRef = f.run, f.backup
	w.receipt.RunID, w.receipt.BackupRef = f.run, f.backup
	w.row.RestoreBinding = &w.receipt
	completed, err := time.Parse(time.RFC3339Nano, ready.CompletedAt)
	pubMust(t, err)
	verified, consumed, finished := b.PublishedAt.Add(time.Minute), b.PublishedAt.Add(2*time.Minute), b.PublishedAt.Add(3*time.Minute)
	w.row.Outcome, w.row.JournalStage = recovery.RecoveryOutcomeSucceeded, recovery.StageConsumed
	w.row.JournalConsumedAt, w.row.CompletedAt = &consumed, &finished
	w.row.ArticleEvidenceVersion, w.row.ArticleEvidenceRef, w.row.ArticleEvidenceDigest = &ready.EvidenceVersion, &ready.EvidenceRef, &ready.EvidenceDigest
	w.row.ArticleEvidenceCompletedAt, w.row.ArticleReplayVerifiedAt, w.row.ArticleNamespace = &completed, &verified, &ready.Namespace
	if !missing {
		w.row.ArticlePublicationRef, w.row.ArticlePublicationDigest = &b.Ref, &b.Digest
	}
	w.consumed = w.begin
	w.consumed.Sequence = 2
	w.consumed.Stage = recovery.StageConsumed
	w.consumed.RecordedAt = consumed
	w.consumed.RecordHMAC = "hmac-sha256:" + strings.Repeat("c", 64)
	w.consumed.PreviousRecordHash = recovery.SHA256Digest([]byte("synthetic-committed-Begin"))
	store, err := NewPostgresReplayEvidenceStore(f.pool)
	pubMust(t, err)
	r, err := recovery.NewRuntime(context.Background(), recovery.RuntimeDependencies{Store: w, Journal: w, ArticleSnapshots: w, Closures: store, Genesis: w, Exclusive: w, Lifecycle: w})
	pubMust(t, err)
	w.runtime = r
	o := f.phase.journal.options
	o.Runtime = r
	keys := DeletionJournalKeyring{Active: DeletionJournalKey{ID: "synthetic-key", Version: "v1", Material: []byte("synthetic-article-core-key-32bytes")}}
	j, err := newAnchoredDeletionJournal(context.Background(), o, keys, f.phase.journal.media)
	pubMust(t, err)
	w.journal = j
	reader, err := NewFileBackupInventoryReader(f.path)
	pubMust(t, err)
	facts, err := NewPostgresReplayFactReader(f.pool)
	pubMust(t, err)
	source, err := NewProductionDeletionReplaySource(j, reader, f.physical, facts, facts, integrationClock{}.Now)
	pubMust(t, err)
	store.publicationTransactions = &publicationPGTransactions{pool: f.pool}
	_, err = NewReplayPublicationPorts(store, ReplayPublicationDependencies{Source: source, Runtime: r, InstallationID: o.InstallationID, PublicationIDs: &publicationIDs{}, Clock: &publicationClock{}})
	pubMust(t, err)
	provider, err := NewDeletionReplayProvider(DeletionReplayDependencies{Store: store, Source: source, EvidenceIDs: &publicationIDs{}, Tokens: &publicationIDs{}, Clock: integrationClock{}})
	pubMust(t, err)
	t.Cleanup(func() { pubMust(t, errors.Join(j.Close(), r.Stop(context.Background()))) })
	return &publicationNormalFixture{store: store, provider: provider, runtime: r, journal: j, world: w}
}

func publicationPGAdvance(t *testing.T) {
	t.Run("ActualNormalRequiredDualRevisionReadsAndHeadAdvance", func(t *testing.T) {
		f := newPublicationPGFixture(t)
		ready := f.replay()
		b := f.binding(ready)
		n := publicationNormalRebuild(t, f, b, ready, false)
		pubMust(t, n.runtime.RunNormal(context.Background()))
		if n.world.storeReads.Load() < 2 || n.world.recoveryReads.Load() < 2 || n.world.articleReads.Load() < 2 {
			t.Fatal("actual Runtime required proof reads not observed")
		}
		lease, err := n.runtime.Acquire(context.Background(), recovery.WorkRequest)
		pubMust(t, err)
		_, err = n.journal.Append(lease.Context(), anchorRequest(4))
		lease.Release()
		pubMust(t, err)
		closure, err := n.store.ReadPublishedClosure(context.Background(), publicationRequest(b))
		pubMust(t, err)
		if closure.HistoricalHead != b.Head || closure.CurrentHead.Sequence != 4 || closure.Ready != ready {
			t.Fatal("business head advance invalidated historical source")
		}
		raw, err := n.provider.ReadStatus(context.Background(), f.run, f.backup)
		pubMust(t, err)
		status, err := recovery.DecodeReplayStatusV1(raw)
		pubMust(t, err)
		if status.Unavailable == nil || status.Unavailable.ReasonCode != replayReasonEvidenceStale {
			t.Fatal("fresh read accepted stale evidence")
		}
	})
	t.Run("NullPublicationBindingDoesNotOpenNormal", func(t *testing.T) {
		f := newPublicationPGFixture(t)
		ready := f.replay()
		b := f.binding(ready)
		n := publicationNormalRebuild(t, f, b, ready, true)
		if err := n.runtime.RunNormal(context.Background()); !errors.Is(err, recovery.ErrAdmissionIncomplete) {
			t.Fatalf("NULL publication normal did not remain closed: %v", err)
		}
		if lease, err := n.runtime.Acquire(context.Background(), recovery.WorkRequest); err == nil {
			lease.Release()
			t.Fatal("NULL Store opened work")
		}
	})
}

type publicationCoordinatorJournal struct{ *publicationReplayConsumer }

func (j publicationCoordinatorJournal) VerifyAgainst(ctx context.Context, run recovery.RecoveryRun) (recovery.JournalEntry, error) {
	if run.RunID != j.begin.RunID || run.BackupRef != j.begin.BackupRef || run.JournalKeyID != j.begin.KeyID || run.JournalKeyVersion != j.begin.KeyVersion || run.JournalSecretFingerprint != j.begin.SecretFingerprint || !run.StartedAt.Equal(j.begin.RecordedAt) {
		return recovery.JournalEntry{}, recovery.ErrAdmissionBinding
	}
	return j.begin, ctx.Err()
}

type publicationEnsureCapture struct {
	*PostgresReplayEvidenceStore
	permit recovery.PhasePermit
}

func (p *publicationEnsureCapture) EnsureCurrentPublication(ctx context.Context, permit recovery.PhasePermit, run, backup, ref, digest string) (recovery.PublicationBinding, error) {
	p.permit = permit
	return p.PostgresReplayEvidenceStore.EnsureCurrentPublication(ctx, permit, run, backup, ref, digest)
}

func publicationSeedLegacy(t *testing.T, pool *pgxpool.Pool, run, backup string, input DeletionReplayInput) DeletionReplayEvidence {
	t.Helper()
	e, err := evidenceFromReplayInput(run, backup, input, "retained-independent-legacy-evidence", integrationClock{}.Now())
	pubMust(t, err)
	tx, err := (publicationPGTransactor{pool}).Begin(context.Background())
	pubMust(t, err)
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(context.Background(), insertReplayRunSQL, run, ArticleDeletionNamespace, backup)
	pubMust(t, err)
	inserted, err := insertReplayEvidenceRecord(context.Background(), tx, e)
	pubMust(t, err)
	if !inserted {
		t.Fatal("new legacy evidence unexpectedly existed")
	}
	_, err = tx.Exec(context.Background(), `UPDATE deletion_replay_runs SET status='ready',input_digest=$1,current_evidence_ref=$2 WHERE run_id=$3`, e.EvidenceDigest, e.EvidenceRef, run)
	pubMust(t, err)
	pubMust(t, tx.Commit(context.Background()))
	return e
}

func publicationPGLegacy(t *testing.T) {
	t.Run("FreshLegacyZeroWriteThenActualCoordinatorEnsureNow", publicationPGEnsure)
	for _, mode := range []string{"ApprovedCheckpoint", "MissingVerifier", "ManifestOnlyApproval", "WrongSource", "WrongScope", "WrongProof", "OrdinaryContext", "ZeroPermit", "BeforeVerifierCancel", "BeforeVerifierLeaseLoss", "AfterVerifierCancel", "AfterVerifierLeaseLoss", "BlockingVerifierCancelJoin", "VerifierFailure", "VerifierPanic",
		"ProofSourceRef", "ProofSourceDigest", "ProofRun", "ProofBackup", "ProofEvidenceRef", "ProofEvidenceDigest", "ProofCompletedAt", "ProofJournal", "ProofEpoch", "ProofHeadSequence", "ProofHeadHash", "ProofOriginalPublishedAt"} {
		t.Run("ActualInstallImport"+mode, func(t *testing.T) { publicationPGImport(t, mode) })
	}
}

type publicationCheckpoint struct {
	InstallationID, Namespace string
	Proof                     LegacyPublicationProof
}
type publicationCheckpointVerifier struct {
	path, digest, installation string
	calls, active              atomic.Int64
	owned, closes              atomic.Int64
	file                       *os.File
	entered, release           chan struct{}
	after                      func()
	failure                    error
	panicValue                 any
}

func (v *publicationCheckpointVerifier) VerifyLegacyPublication(ctx context.Context, p LegacyPublicationProof) (result error) {
	v.calls.Add(1)
	v.active.Add(1)
	defer v.active.Add(-1)
	file, err := os.Open(v.path)
	if err != nil {
		return err
	}
	v.file = file
	v.owned.Add(1)
	defer func() { result = errors.Join(result, file.Close()); v.closes.Add(1); v.owned.Add(-1) }()
	if v.entered != nil {
		close(v.entered)
		<-v.release
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if recovery.SHA256Digest(raw) != v.digest || p.SourceDigest != v.digest {
		return errors.New("approved checkpoint digest mismatch")
	}
	var checkpoint publicationCheckpoint
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&checkpoint); err != nil {
		return err
	}
	checkpoint.Proof.SourceDigest = v.digest
	if checkpoint.InstallationID != v.installation || checkpoint.Namespace != ArticleDeletionNamespace || checkpoint.Proof != p {
		return errors.New("approved checkpoint exact scope mismatch")
	}
	if v.failure != nil {
		return v.failure
	}
	if v.panicValue != nil {
		panic(v.panicValue)
	}
	if v.after != nil {
		v.after()
		return nil
	} // Explicit late-success fault, after actual I/O joined.
	return ctx.Err()
}

func publicationSeedRecoveryWitness(t *testing.T, pool *pgxpool.Pool, p LegacyPublicationProof) {
	t.Helper()
	ctx := context.Background()
	started := p.EvidenceCompletedAt.Add(-time.Hour)
	_, err := pool.Exec(ctx, `INSERT INTO recovery_runs(run_id,backup_ref,journal_key_id,journal_key_version,journal_secret_fingerprint,journal_stage,started_at,outcome)
 VALUES($1,$2,'synthetic-recovery-key','v1',$3,'begun',$4,'running')`, p.RunID, p.BackupRef, "hmac-sha256:"+strings.Repeat("a", 64), started)
	pubMust(t, err)
	for _, outcome := range []string{"identity_ready", "article_deletion_ready", "ready_to_consume"} {
		_, err := pool.Exec(ctx, "UPDATE recovery_runs SET outcome=$1 WHERE run_id=$2", outcome, p.RunID)
		pubMust(t, err)
	}
	_, err = pool.Exec(ctx, `UPDATE recovery_runs SET outcome='journal_consumed',journal_stage='consumed',journal_consumed_at=$1,
 article_evidence_version=$2,article_evidence_digest=$3,article_evidence_ref=$4,article_evidence_completed_at=$5,article_namespace=$6,article_replay_verified_at=$1 WHERE run_id=$7`,
		p.OriginalPublishedAt.Add(time.Minute), DeletionReplayEvidenceVersion, p.EvidenceDigest, p.EvidenceRef, p.EvidenceCompletedAt, ArticleDeletionNamespace, p.RunID)
	pubMust(t, err)
	_, err = pool.Exec(ctx, `UPDATE recovery_runs SET outcome='succeeded',completed_at=$1 WHERE run_id=$2`, p.OriginalPublishedAt.Add(2*time.Minute), p.RunID)
	pubMust(t, err)
}

type publicationImportFixture struct {
	ctx       context.Context
	cancel    context.CancelFunc
	mode      string
	pool      *pgxpool.Pool
	w         *anchorC1World
	store     *PostgresReplayEvidenceStore
	ids       *publicationIDs
	clock     *publicationClock
	v         *publicationCheckpointVerifier
	sentinel  error
	binding   recovery.PublicationBinding
	before    map[string]string
	importErr error
}

func publicationPGImport(t *testing.T, mode string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	x := &publicationImportFixture{ctx: ctx, cancel: cancel, mode: mode, pool: newReplayPublicationPool(t, ctx), w: newAnchorC1World(t, "committed-advanced"), ids: &publicationIDs{}, clock: &publicationClock{}, sentinel: errors.New("independent checkpoint reader cause")}
	var err error
	x.store, err = NewPostgresReplayEvidenceStore(x.pool)
	pubMust(t, err)
	x.v = &publicationCheckpointVerifier{installation: x.w.o.InstallationID}
	if mode == "BlockingVerifierCancelJoin" {
		x.v.entered = make(chan struct{})
		x.v.release = make(chan struct{})
	}
	x.w.execute = func(pc context.Context, permit recovery.PhasePermit) error { return x.execute(t, pc, permit) }
	x.run(t)
	x.verify(t)
}
func (x *publicationImportFixture) execute(t *testing.T, pc context.Context, permit recovery.PhasePermit) error {
	if err := x.w.coordinate(pc, permit); err != nil {
		return err
	}
	source, dir, err := x.source(t)
	if err != nil {
		return err
	}
	proof, err := x.seed(t, pc, source, dir)
	if err != nil {
		return err
	}
	if err := x.configure(pc, source); err != nil {
		return err
	}
	return x.importAndReuse(t, pc, permit, proof)
}
func (x *publicationImportFixture) source(t *testing.T) (*ProductionDeletionReplaySource, string, error) {
	dir, err := os.MkdirTemp("", "han-b13d-a48-legacy-")
	if err != nil {
		return nil, "", err
	}
	path := filepath.Join(dir, "inventory.json")
	writeInventory(t, path, inventoryManifestWithCutoff(t, "legacy-backup", "2026-10-20T00:00:00.000000Z"))
	reader, err := NewFileBackupInventoryReader(path)
	if err != nil {
		return nil, "", err
	}
	facts, err := NewPostgresReplayFactReader(x.pool)
	if err != nil {
		return nil, "", err
	}
	finalizer, err := NewDeletionFinalizer(DeletionFinalizerDependencies{Transactions: publicationPGTransactor{x.pool}, Repository: Repository{}, Journal: x.w.control, Auditor: audit.Store{}, AuditIDs: &publicationIDs{}, Clock: integrationClock{}, WorkerID: "synthetic-legacy", LeaseDuration: time.Minute, Identity: pgA47Identity{}})
	if err != nil {
		return nil, "", err
	}
	source, err := NewProductionDeletionReplaySource(x.w.control, reader, finalizer, facts, facts, integrationClock{}.Now)
	if err != nil {
		return nil, "", err
	}
	return source, dir, nil
}
func (x *publicationImportFixture) seed(t *testing.T, pc context.Context, source *ProductionDeletionReplaySource, dir string) (LegacyPublicationProof, error) {
	fresh, err := source.Snapshot(pc, "legacy-backup")
	if err != nil {
		return LegacyPublicationProof{}, err
	}
	input := DeletionReplayInput{DeletionReplayFacts: fresh, AlreadyAbsent: fresh.TombstonesTotal}
	// Materialize the independent approved synthetic checkpoint x.before target rows exist.
	expected, err := evidenceFromReplayInput("legacy-run", "legacy-backup", input, "retained-independent-legacy-evidence", integrationClock{}.Now())
	if err != nil {
		return LegacyPublicationProof{}, err
	}
	proof := LegacyPublicationProof{SourceRef: "synthetic-preexisting-approved-checkpoint", RunID: expected.RunID, BackupRef: expected.BackupRef, EvidenceRef: expected.EvidenceRef, EvidenceDigest: expected.EvidenceDigest,
		JournalID: x.w.local.JournalID, Epoch: x.w.local.Epoch, Head: x.w.local.Head, EvidenceCompletedAt: expected.CompletedAt, OriginalPublishedAt: expected.CompletedAt.Add(time.Minute)}
	raw, err := json.Marshal(publicationCheckpoint{InstallationID: x.w.o.InstallationID, Namespace: ArticleDeletionNamespace, Proof: proof})
	if err != nil {
		return LegacyPublicationProof{}, err
	}
	x.v.path = filepath.Join(dir, "approved-original-checkpoint.json")
	if err := os.WriteFile(x.v.path, raw, 0600); err != nil {
		return LegacyPublicationProof{}, err
	}
	x.v.digest = recovery.SHA256Digest(raw)
	proof.SourceDigest = x.v.digest
	publicationSeedLegacy(t, x.pool, proof.RunID, proof.BackupRef, input)
	publicationSeedRecoveryWitness(t, x.pool, proof)
	return proof, nil
}
func (x *publicationImportFixture) configure(pc context.Context, source *ProductionDeletionReplaySource) error {
	var verifier LegacyPublicationVerifier = x.v
	if x.mode == "MissingVerifier" || x.mode == "ManifestOnlyApproval" {
		var missing *publicationCheckpointVerifier
		verifier = missing
	}
	if x.mode == "WrongScope" {
		x.v.installation = "wrong-installation"
	}
	if x.mode == "WrongSource" {
		x.v.digest = recovery.SHA256Digest([]byte("unapproved checkpoint"))
	}
	if x.mode == "AfterVerifierCancel" {
		x.v.after = func() { x.cancel(); <-pc.Done() }
	}
	if x.mode == "AfterVerifierLeaseLoss" {
		x.v.after = func() { x.w.h.releaseNow(); <-x.w.h.terminal; x.w.h.driver.Wait() }
	}
	if x.mode == "VerifierFailure" {
		x.v.failure = x.sentinel
	}
	if x.mode == "VerifierPanic" {
		x.v.panicValue = x.sentinel
	}
	x.store.publicationTransactions = &publicationPGTransactions{pool: x.pool}
	_, err := NewReplayPublicationPorts(x.store, ReplayPublicationDependencies{Source: source, Runtime: x.w.runtime, InstallationID: x.w.o.InstallationID, PublicationIDs: x.ids, Clock: x.clock, LegacySources: verifier})
	if err != nil {
		return err
	}
	return nil
}
func (x *publicationImportFixture) importAndReuse(t *testing.T, pc context.Context, permit recovery.PhasePermit, proof LegacyPublicationProof) error {
	x.before = publicationImage(t, x.pool)
	if x.mode == "WrongProof" {
		proof.Head.Hash = recovery.SHA256Digest([]byte("wrong historical head"))
	}
	publicationMutateLegacyProof(&proof, x.mode)
	callContext, callPermit := pc, permit
	if x.mode == "OrdinaryContext" {
		callContext = context.Background()
	}
	if x.mode == "ZeroPermit" {
		callPermit = recovery.PhasePermit{}
	}
	if x.mode == "BeforeVerifierCancel" {
		x.cancel()
		<-pc.Done()
	}
	if x.mode == "BeforeVerifierLeaseLoss" {
		x.w.h.releaseNow()
		<-x.w.h.terminal
		x.w.h.driver.Wait()
	}
	x.binding, x.importErr = x.store.ImportLegacyPublication(callContext, callPermit, proof)
	if x.mode == "ApprovedCheckpoint" && x.importErr == nil {
		again, err := x.store.ImportLegacyPublication(pc, permit, proof)
		if err != nil {
			return err
		}
		if again != x.binding || x.ids.calls.Load() != 1 || x.clock.calls.Load() != 1 {
			return errors.New("Import reuse changed receipt/ref/time")
		}
	}
	return x.importErr
}
func (x *publicationImportFixture) run(t *testing.T) {
	if x.mode == "BlockingVerifierCancelJoin" {
		publicationImportBlockedJoin(t, x.w, x.v, x.ctx)
	} else if x.mode == "VerifierPanic" {
		var recovered any
		func() { defer func() { recovered = recover() }(); _ = x.w.runtime.Install(x.ctx, x.w.manifest) }()
		if recovered != x.sentinel {
			t.Fatal("verifier panic did not naturally propagate")
		}
	} else {
		err := x.w.runtime.Install(x.ctx, x.w.manifest)
		if x.mode == "ApprovedCheckpoint" {
			pubMust(t, err)
		} else if err == nil {
			t.Fatal("untrusted/revoked Import succeeded")
		}
	}

}
func (x *publicationImportFixture) verify(t *testing.T) {
	if x.before == nil || x.w.control == nil {
		t.Fatal("case did not reach actual install/control/source boundary")
	}
	if x.mode == "VerifierFailure" && !errors.Is(x.importErr, x.sentinel) {
		t.Fatal("verifier error cause lost")
	}
	if (x.mode == "BeforeVerifierCancel" || x.mode == "AfterVerifierCancel") && !errors.Is(x.importErr, context.Canceled) {
		t.Fatalf("Import cancellation cause lost: %x.v", x.importErr)
	}
	if (x.mode == "OrdinaryContext" || x.mode == "ZeroPermit" || x.mode == "BeforeVerifierCancel" || x.mode == "BeforeVerifierLeaseLoss") && x.v.calls.Load() != 0 {
		t.Fatal("Import called verifier without current control authority")
	}
	if x.v.active.Load() != 0 || x.v.owned.Load() != 0 {
		t.Fatal("verifier I/O/file still active after return")
	}
	after := publicationImage(t, x.pool)
	for table, image := range x.before {
		if table != "article_replay_publications" && image != after[table] {
			t.Fatalf("Article Import wrote original %s", table)
		}
	}
	count := publicationCount(t, x.pool, "article_replay_publications")
	if x.mode == "ApprovedCheckpoint" {
		if count != 1 || x.binding.Ref == "" || x.binding.PublishedAt.Equal(x.binding.EvidenceCompletedAt) {
			t.Fatal("trusted Import did not return a new exact source")
		}
		var kind string
		pubMust(t, x.pool.QueryRow(context.Background(), "SELECT source_kind FROM article_replay_publications").Scan(&kind))
		if kind != replayPublicationLegacy {
			t.Fatal("import source kind incorrect")
		}
	} else if count != 0 || x.ids.calls.Load() != 0 || x.clock.calls.Load() != 0 {
		t.Fatal("rejected Import started INSERT/receipt generation")
	}
	if (x.mode == "MissingVerifier" || x.mode == "ManifestOnlyApproval") && x.v.calls.Load() != 0 {
		t.Fatal("disabled Import called a verifier")
	}
}

func publicationMutateLegacyProof(p *LegacyPublicationProof, mode string) {
	switch mode {
	case "ProofSourceRef":
		p.SourceRef = "wrong"
	case "ProofSourceDigest":
		p.SourceDigest = recovery.SHA256Digest([]byte("wrong"))
	case "ProofRun":
		p.RunID = "wrong"
	case "ProofBackup":
		p.BackupRef = "wrong"
	case "ProofEvidenceRef":
		p.EvidenceRef = "wrong"
	case "ProofEvidenceDigest":
		p.EvidenceDigest = recovery.SHA256Digest([]byte("wrong"))
	case "ProofCompletedAt":
		p.EvidenceCompletedAt = p.EvidenceCompletedAt.Add(time.Microsecond)
	case "ProofJournal":
		p.JournalID = "wrong"
	case "ProofEpoch":
		p.Epoch++
	case "ProofHeadSequence":
		p.Head.Sequence++
	case "ProofHeadHash":
		p.Head.Hash = recovery.SHA256Digest([]byte("wrong"))
	case "ProofOriginalPublishedAt":
		p.OriginalPublishedAt = p.OriginalPublishedAt.Add(time.Microsecond)
	}
}

func publicationImportBlockedJoin(t *testing.T, w *anchorC1World, v *publicationCheckpointVerifier, ctx context.Context) {
	t.Helper()
	var workers sync.WaitGroup
	install, stop1, stop2 := make(chan error, 1), make(chan error, 1), make(chan error, 1)
	var release sync.Once
	defer func() { release.Do(func() { close(v.release) }); workers.Wait() }()
	workers.Add(1)
	go func() { defer workers.Done(); install <- w.runtime.Install(ctx, w.manifest) }()
	select {
	case <-v.entered:
	case err := <-install:
		t.Fatalf("Install ended before verifier: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if v.owned.Load() != 1 || v.closes.Load() != 0 {
		t.Fatal("blocked verifier did not acquire a real checkpoint file")
	}
	if _, err := v.file.Stat(); err != nil {
		t.Fatal("blocked checkpoint file was not held")
	}
	workers.Add(2)
	go func() { defer workers.Done(); stop1 <- w.runtime.Stop(context.Background()) }()
	go func() { defer workers.Done(); stop2 <- w.runtime.Stop(context.Background()) }()
	<-w.pc.Done()
	for _, done := range []<-chan error{install, stop1, stop2} {
		select {
		case <-done:
			t.Fatal("Stop/Install returned while verifier I/O was blocked")
		default:
		}
	}
	assertAnchorDeploymentCloses(t, w.anchorRuntimeWorld, 0)
	release.Do(func() { close(v.release) })
	workers.Wait()
	err := <-install
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("blocked verifier lost original cancellation: %v", err)
	}
	pubMust(t, <-stop1)
	pubMust(t, <-stop2)
	if v.active.Load() != 0 || v.owned.Load() != 0 || v.closes.Load() != 1 || w.h.rawCloses.Load() != 1 || w.h.closeAttempts.Load() != 1 {
		t.Fatal("verifier/namespace resource not truly joined exactly once")
	}
	if _, err := v.file.Read(make([]byte, 1)); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("checkpoint physical file was not closed: %v", err)
	}
}

type publicationFactGate struct {
	actual           *PostgresReplayFactReader
	mu               sync.Mutex
	armed            bool
	entered, release chan struct{}
	reads            atomic.Int64
}

func (p *publicationFactGate) arm() (chan struct{}, chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.armed = true
	p.entered = make(chan struct{})
	p.release = make(chan struct{})
	return p.entered, p.release
}
func (p *publicationFactGate) ReadDeletionOperations(ctx context.Context) ([]DeletionOperationReplayFact, error) {
	p.reads.Add(1)
	rows, err := p.actual.ReadDeletionOperations(ctx)
	if err != nil {
		return nil, err
	}
	p.mu.Lock()
	armed, entered, release := p.armed, p.entered, p.release
	p.armed = false
	p.mu.Unlock()
	if armed {
		close(entered)
		<-release
	}
	return rows, ctx.Err()
}
func (p *publicationFactGate) ReadDeletionAuditEvents(ctx context.Context) ([]DeletionReplayAuditFact, error) {
	p.reads.Add(1)
	return p.actual.ReadDeletionAuditEvents(ctx)
}
func publicationPGRefreshRecheck(t *testing.T, operation string) {
	f := newPublicationPGFixture(t)
	var request ReplayPublishRequest
	if operation == "Publish" {
		request, _, _ = publicationPrepareRequest(t, f)
	} else {
		_ = f.replay()
	}
	entered, release := f.facts.arm()
	var once sync.Once
	var workers sync.WaitGroup
	result := make(chan error, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		if operation == "Publish" {
			_, err := f.store.Publish(f.ctx, request)
			result <- err
		} else {
			_, _, err := f.store.ReadCurrent(f.ctx, f.run, f.backup, func(c context.Context) (DeletionReplayFacts, error) { return f.source.Snapshot(c, f.backup) })
			result <- err
		}
	}()
	defer func() { once.Do(func() { close(release) }); workers.Wait() }()
	select {
	case <-entered:
	case err := <-result:
		t.Fatalf("did not enter real fresh reader: %v", err)
	case <-f.ctx.Done():
		t.Fatal(f.ctx.Err())
	}
	advancePublicationOwnerNowait(t, f)
	once.Do(func() { close(release) })
	workers.Wait()
	err := <-result
	if operation == "Publish" {
		if !errors.Is(err, ErrDeletionReplayAttemptNotOwner) {
			t.Fatalf("Publish failed to recheck owner: %v", err)
		}
		if publicationCount(t, f.observer, "deletion_replay_evidence") != 0 || publicationCount(t, f.observer, "article_replay_publications") != 0 {
			t.Fatal("stale owner published")
		}
	} else if !errors.Is(err, ErrDeletionReplayInputChanged) {
		t.Fatalf("ReadCurrent failed to recheck pointer/owner: %v", err)
	}
}

func publicationPGFirstCount(t *testing.T) {
	f := newPublicationPGFixture(t)
	first := f.replay()
	binding := f.binding(first)
	if first.TombstonesReplayed != 2 || first.AlreadyAbsent != 1 || publicationCount(t, f.observer, "articles") != 0 || publicationCount(t, f.observer, "audit_events") != 3 {
		t.Fatal("actual Finalizer outcomes were not 2/1 with DELETE/Audit")
	}
	second := f.replay()
	if first != second || f.physical.calls.Load() != 1 {
		t.Fatal("fresh Ready replayed or changed first evidence")
	}
	a, err := f.store.LoadAttempt(f.ctx, f.run, f.backup)
	pubMust(t, err)
	late, err := f.source.ReplayOldBackup(f.ctx, f.backup)
	pubMust(t, err)
	if late.TombstonesReplayed != 0 || late.AlreadyAbsent != 3 {
		t.Fatal("late physical result was not real0/3")
	}
	evidence, err := f.store.Publish(f.ctx, ReplayPublishRequest{RunID: f.run, BackupRef: f.backup, Attempt: a, Input: late, Refresh: func(c context.Context) (DeletionReplayFacts, error) { return f.source.Snapshot(c, f.backup) }, EvidenceIDs: &publicationIDs{}, Clock: integrationClock{}})
	pubMust(t, err)
	if evidence.TombstonesReplayed != 2 || evidence.AlreadyAbsent != 1 || f.binding(first) != binding || f.ids.calls.Load() != 1 || f.clock.calls.Load() != 1 {
		t.Fatal("late winner changed committed evidence/receipt metadata")
	}

}

func publicationPGRequiredReads(t *testing.T) {
	f := newPublicationPGFixture(t)
	ready := f.replay()
	b := f.binding(ready)
	before := publicationImage(t, f.observer)
	calls, facts := f.physical.calls.Load(), f.facts.reads.Load()
	consumes, writes := f.phase.world.calls.Load(), f.txs.statements.Load()
	f.txs.mu.Lock()
	eventStart := len(f.txs.events)
	f.txs.mu.Unlock()
	var lists, opens atomic.Int64
	media := f.phase.journal.media
	f.phase.journal.media.names = func(ctx context.Context) ([]string, error) { lists.Add(1); return media.names(ctx) }
	f.phase.journal.media.open = func(ctx context.Context, name string) (*os.File, error) { opens.Add(1); return media.open(ctx, name) }
	anchor := f.phase.journal.options.Anchor.(*anchorCoreCAS)
	anchor.mu.Lock()
	n := len(anchor.calls)
	anchor.mu.Unlock()
	closure, err := f.store.ReadPublishedClosure(context.Background(), publicationRequest(b))
	pubMust(t, err)
	if closure.Ready != ready || closure.HistoricalHead != b.Head || closure.CurrentHead != b.Head || closure.VerifiedArticleRevision == "" {
		t.Fatal("wrong exact historical projection")
	}
	if calls != f.physical.calls.Load() || facts != f.facts.reads.Load() || !reflect.DeepEqual(before, publicationImage(t, f.observer)) {
		t.Fatal("historical read refreshed Facts, wrote or replayed")
	}
	if consumes != f.phase.world.calls.Load() || writes != f.txs.statements.Load() {
		t.Fatal("historical read consumed or executed a write statement")
	}
	f.txs.mu.Lock()
	events := append([]string(nil), f.txs.events[eventStart:]...)
	f.txs.mu.Unlock()
	if !reflect.DeepEqual(events, []string{"ReadPublication", "ReadEvidence"}) || lists.Load() == 0 || opens.Load() == 0 {
		t.Fatalf("historical required DB/actual source reads absent: %v lists=%d opens=%d", events, lists.Load(), opens.Load())
	}
	anchor.mu.Lock()
	reads := append([]string(nil), anchor.calls[n:]...)
	anchor.mu.Unlock()
	if len(reads) == 0 {
		t.Fatal("required Anchor reads absent")
	}
	for _, call := range reads {
		if call == "cas" {
			t.Fatal("historical read repaired Anchor")
		}
	}
	t.Logf("historical required DB=%v Journal/source lists=%d file opens=%d Anchor/revision reads=%d; fresh Facts=0 ReplayAll=0 Consume=0 SQL Exec(Fail/Publish/receipt write)=0 Anchor CAS/repair=0", events, lists.Load(), opens.Load(), len(reads))

}

func publicationPGCatalog(t *testing.T) {
	f := newPublicationPGFixture(t)
	ready := f.replay()
	b := f.binding(ready)
	var definition string
	pubMust(t, f.observer.QueryRow(f.ctx, `SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='article_replay_publications'::regclass AND conname='article_replay_publications_evidence_fk'`).Scan(&definition))
	if !strings.Contains(definition, "run_id, namespace, backup_ref, evidence_ref, evidence_digest") {
		t.Fatal("incomplete composite FK")
	}
	for _, sql := range []string{`UPDATE article_replay_publications SET source_ref='changed'`, `DELETE FROM article_replay_publications`} {
		_, err := f.pool.Exec(f.ctx, sql)
		var pgerr *pgconn.PgError
		if !errors.As(err, &pgerr) || pgerr.Code != "55000" {
			t.Fatalf("immutable constraint did not reject: %v", err)
		}
	}
	before := publicationImage(t, f.observer)
	assertPublicationSQLMutations(t, f, b)
	if !reflect.DeepEqual(before, publicationImage(t, f.observer)) {
		t.Fatal("negative SQL changed retained rows")
	}

}

func publicationPGEnsure(t *testing.T) {
	f := newPublicationPGFixture(t)
	input, err := f.source.ReplayOldBackup(f.ctx, f.backup)
	pubMust(t, err)
	e := publicationSeedLegacy(t, f.pool, f.run, f.backup, input)
	before := publicationImage(t, f.observer)
	assertUnconfiguredLegacyFresh(t, f, e, before)
	raw, err := f.provider.ReadStatus(f.ctx, f.run, f.backup)
	pubMust(t, err)
	status, err := recovery.DecodeReplayStatusV1(raw)
	pubMust(t, err)
	if status.Ready == nil || status.Ready.TombstonesReplayed != 2 || !reflect.DeepEqual(before, publicationImage(t, f.observer)) {
		t.Fatal("legacy fresh read changed rows")
	}
	port := &publicationEnsureCapture{PostgresReplayEvidenceStore: f.store}
	coordinator, err := recovery.NewCoordinator(f.phase.world, publicationCoordinatorJournal{f.phase.world}, f.provider, port, recovery.NewReplayStatusV1Decoder(recovery.SHA256Digest), integrationClock{}.Now)
	pubMust(t, err)
	_, err = coordinator.Consume(f.ctx, f.run)
	if err == nil {
		t.Fatal("synthetic identity Store must not complete a full recovery")
	}
	b, err := f.store.ReadCurrentPublication(f.ctx, f.run, f.backup, e.EvidenceRef, e.EvidenceDigest)
	pubMust(t, err)
	if b.EvidenceRef != e.EvidenceRef || !b.EvidenceCompletedAt.Equal(e.CompletedAt) || b.PublishedAt.Equal(e.CompletedAt) {
		t.Fatal("reassertion changed old evidence or claimed old publication time")
	}
	ids, clock := f.ids.calls.Load(), f.clock.calls.Load()
	again, err := f.store.EnsureCurrentPublication(f.ctx, port.permit, f.run, f.backup, e.EvidenceRef, e.EvidenceDigest)
	pubMust(t, err)
	if again != b || f.ids.calls.Load() != ids || f.clock.calls.Load() != clock {
		t.Fatal("Ensure reuse consumed metadata")
	}
	if _, err := f.store.EnsureCurrentPublication(f.ctx, recovery.PhasePermit{}, f.run, f.backup, e.EvidenceRef, e.EvidenceDigest); err == nil {
		t.Fatal("zero permit reasserted legacy")
	}

}

func replayPublicationConnectionVerifier(t *testing.T, expectedHost string, expectedPort uint16, expectedDB, expectedUser string) func(context.Context, *pgx.Conn) error {
	var firstConnection atomic.Bool
	return func(ctx context.Context, c *pgx.Conn) error {
		var version, port int
		var host, db, user string
		if err := c.QueryRow(ctx, `SELECT current_setting('server_version_num')::int,host(inet_server_addr()),inet_server_port(),current_database(),current_user`).Scan(&version, &host, &port, &db, &user); err != nil {
			return err
		}
		if version < 180000 || version >= 190000 || host != expectedHost || port != int(expectedPort) || db != expectedDB || user != expectedUser {
			return errors.New("isolated PostgreSQL identity mismatch")
		}
		if firstConnection.CompareAndSwap(false, true) {
			t.Logf("PG preflight version=%d host=%s port=%d database=%s user=%s", version, host, port, db, user)
		}
		return nil
	}
}

func applyReplayPublicationMigrations(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schema string) {
	for _, name := range []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0005_recovery_runs.sql", "0009_article_replay_attempt.sql", "0011_article_replay_publication.sql"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(raw)); err != nil {
			t.Fatalf("apply explicit migration %s: %v", name, err)
		}
		t.Logf("applied migration=%s digest=%s", name, recovery.SHA256Digest(raw))
	}
	var current string
	if err := pool.QueryRow(ctx, "SELECT current_schema()").Scan(&current); err != nil || current != schema {
		t.Fatal("migration schema readback mismatch")
	}
	t.Logf("retained schema=%s explicit migrations=0001,0002,0003,0005,0009,0011", schema)
}

func assertPublicationRollback(t *testing.T, f *publicationPGFixture, request ReplayPublishRequest) (map[string]string, []string) {
	before := publicationImage(t, f.observer)
	owner, err := f.store.LoadAttempt(f.ctx, f.run, f.backup)
	pubMust(t, err)
	events := []string{"Publish"}
	evidence, err := f.store.Publish(f.ctx, request)
	if err == nil {
		t.Fatal("transaction witness did not reject Publish")
	}
	if evidence != (DeletionReplayEvidence{}) {
		t.Fatal("failed Publish returned uncommitted evidence")
	}
	var pgerr *pgconn.PgError
	if !errors.As(err, &pgerr) || pgerr.Code != "23514" {
		t.Fatalf("wrong witness error: %v", err)
	}
	if !reflect.DeepEqual(before, publicationImage(t, f.observer)) {
		t.Fatal("failed publication changed original seven-table/receipt image")
	}
	after, err := f.store.LoadAttempt(f.ctx, f.run, f.backup)
	pubMust(t, err)
	if !matchesAttemptSnapshot(owner, after) {
		t.Fatal("failed publication changed exact owner")
	}
	events = append(events, "ReadCurrent")
	run, pending, err := f.store.ReadCurrent(f.ctx, f.run, f.backup, request.Refresh)
	pubMust(t, err)
	if pending != nil || run.Status != ReplayRunRunning {
		t.Fatal("failed Publish changed current to Ready")
	}
	if publicationCount(t, f.observer, "deletion_replay_evidence") != 0 || publicationCount(t, f.observer, "article_replay_publications") != 0 {
		t.Fatal("partially visible publication")
	}
	return before, events
}

func assertPublicationSQLMutations(t *testing.T, f *publicationPGFixture, b recovery.PublicationBinding) {
	for _, change := range []string{"run_id='other-run'", "backup_ref='other-backup'", "evidence_ref='missing'", "evidence_digest='sha256:" + strings.Repeat("a", 64) + "'", "namespace='foreign/v1'", "attempt_generation=-1", "attempt_token=NULL", "source_kind='caller_approved'"} {
		tx, err := f.pool.Begin(f.ctx)
		pubMust(t, err)
		// clone into a fresh unique reference, then alter only a selected INSERT value.
		columns := strings.Split(strings.ReplaceAll(replayPublicationColumns, "\n", ""), ",")
		values := append([]string(nil), columns...)
		values[0] = "'88888888-8888-4888-8888-888888888888'"
		for i, column := range columns {
			if strings.TrimSpace(column) == "attempt_generation" {
				values[i] = "attempt_generation+1"
			}
		}
		parts := strings.SplitN(change, "=", 2)
		for i, column := range columns {
			if strings.TrimSpace(column) == parts[0] {
				values[i] = parts[1]
			}
		}
		_, err = tx.Exec(f.ctx, "INSERT INTO article_replay_publications ("+replayPublicationColumns+") SELECT "+strings.Join(values, ",")+" FROM article_replay_publications WHERE publication_ref=$1", b.Ref)
		if err == nil {
			_ = tx.Rollback(context.Background())
			t.Fatal("invalid composite/source/generation INSERT accepted")
		}
		var pgerr *pgconn.PgError
		if !errors.As(err, &pgerr) {
			_ = tx.Rollback(context.Background())
			t.Fatal("constraint failure was not PostgreSQL")
		}
		if parts[0] == "run_id" || parts[0] == "backup_ref" || parts[0] == "evidence_ref" || parts[0] == "evidence_digest" {
			if pgerr.Code != "23503" {
				_ = tx.Rollback(context.Background())
				t.Fatal("unique/check masked composite FK oracle")
			}
		}
		pubMust(t, tx.Rollback(context.Background()))
	}
}

func assertUnconfiguredLegacyFresh(t *testing.T, f *publicationPGFixture, e DeletionReplayEvidence, before map[string]string) {
	// The pool-only constructor remains a usable fresh legacy reader, with no bind.
	legacy, err := NewPostgresReplayEvidenceStore(f.pool)
	pubMust(t, err)
	legacyProvider, err := NewDeletionReplayProvider(DeletionReplayDependencies{Store: legacy, Source: f.source, EvidenceIDs: &publicationIDs{}, Tokens: &publicationIDs{}, Clock: integrationClock{}})
	pubMust(t, err)
	legacyRaw, err := legacyProvider.ReadStatus(context.Background(), f.run, f.backup)
	pubMust(t, err)
	legacyStatus, err := recovery.DecodeReplayStatusV1(legacyRaw)
	pubMust(t, err)
	if legacyStatus.Ready == nil || legacyStatus.Ready.EvidenceRef != e.EvidenceRef || !reflect.DeepEqual(before, publicationImage(t, f.observer)) {
		t.Fatal("unconfigured legacy fresh read was blocked or wrote metadata")
	}
}

func publicationPGNextGeneration(t *testing.T) {
	f := newPublicationPGFixture(t)
	a := f.replay()
	first := f.binding(a)
	writeInventory(t, f.path, inventoryManifestWithCutoff(t, f.backup, "2026-10-20T00:00:02.000000Z"))
	b := f.replay()
	if b.EvidenceRef == a.EvidenceRef {
		t.Fatal("drift failed to advance current")
	}
	writeInventory(t, f.path, inventoryManifestWithCutoff(t, f.backup, "2026-10-20T00:00:00.000000Z"))
	again := f.replay()
	last := f.binding(again)
	if again != a || last.Ref == first.Ref || publicationCount(t, f.observer, "deletion_replay_evidence") != 2 || publicationCount(t, f.observer, "article_replay_publications") != 3 {
		t.Fatal("A/B/A duplicated evidence or reused old publication")
	}
	closure, err := f.store.ReadPublishedClosure(context.Background(), publicationRequest(first))
	pubMust(t, err)
	if closure.Ready != a || closure.PublicationRef != first.Ref {
		t.Fatal("old exact publication changed")
	}
}

func assertPublicationRunUnlocked(t *testing.T, f *publicationPGFixture) {
	conn, err := f.observer.Acquire(f.ctx)
	pubMust(t, err)
	defer conn.Release()
	tx, err := conn.Begin(f.ctx)
	pubMust(t, err)
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(f.ctx, `SELECT 1 FROM deletion_replay_runs WHERE run_id=$1 FOR UPDATE NOWAIT`, f.run)
	pubMust(t, err)
}

func advancePublicationOwnerNowait(t *testing.T, f *publicationPGFixture) {
	// The independent observer can own the run row while the source is blocked.
	tx, err := (publicationPGTransactor{f.observer}).Begin(f.ctx)
	pubMust(t, err)
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(f.ctx, `SELECT 1 FROM deletion_replay_runs WHERE run_id=$1 FOR UPDATE NOWAIT`, f.run)
	pubMust(t, err)
	_, err = tx.Exec(f.ctx, `UPDATE deletion_replay_runs SET attempt_generation=attempt_generation+1,attempt_token='synthetic-new-owner',status='running' WHERE run_id=$1`, f.run)
	pubMust(t, err)
	pubMust(t, tx.Commit(f.ctx))
}

func publicationPGTrace(t *testing.T, existing bool) {
	f := newPublicationPGFixture(t)
	ctx := f.ctx
	if existing {
		var err error
		ctx, err = platform.ContextWithTraceID(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAV")
		pubMust(t, err)
	}
	raw, err := f.provider.ReplayAndVerify(ctx, f.run, f.backup)
	pubMust(t, err)
	status, err := recovery.DecodeReplayStatusV1(raw)
	pubMust(t, err)
	if status.Ready == nil {
		t.Fatal("Trace case unavailable")
	}
	f.physical.mu.Lock()
	trace := f.physical.trace
	f.physical.mu.Unlock()
	if trace == "" || existing && trace != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatal("Trace not preserved/generated")
	}
	var count int
	pubMust(t, f.observer.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE trace_id=$1", trace).Scan(&count))
	if count != 3 {
		t.Fatal("real SQL audit lost Trace")
	}
}
