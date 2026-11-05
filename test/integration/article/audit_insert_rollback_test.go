//go:build integration

package articleintegration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const arTrace = "01ARZ3NDEKTSV4RRFFQ69G5FAV"

func TestArticleAuditInsertRollbackContract(t *testing.T) {
	t.Run("MarkRollbackAndRetry", func(t *testing.T) { arModeration(t, false) })
	t.Run("RestoreRollbackAndRetry", func(t *testing.T) { arModeration(t, true) })
	t.Run("TransferRollbackAndRetry", arTransfer)
	t.Run("FinalizerSQLRollbackAndRetry", arFinalize)
}

type arClock struct{ at time.Time }

func (c *arClock) Now() time.Time { return c.at }

type arIDs struct{ remaining []string }

func (ids *arIDs) NewID() (string, error) {
	if len(ids.remaining) == 0 {
		return "", errors.New("audit fixture ID sequence exhausted")
	}
	id := ids.remaining[0]
	ids.remaining = ids.remaining[1:]
	return id, nil
}

func arUUID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	_, err := rand.Read(b[:])
	arMust(t, err)
	b[6], b[8] = b[6]&15|64, b[8]&63|128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func arMust(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		t.Fatalf("SQL failure code=%s constraint=%s", pg.Code, pg.ConstraintName)
	}
	t.Fatalf("operation failed type=%T", err)
}

func arRollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

type arFixture struct {
	t                      *testing.T
	ctx                    context.Context
	cancel                 context.CancelFunc
	pool                   *pgxpool.Pool
	observer               *pgxpool.Conn
	schema, dir            string
	clock                  *arClock
	admin                  session.Principal
	row                    article.Article
	collision, fresh       string
	noticeBad, noticeNew   string
	source, target         string
	sourceMail, targetMail string
	operation              string
	created                bool
	tx                     *arTransactions
	auditor                *arAuditor
	seedAudit              audit.Event
	initialImage           string
}

func arConfig(t *testing.T, schema string) *pgxpool.Config {
	t.Helper()
	dsn := os.Getenv("HAN_TEST_POSTGRES_DSN")
	if strings.TrimSpace(dsn) == "" {
		t.Fatal("HAN_TEST_POSTGRES_DSN missing; selected audit top cannot skip")
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("invalid audit test PostgreSQL configuration")
	}
	arAddress(t, c.ConnConfig.Host, c.ConnConfig.Port)
	for _, fallback := range c.ConnConfig.Fallbacks {
		arAddress(t, fallback.Host, fallback.Port)
	}
	c.MaxConns = 4
	c.ConnConfig.RuntimeParams["search_path"] = pgx.Identifier{schema}.Sanitize()
	return c
}

func arAddress(t *testing.T, host string, port uint16) {
	t.Helper()
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() || port != 5433 {
		t.Fatal("audit test requires literal loopback:5433 for primary and all fallbacks")
	}
}

func arNew(t *testing.T) *arFixture {
	t.Helper()
	f := &arFixture{t: t, schema: "han_s15audit_" + strings.ReplaceAll(arUUID(t), "-", "")}
	f.ctx, f.cancel = context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(f.close)
	var err error
	f.pool, err = pgxpool.NewWithConfig(f.ctx, arConfig(t, f.schema))
	arMust(t, err)
	arPreflight(f)
	_, err = f.pool.Exec(f.ctx, "CREATE SCHEMA "+pgx.Identifier{f.schema}.Sanitize())
	arMust(t, err)
	t.Logf("retained_schema=%s case=%s stage=create_committed", f.schema, t.Name())
	f.created = true
	f.dir = filepath.Join(os.Getenv("HAN_AUDIT_EVIDENCE_DIR"), f.schema)
	if os.Getenv("HAN_AUDIT_EVIDENCE_DIR") == "" {
		t.Fatal("audit evidence directory required")
	}
	arMust(t, os.Mkdir(f.dir, 0700))
	arMigrate(f)
	f.observer, err = f.pool.Acquire(f.ctx)
	arMust(t, err)
	f.ctx, err = platform.ContextWithTraceID(f.ctx, arTrace)
	arMust(t, err)
	f.tx = &arTransactions{f: f}
	f.auditor = &arAuditor{f: f}
	arSeed(f)
	return f
}

func arPreflight(f *arFixture) {
	var host, user string
	var port, version int
	var now time.Time
	err := f.pool.QueryRow(f.ctx, `SELECT host(inet_server_addr()),inet_server_port(),
		current_setting('server_version_num')::int,current_user,clock_timestamp()`).Scan(&host, &port, &version, &user, &now)
	arMust(f.t, err)
	arAddress(f.t, host, uint16(port))
	if version != 180006 || user == "" {
		f.t.Fatal("audit PostgreSQL version/identity preflight mismatch; no schema created")
	}
	f.clock = &arClock{now.UTC().Truncate(time.Microsecond)}
	f.t.Logf("audit_preflight=passed version=%d endpoint=loopback:5433 identity=%s", version, user)
}

func arMigrate(f *arFixture) {
	tx, err := f.pool.Begin(f.ctx)
	arMust(f.t, err)
	defer arRollback(tx)
	names := []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql", "0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql", "0009_article_replay_attempt.sql"}
	for _, name := range names {
		body, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
		arMust(f.t, err)
		_, err = tx.Exec(f.ctx, string(body))
		arMust(f.t, err)
		f.t.Logf("schema=%s migration=%s stage=applied_in_tx", f.schema, name)
	}
	arMust(f.t, tx.Commit(f.ctx))
	f.t.Logf("schema=%s stage=migrations_committed", f.schema)
}

