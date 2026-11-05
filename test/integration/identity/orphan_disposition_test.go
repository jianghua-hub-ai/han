//go:build integration

package identity

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	identitypkg "han/internal/identity"
	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"
	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const orphanPGTrace = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

// Historical RED (Turing, before the authorized T1 normalization): two real PG
// runs returned journal_corrupt, exact timestamp equality=false, microsecond
// equality=true, operation=journaled, Article=pending and completion audits=0.
// Retained schemas: han_s62_3cff71eb4b1f09dd, han_s62_40ea3b9a7261f60a.
// Do not normalize all fixture clocks: the precision regression supplies nanos.
func newOrphanPGHarness(t *testing.T) *schemaHarness {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)
	config, err := pgxpool.ParseConfig(testenv.RequirePostgresDSN(t))
	if err != nil {
		t.Fatal("configure orphan PostgreSQL harness")
	}
	schema := "han_s62_" + randomHex(t, 8)
	quoted := pgx.Identifier{schema}.Sanitize()
	config.ConnConfig.RuntimeParams["search_path"] = quoted
	config.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open orphan PostgreSQL harness")
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal("connect orphan PostgreSQL harness")
	}
	requirePostgres18(t, ctx, pool)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal("begin orphan schema")
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal("create orphan schema")
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+quoted); err != nil {
		t.Fatal("bind orphan schema")
	}
	for _, file := range migrationFiles {
		if _, err := tx.Exec(ctx, readMigrationFile(t, file)); err != nil {
			t.Fatalf("apply %s: %v", file, err)
		}
	}
	// 0010 requires a bound installation before any recovery run is inserted.
	if _, err := tx.Exec(ctx, `INSERT INTO recovery_installation(singleton,metadata_version,deployment_id,
		installation_id,target_digest,commitment_ref,commitment_digest,recovery_journal_id,article_journal_id,
		recovery_epoch,article_epoch,recovery_initial_sequence,article_initial_sequence,recovery_initial_hash,
		article_initial_hash,revision,installed_at)
		VALUES(true,'han-recovery-install/v1','s62-synthetic-deployment','s62-synthetic-installation',
		'sha256:0000000000000000000000000000000000000000000000000000000000000000',
		'recovery/v1/installation/s62-synthetic-deployment',
		'sha256:1111111111111111111111111111111111111111111111111111111111111111',
		's62-recovery-journal','s62-article-journal',1,1,0,0,'','',1,'2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("bind orphan recovery installation: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal("commit orphan schema")
	}
	t.Logf("retained_schema=%s", schema)
	return &schemaHarness{pool: pool, schema: schema, quoted: quoted}
}

type orphanPGTransactor struct {
	harness *schemaHarness
	wrap    func(pgx.Tx) pgx.Tx
}

func (s orphanPGTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := s.harness.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, "SET LOCAL search_path TO "+s.harness.quoted); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, err
	}
	// No SQL read before production SET TRANSACTION. PID is available without SQL.
	if s.wrap != nil {
		return s.wrap(tx), nil
	}
	return tx, nil
}

type orphanPGClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *orphanPGClock) Now() time.Time    { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *orphanPGClock) set(now time.Time) { c.mu.Lock(); defer c.mu.Unlock(); c.now = now }

// This in-memory journal isolates the real PG matrix from unsupported Windows
// directory fsync. Group 10 also requires the concrete file journal separately;
// that evidence must never be claimed from this adapter.
type orphanPGJournal struct {
	mu      sync.Mutex
	records []article.ArticleDeletionTombstone
}

func (j *orphanPGJournal) Append(ctx context.Context, r article.DeletionTombstoneRequest) (article.ArticleDeletionTombstone, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, v := range j.records {
		if v.DeletionOperationID == r.DeletionOperationID {
			return v, nil
		}
	}
	v := article.ArticleDeletionTombstone{Namespace: article.ArticleDeletionNamespace, Sequence: int64(len(j.records) + 1), ArticleID: r.ArticleID, ArticleVersion: r.ArticleVersion, DeletionOperationID: r.DeletionOperationID, DeletedAt: r.DeletedAt, KeyID: "fixture-key", KeyVersion: "v1"}
	j.records = append(j.records, v)
	return v, nil
}
func (j *orphanPGJournal) FindByOperation(ctx context.Context, id string) (article.ArticleDeletionTombstone, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, v := range j.records {
		if v.DeletionOperationID == id {
			return v, nil
		}
	}
	return article.ArticleDeletionTombstone{}, article.ErrDeletionTombstoneNotFound
}

type orphanPGAudit struct {
	after func(context.Context, pgx.Tx, audit.Event) error
}

func (a orphanPGAudit) Append(ctx context.Context, tx pgx.Tx, e audit.Event) error {
	if err := (audit.Store{}).Append(ctx, tx, e); err != nil {
		return err
	}
	if a.after != nil {
		return a.after(ctx, tx, e)
	}
	return nil
}

// Only DTO mapping and native proof binding live in this test adapter. Neither
// it nor its verifier implement a provider, count query or transaction authority.
type orphanPGCompletionHook func(context.Context, pgx.Tx, *article.DispositionCompletionProof, *identitypkg.ArticleIdentityLifecycle, identitypkg.ArticleCompletionReference) error

type orphanPGIdentity struct {
	audit         identitypkg.CanonicalDeleteAuditAppender
	clock         *orphanPGClock
	hook          orphanPGCompletionHook
	mu            sync.Mutex
	calls         int
	lastProof     *article.DispositionCompletionProof
	lastReference identitypkg.ArticleCompletionReference
	lastFacts     identitypkg.VerifiedArticleCompletion
}

func orphanPGBoundary(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, identitypkg.ErrArticleTransferTargetInvalid) {
		return errors.Join(article.ErrTransferTargetInvalid, err)
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		return platform.NewPublicError(platform.CodePlatformDatabaseUnavailable, err)
	}
	return err
}
func (a *orphanPGIdentity) LockTransferParties(ctx context.Context, tx pgx.Tx, target *string, source string) (article.DispositionParties, error) {
	p, err := (&identitypkg.ArticleIdentityLifecycle{}).LockTransferParties(ctx, tx, target, source)
	return article.DispositionParties{SourceID: p.SourceID, SourceEmail: p.SourceEmail, TargetID: p.TargetID, TargetEmail: p.TargetEmail}, orphanPGBoundary(err)
}
func (a *orphanPGIdentity) LockDeletionCandidate(ctx context.Context, tx pgx.Tx, id string) (article.DispositionDeletionCandidate, error) {
	p, err := (&identitypkg.ArticleIdentityLifecycle{}).LockDeletionCandidate(ctx, tx, id)
	return article.DispositionDeletionCandidate{UserID: p.UserID, Deleted: p.Deleted}, orphanPGBoundary(err)
}
func (a *orphanPGIdentity) ClearRetainedEmailIfEligible(ctx context.Context, tx pgx.Tx, source string, proof *article.DispositionCompletionProof) (bool, error) {
	ref := orphanPGReference(proof.Reference())
	verifier := &orphanPGBoundVerifier{proof: proof, owner: a}
	lifecycle, err := identitypkg.NewArticleIdentityLifecycle(verifier, a.audit, identitypkg.RandomUUIDGenerator{}, a.clock)
	if err != nil {
		return false, err
	}
	a.mu.Lock()
	a.lastProof = proof
	a.lastReference = ref
	a.mu.Unlock()
	if a.hook != nil {
		if err := a.hook(ctx, tx, proof, lifecycle, ref); err != nil {
			return false, err
		}
	}
	cleared, err := lifecycle.ClearRetainedEmailIfEligible(ctx, tx, source, ref)
	return cleared, orphanPGBoundary(err)
}

type orphanPGBoundVerifier struct {
	proof *article.DispositionCompletionProof
	owner *orphanPGIdentity
}

func (v *orphanPGBoundVerifier) VerifyCompletion(ctx context.Context, tx pgx.Tx, ref identitypkg.ArticleCompletionReference) (identitypkg.VerifiedArticleCompletion, error) {
	if v.proof == nil || tx == nil || tx != ref.Transaction || !reflect.DeepEqual(ref, orphanPGReference(v.proof.Reference())) {
		return identitypkg.VerifiedArticleCompletion{}, errors.New("native reference field or exact transaction mismatch")
	}
	facts, err := v.proof.VerifyCompletion(ctx, tx)
	if err != nil {
		return identitypkg.VerifiedArticleCompletion{}, err
	}
	mapped := orphanPGFacts(facts)
	if v.owner != nil {
		v.owner.mu.Lock()
		v.owner.calls++
		v.owner.lastFacts = mapped
		v.owner.mu.Unlock()
	}
	return mapped, nil
}
func orphanPGReference(r article.DispositionCompletionReference) identitypkg.ArticleCompletionReference {
	return identitypkg.ArticleCompletionReference{Transaction: r.Transaction, ExpectedArticleID: r.ExpectedArticleID, SourceID: r.SourceID, BeforeVersion: r.BeforeVersion, Action: identitypkg.ArticleDispositionAction(r.Action), NoticeID: r.NoticeID, NewAuthorID: r.NewAuthorID, OperationID: r.OperationID, AuditEventID: r.AuditEventID, BeforeArticle: orphanPGState(r.BeforeArticle)}
}
func orphanPGState(s article.DispositionArticleState) identitypkg.ArticleDispositionState {
	var category *string
	if s.Category != nil {
		v := string(*s.Category)
		category = &v
	}
	return identitypkg.ArticleDispositionState{Exists: s.Exists, ID: s.ID, Version: s.Version, AuthorID: s.AuthorID, SourceID: s.SourceID, Publication: string(s.Publication), Violation: string(s.Violation), Category: category, Deletion: string(s.Deletion), OperationID: s.OperationID}
}
func orphanPGFacts(f article.DispositionCompletionFacts) identitypkg.VerifiedArticleCompletion {
	r := identitypkg.VerifiedArticleCompletion{AfterArticle: orphanPGState(f.AfterArticle), RemainingActiveCount: f.RemainingActiveCount, RemainingPendingCount: f.RemainingPendingCount, Audit: identitypkg.ArticleDispositionAudit{ID: f.Audit.ID, ArticleID: f.Audit.TargetID, Action: f.Audit.Action, Outcome: f.Audit.Outcome, TraceID: f.Audit.TraceID}}
	if f.Notice != nil {
		r.Notice = &identitypkg.ArticleDispositionNotice{ID: f.Notice.ID, ArticleID: f.Notice.ArticleRef, NewAuthorID: f.Notice.NewAuthorID, FormerDeletedUserID: f.Notice.FormerDeletedUserID}
	}
	for _, v := range f.Recipients {
		r.Recipients = append(r.Recipients, identitypkg.ArticleDispositionRecipient{Kind: string(v.Kind), EmailSnapshot: v.EmailSnapshot, Status: string(v.Status)})
	}
	if o := f.CompletedOperation; o != nil {
		r.CompletedOperation = &identitypkg.ArticleDispositionOperation{ID: o.ID, ArticleID: o.ArticleID, ArticleVersion: o.ArticleVersion, DeleteKind: string(o.DeleteKind), SourceID: o.OrphanedFromUserID, Status: string(o.Status)}
		if o.AuditEventID != nil {
			r.CompletedOperation.AuditEventID = *o.AuditEventID
		}
	}
	return r
}

type orphanPGCase struct {
	t          *testing.T
	h          *schemaHarness
	ctx        context.Context
	ids        fixtureIDs
	clock      *orphanPGClock
	identity   *orphanPGIdentity
	transactor orphanPGTransactor
	command    *article.CommandService
	journal    article.ArticleDeletionJournal
	wg         sync.WaitGroup
}

func newOrphanPGCase(t *testing.T) *orphanPGCase {
	t.Helper()
	h := newOrphanPGHarness(t)
	base, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	ctx, err := platform.ContextWithTraceID(base, orphanPGTrace)
	orphanPGMust(t, err)
	c := &orphanPGCase{t: t, h: h, ctx: ctx, ids: seedFixtures(t, h, ctx), journal: &orphanPGJournal{}}
	c.clock = &orphanPGClock{now: c.ids.now.Add(time.Hour + 123*time.Nanosecond)}
	c.identity = &orphanPGIdentity{audit: audit.Store{}, clock: c.clock}
	c.transactor = orphanPGTransactor{harness: h}
	c.command = c.newCommand(c.transactor, audit.Store{})
	t.Cleanup(func() { cancel(); c.wg.Wait() })
	return c
}
func (c *orphanPGCase) launch(run func()) {
	c.wg.Add(1)
	go func() { defer c.wg.Done(); run() }()
}
func orphanPGMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("operation failed (SQLSTATE=%s): %v", orphanPGSQLState(err), err)
	}
}
func orphanPGSQLState(err error) string {
	var p *pgconn.PgError
	if errors.As(err, &p) {
		return p.Code
	}
	return ""
}
func (c *orphanPGCase) newCommand(tx article.ArticleTransactor, appender article.DeletionAuditor) *article.CommandService {
	c.t.Helper()
	s, err := article.NewCommandService(article.CommandDependencies{Transactions: tx, Repository: article.Repository{}, IDs: identitypkg.RandomUUIDGenerator{}, Clock: c.clock,
		Disposition: &article.OrphanDispositionDependencies{Identity: c.identity, Auditor: appender, RecipientMaxAttempts: 3},
		Orphaning:   &article.OrphanArticlesCanonicalDependencies{Repository: article.CanonicalOrphanRepository{}, Auditor: audit.Store{}, AuditIDs: identitypkg.RandomUUIDGenerator{}, Clock: c.clock}})
	orphanPGMust(c.t, err)
	return s
}
func (c *orphanPGCase) admin() session.Principal {
	return session.Principal{UserID: c.ids.initialAdmin, Role: "admin", Status: "active", IsInitialAdmin: true}
}
func (c *orphanPGCase) insert(source *string, author *string, violating bool) string {
	c.t.Helper()
	a := article.Article{ID: randomUUID(c.t), AuthorID: author, OrphanedFromUserID: source, Title: "synthetic", Body: "synthetic", PublicationState: article.PublicationPublished, ViolationState: article.ViolationClear, DeletionState: article.DeletionActive, Version: 1, CreatedAt: c.clock.Now(), UpdatedAt: c.clock.Now()}
	if source != nil {
		a.PublicationState = article.PublicationHeld
	}
	if violating {
		v := article.ViolationViolence
		a.ViolationState = article.ViolationViolating
		a.ViolationCategory = &v
	}
	tx := c.h.begin(c.t, c.ctx)
	defer func() { _ = tx.Rollback(context.Background()) }()
	orphanPGMust(c.t, (article.Repository{}).InsertArticle(c.ctx, tx, a))
	orphanPGMust(c.t, tx.Commit(c.ctx))
	return a.ID
}
func (c *orphanPGCase) orphan() string { return c.insert(&c.ids.deletedMember, nil, false) }
func (c *orphanPGCase) publish(id string) error {
	_, err := c.command.PublishOrphan(c.ctx, c.admin(), id, 1)
	return err
}
func (c *orphanPGCase) transfer(id string) error {
	_, err := c.command.TransferOrphan(c.ctx, c.admin(), id, 1, c.ids.member)
	return err
}
func (c *orphanPGCase) finalizer(tx article.ArticleTransactor, repo article.DeletionFinalizerRepository, appender article.DeletionAuditor) *article.DeletionFinalizer {
	c.t.Helper()
	f, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{Transactions: tx, Repository: repo, Journal: c.journal, Auditor: appender, AuditIDs: identitypkg.RandomUUIDGenerator{}, Clock: c.clock, WorkerID: "s62-pg", LeaseDuration: time.Minute, Identity: c.identity})
	orphanPGMust(c.t, err)
	return f
}
func (c *orphanPGCase) coordinator(f article.DeletionOperationFinalizer) *article.PermanentDeletionCoordinator {
	c.t.Helper()
	d, err := article.NewPermanentDeletionCoordinator(article.PermanentDeletionDependencies{Transactions: c.transactor, Repository: article.Repository{}, Journal: c.journal, Finalizer: f, OperationIDs: identitypkg.RandomUUIDGenerator{}, Clock: c.clock})
	orphanPGMust(c.t, err)
	return d
}

type orphanPGDeferredFinalizer struct{}

func (orphanPGDeferredFinalizer) Finalize(context.Context, string, string) error {
	return errors.New("fixture stops after committed T1 and journal")
}
func (c *orphanPGCase) deletionRequest(id string, kind article.DeleteKind) article.PermanentDeletionRequest {
	return article.PermanentDeletionRequest{ArticleID: id, IfMatch: `"article-` + id + `-v1"`, DeleteKind: kind, TraceID: orphanPGTrace}
}
func (c *orphanPGCase) prepareDeletion(id string, kind article.DeleteKind) string {
	c.t.Helper()
	err := c.coordinator(orphanPGDeferredFinalizer{}).DeletePermanent(c.ctx, c.admin(), c.deletionRequest(id, kind))
	if !errors.Is(err, article.ErrDeletionInProgress) {
		c.t.Fatalf("T1 control did not stop before T2: %v", err)
	}
	var op string
	orphanPGMust(c.t, c.h.queryRow(c.t, c.ctx, "SELECT deletion_operation_id FROM articles WHERE id=$1", id).Scan(&op))
	return op
}
func (c *orphanPGCase) scalar(query string, args ...any) int64 {
	c.t.Helper()
	var n int64
	orphanPGMust(c.t, c.h.queryRow(c.t, c.ctx, query, args...).Scan(&n))
	return n
}
func (c *orphanPGCase) retained(source string, want bool) {
	c.t.Helper()
	var kept bool
	var emails, other int64
	orphanPGMust(c.t, c.h.queryRow(c.t, c.ctx, `SELECT retained_email IS NOT NULL,(SELECT count(*) FROM identity_identifiers WHERE user_id=$1 AND kind='email'),(SELECT count(*) FROM identity_identifiers WHERE user_id=$1 AND kind IN ('username','phone')) FROM users WHERE id=$1`, source).Scan(&kept, &emails, &other))
	if kept != want || (emails == 1) != want || other != 2 {
		c.t.Fatalf("identity release invariant: retained=%t emails=%d other=%d want_retained=%t", kept, emails, other, want)
	}
}
func (c *orphanPGCase) releases(source string) int64 {
	return c.scalar("SELECT count(*) FROM audit_events WHERE action='identity.identifier.email.release' AND target_id=$1", source)
}
func (c *orphanPGCase) snapshot() [32]byte {
	c.t.Helper()
	var body string
	orphanPGMust(c.t, c.h.queryRow(c.t, c.ctx, `SELECT jsonb_build_array(
 (SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM users v),
 (SELECT jsonb_agg(to_jsonb(v) ORDER BY user_id,kind) FROM identity_identifiers v),
 (SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM articles v),
 (SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM deletion_operations v),
 (SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM transfer_notices v),
 (SELECT jsonb_agg(to_jsonb(v) ORDER BY notice_id,recipient_kind) FROM transfer_notice_recipients v),
 (SELECT jsonb_agg(to_jsonb(v) ORDER BY id) FROM audit_events v))::text`).Scan(&body))
	return sha256.Sum256([]byte(body))
}
func (c *orphanPGCase) rolledBack(before [32]byte, err error) {
	c.t.Helper()
	if err == nil {
		c.t.Fatal("fault unexpectedly committed")
	}
	if c.snapshot() != before {
		c.t.Fatal("fault left a partial database mutation (snapshot hashes differ)")
	}
}
func (c *orphanPGCase) canonical(target string, beforeArticle func(context.Context, pgx.Tx) error) error {
	port := identitypkg.ArticleAccountPortFunc(func(ctx context.Context, tx pgx.Tx, id string) (identitypkg.CanonicalDeleteArticleResult, error) {
		if beforeArticle != nil {
			if err := beforeArticle(ctx, tx); err != nil {
				return identitypkg.CanonicalDeleteArticleResult{}, err
			}
		}
		r, err := c.command.OrphanArticlesCanonical(ctx, tx, id)
		return identitypkg.CanonicalDeleteArticleResult{NewlyOrphanedCount: r.NewlyOrphanedCount, AlreadyOrphanedCount: r.AlreadyOrphanedCount, PendingOrphanCount: r.PendingOrphanCount}, err
	})
	d, err := identitypkg.NewCanonicalDeleteService(identitypkg.CanonicalDeleteDependencies{Transactions: c.transactor, Repository: identitypkg.PostgresCanonicalDeleteRepository{}, Articles: port, Audit: audit.Store{}, IDs: identitypkg.RandomUUIDGenerator{}, Clock: c.clock})
	if err != nil {
		return err
	}
	_, err = d.CanonicalDelete(c.ctx, identitypkg.CanonicalDeleteRequest{Actor: c.admin(), TargetID: target, TargetRole: identitypkg.RoleMember, Mode: identitypkg.CanonicalDeleteDailyGovernance, Reason: identitypkg.AccountDeletionGovernance, TraceID: orphanPGTrace})
	return err
}

func TestOrphanDispositionPostgres(t *testing.T) {
	groups := []struct {
		name string
		run  func(*testing.T)
	}{
		{"01_CanonicalCompatibility", testOrphanPGCanonical},
		{"02_T2ReverseLockRetreat", testOrphanPGLocks},
		{"03_HTTPWorkerConsumption", testOrphanPGConsumers},
		{"04_LastOfTwoAndPending", testOrphanPGLastTwo},
		{"05_CompletionReferenceBinding", testOrphanPGReferences},
		{"06_SnapshotsZeroRowsDrift", testOrphanPGDrift},
		{"07_AtomicFaultsAndUnknownCommit", testOrphanPGAtomic},
		{"08_N001ContactsAndReplay", testOrphanPGN001},
		{"09_F001PendingCanonicalCompletion", testOrphanPGF001},
		{"10_N002PrecisionLeaseAndRealJournal", testOrphanPGN002},
	}
	for _, g := range groups {
		t.Run(g.name, g.run)
	}
}
func orphanPGWait(t *testing.T, ctx context.Context, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("barrier deadline")
	}
}
func orphanPGJoin(t *testing.T, ctx context.Context, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		t.Fatal("goroutine join deadline")
		return ctx.Err()
	}
}
func (c *orphanPGCase) hold(query string, args ...any) pgx.Tx {
	c.t.Helper()
	tx := c.h.begin(c.t, c.ctx)
	c.t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	_, err := tx.Exec(c.ctx, query, args...)
	orphanPGMust(c.t, err)
	return tx
}
func (c *orphanPGCase) nowait(err error) {
	c.t.Helper()
	if orphanPGSQLState(err) != "55P03" {
		c.t.Fatalf("expected native NOWAIT SQLSTATE 55P03; got %s: %v", orphanPGSQLState(err), err)
	}
	c.t.Log("native SQLSTATE=55P03; no lock_timeout configured")
}
func (c *orphanPGCase) checkConnection() {
	c.t.Helper()
	tx1 := c.h.begin(c.t, c.ctx)
	defer func() { _ = tx1.Rollback(context.Background()) }()
	tx2 := c.h.begin(c.t, c.ctx)
	defer func() { _ = tx2.Rollback(context.Background()) }()
	p1, p2 := tx1.Conn().PgConn().PID(), tx2.Conn().PgConn().PID()
	if p1 == p2 {
		c.t.Fatal("independent connections required")
	}
	c.t.Logf("independent_backend_pids=%d,%d", p1, p2)
}
func (c *orphanPGCase) cloneTarget(id string) {
	c.t.Helper()
	tx := c.h.begin(c.t, c.ctx)
	defer func() { _ = tx.Rollback(context.Background()) }()
	_, err := tx.Exec(c.ctx, `INSERT INTO users(id,username,email,phone,role,status,credential_mode,created_at,updated_at)
 SELECT $1,'s62-target','s62-target@example.test','+15555550001','member','active','permanent',created_at,updated_at FROM users WHERE id=$2`, id, c.ids.member)
	orphanPGMust(c.t, err)
	_, err = tx.Exec(c.ctx, `INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at)
 VALUES($1,'username','s62-target','active',$2),($1,'email','s62-target@example.test','active',$2),($1,'phone','+15555550001','active',$2)`, id, c.clock.Now())
	orphanPGMust(c.t, err)
	orphanPGMust(c.t, tx.Commit(c.ctx))
	c.ids.member = id
}
func testOrphanPGCanonical(t *testing.T) {
	t.Run("transfer_first_canonical_waits", testOrphanPGTransferFirst)
	t.Run("kept_deleted_source_competition", testOrphanPGKeptDeleted)
	for _, target := range []string{"00000000-0000-4000-8000-000000000001", "ffffffff-ffff-4fff-bfff-ffffffffffff"} {
		t.Run("canonical_first_"+target[:1], func(t *testing.T) { testOrphanPGCanonicalFirst(t, target) })
	}
	for _, lock := range []string{"users", "identity_identifiers"} {
		t.Run("source_"+lock, func(t *testing.T) {
			c := newOrphanPGCase(t)
			id := c.orphan()
			column := "id"
			if lock != "users" {
				column = "user_id"
			}
			holder := c.hold("SELECT 1 FROM "+lock+" WHERE "+column+"=$1 FOR UPDATE", c.ids.deletedMember)
			before := c.snapshot()
			err := c.publish(id)
			c.nowait(err)
			c.rolledBack(before, err)
			orphanPGMust(t, holder.Rollback(c.ctx))
			orphanPGMust(t, c.publish(id))
			c.retained(c.ids.deletedMember, false)
		})
	}
}

type orphanPGFinalizerRepository struct {
	article.Repository
	beforeArticle  func(context.Context, pgx.Tx) error
	afterOperation func(context.Context, pgx.Tx, article.DeletionOperation) error
	afterComplete  func(context.Context, pgx.Tx) error
}

func (r orphanPGFinalizerRepository) CompleteDeletionOperation(ctx context.Context, tx pgx.Tx, id, owner, auditID string, at time.Time) error {
	if err := r.Repository.CompleteDeletionOperation(ctx, tx, id, owner, auditID, at); err != nil {
		return err
	}
	if r.afterComplete != nil {
		return r.afterComplete(ctx, tx)
	}
	return nil
}

func (r orphanPGFinalizerRepository) LockArticleForDisposition(ctx context.Context, tx pgx.Tx, id string) (article.Article, error) {
	if r.beforeArticle != nil {
		if err := r.beforeArticle(ctx, tx); err != nil {
			return article.Article{}, err
		}
	}
	return r.Repository.LockArticleForDisposition(ctx, tx, id)
}
func (r orphanPGFinalizerRepository) LoadDeletionOperationForUpdate(ctx context.Context, tx pgx.Tx, id string) (article.DeletionOperation, error) {
	o, err := r.Repository.LoadDeletionOperationForUpdate(ctx, tx, id)
	if err == nil && r.afterOperation != nil {
		err = r.afterOperation(ctx, tx, o)
	}
	return o, err
}
func (c *orphanPGCase) pendingUnchanged(id, op string) {
	c.t.Helper()
	var status string
	var owner *string
	var until *time.Time
	orphanPGMust(c.t, c.h.queryRow(c.t, c.ctx, "SELECT status,claimed_by,locked_until FROM deletion_operations WHERE id=$1", op).Scan(&status, &owner, &until))
	if status != "finalizing" || owner == nil || *owner != "s62-pg" || until == nil {
		c.t.Fatal("committed lease was lost")
	}
	if c.scalar("SELECT count(*) FROM articles WHERE id=$1 AND deletion_state='deletion_pending'", id) != 1 || c.scalar("SELECT count(*) FROM audit_events WHERE target_id=$1", id) != 0 {
		c.t.Fatal("failed T2 partially completed")
	}
	probe := c.hold("SELECT 1 FROM deletion_operations WHERE id=$1 FOR UPDATE NOWAIT", op)
	orphanPGMust(c.t, probe.Rollback(c.ctx))
}
func testOrphanPGLocks(t *testing.T) {
	t.Run("real_coordinator_operation_article_reverse", testOrphanPGCoordinatorRetreat)
	for _, kind := range []article.DeleteKind{article.DeleteOrdinary, article.DeleteViolating, article.DeleteOrphan} {
		t.Run(string(kind)+"_Article", func(t *testing.T) {
			c := newOrphanPGCase(t)
			var id string
			if kind == article.DeleteOrphan {
				id = c.orphan()
			} else {
				id = c.insert(nil, &c.ids.member, kind == article.DeleteViolating)
			}
			op := c.prepareDeletion(id, kind)
			holder := c.hold("SELECT 1 FROM articles WHERE id=$1 FOR UPDATE", id)
			err := c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
			c.nowait(err)
			c.pendingUnchanged(id, op)
			orphanPGMust(t, holder.Rollback(c.ctx))
			c.clock.set(c.clock.Now().Add(2 * time.Minute))
			orphanPGMust(t, c.finalizer(c.transactor, article.Repository{}, audit.Store{}).ConvergeDeletionOperation(c.ctx, op, orphanPGTrace))
		})
	}
	for _, table := range []string{"users", "identity_identifiers"} {
		t.Run("orphan_"+table, func(t *testing.T) {
			c := newOrphanPGCase(t)
			id := c.orphan()
			op := c.prepareDeletion(id, article.DeleteOrphan)
			column := "id"
			if table != "users" {
				column = "user_id"
			}
			holder := c.hold("SELECT 1 FROM "+table+" WHERE "+column+"=$1 FOR UPDATE", c.ids.deletedMember)
			err := c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
			c.nowait(err)
			c.pendingUnchanged(id, op)
			c.retained(c.ids.deletedMember, true)
			orphanPGMust(t, holder.Rollback(c.ctx))
		})
	}
}

type orphanPGResolver struct{ principal session.Principal }

func (r orphanPGResolver) ResolvePrincipal(context.Context, session.ResolveInput) (session.Principal, error) {
	return r.principal, nil
}

type orphanPGCountedCommands struct {
	*article.CommandService
	calls int
}

func (c *orphanPGCountedCommands) PublishOrphan(ctx context.Context, p session.Principal, id string, v int64) (article.OrphanDispositionResult, error) {
	c.calls++
	return c.CommandService.PublishOrphan(ctx, p, id, v)
}
func (c *orphanPGCountedCommands) TransferOrphan(ctx context.Context, p session.Principal, id string, v int64, target string) (article.OrphanDispositionResult, error) {
	c.calls++
	return c.CommandService.TransferOrphan(ctx, p, id, v, target)
}
func (c *orphanPGCase) httpDisposition(id string, transfer bool) (int, string, int) {
	c.t.Helper()
	commands := &orphanPGCountedCommands{CommandService: c.command}
	handler, err := article.NewOrphanCommandHandler(&article.Handler{}, commands, nil)
	orphanPGMust(c.t, err)
	auth, err := platform.NewHTTPAuthorizer(orphanPGResolver{principal: c.admin()}, c.clock.Now)
	orphanPGMust(c.t, err)
	action, body := "publish", ""
	var serve http.Handler = http.HandlerFunc(handler.PublishOrphan)
	if transfer {
		action = "transfer"
		body = `{"new_author_id":"` + c.ids.member + `"}`
		serve = http.HandlerFunc(handler.TransferOrphan)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orphan-articles/"+id+"/"+action, strings.NewReader(body)).WithContext(c.ctx)
	req.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", `"article-`+id+`-v1"`)
	response := httptest.NewRecorder()
	auth.Protect(session.AccessGeneral, serve).ServeHTTP(response, req)
	return response.Code, response.Body.String(), commands.calls
}
func (c *orphanPGCase) worker() *article.DeletionWorker {
	c.t.Helper()
	discoverer, err := article.NewPostgresDeletionOperationDiscoverer(c.h.pool)
	orphanPGMust(c.t, err)
	worker, err := article.NewDeletionWorker(article.DeletionWorkerDependencies{Discoverer: discoverer, Converger: c.finalizer(c.transactor, article.Repository{}, audit.Store{}), Now: c.clock.Now, NewTraceID: func() (string, error) { return orphanPGTrace, nil }, Sleep: func(ctx context.Context, d time.Duration) error { return nil }, Jitter: func(d time.Duration) time.Duration { return d }}, article.DeletionWorkerConfig{PollInterval: time.Second, BatchSize: 8, OperationBudget: time.Second, BackoffInitial: time.Millisecond, BackoffMax: time.Second, MaxBatchesPerWake: 1, ShutdownBudget: time.Second})
	orphanPGMust(c.t, err)
	return worker
}
func testOrphanPGConsumers(t *testing.T) {
	for _, transfer := range []bool{false, true} {
		t.Run(fmt.Sprintf("HTTP_transfer_%t", transfer), func(t *testing.T) {
			c := newOrphanPGCase(t)
			id := c.orphan()
			holder := c.hold("SELECT 1 FROM users WHERE id=$1 FOR UPDATE", c.ids.deletedMember)
			status, body, calls := c.httpDisposition(id, transfer)
			if status != 503 || !strings.Contains(body, "platform.database_unavailable") || calls != 1 {
				t.Fatalf("HTTP lock consumption status=%d calls=%d", status, calls)
			}
			orphanPGMust(t, holder.Rollback(c.ctx))
			status, _, calls = c.httpDisposition(id, transfer)
			if status != 200 || calls != 1 {
				t.Fatalf("new HTTP retry failed status=%d calls=%d", status, calls)
			}
		})
	}
	t.Run("worker_transient_lease_expiry_retry", func(t *testing.T) { testOrphanPGWorkerTransient(t) })
}

func testOrphanPGLastTwo(t *testing.T) {
	for _, deletion := range []bool{false, true} {
		t.Run(fmt.Sprintf("publish_then_delete_%t", deletion), func(t *testing.T) { testOrphanPGTwoConcurrentDispositions(t, deletion) })
	}
	for _, anomaly := range []bool{false, true} {
		t.Run(fmt.Sprintf("third_pending_or_anomalous_%t", anomaly), func(t *testing.T) {
			c := newOrphanPGCase(t)
			a, b, third := c.orphan(), c.orphan(), c.orphan()
			if anomaly {
				c.h.exec(t, c.ctx, "UPDATE articles SET author_id=$2,orphaned_from_user_id=NULL,publication_state='published' WHERE id=$1", third, c.ids.deletedMember)
			} else {
				c.prepareDeletion(third, article.DeleteOrphan)
			}
			orphanPGMust(t, c.publish(a))
			orphanPGMust(t, c.transfer(b))
			c.retained(c.ids.deletedMember, true)
			if c.releases(c.ids.deletedMember) != 0 {
				t.Fatal("pending/anomalous association was omitted from counts")
			}
			facts := c.identity.lastFacts
			if anomaly && facts.RemainingActiveCount != 1 || !anomaly && facts.RemainingPendingCount != 1 {
				t.Fatal("same-snapshot association counts mismatch")
			}
		})
	}
}

type orphanPGReferenceMutation struct {
	name   string
	change func(*identitypkg.ArticleCompletionReference)
}

func orphanPGReferenceMutations() []orphanPGReferenceMutation {
	other := "00000000-0000-4000-8000-000000000099"
	return []orphanPGReferenceMutation{
		{"article", func(r *identitypkg.ArticleCompletionReference) { r.ExpectedArticleID = other }},
		{"source", func(r *identitypkg.ArticleCompletionReference) { r.SourceID = other }},
		{"version", func(r *identitypkg.ArticleCompletionReference) { r.BeforeVersion++ }},
		{"action", func(r *identitypkg.ArticleCompletionReference) { r.Action = "untrusted" }},
		{"audit", func(r *identitypkg.ArticleCompletionReference) { r.AuditEventID = other }},
		{"notice", func(r *identitypkg.ArticleCompletionReference) { r.NoticeID = &other }},
		{"new_author", func(r *identitypkg.ArticleCompletionReference) { r.NewAuthorID = &other }},
		{"operation", func(r *identitypkg.ArticleCompletionReference) { r.OperationID = &other }},
		{"before_exists", func(r *identitypkg.ArticleCompletionReference) { r.BeforeArticle.Exists = false }},
		{"before_id", func(r *identitypkg.ArticleCompletionReference) { r.BeforeArticle.ID = other }},
		{"before_version", func(r *identitypkg.ArticleCompletionReference) { r.BeforeArticle.Version++ }},
		{"before_author", func(r *identitypkg.ArticleCompletionReference) { r.BeforeArticle.AuthorID = &other }},
		{"before_source", func(r *identitypkg.ArticleCompletionReference) { r.BeforeArticle.SourceID = &other }},
		{"before_publication", func(r *identitypkg.ArticleCompletionReference) { r.BeforeArticle.Publication = "published" }},
		{"before_violation", func(r *identitypkg.ArticleCompletionReference) { r.BeforeArticle.Violation = "violating" }},
		{"before_category", func(r *identitypkg.ArticleCompletionReference) { v := "sexual"; r.BeforeArticle.Category = &v }},
		{"before_deletion", func(r *identitypkg.ArticleCompletionReference) { r.BeforeArticle.Deletion = "invalid" }},
		{"before_operation", func(r *identitypkg.ArticleCompletionReference) { r.BeforeArticle.OperationID = &other }},
	}
}
func testOrphanPGReferences(t *testing.T) {
	t.Run("persisted_operation_fields", testOrphanPGOperationBindings)
	for _, action := range []string{"publish", "transfer", "delete"} {
		t.Run(action, func(t *testing.T) {
			for _, mutation := range orphanPGReferenceMutations() {
				t.Run(mutation.name, func(t *testing.T) { testOrphanPGReferenceField(t, action, mutation) })
			}
			for _, life := range []string{"foreign_tx", "ended_tx", "trace"} {
				t.Run(life, func(t *testing.T) { testOrphanPGReferenceLifetime(t, action, life) })
			}
			t.Run("success_and_reentry_then_ended", func(t *testing.T) { testOrphanPGReferenceSuccess(t, action) })
		})
	}
}

type orphanPGSQLFault struct{ name, sql string }

func orphanPGTransferFaults() []orphanPGSQLFault {
	return []orphanPGSQLFault{
		{"notice_article", `UPDATE transfer_notices SET article_ref='00000000-0000-4000-8000-000000000099' WHERE id=(SELECT id FROM transfer_notices LIMIT 1)`},
		{"notice_source", `UPDATE transfer_notices SET former_deleted_user_id=new_author_id WHERE id=(SELECT id FROM transfer_notices LIMIT 1)`},
		{"notice_target", `UPDATE transfer_notices SET new_author_id=former_deleted_user_id WHERE id=(SELECT id FROM transfer_notices LIMIT 1)`},
		{"missing_new_recipient", `DELETE FROM transfer_notice_recipients WHERE recipient_kind='new_author'`},
		{"missing_former_recipient", `DELETE FROM transfer_notice_recipients WHERE recipient_kind='former_deleted_author'`},
		{"duplicate_kind", `UPDATE transfer_notice_recipients SET recipient_kind='new_author' WHERE recipient_kind='former_deleted_author'`},
		{"new_address", `UPDATE transfer_notice_recipients SET email_snapshot='wrong@example.test' WHERE recipient_kind='new_author'`},
		{"former_address", `UPDATE transfer_notice_recipients SET email_snapshot='wrong@example.test' WHERE recipient_kind='former_deleted_author'`},
		{"recipient_not_pending", `UPDATE transfer_notice_recipients SET status='failed_terminal'`},
	}
}
func (c *orphanPGCase) auditSQL(sql string) orphanPGAudit {
	return orphanPGAudit{after: func(ctx context.Context, tx pgx.Tx, e audit.Event) error { _, err := tx.Exec(ctx, sql); return err }}
}
func (c *orphanPGCase) trigger(table, event, body string) {
	c.t.Helper()
	sql := `CREATE FUNCTION s62_fault() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN ` + body + `; END $$; CREATE TRIGGER s62_fault BEFORE ` + event + ` ON ` + table + ` FOR EACH ROW EXECUTE FUNCTION s62_fault()`
	c.h.exec(c.t, c.ctx, sql)
}
func testOrphanPGDrift(t *testing.T) {
	for _, fault := range orphanPGTransferFaults() {
		t.Run(fault.name, func(t *testing.T) {
			c := newOrphanPGCase(t)
			id := c.orphan()
			c.command = c.newCommand(c.transactor, c.auditSQL(fault.sql))
			before := c.snapshot()
			c.rolledBack(before, c.transfer(id))
		})
	}
	for _, table := range []string{"articles", "transfer_notices", "transfer_notice_recipients", "users", "identity_identifiers"} {
		t.Run("zero_"+table, func(t *testing.T) { testOrphanPGZeroRows(t, table) })
	}
	for _, drift := range []string{"retained_absent", "identifier_absent", "binding", "user_active", "source_missing"} {
		t.Run(drift, func(t *testing.T) { testOrphanPGIdentityDrift(t, drift) })
	}
	for _, absent := range []bool{false, true} {
		t.Run(fmt.Sprintf("delete_absent_%t", absent), func(t *testing.T) { testOrphanPGDeleteAbsence(t, absent) })
	}
}

func testOrphanPGAtomic(t *testing.T) {
	t.Run("T2_commit_outcome_unknown", testOrphanPGT2CommitUnknown)
	for _, point := range []string{"disposition_audit", "verifier", "release_audit", "article_update", "notice_insert", "recipient_insert"} {
		t.Run("transfer_"+point, func(t *testing.T) { testOrphanPGTransferFault(t, point) })
	}
	for _, ackLoss := range []bool{false, true} {
		t.Run(fmt.Sprintf("commit_ack_loss_%t", ackLoss), func(t *testing.T) { testOrphanPGTransferCommitOutcome(t, ackLoss) })
	}
	for _, point := range []string{"completion_audit", "completed_operation", "verifier", "release_audit"} {
		t.Run("T2_"+point, func(t *testing.T) { testOrphanPGT2Fault(t, point) })
	}
}
func testOrphanPGN001(t *testing.T) {
	t.Run("violating_transfer_preserves_classification", testOrphanPGViolatingTransfer)
	for _, action := range []string{"publish", "delete"} {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_empty_%t", action, empty), func(t *testing.T) {
				c := newOrphanPGCase(t)
				id := c.orphan()
				if empty {
					c.h.exec(t, c.ctx, "DELETE FROM identity_identifiers WHERE user_id=$1 AND kind='email'", c.ids.deletedMember)
					c.h.exec(t, c.ctx, "UPDATE users SET retained_email=NULL WHERE id=$1", c.ids.deletedMember)
				}
				var err error
				if action == "publish" {
					err = c.publish(id)
				} else {
					op := c.prepareDeletion(id, article.DeleteOrphan)
					err = c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
				}
				orphanPGMust(t, err)
				c.retained(c.ids.deletedMember, false)
				if c.scalar("SELECT count(*) FROM transfer_notices") != 0 || c.releases(c.ids.deletedMember) != map[bool]int64{true: 0, false: 1}[empty] {
					t.Fatal("nontransfer unexpectedly required contacts or Notice")
				}
			})
		}
	}
	for _, missing := range []string{"none", "source", "target", "binding", "suspended", "deleted"} {
		t.Run("transfer_"+missing, func(t *testing.T) { testOrphanPGTransferContact(t, missing) })
	}
	t.Run("absent_source_free_replay_does_not_guess", testOrphanPGSourceFreeReplay)
}

func testOrphanPGF001(t *testing.T) {
	t.Run("active_author_lock_blocks_canonical_until_commit", testOrphanPGActiveAuthorCanonicalWait)
	t.Run("absent_ordinary_does_not_guess_source", testOrphanPGAbsentOrdinary)
	for _, kind := range []article.DeleteKind{article.DeleteOrdinary, article.DeleteViolating} {
		t.Run(string(kind), func(t *testing.T) { testOrphanPGAuthoredDeletionKind(t, kind) })
	}
}

func testOrphanPGN002(t *testing.T) {
	t.Run("clock_expires_during_real_operation_lock_wait", testOrphanPGFreshClockWait)
	t.Run("submicrosecond_T1_native_identity", func(t *testing.T) {
		c := newOrphanPGCase(t)
		c.clock.set(time.Date(2026, 10, 27, 11, 0, 0, 123456789, time.FixedZone("fixture", 9*3600)))
		id := c.orphan()
		f := c.finalizer(c.transactor, article.Repository{}, audit.Store{})
		orphanPGMust(t, c.coordinator(f).DeletePermanent(c.ctx, c.admin(), c.deletionRequest(id, article.DeleteOrphan)))
		var op string
		var started time.Time
		orphanPGMust(t, c.h.queryRow(t, c.ctx, "SELECT id,started_at FROM deletion_operations WHERE article_id=$1", id).Scan(&op, &started))
		tombstone, err := c.journal.FindByOperation(c.ctx, op)
		orphanPGMust(t, err)
		if c.clock.Now().Nanosecond()%1000 == 0 || !started.Equal(tombstone.DeletedAt) || tombstone.DeletedAt.Nanosecond()%1000 != 0 {
			t.Fatal("T1 normalization lost nanosecond input control or persisted/journal equality")
		}
		c.retained(c.ids.deletedMember, false)
		if c.identity.calls == 0 || c.scalar("SELECT count(*) FROM articles WHERE id=$1", id) != 0 {
			t.Fatal("precision success omitted real cleanup/delete")
		}
		orphanPGMust(t, c.worker().Reconcile(c.ctx))
		if c.releases(c.ids.deletedMember) != 1 {
			t.Fatal("precision followup duplicated release")
		}
	})
	t.Run("PG_returned_lease_is_authority", func(t *testing.T) {
		c := newOrphanPGCase(t)
		id := c.orphan()
		op := c.prepareDeletion(id, article.DeleteOrphan)
		tx := c.h.begin(t, c.ctx)
		defer func() { _ = tx.Rollback(context.Background()) }()
		now := c.clock.Now().In(time.FixedZone("fixture", -5*3600))
		candidate := now.Add(time.Minute)
		got, claimed, err := (article.Repository{}).ClaimDeletionOperation(c.ctx, tx, op, "s62-pg", now, candidate)
		orphanPGMust(t, err)
		if !claimed || got.LockedUntil == nil || got.LockedUntil.Location() != time.UTC || got.LockedUntil.Nanosecond()%1000 != 0 || !got.LockedUntil.Equal(candidate.UTC().Truncate(time.Microsecond)) {
			t.Fatal("claim did not return database UTC microsecond authority")
		}
		var saved time.Time
		orphanPGMust(t, tx.QueryRow(c.ctx, "SELECT locked_until FROM deletion_operations WHERE id=$1", op).Scan(&saved))
		if !saved.Equal(*got.LockedUntil) {
			t.Fatal("claim lease differs from PostgreSQL readback")
		}
		orphanPGMust(t, tx.Commit(c.ctx))
	})
	for _, mismatch := range []string{"expired_after_lock", "owner", "status", "lease"} {
		t.Run(mismatch, func(t *testing.T) { testOrphanPGRejectedLease(t, mismatch) })
	}
	t.Run("production_file_journal_host_contract", testOrphanPGFileJournal)
}
func testOrphanPGFileJournal(t *testing.T) {
	c := newOrphanPGCase(t)
	key := article.DeletionJournalKey{ID: "s62-fixture", Version: "v1", Material: []byte("synthetic-s62-journal-key-material")}
	directory, err := os.MkdirTemp("", "han_s62_journal_")
	orphanPGMust(t, err)
	t.Logf("retained_journal_directory=%s", directory)
	journal, err := article.NewFileArticleDeletionJournal(article.ArticleDeletionJournalOptions{Directory: directory, FileMode: 0o600}, article.DeletionJournalKeyring{Active: key, Retained: []article.DeletionJournalKey{key}})
	if errors.Is(err, article.ErrDeletionJournalUnsupportedPlatform) {
		t.Log("production Journal correctly fails closed on this Host; real fsync success NOT VERIFIED; PG matrix uses explicit Journal fixture")
		return
	}
	orphanPGMust(t, err)
	t.Cleanup(func() { orphanPGMust(t, journal.Close()) })
	c.journal = journal
	id := c.orphan()
	orphanPGMust(t, c.coordinator(c.finalizer(c.transactor, article.Repository{}, audit.Store{})).DeletePermanent(c.ctx, c.admin(), c.deletionRequest(id, article.DeleteOrphan)))
	records, err := journal.ReadAllVerified(c.ctx)
	orphanPGMust(t, err)
	if len(records) != 1 {
		t.Fatal("real journal record absent")
	}
	c.retained(c.ids.deletedMember, false)
}

// Observation-only poll: the lock authority remains PostgreSQL. A deadline only
// prevents a hung test; neither lock_timeout nor a sleep decides who holds rows.
func (c *orphanPGCase) waitBlocked(blocker, waiter uint32) {
	c.t.Helper()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		err := c.h.pool.QueryRow(c.ctx, "SELECT $1::int = ANY(pg_blocking_pids($2::int))", blocker, waiter).Scan(&blocked)
		orphanPGMust(c.t, err)
		if blocked {
			c.t.Logf("observed_blocker_pid=%d waiter_pid=%d", blocker, waiter)
			return
		}
		select {
		case <-ticker.C:
		case <-c.ctx.Done():
			c.t.Fatal("expected PostgreSQL lock wait not observed")
		}
	}
}
func testOrphanPGCoordinatorRetreat(t *testing.T) {
	c := newOrphanPGCase(t)
	id := c.insert(nil, &c.ids.member, false)
	op := c.prepareDeletion(id, article.DeleteOrdinary)
	pid := make(chan uint32, 2)
	done := make(chan error, 1)
	retryTx := c.transactor
	retryTx.wrap = func(tx pgx.Tx) pgx.Tx { pid <- tx.Conn().PgConn().PID(); return tx }
	retry, err := article.NewPermanentDeletionCoordinator(article.PermanentDeletionDependencies{Transactions: retryTx, Repository: article.Repository{}, Journal: c.journal, Finalizer: orphanPGDeferredFinalizer{}, OperationIDs: identitypkg.RandomUUIDGenerator{}, Clock: c.clock})
	orphanPGMust(t, err)
	repo := orphanPGFinalizerRepository{beforeArticle: func(ctx context.Context, tx pgx.Tx) error {
		c.launch(func() { done <- retry.DeletePermanent(ctx, c.admin(), c.deletionRequest(id, article.DeleteOrdinary)) })
		var waiter uint32
		select {
		case waiter = <-pid:
		case <-ctx.Done():
			return ctx.Err()
		}
		c.waitBlocked(tx.Conn().PgConn().PID(), waiter)
		return nil
	}}
	err = c.finalizer(c.transactor, repo, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
	c.nowait(err)
	retryErr := orphanPGJoin(t, c.ctx, done)
	if !errors.Is(retryErr, article.ErrDeletionInProgress) {
		t.Fatalf("coordinator did not advance after T2 rollback: %v", retryErr)
	}
	c.pendingUnchanged(id, op)
}
func testOrphanPGTransferFirst(t *testing.T) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	ready, release := make(chan struct{}), make(chan struct{})
	transferDone := make(chan error, 1)
	var blocker uint32
	c.identity.hook = func(ctx context.Context, tx pgx.Tx, p *article.DispositionCompletionProof, l *identitypkg.ArticleIdentityLifecycle, r identitypkg.ArticleCompletionReference) error {
		blocker = tx.Conn().PgConn().PID()
		close(ready)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	c.launch(func() { transferDone <- c.transfer(id) })
	orphanPGWait(t, c.ctx, ready)
	pid := make(chan uint32, 1)
	canonicalDone := make(chan error, 1)
	c.transactor.wrap = func(tx pgx.Tx) pgx.Tx { pid <- tx.Conn().PgConn().PID(); return tx }
	c.launch(func() { canonicalDone <- c.canonical(c.ids.member, nil) })
	var waiter uint32
	select {
	case waiter = <-pid:
	case <-c.ctx.Done():
		t.Fatal("canonical begin deadline")
	}
	c.waitBlocked(blocker, waiter)
	close(release)
	orphanPGMust(t, orphanPGJoin(t, c.ctx, transferDone))
	orphanPGMust(t, orphanPGJoin(t, c.ctx, canonicalDone))
	if c.scalar("SELECT count(*) FROM articles WHERE id=$1 AND orphaned_from_user_id=$2 AND publication_state='held' AND author_id IS NULL", id, c.ids.member) != 1 {
		t.Fatal("canonical did not see committed new ownership")
	}
	c.retained(c.ids.deletedMember, false)
	c.retained(c.ids.member, true)
}
func testOrphanPGFreshClockWait(t *testing.T) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	op := c.prepareDeletion(id, article.DeleteOrphan)
	ready := make(chan uint32, 1)
	done := make(chan error, 1)
	var holder pgx.Tx
	begins := 0
	tx := c.transactor
	tx.wrap = func(inner pgx.Tx) pgx.Tx {
		begins++
		if begins == 2 {
			holder = c.h.begin(t, c.ctx)
			t.Cleanup(func() { _ = holder.Rollback(context.Background()) })
			_, err := holder.Exec(c.ctx, "SELECT 1 FROM deletion_operations WHERE id=$1 FOR UPDATE", op)
			orphanPGMust(t, err)
			ready <- inner.Conn().PgConn().PID()
		}
		return inner
	}
	c.launch(func() {
		done <- c.finalizer(tx, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
	})
	var waiter uint32
	select {
	case waiter = <-ready:
	case <-c.ctx.Done():
		t.Fatal("T2 begin deadline")
	}
	c.waitBlocked(holder.Conn().PgConn().PID(), waiter)
	var lease time.Time
	orphanPGMust(t, holder.QueryRow(c.ctx, "SELECT locked_until FROM deletion_operations WHERE id=$1", op).Scan(&lease))
	c.clock.set(lease.Add(time.Microsecond))
	orphanPGMust(t, holder.Rollback(c.ctx))
	err := orphanPGJoin(t, c.ctx, done)
	if !errors.Is(err, article.ErrDeletionInProgress) {
		t.Fatalf("clock was not refreshed after lock acquisition: %v", err)
	}
	c.pendingUnchanged(id, op)
	c.retained(c.ids.deletedMember, true)
}
func testOrphanPGActiveAuthorCanonicalWait(t *testing.T) {
	c := newOrphanPGCase(t)
	id := c.insert(nil, &c.ids.member, false)
	op := c.prepareDeletion(id, article.DeleteOrdinary)
	ready, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	var blocker uint32
	appender := orphanPGAudit{after: func(ctx context.Context, tx pgx.Tx, e audit.Event) error {
		blocker = tx.Conn().PgConn().PID()
		close(ready)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	c.launch(func() {
		done <- c.finalizer(c.transactor, article.Repository{}, appender).Finalize(c.ctx, op, orphanPGTrace)
	})
	orphanPGWait(t, c.ctx, ready)
	pid := make(chan uint32, 1)
	canonicalDone := make(chan error, 1)
	c.transactor.wrap = func(tx pgx.Tx) pgx.Tx { pid <- tx.Conn().PgConn().PID(); return tx }
	c.launch(func() { canonicalDone <- c.canonical(c.ids.member, nil) })
	var waiter uint32
	select {
	case waiter = <-pid:
	case <-c.ctx.Done():
		t.Fatal("canonical begin deadline")
	}
	c.waitBlocked(blocker, waiter)
	close(release)
	orphanPGMust(t, orphanPGJoin(t, c.ctx, done))
	orphanPGMust(t, orphanPGJoin(t, c.ctx, canonicalDone))
	c.retained(c.ids.member, false)
	if c.identity.calls != 0 {
		t.Fatal("active-author T2 generated cleanup proof")
	}
}

// A recovery environment input, not an article verifier or run authority. The
// concrete RecoveryRunStore below still locks and validates the persisted run.
type orphanPGMaintenanceGate struct{}

func (orphanPGMaintenanceGate) CheckRecoveryOpenGate(context.Context, pgx.Tx) error { return nil }
func testOrphanPGKeptDeleted(t *testing.T) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	runID, backup := "s62-run-"+randomHex(t, 8), "synthetic-backup"
	store, err := identitypkg.NewRecoveryRunStore(c.h.pool)
	orphanPGMust(t, err)
	orphanPGMust(t, store.Create(c.ctx, recovery.RecoveryRun{RunID: runID, BackupRef: backup, JournalKeyID: "fixture-key", JournalKeyVersion: "v1", JournalSecretFingerprint: "synthetic-fingerprint", JournalStage: recovery.StageBegun, StartedAt: c.ids.now, Outcome: recovery.RecoveryOutcomeRunning}))
	c.h.exec(t, c.ctx, `INSERT INTO recovery_account_reviews(recovery_run_id,user_id,backup_role,backup_status,backup_deleted_at,decision) VALUES($1,$2,'member','deleted',$3,'pending')`, runID, c.ids.deletedMember, c.ids.now)
	ready, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	port := identitypkg.ArticleAccountPortFunc(func(ctx context.Context, tx pgx.Tx, source string) (identitypkg.CanonicalDeleteArticleResult, error) {
		close(ready)
		select {
		case <-release:
		case <-ctx.Done():
			return identitypkg.CanonicalDeleteArticleResult{}, ctx.Err()
		}
		r, err := c.command.OrphanArticlesCanonical(ctx, tx, source)
		return identitypkg.CanonicalDeleteArticleResult{NewlyOrphanedCount: r.NewlyOrphanedCount, AlreadyOrphanedCount: r.AlreadyOrphanedCount, PendingOrphanCount: r.PendingOrphanCount}, err
	})
	coordinator, err := identitypkg.NewCanonicalDeleteService(identitypkg.CanonicalDeleteDependencies{Transactions: c.transactor, Repository: identitypkg.PostgresCanonicalDeleteRepository{}, Articles: port, Audit: audit.Store{}, IDs: identitypkg.RandomUUIDGenerator{}, Clock: c.clock, RecoveryRuns: store, RecoveryGate: orphanPGMaintenanceGate{}})
	orphanPGMust(t, err)
	capability, err := identitypkg.NewRecoveryDeleteCapability(runID, backup)
	orphanPGMust(t, err)
	c.launch(func() {
		_, err := coordinator.CanonicalDelete(c.ctx, identitypkg.CanonicalDeleteRequest{Actor: c.admin(), TargetID: c.ids.deletedMember, TargetRole: identitypkg.RoleMember, Mode: identitypkg.CanonicalDeleteRecoveryKept, Reason: identitypkg.AccountDeletionGovernance, TraceID: orphanPGTrace, Recovery: &identitypkg.RecoveryKeptDeletedRequest{Capability: capability}})
		done <- err
	})
	orphanPGWait(t, c.ctx, ready)
	err = c.transfer(id)
	close(release)
	canonicalErr := orphanPGJoin(t, c.ctx, done)
	c.nowait(err)
	orphanPGMust(t, canonicalErr)
	orphanPGMust(t, c.transfer(id))
	c.retained(c.ids.deletedMember, false)
}
func testOrphanPGAbsentOrdinary(t *testing.T) {
	c := newOrphanPGCase(t)
	id := c.insert(nil, &c.ids.member, false)
	op := c.prepareDeletion(id, article.DeleteOrdinary)
	orphanPGMust(t, c.canonical(c.ids.member, nil))
	c.h.exec(t, c.ctx, "DELETE FROM articles WHERE id=$1", id)
	f := c.finalizer(c.transactor, article.Repository{}, audit.Store{})
	orphanPGMust(t, f.Finalize(c.ctx, op, orphanPGTrace))
	orphanPGMust(t, f.Finalize(c.ctx, op, orphanPGTrace))
	c.retained(c.ids.member, true)
	if c.identity.calls != 0 || c.releases(c.ids.member) != 0 {
		t.Fatal("absent source-free operation guessed author")
	}
}

func testOrphanPGOperationBindings(t *testing.T) {
	for _, field := range []string{"article", "version", "operation", "audit", "source"} {
		t.Run(field, func(t *testing.T) { testOrphanPGPersistedOperationField(t, field) })
	}
}
func testOrphanPGViolatingTransfer(t *testing.T) {
	c := newOrphanPGCase(t)
	id := c.insert(&c.ids.deletedMember, nil, true)
	orphanPGMust(t, c.transfer(id))
	f := c.identity.lastFacts
	if f.AfterArticle.Violation != "violating" || f.AfterArticle.Category == nil || *f.AfterArticle.Category != "violence" || f.AfterArticle.AuthorID == nil || *f.AfterArticle.AuthorID != c.ids.member || len(f.Recipients) != 2 {
		t.Fatal("transfer changed violation or failed dual snapshot")
	}
	c.retained(c.ids.deletedMember, false)
}
func testOrphanPGT2CommitUnknown(t *testing.T) {
	for _, ackLoss := range []bool{false, true} {
		t.Run(fmt.Sprintf("ack_loss_%t", ackLoss), func(t *testing.T) { testOrphanPGT2CommitOutcome(t, ackLoss) })
	}
}

// A transaction-local SQL fault after the first native completion check. The
// Identity callback must invoke the native provider again and reject the actual
// reappearing Article; no fabricated VerifiedCompletionFacts are supplied.
func (c *orphanPGCase) corruptDeletionReadback(id string) {
	c.identity.hook = func(ctx context.Context, tx pgx.Tx, p *article.DispositionCompletionProof, l *identitypkg.ArticleIdentityLifecycle, r identitypkg.ArticleCompletionReference) error {
		now := c.clock.Now()
		return (article.Repository{}).InsertArticle(ctx, tx, article.Article{ID: id, Title: "synthetic", Body: "synthetic", PublicationState: article.PublicationPublished, ViolationState: article.ViolationClear, DeletionState: article.DeletionActive, Version: 1, CreatedAt: now, UpdatedAt: now})
	}
}

func testOrphanPGT2CommitOutcome(t *testing.T, ackLoss bool) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	op := c.prepareDeletion(id, article.DeleteOrphan)
	tx := c.transactor
	begins := 0
	tx.wrap = func(inner pgx.Tx) pgx.Tx {
		begins++
		if begins == 2 {
			if ackLoss {
				return commitAckLossTx{Tx: inner}
			}
			return commitErrorTx{Tx: inner}
		}
		return inner
	}
	err := c.finalizer(tx, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
	if err == nil {
		t.Fatal("T2 commit uncertainty reported success")
	}
	if ackLoss {
		c.retained(c.ids.deletedMember, false)
	} else {
		c.pendingUnchanged(id, op)
		c.retained(c.ids.deletedMember, true)
		c.clock.set(c.clock.Now().Add(2 * time.Minute))
	}
	orphanPGMust(t, c.worker().Reconcile(c.ctx))
	orphanPGMust(t, c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace))
	if c.releases(c.ids.deletedMember) != 1 {
		t.Fatal("T2 unknown outcome retry duplicated release")
	}
}

func testOrphanPGPersistedOperationField(t *testing.T, field string) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	op := c.prepareDeletion(id, article.DeleteOrphan)
	repo := orphanPGFinalizerRepository{afterComplete: func(ctx context.Context, tx pgx.Tx) error {
		query := "UPDATE deletion_operations SET "
		switch field {
		case "article":
			query += "article_id='00000000-0000-4000-8000-000000000099'"
		case "version":
			query += "article_version=article_version+1"
		case "operation":
			query += "id='00000000-0000-4000-8000-000000000099'"
		case "audit":
			query += "audit_event_id=NULL"
		default:
			query += "orphaned_from_user_id='00000000-0000-4000-8000-000000000099'"
		}
		_, err := tx.Exec(ctx, query+" WHERE id=$1", op)
		return err
	}}
	err := c.finalizer(c.transactor, repo, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
	if err == nil {
		t.Fatal("persisted operation binding drift accepted")
	}
	c.pendingUnchanged(id, op)
	c.retained(c.ids.deletedMember, true)
}

func testOrphanPGRejectedLease(t *testing.T, mismatch string) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	op := c.prepareDeletion(id, article.DeleteOrphan)
	tx := c.transactor
	begins := 0
	tx.wrap = func(inner pgx.Tx) pgx.Tx {
		begins++
		if begins == 2 {
			switch mismatch {
			case "owner":
				c.h.exec(t, c.ctx, "UPDATE deletion_operations SET claimed_by='other-owner' WHERE id=$1", op)
			case "status":
				c.h.exec(t, c.ctx, "UPDATE deletion_operations SET status='journaled',claimed_by=NULL,locked_until=NULL WHERE id=$1", op)
			case "lease":
				c.h.exec(t, c.ctx, "UPDATE deletion_operations SET locked_until=locked_until+interval '1 second' WHERE id=$1", op)
			}
		}
		return inner
	}
	repo := orphanPGFinalizerRepository{}
	if mismatch == "expired_after_lock" {
		repo.afterOperation = func(ctx context.Context, tx pgx.Tx, o article.DeletionOperation) error {
			if o.Status == article.DeletionOperationFinalizing {
				c.clock.set(o.LockedUntil.Add(time.Microsecond))
			}
			return nil
		}
	}
	err := c.finalizer(tx, repo, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
	if !errors.Is(err, article.ErrDeletionInProgress) {
		t.Fatalf("lost/expired claim not rejected: %v", err)
	}
	if c.scalar("SELECT count(*) FROM articles WHERE id=$1 AND deletion_state='deletion_pending'", id) != 1 || c.scalar("SELECT count(*) FROM audit_events WHERE target_id=$1", id) != 0 || c.identity.calls != 0 {
		t.Fatal("invalid lease produced deletion side effects")
	}
	c.retained(c.ids.deletedMember, true)
}

func testOrphanPGAuthoredDeletionFault(t *testing.T, kind article.DeleteKind, point string) {
	c := newOrphanPGCase(t)
	id := c.insert(nil, &c.ids.member, kind == article.DeleteViolating)
	op := c.prepareDeletion(id, kind)
	orphanPGMust(t, c.canonical(c.ids.member, nil))
	beforeAudits := c.scalar("SELECT count(*) FROM audit_events")
	var appender article.DeletionAuditor = audit.Store{}
	switch point {
	case "completion_audit":
		appender = orphanPGAudit{after: func(context.Context, pgx.Tx, audit.Event) error {
			return errors.New("synthetic F001 completion failure")
		}}
	case "release_audit":
		c.identity.audit = orphanPGAudit{after: func(context.Context, pgx.Tx, audit.Event) error { return errors.New("synthetic F001 release failure") }}
	default:
		c.corruptDeletionReadback(id)
	}
	err := c.finalizer(c.transactor, article.Repository{}, appender).Finalize(c.ctx, op, orphanPGTrace)
	if err == nil {
		t.Fatal("F001 fault committed")
	}
	c.pendingUnchanged(id, op)
	c.retained(c.ids.member, true)
	if c.scalar("SELECT count(*) FROM audit_events") != beforeAudits {
		t.Fatal("F001 fault leaked an audit")
	}
	c.identity.hook = nil
	c.identity.audit = audit.Store{}
	c.clock.set(c.clock.Now().Add(2 * time.Minute))
	orphanPGMust(t, c.worker().Reconcile(c.ctx))
	c.retained(c.ids.member, false)
}

func testOrphanPGAuthoredDeletionKind(t *testing.T, kind article.DeleteKind) {
	for _, otherPending := range []bool{false, true} {
		t.Run(fmt.Sprintf("other_pending_%t", otherPending), func(t *testing.T) { testOrphanPGRemainingPending(t, kind, otherPending) })
	}
	t.Run("active_author_then_canonical", func(t *testing.T) { testOrphanPGActiveAuthoredDeletion(t, kind) })
	for _, table := range []string{"users", "identity_identifiers"} {
		t.Run("NOWAIT_"+table, func(t *testing.T) { testOrphanPGAuthoredNowait(t, kind, table) })
	}
	for _, point := range []string{"completion_audit", "release_audit", "verifier"} {
		t.Run("rollback_"+point, func(t *testing.T) { testOrphanPGAuthoredDeletionFault(t, kind, point) })
	}
}

func testOrphanPGTransferContact(t *testing.T, missing string) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	switch missing {
	case "source":
		c.h.exec(t, c.ctx, "DELETE FROM identity_identifiers WHERE user_id=$1 AND kind='email'", c.ids.deletedMember)
		c.h.exec(t, c.ctx, "UPDATE users SET retained_email=NULL WHERE id=$1", c.ids.deletedMember)
	case "target":
		c.ids.member = c.ids.initialAdmin
		c.h.exec(t, c.ctx, "DELETE FROM identity_identifiers WHERE user_id=$1 AND kind='email'", c.ids.member)
		c.h.exec(t, c.ctx, "UPDATE users SET email=NULL WHERE id=$1", c.ids.member)
	case "binding":
		c.h.exec(t, c.ctx, "UPDATE identity_identifiers SET normalized_value='wrong@example.test' WHERE user_id=$1 AND kind='email'", c.ids.member)
	case "suspended":
		c.h.exec(t, c.ctx, "UPDATE users SET status='suspended' WHERE id=$1", c.ids.member)
	case "deleted":
		orphanPGMust(t, c.canonical(c.ids.member, nil))
	}
	before := c.snapshot()
	status, body, _ := c.httpDisposition(id, true)
	if missing == "none" {
		if status != 200 || !strings.Contains(body, `"delivery_capability":"closed"`) {
			t.Fatal("closed ProcessingGate blocked transfer")
		}
		c.retained(c.ids.deletedMember, false)
	} else {
		if status == 200 {
			t.Fatal("invalid transfer contact/target accepted")
		}
		if c.snapshot() != before {
			t.Fatal("invalid transfer partially committed")
		}
	}
}

func testOrphanPGT2Fault(t *testing.T, point string) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	op := c.prepareDeletion(id, article.DeleteOrphan)
	var appender article.DeletionAuditor = audit.Store{}
	switch point {
	case "completion_audit":
		appender = orphanPGAudit{after: func(context.Context, pgx.Tx, audit.Event) error {
			return errors.New("synthetic completion audit failure")
		}}
	case "completed_operation":
		c.trigger("deletion_operations", "UPDATE", `IF NEW.status='completed' THEN RAISE EXCEPTION 'synthetic completion failure'; END IF; RETURN NEW`)
	case "verifier":
		c.corruptDeletionReadback(id)
	case "release_audit":
		c.identity.audit = orphanPGAudit{after: func(context.Context, pgx.Tx, audit.Event) error { return errors.New("synthetic release audit failure") }}
	}
	err := c.finalizer(c.transactor, article.Repository{}, appender).Finalize(c.ctx, op, orphanPGTrace)
	if err == nil {
		t.Fatal("T2 fault committed")
	}
	c.pendingUnchanged(id, op)
	c.retained(c.ids.deletedMember, true)
	if c.releases(c.ids.deletedMember) != 0 {
		t.Fatal("T2 failure leaked release audit")
	}
	if _, err := c.journal.FindByOperation(c.ctx, op); err != nil {
		t.Fatal("committed tombstone lost")
	}
	if point == "completed_operation" {
		c.h.exec(t, c.ctx, "ALTER TABLE deletion_operations DISABLE TRIGGER s62_fault")
	}
	c.identity.audit = audit.Store{}
	c.identity.hook = nil
	c.clock.set(c.clock.Now().Add(2 * time.Minute))
	orphanPGMust(t, c.worker().Reconcile(c.ctx))
	c.retained(c.ids.deletedMember, false)
	if c.releases(c.ids.deletedMember) != 1 {
		t.Fatal("worker after T2 fault duplicated release")
	}
}

func testOrphanPGTransferCommitOutcome(t *testing.T, ackLoss bool) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	before := c.snapshot()
	tx := c.transactor
	tx.wrap = func(tx pgx.Tx) pgx.Tx {
		if ackLoss {
			return commitAckLossTx{Tx: tx}
		}
		return commitErrorTx{Tx: tx}
	}
	c.command = c.newCommand(tx, audit.Store{})
	err := c.transfer(id)
	if err == nil {
		t.Fatal("unknown commit reported success")
	}
	c.command = c.newCommand(c.transactor, audit.Store{})
	if ackLoss {
		c.retained(c.ids.deletedMember, false)
		if c.scalar("SELECT count(*) FROM transfer_notices") != 1 {
			t.Fatal("ack loss did not actually commit")
		}
		status, _, _ := c.httpDisposition(id, true)
		if status == 200 || c.scalar("SELECT count(*) FROM transfer_notices") != 1 {
			t.Fatal("new HTTP request duplicated committed Notice")
		}
	} else {
		c.rolledBack(before, err)
		status, _, _ := c.httpDisposition(id, true)
		if status != 200 {
			t.Fatal("new HTTP retry after rolled-back commit failed")
		}
	}
}

func testOrphanPGTransferFault(t *testing.T, point string) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	fault := errors.New("synthetic atomic boundary failure")
	switch point {
	case "disposition_audit":
		c.command = c.newCommand(c.transactor, orphanPGAudit{after: func(context.Context, pgx.Tx, audit.Event) error { return fault }})
	case "verifier":
		c.command = c.newCommand(c.transactor, c.auditSQL(`UPDATE articles SET version=version+1 WHERE orphaned_from_user_id IS NULL AND author_id IS NOT NULL`))
	case "release_audit":
		c.identity.audit = orphanPGAudit{after: func(context.Context, pgx.Tx, audit.Event) error { return fault }}
	default:
		table := "articles"
		event := "UPDATE"
		if point == "notice_insert" {
			table = "transfer_notices"
			event = "INSERT"
		}
		if point == "recipient_insert" {
			table = "transfer_notice_recipients"
			event = "INSERT"
		}
		c.trigger(table, event, `RAISE EXCEPTION 'synthetic atomic failure'`)
	}
	before := c.snapshot()
	c.rolledBack(before, c.transfer(id))
	c.retained(c.ids.deletedMember, true)
	c.identity.audit = audit.Store{}
	c.command = c.newCommand(c.transactor, audit.Store{})
	if point == "article_update" {
		c.h.exec(t, c.ctx, "ALTER TABLE articles DISABLE TRIGGER s62_fault")
	}
	if point == "notice_insert" {
		c.h.exec(t, c.ctx, "ALTER TABLE transfer_notices DISABLE TRIGGER s62_fault")
	}
	if point == "recipient_insert" {
		c.h.exec(t, c.ctx, "ALTER TABLE transfer_notice_recipients DISABLE TRIGGER s62_fault")
	}
	status, _, _ := c.httpDisposition(id, true)
	if status != http.StatusOK {
		t.Fatalf("new HTTP retry after fault status=%d", status)
	}
}

func testOrphanPGReferenceSuccess(t *testing.T, action string) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	c.identity.hook = func(ctx context.Context, tx pgx.Tx, p *article.DispositionCompletionProof, l *identitypkg.ArticleIdentityLifecycle, r identitypkg.ArticleCompletionReference) error {
		_, err := l.ClearRetainedEmailIfEligible(ctx, tx, r.SourceID, r)
		if err != nil {
			return err
		}
		again, err := l.ClearRetainedEmailIfEligible(ctx, tx, r.SourceID, r)
		if again {
			return errors.New("same-tx reentry released twice")
		}
		return err
	}
	var err error
	switch action {
	case "transfer":
		err = c.transfer(id)
	case "delete":
		op := c.prepareDeletion(id, article.DeleteOrphan)
		err = c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
	default:
		err = c.publish(id)
	}
	orphanPGMust(t, err)
	c.retained(c.ids.deletedMember, false)
	if c.identity.calls < 2 || c.releases(c.ids.deletedMember) != 1 {
		t.Fatal("native verifier/reentry evidence missing")
	}
	if _, err := c.identity.lastProof.VerifyCompletion(c.ctx, c.identity.lastReference.Transaction); err == nil {
		t.Fatal("committed transaction reference remained usable")
	}
}

func testOrphanPGReferenceLifetime(t *testing.T, action string, life string) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	var op string
	if action == "delete" {
		op = c.prepareDeletion(id, article.DeleteOrphan)
	}
	c.identity.hook = c.referenceLifetimeFault(t, life)
	if action == "delete" {
		err := c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
		if err == nil {
			t.Fatal("old deletion reference accepted")
		}
		c.pendingUnchanged(id, op)
	} else {
		before := c.snapshot()
		var err error
		if action == "transfer" {
			err = c.transfer(id)
		} else {
			err = c.publish(id)
		}
		c.rolledBack(before, err)
	}
}

func testOrphanPGReferenceField(t *testing.T, action string, mutation orphanPGReferenceMutation) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	var op string
	if action == "delete" {
		op = c.prepareDeletion(id, article.DeleteOrphan)
	}
	c.identity.hook = func(ctx context.Context, tx pgx.Tx, p *article.DispositionCompletionProof, l *identitypkg.ArticleIdentityLifecycle, r identitypkg.ArticleCompletionReference) error {
		mutation.change(&r)
		_, err := l.ClearRetainedEmailIfEligible(ctx, tx, r.SourceID, r)
		if err == nil {
			t.Error("mutated reference was accepted")
			return errors.New("TEST FAILURE: mutated reference accepted")
		}
		return err
	}
	if action == "delete" {
		err := c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
		if err == nil {
			t.Fatal("forged deletion reference accepted")
		}
		c.pendingUnchanged(id, op)
		c.retained(c.ids.deletedMember, true)
	} else {
		before := c.snapshot()
		var err error
		if action == "transfer" {
			err = c.transfer(id)
		} else {
			err = c.publish(id)
		}
		c.rolledBack(before, err)
	}
}

func testOrphanPGTwoConcurrentDispositions(t *testing.T, deletion bool) {
	c := newOrphanPGCase(t)
	first, second := c.orphan(), c.orphan()
	var op string
	if deletion {
		op = c.prepareDeletion(second, article.DeleteOrphan)
	}
	ready, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	c.identity.hook = orphanPGHoldCompletion(first, ready, release)
	c.launch(func() { done <- c.publish(first) })
	orphanPGWait(t, c.ctx, ready)
	var competing error
	if deletion {
		competing = c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
	} else {
		competing = c.transfer(second)
	}
	close(release)
	firstErr := orphanPGJoin(t, c.ctx, done)
	c.nowait(competing)
	orphanPGMust(t, firstErr)
	c.retained(c.ids.deletedMember, true)
	wantActive, wantPending := int64(1), int64(0)
	if deletion {
		wantActive, wantPending = 0, 1
	}
	if c.identity.lastFacts.RemainingActiveCount != wantActive || c.identity.lastFacts.RemainingPendingCount != wantPending {
		t.Fatal("first completion counts did not retain other association")
	}
	c.identity.hook = nil
	if deletion {
		c.clock.set(c.clock.Now().Add(2 * time.Minute))
		orphanPGMust(t, c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace))
	} else {
		orphanPGMust(t, c.transfer(second))
	}
	c.retained(c.ids.deletedMember, false)
	if c.releases(c.ids.deletedMember) != 1 {
		t.Fatal("last-of-two release must happen exactly once")
	}
}

func testOrphanPGWorkerTransient(t *testing.T) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	op := c.prepareDeletion(id, article.DeleteOrphan)
	holder := c.hold("SELECT 1 FROM articles WHERE id=$1 FOR UPDATE", id)
	// A real 55P03 from Finalize must reach the worker's backoff branch. The Sleep
	// callback is an observation barrier, not an alternative classification.
	discoverer, err := article.NewPostgresDeletionOperationDiscoverer(c.h.pool)
	orphanPGMust(t, err)
	sleeps := 0
	worker, err := article.NewDeletionWorker(article.DeletionWorkerDependencies{Discoverer: discoverer, Converger: c.finalizer(c.transactor, article.Repository{}, audit.Store{}), Now: c.clock.Now, NewTraceID: func() (string, error) { return orphanPGTrace, nil }, Sleep: func(context.Context, time.Duration) error { sleeps++; return nil }, Jitter: func(d time.Duration) time.Duration { return d }}, article.DeletionWorkerConfig{PollInterval: time.Second, BatchSize: 8, OperationBudget: time.Second, BackoffInitial: time.Millisecond, BackoffMax: time.Second, MaxBatchesPerWake: 1, ShutdownBudget: time.Second})
	orphanPGMust(t, err)
	orphanPGMust(t, worker.Reconcile(c.ctx))
	if sleeps != 1 {
		t.Fatalf("55P03 was not transient/backoff: sleeps=%d", sleeps)
	}
	c.pendingUnchanged(id, op)
	orphanPGMust(t, holder.Rollback(c.ctx))
	orphanPGMust(t, worker.Reconcile(c.ctx))
	if sleeps != 1 {
		t.Fatal("unexpired claim was retried")
	}
	c.clock.set(c.clock.Now().Add(2 * time.Minute))
	orphanPGMust(t, worker.Reconcile(c.ctx))
	c.retained(c.ids.deletedMember, false)
	if c.releases(c.ids.deletedMember) != 1 {
		t.Fatal("worker retry duplicated release")
	}
}

func testOrphanPGCanonicalFirst(t *testing.T, target string) {
	c := newOrphanPGCase(t)
	c.cloneTarget(target)
	id := c.orphan()
	c.checkConnection()
	ready, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	c.launch(func() {
		done <- c.canonical(target, func(ctx context.Context, tx pgx.Tx) error {
			close(ready)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	})
	orphanPGWait(t, c.ctx, ready)
	// Canonical owns target User and Identifier; both UUID sort directions run.
	err := c.transfer(id)
	close(release)
	canonicalErr := orphanPGJoin(t, c.ctx, done)
	c.nowait(err)
	orphanPGMust(t, canonicalErr)
	c.retained(c.ids.deletedMember, true)
	if err := c.transfer(id); !errors.Is(err, article.ErrTransferTargetInvalid) {
		t.Fatal("fresh retry did not revalidate deleted target")
	}
}

func testOrphanPGAuthoredNowait(t *testing.T, kind article.DeleteKind, table string) {
	c := newOrphanPGCase(t)
	id := c.insert(nil, &c.ids.member, kind == article.DeleteViolating)
	op := c.prepareDeletion(id, kind)
	orphanPGMust(t, c.canonical(c.ids.member, nil))
	column := "id"
	if table != "users" {
		column = "user_id"
	}
	holder := c.hold("SELECT 1 FROM "+table+" WHERE "+column+"=$1 FOR UPDATE", c.ids.member)
	err := c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
	c.nowait(err)
	c.pendingUnchanged(id, op)
	c.retained(c.ids.member, true)
	articleProbe := c.hold("SELECT 1 FROM articles WHERE id=$1 FOR UPDATE NOWAIT", id)
	orphanPGMust(t, articleProbe.Rollback(c.ctx))
	orphanPGMust(t, holder.Rollback(c.ctx))
	c.clock.set(c.clock.Now().Add(2 * time.Minute))
	orphanPGMust(t, c.worker().Reconcile(c.ctx))
	c.retained(c.ids.member, false)
}

func testOrphanPGActiveAuthoredDeletion(t *testing.T, kind article.DeleteKind) {
	c := newOrphanPGCase(t)
	id := c.insert(nil, &c.ids.member, kind == article.DeleteViolating)
	op := c.prepareDeletion(id, kind)
	orphanPGMust(t, c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace))
	if c.identity.calls != 0 {
		t.Fatal("active author cleanup was invoked")
	}
	orphanPGMust(t, c.canonical(c.ids.member, nil))
	c.retained(c.ids.member, false)
}

func testOrphanPGRemainingPending(t *testing.T, kind article.DeleteKind, otherPending bool) {
	c := newOrphanPGCase(t)
	id := c.insert(nil, &c.ids.member, kind == article.DeleteViolating)
	op := c.prepareDeletion(id, kind)
	var second string
	if otherPending {
		second = c.insert(nil, &c.ids.member, false)
		c.prepareDeletion(second, article.DeleteOrdinary)
	}
	orphanPGMust(t, c.canonical(c.ids.member, nil))
	c.retained(c.ids.member, true)
	if c.scalar("SELECT count(*) FROM articles WHERE author_id=$1 AND deletion_state='deletion_pending'", c.ids.member) != map[bool]int64{false: 1, true: 2}[otherPending] {
		t.Fatal("canonical modified pending author provenance")
	}
	f := c.finalizer(c.transactor, article.Repository{}, audit.Store{})
	orphanPGMust(t, f.Finalize(c.ctx, op, orphanPGTrace))
	c.retained(c.ids.member, otherPending)
	if c.scalar("SELECT count(*) FROM deletion_operations WHERE id=$1 AND delete_kind=$2 AND orphaned_from_user_id IS NULL AND status='completed'", op, string(kind)) != 1 {
		t.Fatal("F001 rewrote deletion history")
	}
	orphanPGMust(t, f.Finalize(c.ctx, op, orphanPGTrace))
	if c.releases(c.ids.member) != map[bool]int64{false: 1, true: 0}[otherPending] {
		t.Fatal("F001 duplicate or premature release")
	}
}

func testOrphanPGDeleteAbsence(t *testing.T, absent bool) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	op := c.prepareDeletion(id, article.DeleteOrphan)
	if absent {
		c.h.exec(t, c.ctx, "DELETE FROM articles WHERE id=$1", id)
	} else {
		c.h.exec(t, c.ctx, "UPDATE articles SET version=version+1 WHERE id=$1", id)
	}
	err := c.finalizer(c.transactor, article.Repository{}, audit.Store{}).Finalize(c.ctx, op, orphanPGTrace)
	if absent {
		orphanPGMust(t, err)
		c.retained(c.ids.deletedMember, false)
	} else {
		if !errors.Is(err, article.ErrDeletionArticleMismatch) {
			t.Fatal("present mismatch was treated as absent")
		}
		c.pendingUnchanged(id, op)
	}
}

func testOrphanPGIdentityDrift(t *testing.T, drift string) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	switch drift {
	case "retained_absent":
		c.h.exec(t, c.ctx, "UPDATE users SET retained_email=NULL WHERE id=$1", c.ids.deletedMember)
	case "identifier_absent":
		c.h.exec(t, c.ctx, "DELETE FROM identity_identifiers WHERE user_id=$1 AND kind='email'", c.ids.deletedMember)
	case "binding":
		c.h.exec(t, c.ctx, "UPDATE identity_identifiers SET normalized_value='wrong@example.test' WHERE user_id=$1 AND kind='email'", c.ids.deletedMember)
	case "user_active":
		c.h.exec(t, c.ctx, "UPDATE users SET status='active',deleted_at=NULL,email=retained_email,retained_email=NULL,phone=(SELECT normalized_value FROM identity_identifiers WHERE user_id=$1 AND kind='phone') WHERE id=$1", c.ids.deletedMember)
	default:
		before := c.snapshot()
		tx := c.h.begin(t, c.ctx)
		_, err := c.identity.LockTransferParties(c.ctx, tx, nil, randomUUID(t))
		orphanPGMust(t, tx.Rollback(c.ctx))
		c.rolledBack(before, err)
		return
	}
	before := c.snapshot()
	c.rolledBack(before, c.publish(id))
}

func testOrphanPGZeroRows(t *testing.T, table string) {
	c := newOrphanPGCase(t)
	id := c.orphan()
	event := "UPDATE"
	action := "RETURN NULL"
	if table == "transfer_notices" || table == "transfer_notice_recipients" {
		event = "INSERT"
	}
	if table == "identity_identifiers" {
		event = "DELETE"
	}
	c.trigger(table, event, action)
	before := c.snapshot()
	c.rolledBack(before, c.transfer(id))
}

func (c *orphanPGCase) referenceLifetimeFault(t *testing.T, life string) orphanPGCompletionHook {
	return func(ctx context.Context, tx pgx.Tx, p *article.DispositionCompletionProof, l *identitypkg.ArticleIdentityLifecycle, r identitypkg.ArticleCompletionReference) error {
		switch life {
		case "foreign_tx":
			other := c.h.begin(t, ctx)
			defer func() { _ = other.Rollback(context.Background()) }()
			r.Transaction = other
			_, err := l.ClearRetainedEmailIfEligible(ctx, other, r.SourceID, r)
			if err == nil {
				t.Fatal("foreign tx accepted")
			}
			return err
		case "ended_tx":
			orphanPGMust(t, tx.Rollback(ctx))
			_, err := p.VerifyCompletion(ctx, tx)
			if err == nil {
				t.Fatal("ended transaction proof accepted")
			}
			return err
		default:
			other, err := platform.ContextWithTraceID(ctx, "01ARZ3NDEKTSV4RRFFQ69G5FAW")
			if err != nil {
				return err
			}
			_, err = p.VerifyCompletion(other, tx)
			if err == nil {
				t.Fatal("different trace accepted")
			}
			return err
		}
	}
}

func orphanPGHoldCompletion(id string, ready chan<- struct{}, release <-chan struct{}) orphanPGCompletionHook {
	return func(ctx context.Context, tx pgx.Tx, p *article.DispositionCompletionProof, l *identitypkg.ArticleIdentityLifecycle, r identitypkg.ArticleCompletionReference) error {
		if r.ExpectedArticleID == id {
			close(ready)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
}

func testOrphanPGSourceFreeReplay(t *testing.T) {
	c := newOrphanPGCase(t)
	op := randomUUID(t)
	id := randomUUID(t)
	tombstone, err := c.journal.Append(c.ctx, article.DeletionTombstoneRequest{ArticleID: id, ArticleVersion: 1, DeletionOperationID: op, DeletedAt: c.clock.Now().UTC().Truncate(time.Microsecond)})
	orphanPGMust(t, err)
	f := c.finalizer(c.transactor, article.Repository{}, audit.Store{})
	_, err = f.ReplayAllTombstones(c.ctx, []article.ArticleDeletionTombstone{tombstone}, orphanPGTrace)
	if err != nil {
		// Real PG RED: the rebuilt source-free operation has no actor, while
		// the current finalizer attempts actor_kind=user. Keep the success
		// expectation and residual evidence; do not fabricate an actor.
		c.retained(c.ids.deletedMember, true)
		var status string
		var actorAbsent, sourceAbsent bool
		orphanPGMust(t, c.h.queryRow(t, c.ctx, "SELECT status,actor_user_id IS NULL,orphaned_from_user_id IS NULL FROM deletion_operations WHERE id=$1", op).Scan(&status, &actorAbsent, &sourceAbsent))
		t.Logf("source_free_replay_red SQLSTATE=%s status=%s actor_absent=%t source_absent=%t cleanup_calls=%d completion_audits=%d", orphanPGSQLState(err), status, actorAbsent, sourceAbsent, c.identity.calls, c.scalar("SELECT count(*) FROM audit_events WHERE target_id=$1", id))
	}
	orphanPGMust(t, err)
	c.retained(c.ids.deletedMember, true)
	if c.identity.calls != 0 || c.releases(c.ids.deletedMember) != 0 {
		t.Fatal("source-free absent replay invented cleanup provenance")
	}
	assertOrphanPGSourceFreeCompletion(c, op, id)
	before := c.snapshot()
	result, err := f.ReplayAllTombstones(c.ctx, []article.ArticleDeletionTombstone{tombstone}, orphanPGTrace)
	orphanPGMust(t, err)
	if result.AlreadyAbsent != 1 || result.Replayed != 0 || before != c.snapshot() || c.identity.calls != 0 {
		t.Fatal("source-free replay reentry changed persisted facts or invented cleanup provenance")
	}
	assertOrphanPGSourceFreeCompletion(c, op, id)
}

func assertOrphanPGSourceFreeCompletion(testCase *orphanPGCase, operationID, articleID string) {
	testCase.t.Helper()
	const completionSQL = `
		SELECT count(*) FROM deletion_operations operation
		JOIN audit_events event ON event.id=operation.audit_event_id
		WHERE operation.id=$1 AND operation.article_id=$2::uuid
			AND operation.status='completed' AND operation.completed_at=event.occurred_at
			AND operation.delete_kind='ordinary' AND operation.article_version=1
			AND operation.actor_user_id IS NULL AND operation.actor_role IS NULL
			AND operation.orphaned_from_user_id IS NULL
			AND operation.claimed_by IS NULL AND operation.locked_until IS NULL
			AND event.actor_kind='system' AND event.actor_system='article-deletion-replay'
			AND event.actor_user_id IS NULL AND event.actor_role IS NULL
			AND event.action='article.deletion.replay' AND event.target_type='article'
			AND event.target_id=$2::text AND event.trace_id=$3 AND event.outcome='succeeded'
	`
	if testCase.scalar(completionSQL, operationID, articleID, orphanPGTrace) != 1 ||
		testCase.scalar("SELECT count(*) FROM audit_events WHERE target_id=$1", articleID) != 1 ||
		testCase.scalar("SELECT count(*) FROM articles WHERE id=$1", articleID) != 0 ||
		testCase.identity.calls != 0 || testCase.releases(testCase.ids.deletedMember) != 0 {
		testCase.t.Fatal("source-free replay lacks unique completed/system-audit binding or guessed identity cleanup")
	}
	testCase.retained(testCase.ids.deletedMember, true)
}