func (f *arFixture) close() {
	if f.observer != nil {
		f.observer.Release()
	}
	f.cancel()
	if f.pool != nil {
		if n := f.pool.Stat().AcquiredConns(); n != 0 {
			f.t.Errorf("audit resources acquired=%d; expected0", n)
		}
		f.pool.Close()
	}
	f.t.Logf("schema=%s resources=cancelled_observer_released_acquired0_pool_closed workers=none retained=%t", f.schema, f.created)
}

func (f *arFixture) save(stage string, value any) {
	data, err := json.MarshalIndent(value, "", "  ")
	arMust(f.t, err)
	file, err := os.OpenFile(filepath.Join(f.dir, stage+".json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	arMust(f.t, err)
	_, err = file.Write(data)
	arMust(f.t, err)
	arMust(f.t, file.Close())
}

func arSeed(f *arFixture) {
	tx, err := f.pool.Begin(f.ctx)
	arMust(f.t, err)
	defer arRollback(tx)
	f.admin = session.Principal{UserID: arUser(f, tx, "active", "admin"), Role: "admin", Status: "active"}
	f.collision, f.fresh = arUUID(f.t), arUUID(f.t)
	f.noticeBad, f.noticeNew = arUUID(f.t), arUUID(f.t)
	e := audit.Event{ID: f.collision, OccurredAt: f.clock.at, ActorKind: audit.ActorSystem, ActorSystem: arPtr("audit-fixture"),
		Action: "synthetic.seed", TargetType: "synthetic", TargetID: "collision", Outcome: "succeeded", TraceID: arTrace, Metadata: map[string]any{"fixture": true}}
	arMust(f.t, (audit.Store{}).Append(f.ctx, tx, e))
	f.seedAudit = e
	arMust(f.t, tx.Commit(f.ctx))
	f.clock.at = f.clock.at.Add(time.Second)
	f.row = article.Article{ID: arUUID(f.t), Title: " 合成标题🙂 ", Body: " literal <b>合成正文</b> ",
		PublicationState: article.PublicationPublished, ViolationState: article.ViolationClear, DeletionState: article.DeletionActive,
		Version: 3, CreatedAt: f.clock.at.Add(-time.Hour), UpdatedAt: f.clock.at.Add(-time.Minute)}
}

func arPtr(s string) *string { return &s }

func arUser(f *arFixture, tx pgx.Tx, status, role string) string {
	id := arUUID(f.t)
	username, email, phone := "audit-"+id, "audit-"+id+"@example.test", "synthetic-"+id
	var liveMail, livePhone, retained *string
	var deleted *time.Time
	lifecycle := "active"
	if status == "deleted" {
		retained, deleted, lifecycle = &email, &f.clock.at, "reserved"
	} else {
		liveMail, livePhone = &email, &phone
	}
	_, err := tx.Exec(f.ctx, `INSERT INTO users(id,username,email,phone,role,status,credential_mode,retained_email,deleted_at,created_at,updated_at)
		VALUES($1,$2,$3,$4,$5,$6,'permanent',$7,$8,$9,$9)`, id, username, liveMail, livePhone, role, status, retained, deleted, f.clock.at)
	arMust(f.t, err)
	values := map[string]string{"username": username, "email": email}
	if livePhone != nil {
		values["phone"] = phone
	}
	for kind, value := range values {
		_, err = tx.Exec(f.ctx, `INSERT INTO identity_identifiers(user_id,kind,normalized_value,lifecycle,created_at) VALUES($1,$2,$3,$4,$5)`, id, kind, value, lifecycle, f.clock.at)
		arMust(f.t, err)
	}
	return id
}

type arQuery interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

const arImageSQL = `SELECT jsonb_build_object(
	'articles',(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY id),'[]') FROM articles x),
	'audit',(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY id),'[]') FROM audit_events x),
	'notices',(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY id),'[]') FROM transfer_notices x),
	'recipients',(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY notice_id,recipient_kind),'[]') FROM transfer_notice_recipients x),
	'operations',(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY id),'[]') FROM deletion_operations x),
	'users',(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY id),'[]') FROM users x),
	'identifiers',(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY user_id,kind),'[]') FROM identity_identifiers x))::text`

func (f *arFixture) image(q arQuery, stage string) string {
	var raw string
	arMust(f.t, q.QueryRow(f.ctx, arImageSQL).Scan(&raw))
	if stage == "before" {
		f.initialImage = raw
	}
	f.save(stage, json.RawMessage(raw))
	f.t.Logf("schema=%s stage=%s image_sha256=%x", f.schema, stage, sha256.Sum256([]byte(raw)))
	return raw
}

type arTransactions struct {
	f           *arFixture
	seen        []*arTx
	beforeBegin func(int)
}

func (s *arTransactions) Begin(ctx context.Context) (pgx.Tx, error) {
	if s.beforeBegin != nil {
		s.beforeBegin(len(s.seen) + 1)
	}
	tx, err := s.f.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	pid, observer := tx.Conn().PgConn().PID(), s.f.observer.Conn().PgConn().PID()
	if pid == observer {
		s.f.t.Fatal("observer reused business backend")
	}
	s.f.t.Logf("schema=%s business_pid=%d observer_pid=%d independent=true tx=%d", s.f.schema, pid, observer, len(s.seen)+1)
	wrapped := &arTx{Tx: tx, f: s.f}
	s.seen = append(s.seen, wrapped)
	return wrapped, nil
}

type arTx struct {
	pgx.Tx
	f         *arFixture
	ended     bool
	committed bool
	auditErr  error
	auditExec int
}

func (tx *arTx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tag, err := tx.Tx.Exec(ctx, sql, args...)
	if strings.Contains(sql, "INSERT INTO audit_events") {
		tx.auditExec++
		tx.auditErr = err
		if err != nil {
			var pg *pgconn.PgError
			if errors.As(err, &pg) {
				tx.f.t.Logf("schema=%s stage=actual_audit_exec code=%s table=%s constraint=%s", tx.f.schema, pg.Code, pg.TableName, pg.ConstraintName)
			}
		}
	}
	return tag, err
}

func (tx *arTx) Rollback(context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := tx.Tx.Rollback(ctx)
	if err == nil || errors.Is(err, pgx.ErrTxClosed) {
		tx.ended = true
	}
	return err
}

func (tx *arTx) Commit(ctx context.Context) error {
	err := tx.Tx.Commit(ctx)
	if err == nil {
		tx.ended, tx.committed = true, true
	}
	return err
}

type arAuditor struct {
	f     *arFixture
	mode  string
	calls int
}

func (a *arAuditor) Append(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	if len(a.f.tx.seen) == 0 || tx != a.f.tx.seen[len(a.f.tx.seen)-1] {
		a.f.t.Fatal("audit did not receive exact business transaction")
	}
	a.calls++
	arWitness(a.f, tx, a.mode, event)
	a.f.image(tx, "before-append-"+event.ID)
	return (audit.Store{}).Append(ctx, tx, event)
}

func arWitness(f *arFixture, tx pgx.Tx, mode string, event audit.Event) {
	if trace, ok := platform.TraceIDFromContext(f.ctx); !ok || trace != event.TraceID {
		f.t.Fatal("audit trace mismatch")
	}
	if mode == "finalizer" {
		var present bool
		var status string
		var auditID *string
		arMust(f.t, tx.QueryRow(f.ctx, "SELECT EXISTS(SELECT 1 FROM articles WHERE id=$1)", f.row.ID).Scan(&present))
		arMust(f.t, tx.QueryRow(f.ctx, "SELECT status,audit_event_id::text FROM deletion_operations WHERE id=$1", f.operation).Scan(&status, &auditID))
		if present || status != "finalizing" || auditID != nil {
			f.t.Fatal("finalizer audit preceded actual DELETE or completion ordering changed")
		}
		return
	}
	row := arReadArticle(f, tx)
	if row.Version != f.row.Version+1 {
		f.t.Fatal("audit preceded actual Article UPDATE")
	}
	if mode == "transfer" {
		if row.AuthorID == nil || *row.AuthorID != f.target || row.OrphanedFromUserID != nil || row.PublicationState != article.PublicationPublished {
			f.t.Fatal("transfer audit preceded association UPDATE")
		}
		notice := f.noticeBad
		if event.ID == f.fresh {
			notice = f.noticeNew
		}
		arNotice(f, tx, notice)
	} else {
		want := article.ViolationViolating
		if mode == "restore" {
			want = article.ViolationClear
		}
		if row.ViolationState != want || (mode == "restore" && row.ViolationCategory != nil) || (mode == "mark" && (row.ViolationCategory == nil || *row.ViolationCategory != article.ViolationViolence)) {
			f.t.Fatal("audit preceded moderation state UPDATE")
		}
	}
}

func arReadArticle(f *arFixture, q arQuery) article.Article {
	var row article.Article
	arMust(f.t, q.QueryRow(f.ctx, `SELECT id,author_id,orphaned_from_user_id,title,body,publication_state,violation_state,
		violation_category,deletion_state,deletion_operation_id,version,created_at,updated_at FROM articles WHERE id=$1`, f.row.ID).Scan(
		&row.ID, &row.AuthorID, &row.OrphanedFromUserID, &row.Title, &row.Body, &row.PublicationState, &row.ViolationState,
		&row.ViolationCategory, &row.DeletionState, &row.DeletionOperationID, &row.Version, &row.CreatedAt, &row.UpdatedAt))
	row.CreatedAt, row.UpdatedAt = row.CreatedAt.UTC(), row.UpdatedAt.UTC()
	return row
}

func arCollision(f *arFixture, err error) {
	tx := f.tx.seen[len(f.tx.seen)-1]
	var actual, returned *pgconn.PgError
	if !errors.As(tx.auditErr, &actual) || actual.Code != "23505" || actual.SchemaName != f.schema || actual.TableName != "audit_events" || actual.ConstraintName != "audit_events_pkey" || tx.auditExec != 1 {
		f.t.Fatal("actual audit INSERT did not produce the required PostgreSQL collision")
	}
	f.t.Logf("schema=%s stage=collision_observed actual_sqlstate=23505 tx_released=%t", f.schema, tx.ended)
	if !errors.As(err, &returned) {
		f.t.Fatal("returned error chain lost PostgreSQL audit INSERT cause")
	}
	if returned != actual || !tx.ended || tx.committed {
		f.t.Fatal("audit collision chain or transaction ownership changed")
	}
	f.save("collision", map[string]any{"code": actual.Code, "schema": actual.SchemaName, "table": actual.TableName, "constraint": actual.ConstraintName, "same_error_pointer": returned == actual, "rolled_back": tx.ended})
}

func arInsert(f *arFixture) {
	tx, err := f.pool.Begin(f.ctx)
	arMust(f.t, err)
	defer arRollback(tx)
	arMust(f.t, (article.Repository{}).InsertArticle(f.ctx, tx, f.row))
	arMust(f.t, tx.Commit(f.ctx))
}

func arCommand(f *arFixture, mode string, ids *arIDs, port article.DispositionIdentityPort) *article.CommandService {
	f.auditor.mode = mode
	d := article.CommandDependencies{Transactions: f.tx, Repository: article.Repository{}, IDs: ids, Clock: f.clock,
		Moderation: &article.ModerationDependencies{Auditor: f.auditor, AuthorStates: arAuthorStates}}
	if mode == "transfer" {
		d.Disposition = &article.OrphanDispositionDependencies{Identity: port, Auditor: f.auditor, RecipientMaxAttempts: 3}
	}
	s, err := article.NewCommandService(d)
	arMust(f.t, err)
	return s
}

func arAuthorStates(ctx context.Context, tx pgx.Tx, ids []string) (map[string]string, error) {
	r, err := identity.NewArticleIdentityReader(tx)
	if err != nil {
		return nil, err
	}
	states, err := r.GetAuthorStates(ctx, ids)
	result := make(map[string]string, len(states))
	for id, state := range states {
		result[id] = string(state)
	}
	return result, err
}

func arModeration(t *testing.T, restore bool) {
	f := arNew(t)
	mode, action, category := "mark", "article.violation.mark", article.ViolationViolence
	var input *article.ViolationCategory = &category
	if restore {
		mode, action, category = "restore", "article.violation.clear", article.ViolationSexual
		f.row.ViolationState, f.row.ViolationCategory, input = article.ViolationViolating, &category, nil
	}
	arInsert(f)
	before := f.image(f.observer, "before")
	ids := &arIDs{[]string{f.collision, f.fresh}}
	s := arCommand(f, mode, ids, nil)
	view, etag, err := s.Moderate(f.ctx, f.admin, f.row.ID, f.row.Version, input)
	arCollision(f, err)
	if !reflect.DeepEqual(view, article.AdminArticleView{}) || etag != "" {
		t.Fatal("failed moderation returned success payload")
	}
	if f.image(f.observer, "after-failure") != before {
		t.Fatal("moderation collision persisted backend changes")
	}
	arUnlock(f, mode)
	_, etag, err = s.Moderate(f.ctx, f.admin, f.row.ID, f.row.Version, input)
	arMust(t, err)
	expected := f.row
	expected.Version++
	expected.UpdatedAt = f.clock.at
	expected.ViolationState, expected.ViolationCategory = article.ViolationClear, nil
	if !restore {
		expected.ViolationState, expected.ViolationCategory = article.ViolationViolating, &category
	}
	if !reflect.DeepEqual(arReadArticle(f, f.observer), expected) || etag == "" {
		t.Fatal("retry Article full-value oracle mismatch")
	}
	arAudit(f, action, map[string]any{"article_version": float64(expected.Version), "category": string(category)})
	arFinalChecks(f, ids, 2)
}

func arUnlock(f *arFixture, mode string) {
	tx, err := f.observer.Begin(f.ctx)
	arMust(f.t, err)
	defer arRollback(tx)
	if mode == "transfer" {
		ids := []string{f.source, f.target}
		sort.Strings(ids)
		for _, id := range ids {
			arLockRows(f, tx, "SELECT id::text FROM users WHERE id=$1 FOR UPDATE NOWAIT", id)
		}
		for _, id := range ids {
			arLockRows(f, tx, "SELECT user_id::text FROM identity_identifiers WHERE user_id=$1 ORDER BY kind FOR UPDATE NOWAIT", id)
		}
	}
	if mode == "finalizer" {
		arLockRows(f, tx, "SELECT id::text FROM deletion_operations WHERE id=$1 FOR UPDATE NOWAIT", f.operation)
	}
	arLockRows(f, tx, "SELECT id::text FROM articles WHERE id=$1 FOR UPDATE NOWAIT", f.row.ID)
	arMust(f.t, tx.Rollback(f.ctx))
	f.t.Logf("schema=%s stage=independent_nowait_unlocked mode=%s", f.schema, mode)
}

func arLockRows(f *arFixture, tx pgx.Tx, sql, id string) {
	rows, err := tx.Query(f.ctx, sql, id)
	arMust(f.t, err)
	defer rows.Close()
	n := 0
	for rows.Next() {
		var got string
		arMust(f.t, rows.Scan(&got))
		if got != id {
			f.t.Fatal("NOWAIT target binding changed")
		}
		n++
	}
	arMust(f.t, rows.Err())
	if n == 0 {
		f.t.Fatal("NOWAIT probe did not lock a real row")
	}
}

func arFinalChecks(f *arFixture, ids *arIDs, audits int) {
	if len(ids.remaining) != 0 || f.auditor.calls != audits {
		f.t.Fatal("unexpected ID/audit invocation count")
	}
	for _, tx := range f.tx.seen {
		if !tx.ended {
			f.t.Fatal("business transaction not released")
		}
	}
	if !f.tx.seen[len(f.tx.seen)-1].committed {
		f.t.Fatal("retry did not commit")
	}
	after := f.image(f.observer, "after-retry")
	arRetryPreservation(f, after)
	f.t.Logf("schema=%s stage=fresh_id_retry_committed business_transactions_released=true", f.schema)
}

func arRetryPreservation(f *arFixture, after string) {
	if !reflect.DeepEqual(arReadAudit(f, f.collision), f.seedAudit) {
		f.t.Fatal("collision seed audit changed")
	}
	var before, current map[string]json.RawMessage
	arMust(f.t, json.Unmarshal([]byte(f.initialImage), &before))
	arMust(f.t, json.Unmarshal([]byte(after), &current))
	for _, key := range []string{"notices", "recipients", "users", "identifiers", "operations"} {
		if f.auditor.mode == "transfer" && key != "operations" {
			continue
		}
		if f.auditor.mode == "finalizer" && key == "operations" {
			arOperationImmutable(f, before[key], current[key])
			continue
		}
		if !reflect.DeepEqual(before[key], current[key]) {
			f.t.Fatal("retry changed unrelated backend table: " + key)
		}
	}
	if f.auditor.mode == "transfer" {
		arIdentityPreserved(f, before, current)
	}
	var audits int
	arMust(f.t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM audit_events").Scan(&audits))
	want := 2
	if f.auditor.mode == "transfer" {
		want = 3
	}
	if audits != want {
		f.t.Fatal("retry committed unexpected audit rows")
	}
}

func arOperationImmutable(f *arFixture, before, after json.RawMessage) {
	var b, a []map[string]any
	arMust(f.t, json.Unmarshal(before, &b))
	arMust(f.t, json.Unmarshal(after, &a))
	if len(b) != 1 || len(a) != 1 {
		f.t.Fatal("operation cardinality changed")
	}
	for _, key := range []string{"status", "claimed_by", "locked_until", "audit_event_id", "completed_at"} {
		delete(b[0], key)
		delete(a[0], key)
	}
	if !reflect.DeepEqual(b, a) {
		f.t.Fatal("Finalize changed immutable operation fields")
	}
}

func arIdentityPreserved(f *arFixture, before, after map[string]json.RawMessage) {
	var users, actual []map[string]any
	arMust(f.t, json.Unmarshal(before["users"], &users))
	arMust(f.t, json.Unmarshal(after["users"], &actual))
	if len(users) != len(actual) {
		f.t.Fatal("transfer changed user cardinality")
	}
	for i, user := range users {
		if user["id"] == f.source {
			value, ok := actual[i]["updated_at"].(string)
			if !ok {
				f.t.Fatal("identity cleanup updated_at absent")
			}
			stamp, err := time.Parse(time.RFC3339Nano, value)
			arMust(f.t, err)
			if !stamp.Equal(f.clock.at) {
				f.t.Fatal("identity cleanup time mismatch")
			}
			user["retained_email"], user["updated_at"] = nil, value
		}
	}
	if !reflect.DeepEqual(users, actual) {
		f.t.Fatal("transfer changed source/target identity outside real cleanup")
	}
	var identifiers, remaining []map[string]any
	arMust(f.t, json.Unmarshal(before["identifiers"], &identifiers))
	arMust(f.t, json.Unmarshal(after["identifiers"], &remaining))
	var expected []map[string]any
	for _, identifier := range identifiers {
		if identifier["user_id"] != f.source || identifier["kind"] != "email" {
			expected = append(expected, identifier)
		}
	}
	if !reflect.DeepEqual(expected, remaining) {
		f.t.Fatal("transfer changed identifier rows beyond source reserved email release")
	}
}

func arNotice(f *arFixture, q arQuery, id string) {
	var notice article.TransferNotice
	arMust(f.t, q.QueryRow(f.ctx, `SELECT id,article_ref,new_author_id,former_deleted_user_id,created_at,updated_at FROM transfer_notices WHERE id=$1`, id).Scan(
		&notice.ID, &notice.ArticleRef, &notice.NewAuthorID, &notice.FormerDeletedUserID, &notice.CreatedAt, &notice.UpdatedAt))
	notice.CreatedAt, notice.UpdatedAt = notice.CreatedAt.UTC(), notice.UpdatedAt.UTC()
	want := article.TransferNotice{ID: id, ArticleRef: f.row.ID, NewAuthorID: f.target, FormerDeletedUserID: f.source, CreatedAt: f.clock.at, UpdatedAt: f.clock.at}
	if !reflect.DeepEqual(notice, want) {
		f.t.Fatal("Notice full-value SQL oracle mismatch")
	}
	var count int
	arMust(f.t, q.QueryRow(f.ctx, "SELECT count(*) FROM transfer_notice_recipients WHERE notice_id=$1", id).Scan(&count))
	if count != 2 {
		f.t.Fatal("transfer did not write exactly two snapshots")
	}
	for kind, email := range map[article.RecipientKind]string{article.RecipientNewAuthor: f.targetMail, article.RecipientFormerDeletedAuthor: f.sourceMail} {
		arRecipient(f, q, id, kind, email)
	}
}

func arRecipient(f *arFixture, q arQuery, id string, kind article.RecipientKind, email string) {
	var row article.TransferNoticeRecipient
	arMust(f.t, q.QueryRow(f.ctx, `SELECT notice_id,recipient_kind,email_snapshot,status,claimed_by,locked_until,next_attempt_at,attempt_count,max_attempts,
		last_attempt_at,accepted_at,failure_bucket,created_at,updated_at FROM transfer_notice_recipients WHERE notice_id=$1 AND recipient_kind=$2`, id, kind).Scan(
		&row.NoticeID, &row.Kind, &row.EmailSnapshot, &row.Status, &row.ClaimedBy, &row.LockedUntil, &row.NextAttemptAt, &row.AttemptCount, &row.MaxAttempts,
		&row.LastAttemptAt, &row.AcceptedAt, &row.FailureBucket, &row.CreatedAt, &row.UpdatedAt))
	row.CreatedAt, row.UpdatedAt = row.CreatedAt.UTC(), row.UpdatedAt.UTC()
	want := article.TransferNoticeRecipient{NoticeID: id, Kind: kind, EmailSnapshot: email, Status: article.RecipientPending, MaxAttempts: 3, CreatedAt: f.clock.at, UpdatedAt: f.clock.at}
	if !reflect.DeepEqual(row, want) {
		f.t.Fatal("recipient full-value SQL oracle mismatch")
	}
}

func arReadAudit(f *arFixture, id string) audit.Event {
	var e audit.Event
	var metadata []byte
	arMust(f.t, f.observer.QueryRow(f.ctx, `SELECT id,occurred_at,actor_kind,actor_user_id,actor_system,actor_role,action,target_type,target_id,
		outcome,error_code,trace_id,source_fingerprint,metadata FROM audit_events WHERE id=$1`, id).Scan(&e.ID, &e.OccurredAt, &e.ActorKind, &e.ActorUserID,
		&e.ActorSystem, &e.ActorRole, &e.Action, &e.TargetType, &e.TargetID, &e.Outcome, &e.ErrorCode, &e.TraceID, &e.SourceFingerprint, &metadata))
	arMust(f.t, json.Unmarshal(metadata, &e.Metadata))
	e.OccurredAt = e.OccurredAt.UTC()
	return e
}

func arAudit(f *arFixture, action string, metadata map[string]any) {
	e := arReadAudit(f, f.fresh)
	want := audit.Event{ID: f.fresh, OccurredAt: f.clock.at, ActorKind: audit.ActorUser, ActorUserID: &f.admin.UserID, ActorRole: &f.admin.Role,
		Action: action, TargetType: "article", TargetID: f.row.ID, Outcome: "succeeded", TraceID: arTrace, Metadata: metadata}
	if !reflect.DeepEqual(e, want) {
		f.t.Fatal("retry full audit SQL oracle mismatch")
	}
	var n int
	arMust(f.t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM audit_events WHERE target_type='article' AND target_id=$1", f.row.ID).Scan(&n))
	if n != 1 {
		f.t.Fatal("retry produced duplicate or missing business audit")
	}
}

func arTransfer(t *testing.T) {
	f := arNew(t)
	tx, err := f.pool.Begin(f.ctx)
	arMust(t, err)
	defer arRollback(tx)
	f.source, f.target = arUser(f, tx, "deleted", "member"), arUser(f, tx, "active", "member")
	f.sourceMail, f.targetMail = "audit-"+f.source+"@example.test", "audit-"+f.target+"@example.test"
	f.row.OrphanedFromUserID, f.row.PublicationState = &f.source, article.PublicationHeld
	arMust(t, (article.Repository{}).InsertArticle(f.ctx, tx, f.row))
	arMust(t, tx.Commit(f.ctx))
	before := f.image(f.observer, "before")
	ids := &arIDs{[]string{f.collision, f.noticeBad, f.fresh, f.noticeNew}}
	cleanupID := arUUID(t)
	port := &arIdentity{f: f, ids: &arIDs{[]string{cleanupID}}}
	s := arCommand(f, "transfer", ids, port)
	result, err := s.TransferOrphan(f.ctx, f.admin, f.row.ID, f.row.Version, f.target)
	arCollision(f, err)
	if !reflect.DeepEqual(result, article.OrphanDispositionResult{}) {
		t.Fatal("failed transfer returned success payload")
	}
	if f.image(f.observer, "after-failure") != before {
		t.Fatal("transfer conflict persisted business/identity/snapshot changes")
	}
	if port.proofCalls != 0 {
		t.Fatal("failed business audit reached identity cleanup")
	}
	arUnlock(f, "transfer")
	result, err = s.TransferOrphan(f.ctx, f.admin, f.row.ID, f.row.Version, f.target)
	arMust(t, err)
	expected := f.row
	expected.AuthorID = &f.target
	expected.OrphanedFromUserID = nil
	expected.PublicationState = article.PublicationPublished
	expected.Version++
	expected.UpdatedAt = f.clock.at
	if !reflect.DeepEqual(arReadArticle(f, f.observer), expected) || result.NoticeID == nil || *result.NoticeID != f.noticeNew || result.ETag == "" {
		t.Fatal("transfer retry Article/result oracle mismatch")
	}
	arNotice(f, f.observer, f.noticeNew)
	arAudit(f, "article.orphan.transfer", map[string]any{"article_version": float64(expected.Version)})
	arIdentityAfter(f, port, cleanupID)
	arFinalChecks(f, ids, 2)
}

func arIdentityAfter(f *arFixture, port *arIdentity, cleanupID string) {
	var retained *string
	var count int
	arMust(f.t, f.observer.QueryRow(f.ctx, "SELECT retained_email FROM users WHERE id=$1", f.source).Scan(&retained))
	arMust(f.t, f.observer.QueryRow(f.ctx, "SELECT count(*) FROM identity_identifiers WHERE user_id=$1 AND kind='email'", f.source).Scan(&count))
	if retained != nil || count != 0 || port.proofCalls != 1 || len(port.ids.remaining) != 0 {
		f.t.Fatal("real proof/identity cleanup result mismatch")
	}
	e := arReadAudit(f, cleanupID)
	want := audit.Event{ID: cleanupID, OccurredAt: f.clock.at, ActorKind: audit.ActorSystem, ActorSystem: arPtr("article_disposition"),
		Action: "identity.identifier.email.release", TargetType: "identity_identifier", TargetID: f.source, Outcome: "succeeded", TraceID: arTrace, Metadata: map[string]any{"kind": "email"}}
	if !reflect.DeepEqual(e, want) {
		f.t.Fatal("identity cleanup audit oracle mismatch")
	}
	var notices, recipients, audits int
	arMust(f.t, f.observer.QueryRow(f.ctx, "SELECT (SELECT count(*) FROM transfer_notices),(SELECT count(*) FROM transfer_notice_recipients),(SELECT count(*) FROM audit_events)").Scan(&notices, &recipients, &audits))
	if notices != 1 || recipients != 2 || audits != 3 {
		f.t.Fatal("transfer retry committed extra/missing rows")
	}
}

type arIdentity struct {
	f          *arFixture
	ids        *arIDs
	proofCalls int
}

func (p *arIdentity) LockTransferParties(ctx context.Context, tx pgx.Tx, target *string, source string) (article.DispositionParties, error) {
	parties, err := (&identity.ArticleIdentityLifecycle{}).LockTransferParties(ctx, tx, target, source)
	return article.DispositionParties{SourceID: parties.SourceID, SourceEmail: parties.SourceEmail, TargetID: parties.TargetID, TargetEmail: parties.TargetEmail}, err
}

func (p *arIdentity) LockDeletionCandidate(ctx context.Context, tx pgx.Tx, id string) (article.DispositionDeletionCandidate, error) {
	c, err := (&identity.ArticleIdentityLifecycle{}).LockDeletionCandidate(ctx, tx, id)
	return article.DispositionDeletionCandidate{UserID: c.UserID, Deleted: c.Deleted}, err
}

func (p *arIdentity) ClearRetainedEmailIfEligible(ctx context.Context, tx pgx.Tx, source string, proof *article.DispositionCompletionProof) (bool, error) {
	if proof == nil || proof.Reference().Transaction != tx {
		return false, errors.New("real completion proof required")
	}
	v := &arBoundVerifier{p: p, proof: proof, tx: tx}
	lifecycle, err := identity.NewArticleIdentityLifecycle(v, audit.Store{}, p.ids, p.f.clock)
	if err != nil {
		return false, err
	}
	return lifecycle.ClearRetainedEmailIfEligible(ctx, tx, source, arReference(proof.Reference()))
}

type arBoundVerifier struct {
	p     *arIdentity
	proof *article.DispositionCompletionProof
	tx    pgx.Tx
}

func (v *arBoundVerifier) VerifyCompletion(ctx context.Context, tx pgx.Tx, reference identity.ArticleCompletionReference) (identity.VerifiedArticleCompletion, error) {
	if tx != v.tx || !reflect.DeepEqual(reference, arReference(v.proof.Reference())) {
		return identity.VerifiedArticleCompletion{}, errors.New("proof reference transaction mismatch")
	}
	v.p.proofCalls++
	facts, err := v.proof.VerifyCompletion(ctx, tx)
	if err != nil {
		return identity.VerifiedArticleCompletion{}, err
	}
	result := identity.VerifiedArticleCompletion{AfterArticle: arState(facts.AfterArticle), RemainingActiveCount: facts.RemainingActiveCount,
		RemainingPendingCount: facts.RemainingPendingCount, Audit: identity.ArticleDispositionAudit{ID: facts.Audit.ID, ArticleID: facts.Audit.TargetID,
			Action: facts.Audit.Action, Outcome: facts.Audit.Outcome, TraceID: facts.Audit.TraceID}}
	if n := facts.Notice; n != nil {
		result.Notice = &identity.ArticleDispositionNotice{ID: n.ID, ArticleID: n.ArticleRef, NewAuthorID: n.NewAuthorID, FormerDeletedUserID: n.FormerDeletedUserID}
	}
	for _, r := range facts.Recipients {
		result.Recipients = append(result.Recipients, identity.ArticleDispositionRecipient{Kind: string(r.Kind), EmailSnapshot: r.EmailSnapshot, Status: string(r.Status)})
	}
	return result, nil
}

func arReference(r article.DispositionCompletionReference) identity.ArticleCompletionReference {
	return identity.ArticleCompletionReference{Transaction: r.Transaction, ExpectedArticleID: r.ExpectedArticleID, SourceID: r.SourceID, BeforeVersion: r.BeforeVersion,
		Action: identity.ArticleDispositionAction(r.Action), NoticeID: r.NoticeID, NewAuthorID: r.NewAuthorID, OperationID: r.OperationID, AuditEventID: r.AuditEventID, BeforeArticle: arState(r.BeforeArticle)}
}

func arState(s article.DispositionArticleState) identity.ArticleDispositionState {
	var category *string
	if s.Category != nil {
		category = arPtr(string(*s.Category))
	}
	return identity.ArticleDispositionState{Exists: s.Exists, ID: s.ID, Version: s.Version, AuthorID: s.AuthorID, SourceID: s.SourceID, Publication: string(s.Publication),
		Violation: string(s.Violation), Category: category, Deletion: string(s.Deletion), OperationID: s.OperationID}
}

type arJournal struct {
	t         *testing.T
	tombstone article.ArticleDeletionTombstone
	calls     int
}

func (j *arJournal) Append(context.Context, article.DeletionTombstoneRequest) (article.ArticleDeletionTombstone, error) {
	j.t.Fatal("Finalize unexpectedly attempted Journal.Append")
	return article.ArticleDeletionTombstone{}, errors.New("unexpected append")
}

func (j *arJournal) FindByOperation(_ context.Context, id string) (article.ArticleDeletionTombstone, error) {
	if id != j.tombstone.DeletionOperationID {
		j.t.Fatal("Journal operation binding changed")
	}
	j.calls++
	return j.tombstone, nil
}

func arPending(f *arFixture) *arJournal {
	f.operation = arUUID(f.t)
	started := f.clock.at.Add(-time.Minute)
	j := &arJournal{t: f.t, tombstone: article.ArticleDeletionTombstone{Namespace: article.ArticleDeletionNamespace, Sequence: 1,
		ArticleID: f.row.ID, ArticleVersion: f.row.Version, DeletionOperationID: f.operation, DeletedAt: started, KeyID: "synthetic-key", KeyVersion: "test-v1", Stage: "tombstoned"}}
	tx, err := f.pool.Begin(f.ctx)
	arMust(f.t, err)
	defer arRollback(tx)
	_, err = tx.Exec(f.ctx, `INSERT INTO deletion_operations(id,article_id,article_version,delete_kind,actor_user_id,actor_role,status,
		tombstone_sequence,journal_key_id,journal_key_version,started_at) VALUES($1,$2,$3,'ordinary',$4,'admin','journaled',1,'synthetic-key','test-v1',$5)`,
		f.operation, f.row.ID, f.row.Version, f.admin.UserID, started)
	arMust(f.t, err)
	f.row.Version++
	f.row.DeletionState = article.DeletionPending
	f.row.DeletionOperationID = &f.operation
	arMust(f.t, (article.Repository{}).InsertArticle(f.ctx, tx, f.row))
	arMust(f.t, tx.Commit(f.ctx))
	return j
}

func arFinalize(t *testing.T) {
	f := arNew(t)
	journal := arPending(f)
	f.auditor.mode = "finalizer"
	ids := &arIDs{[]string{f.collision, f.fresh}}
	finalizer, err := article.NewDeletionFinalizer(article.DeletionFinalizerDependencies{Transactions: f.tx, Repository: article.Repository{}, Journal: journal,
		Auditor: f.auditor, AuditIDs: ids, Clock: f.clock, WorkerID: "audit-sql-worker", LeaseDuration: time.Minute})
	arMust(t, err)
	before := f.image(f.observer, "before")
	var claim string
	f.tx.beforeBegin = func(n int) {
		if n == 2 {
			claim = f.image(f.observer, "claim-committed")
		}
	}
	err = finalizer.Finalize(f.ctx, f.operation, arTrace)
	arCollision(f, err)
	after := f.image(f.observer, "after-failure")
	if claim == "" || after != claim {
		t.Fatal("completion failure changed claim-committed backend image")
	}
	arClaimBoundary(f, before, after)
	if !reflect.DeepEqual(arReadArticle(f, f.observer), f.row) {
		t.Fatal("physical DELETE was not rolled back")
	}
	arUnlock(f, "finalizer")
	var until time.Time
	arMust(t, f.observer.QueryRow(f.ctx, "SELECT locked_until FROM deletion_operations WHERE id=$1", f.operation).Scan(&until))
	f.clock.at = until.UTC().Add(time.Microsecond)
	arMust(t, finalizer.Finalize(f.ctx, f.operation, arTrace))
	arCompleted(f)
	arAudit(f, "article.delete", map[string]any{"article_version": float64(f.row.Version - 1), "delete_kind": "ordinary", "authority": "administrator"})
	if journal.calls != 2 || len(f.tx.seen) != 4 {
		t.Fatal("Finalize retry did not reclaim through public flow")
	}
	arFinalChecks(f, ids, 2)
}

func arClaimBoundary(f *arFixture, before, after string) {
	var b, a map[string]json.RawMessage
	arMust(f.t, json.Unmarshal([]byte(before), &b))
	arMust(f.t, json.Unmarshal([]byte(after), &a))
	delete(b, "operations")
	delete(a, "operations")
	if !reflect.DeepEqual(b, a) {
		f.t.Fatal("claim changed unrelated backend rows")
	}
	var status, owner string
	var auditID *string
	var completed *time.Time
	var until time.Time
	arMust(f.t, f.observer.QueryRow(f.ctx, "SELECT status,claimed_by,locked_until,audit_event_id::text,completed_at FROM deletion_operations WHERE id=$1", f.operation).Scan(&status, &owner, &until, &auditID, &completed))
	if status != "finalizing" || owner != "audit-sql-worker" || auditID != nil || completed != nil || !until.UTC().Equal(f.clock.at.Add(time.Minute)) {
		f.t.Fatal("claim/completion phase oracle mismatch")
	}
}

func arCompleted(f *arFixture) {
	var present bool
	var status, auditID string
	var owner *string
	var until *time.Time
	var completed time.Time
	arMust(f.t, f.observer.QueryRow(f.ctx, "SELECT EXISTS(SELECT 1 FROM articles WHERE id=$1)", f.row.ID).Scan(&present))
	arMust(f.t, f.observer.QueryRow(f.ctx, "SELECT status,audit_event_id::text,claimed_by,locked_until,completed_at FROM deletion_operations WHERE id=$1", f.operation).Scan(&status, &auditID, &owner, &until, &completed))
	if present || status != "completed" || auditID != f.fresh || owner != nil || until != nil || !completed.UTC().Equal(f.clock.at) {
		f.t.Fatal("Finalizer retry independent completion oracle mismatch")
	}
}
