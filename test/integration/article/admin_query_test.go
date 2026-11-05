//go:build integration

package articleintegration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This top is executed only after a separate LOCALPG START. No package init connects.
func TestAdminQueryPostgres(t *testing.T) {
	t.Run("RawQueryAndPresence", adminPGRawQuery)
	t.Run("FiveViewIsolation", adminPGCursors)
	t.Run("ViewStateMatrix", adminPGViews)
	t.Run("StaticCompositeKeyset", adminPGKeyset)
	t.Run("ScanAndPublicCursor", adminPGScanCursor)
	t.Run("LimitsAndExhaustion", adminPGLimits)
	t.Run("CurrentAuthorTransitions", adminPGAuthorTransitions)
	t.Run("CurrentModerationTransitions", adminPGModerationTransitions)
	t.Run("StrictIdentityWholeFailure", adminPGStrictIdentity)
	t.Run("IdentityScope", adminPGIdentityScope)
	t.Run("DetailPriority", adminPGDetailPriority)
	t.Run("SQLRowsCancel", adminPGSQLFaults)
	t.Run("ZeroBusinessWrites", adminPGReadOnly)
	t.Run("TwelveFieldsAndPrivacy", adminPGFields)
	t.Run("SameArticleETag", adminPGETag)
	t.Run("ErrorAndOldConsumerCompatibility", adminPGCompatibility)
}

type adminPGFixture struct {
	s      *s22Fixture
	db     *adminPGDatabase
	reader *adminPGReader
	query  *article.ArticleQueryService
	admin  *article.Handler
}

func newAdminPG(t *testing.T) *adminPGFixture {
	t.Helper()
	h := &noticeHarness{t: t, schema: "han_admin2_" + strings.ReplaceAll(noticeUUID(), "-", ""), log: &noticeLog{}}
	created := false
	t.Logf("admin2_case=%s planned_schema=%s stage=before_create", t.Name(), h.schema)
	h.ctx, h.cancel = context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(func() {
		h.cancel()
		h.group.Wait()
		if h.pool != nil {
			if h.pool.Stat().AcquiredConns() != 0 {
				t.Error("Admin2 leaked acquired connections")
			}
			h.pool.Close()
		}
		t.Logf("admin2_schema=%s created=%t retained=%t resources=cancelled_joined_pool_closed", h.schema, created, created)
	})
	pool, err := newAdminPGPool(h)
	noticeMust(t, err)
	h.pool = pool
	// CREATE SCHEMA is a standalone autocommit statement, before the migration transaction.
	_, err = h.pool.Exec(h.ctx, "CREATE SCHEMA "+pgx.Identifier{h.schema}.Sanitize())
	noticeMust(t, err)
	created = true
	t.Logf("retained_schema=%s stage=create_committed", h.schema)
	adminPGMigrate(h)
	ctx, err := platform.ContextWithTraceID(h.ctx, noticeTrace)
	noticeMust(t, err)
	s := &s22Fixture{t: t, h: h, ctx: ctx, now: h.now()}
	s.author, s.other, s.suspended, s.deleted, s.admin = s.user("member", "active"), s.user("member", "active"), s.user("member", "suspended"), s.user("member", "deleted"), s.user("admin", "active")
	s.reader, err = identity.NewArticleIdentityReader(h.pool)
	noticeMust(t, err)
	s.repo, err = article.NewPostgresArticleQueryRepository(h.pool)
	noticeMust(t, err)
	f := &adminPGFixture{s: s, db: &adminPGDatabase{real: h.pool}, reader: &adminPGReader{real: s.reader}}
	f.bind(t)
	return f
}

func newAdminPGPool(h *noticeHarness) (*pgxpool.Pool, error) {
	pool, err := pgxpool.NewWithConfig(h.ctx, noticePoolConfig(h.t, h.schema))
	if err != nil {
		return nil, err
	}
	var host string
	var port, version int
	err = pool.QueryRow(h.ctx, "SELECT host(inet_server_addr()),inet_server_port(),current_setting('server_version_num')::int").Scan(&host, &port, &version)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || port != 5433 || version != 180006 {
		pool.Close()
		return nil, errors.New("Admin2 requires the authorized loopback PostgreSQL 18.6 endpoint")
	}
	return pool, nil
}

func adminPGMigrate(h *noticeHarness) {
	h.t.Helper()
	tx, err := h.pool.Begin(h.ctx)
	noticeMust(h.t, err)
	defer tx.Rollback(context.Background())
	for _, name := range []string{"0001_shared.sql", "0002_identity.sql", "0003_article.sql", "0004_idempotency.sql", "0005_recovery_runs.sql", "0006_user_governance_indexes.sql", "0007_idempotency_response_snapshot_v1.sql", "0008_password_reset_durable.sql", "0009_article_replay_attempt.sql"} {
		h.t.Logf("admin2_schema=%s migration=%s stage=begin", h.schema, name)
		body, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", name))
		noticeMust(h.t, err)
		_, err = tx.Exec(h.ctx, string(body))
		noticeMust(h.t, err)
	}
	noticeMust(h.t, tx.Commit(h.ctx))
	h.t.Logf("admin2_schema=%s stage=migrations_committed", h.schema)
}

func (f *adminPGFixture) bind(t *testing.T) {
	t.Helper()
	repo, err := article.NewPostgresArticleQueryRepository(f.db)
	noticeMust(t, err)
	f.query, err = article.NewArticleQueryService(repo, f.reader, s22QueryConfig())
	noticeMust(t, err)
	f.admin, err = article.NewAdminQueryHandler(f.query)
	noticeMust(t, err)
}

type adminPGDatabase struct {
	real     article.ArticleQueryDatabase
	mode     string
	failAt   int64
	calls    atomic.Int64
	gets     atomic.Int64
	closed   atomic.Int64
	scans    atomic.Int64
	rowErrs  atomic.Int64
	mu       sync.Mutex
	args     [][]any
	afterSQL func(context.Context, int64) error
	afterRow func()
}

func (d *adminPGDatabase) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	rows, err := d.real.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	call := d.calls.Add(1)
	d.mu.Lock()
	d.args = append(d.args, append([]any(nil), args...))
	d.mu.Unlock()
	if d.afterSQL != nil {
		err = d.afterSQL(ctx, call)
	}
	if err != nil || d.mode == "query" && (d.failAt == 0 || call == d.failAt) {
		rows.Close()
		d.closed.Add(1)
		return nil, errors.New("synthetic failure after real candidate SQL")
	}
	mode := ""
	if d.failAt == 0 || call == d.failAt {
		mode = d.mode
	}
	return &adminPGRows{Rows: rows, owner: d, mode: mode}, nil
}

func (d *adminPGDatabase) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	d.gets.Add(1)
	return adminPGRow{real: d.real.QueryRow(ctx, sql, args...), owner: d}
}

type adminPGRow struct {
	real  pgx.Row
	owner *adminPGDatabase
}

func (r adminPGRow) Scan(dest ...any) error {
	err := r.real.Scan(dest...)
	if err != nil {
		return err
	}
	if r.owner.afterRow != nil {
		r.owner.afterRow()
	}
	if r.owner.mode == "detail_scan" {
		return errors.New("synthetic failure after real detail Scan")
	}
	return nil
}

type adminPGRows struct {
	pgx.Rows
	owner *adminPGDatabase
	mode  string
	once  atomic.Bool
}

func (r *adminPGRows) Scan(dest ...any) error {
	err := r.Rows.Scan(dest...)
	r.owner.scans.Add(1)
	if err == nil && r.mode == "scan" {
		return errors.New("synthetic failure after real Scan")
	}
	return err
}

func (r *adminPGRows) Err() error {
	err := r.Rows.Err()
	r.owner.rowErrs.Add(1)
	if err == nil && r.mode == "rows" {
		return errors.New("synthetic failure after real Rows.Err")
	}
	return err
}

func (r *adminPGRows) Close() {
	r.Rows.Close()
	if r.once.CompareAndSwap(false, true) {
		r.owner.closed.Add(1)
	}
}

type adminPGReader struct {
	real   *identity.ArticleIdentityReader
	mode   string
	failAt int
	ids    [][]string
	hook   func(context.Context, []string) error
}

func (r *adminPGReader) GetAuthorStates(ctx context.Context, ids []string) (map[string]identity.UserStatus, error) {
	if r.hook != nil {
		if err := r.hook(ctx, ids); err != nil {
			return nil, err
		}
	}
	states, err := r.real.GetAuthorStates(ctx, ids)
	if err != nil {
		return nil, err
	}
	r.ids = append(r.ids, append([]string(nil), ids...))
	if r.failAt > 0 && len(r.ids) != r.failAt {
		return states, nil
	}
	switch r.mode {
	case "nil":
		return nil, nil
	case "missing":
		if len(ids) > 0 {
			delete(states, ids[0])
		}
	case "empty", "invalid", "deleted":
		if len(ids) > 0 {
			value := r.mode
			if value == "empty" {
				value = ""
			}
			states[ids[0]] = identity.UserStatus(value)
		}
	case "error":
		return nil, errors.New("synthetic failure after real identity SQL")
	}
	return states, nil
}

func (f *adminPGFixture) http(t *testing.T, raw, id string) *httptest.ResponseRecorder {
	t.Helper()
	path, fn := "/api/v1/admin/articles", f.admin.ListAdminArticles
	if id != "" {
		path += "/" + id
		fn = f.admin.GetAdminArticle
	}
	auth, err := platform.NewHTTPAuthorizer(s22Principal{f.s.admin}, func() time.Time { return f.s.now })
	noticeMust(t, err)
	r := httptest.NewRequest(http.MethodGet, path, nil).WithContext(f.s.ctx)
	r.URL.RawQuery = raw
	r.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-admin-session"})
	w := httptest.NewRecorder()
	platform.TraceMiddleware(auth.Protect(session.AccessGeneral, http.HandlerFunc(fn))).ServeHTTP(w, r)
	return w
}

func adminPGPage(t *testing.T, w *httptest.ResponseRecorder) article.AdminArticlesPage {
	t.Helper()
	s22AssertStatus(t, w, 200)
	var page article.AdminArticlesPage
	noticeMust(t, json.Unmarshal(w.Body.Bytes(), &page))
	return page
}

func adminPGError(t *testing.T, w *httptest.ResponseRecorder, status int, code platform.Code) {
	t.Helper()
	s22AssertStatus(t, w, status)
	var envelope platform.Envelope
	noticeMust(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	if envelope.Code != code || envelope.TraceID == "" || w.Header().Get(platform.TraceHeader) != envelope.TraceID {
		t.Fatal("public error code/trace mismatch")
	}
	if w.Header().Get("ETag") != "" || strings.Contains(w.Body.String(), "items") {
		t.Fatal("failure exposed successful data/ETag")
	}
}

func (f *adminPGFixture) seedMatrix() []article.Article {
	rows := []article.Article{}
	for i := 8; i >= 1; i-- {
		rows = append(rows, f.s.base(i))
	}
	rows[0].DeletionState, rows[0].DeletionOperationID = article.DeletionPending, &[]string{noticeUUID()}[0]
	rows[2].AuthorID = &f.s.suspended.UserID
	rows[3].AuthorID = nil
	rows[4].AuthorID, rows[4].OrphanedFromUserID, rows[4].PublicationState = nil, &f.s.deleted.UserID, article.PublicationHeld
	for _, i := range []int{5, 6, 7} {
		category := article.ViolationViolence
		rows[i].ViolationState, rows[i].ViolationCategory = article.ViolationViolating, &category
	}
	rows[6].AuthorID = &f.s.suspended.UserID
	rows[7].AuthorID, rows[7].OrphanedFromUserID, rows[7].PublicationState = nil, &f.s.deleted.UserID, article.PublicationHeld
	for _, r := range rows {
		f.s.insert(r)
	}
	return rows
}

func (f *adminPGFixture) seedDense() {
	for i := 15; i >= 1; i-- {
		r := f.s.base(i)
		if i == 15 || i == 8 || i == 4 || i == 1 {
			category := article.ViolationSexual
			r.ViolationState, r.ViolationCategory = article.ViolationViolating, &category
		}
		f.s.insert(r)
	}
}

func (f *adminPGFixture) traverse(t *testing.T, view string) []string {
	t.Helper()
	cursor, ids := "", []string{}
	for i := 0; i < 40; i++ {
		raw := "view=" + view
		if cursor != "" {
			raw += "&cursor=" + cursor
		}
		page := adminPGPage(t, f.http(t, raw, ""))
		for _, r := range page.Items {
			ids = append(ids, r.ID)
		}
		if page.NextCursor == nil {
			return ids
		}
		cursor = *page.NextCursor
	}
	t.Fatal("Admin2 pagination did not converge")
	return nil
}

func (f *adminPGFixture) oracle(t *testing.T, view string) []string {
	t.Helper()
	filter := "TRUE"
	if view == "violating" {
		filter = "a.violation_state='violating'"
	}
	if view == "hidden" {
		filter = "(u.status='suspended' OR a.violation_state='violating' OR a.publication_state='held')"
	}
	rows, err := f.s.h.pool.Query(f.s.ctx, `SELECT a.id::text FROM articles a LEFT JOIN users u ON u.id=a.author_id WHERE a.deletion_state='active' AND `+filter+` ORDER BY a.created_at DESC,a.id DESC`)
	noticeMust(t, err)
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		noticeMust(t, rows.Scan(&id))
		ids = append(ids, id)
	}
	noticeMust(t, rows.Err())
	return ids
}

func adminPGRawQuery(t *testing.T) {
	f := newAdminPG(t)
	f.s.insert(f.s.base(1))
	for index, raw := range []string{"view=%ZZ", "view=hidden&view=%ZZ", "view=all&cursor=%ZZ", "view=hidden&limit=%ZZ", "view=hidden;view=violating", "view=all&other=%ZZ", "view=", "view=all&view=hidden", "v%69ew=all&view=hidden", "view=ALL"} {
		t.Run(fmt.Sprintf("invalid_%02d", index), func(t *testing.T) {
			before := f.db.calls.Load()
			identities := len(f.reader.ids)
			adminPGError(t, f.http(t, raw, ""), 422, article.CodeArticleValidationFailed)
			if f.db.calls.Load() != before || len(f.reader.ids) != identities {
				t.Fatal("bad RawQuery executed SQL")
			}
		})
	}
	for index, raw := range []string{"", "view=all", "view=hidden", "view=violating", "limit=01", "limit=%2B1"} {
		t.Run(fmt.Sprintf("valid_%02d", index), func(t *testing.T) { adminPGPage(t, f.http(t, raw, "")) })
	}
	for index, raw := range []string{"cursor=", "cursor=x&cursor=y", "cursor=%25ZZ"} {
		t.Run(fmt.Sprintf("malformed_%02d", index), func(t *testing.T) { adminPGError(t, f.http(t, raw, ""), 400, platform.CodeRequestInvalidCursor) })
	}
}

func adminPGCursors(t *testing.T) {
	f := newAdminPG(t)
	f.s.insert(f.s.base(1))
	views := []article.CursorView{article.CursorViewMember, article.CursorViewAdminAll, article.CursorViewAdminViolating, article.CursorViewAdminHidden, article.CursorViewAdminOrphan}
	for _, view := range views {
		t.Run(string(view), func(t *testing.T) {
			encoded, err := (article.CursorCodec{}).Encode(article.ArticleCursor{CreatedAt: f.s.now, ID: f.s.base(9).ID, View: view})
			noticeMust(t, err)
			for _, target := range []struct {
				view  article.CursorView
				query string
			}{{article.CursorViewAdminAll, "all"}, {article.CursorViewAdminViolating, "violating"}, {article.CursorViewAdminHidden, "hidden"}} {
				t.Run(target.query, func(t *testing.T) {
					before := f.db.calls.Load()
					w := f.http(t, "view="+target.query+"&cursor="+encoded, "")
					if view == target.view {
						adminPGPage(t, w)
					} else {
						adminPGError(t, w, 400, platform.CodeRequestInvalidCursor)
						if before != f.db.calls.Load() {
							t.Fatal("cross-view cursor queried SQL")
						}
					}
				})
			}
		})
	}
}

func adminPGViews(t *testing.T) {
	f := newAdminPG(t)
	f.seedMatrix()
	for _, view := range []string{"all", "violating", "hidden"} {
		t.Run(view, func(t *testing.T) {
			s22AssertSame(t, f.traverse(t, view), f.oracle(t, view), "Admin2 view differs from independent SQL")
		})
	}
}

func adminPGKeyset(t *testing.T) {
	f := newAdminPG(t)
	want := []string{}
	for i, id := range []int{2, 1, 99, 98, 199, 198} {
		r := f.s.base(id)
		r.CreatedAt = f.s.now.Add(-time.Duration(i/2) * time.Second)
		r.UpdatedAt = r.CreatedAt
		category := article.ViolationSexual
		r.ViolationState, r.ViolationCategory = article.ViolationViolating, &category
		f.s.insert(r)
		want = append(want, r.ID)
	}
	for _, view := range []string{"all", "violating", "hidden"} {
		t.Run(view, func(t *testing.T) {
			s22AssertSame(t, f.oracle(t, view), want, "three-timestamp fixture oracle invalid")
			s22AssertSame(t, f.traverse(t, view), want, "Admin2 composite keyset lost/reordered rows")
		})
	}
}

func adminPGScanCursor(t *testing.T) {
	f := newAdminPG(t)
	f.seedDense()
	first := adminPGPage(t, f.http(t, "view=violating", ""))
	if len(first.Items) != 2 || first.NextCursor == nil {
		t.Fatal("filtered page lost lookahead")
	}
	decoded, err := (article.CursorCodec{}).Decode(*first.NextCursor, article.CursorViewAdminViolating)
	noticeMust(t, err)
	if decoded.ID != first.Items[1].ID || decoded.ID != f.s.base(8).ID {
		t.Fatal("public cursor used last scanned instead of returned")
	}
	if f.db.calls.Load() < 4 {
		t.Fatal("fixture did not exercise multiple filtered batches")
	}
	for i, args := range f.db.args {
		if len(args) != 3 || args[2] != 3 {
			t.Fatal("candidate LIMIT/SQL parameter count changed")
		}
		if i > 0 && (args[0] != f.s.now || args[1] != f.s.base(15-i*3+1).ID) {
			t.Fatal("internal SQL scan boundary changed")
		}
	}
	s22AssertSame(t, f.traverse(t, "violating"), f.oracle(t, "violating"), "scan/public cursor skipped lookahead")
}

func adminPGLimits(t *testing.T) {
	f := newAdminPG(t)
	for index, raw := range []string{"limit=", "limit=1&limit=2", "limit=x", "limit=0", "limit=-1", "limit=3"} {
		t.Run(fmt.Sprintf("invalid_%02d", index), func(t *testing.T) {
			before := f.db.calls.Load()
			adminPGError(t, f.http(t, raw, ""), 422, article.CodeArticleValidationFailed)
			if before != f.db.calls.Load() {
				t.Fatal("invalid limit reached SQL")
			}
		})
	}
	for index, raw := range []string{"", "view=hidden", "limit=1", "limit=2"} {
		t.Run(fmt.Sprintf("empty_%02d", index), func(t *testing.T) {
			w := f.http(t, raw, "")
			page := adminPGPage(t, w)
			if page.Items == nil || len(page.Items) != 0 || page.NextCursor != nil || w.Header().Get("ETag") != "" {
				t.Fatal("empty envelope/ETag changed")
			}
		})
	}
	f.s.insert(f.s.base(1))
	page := adminPGPage(t, f.http(t, "view=hidden", ""))
	if len(page.Items) != 0 || page.NextCursor != nil {
		t.Fatal("all-filtered page not exhausted")
	}
}

func adminPGAuthorTransitions(t *testing.T) {
	f := newAdminPG(t)
	r := f.s.base(1)
	f.s.insert(r)
	for _, state := range []string{"suspended", "active"} {
		f.s.h.exec("UPDATE users SET status=$2 WHERE id=$1", f.s.author.UserID, state)
		page := adminPGPage(t, f.http(t, "view=hidden", ""))
		if (len(page.Items) != 0) != (state == "suspended") {
			t.Fatal("Admin2 cached identity state")
		}
	}
	queryDone, committed := make(chan struct{}), make(chan error, 1)
	f.s.h.group.Add(1)
	go func() {
		defer f.s.h.group.Done()
		select {
		case <-queryDone:
		case <-f.s.ctx.Done():
			committed <- f.s.ctx.Err()
			return
		}
		_, err := f.s.h.pool.Exec(f.s.ctx, "UPDATE users SET status='suspended' WHERE id=$1", f.s.author.UserID)
		committed <- err
	}()
	var once sync.Once
	f.db.afterSQL = func(ctx context.Context, _ int64) error {
		var err error
		once.Do(func() {
			close(queryDone)
			select {
			case err = <-committed:
			case <-ctx.Done():
				err = ctx.Err()
			}
		})
		return err
	}
	page := adminPGPage(t, f.http(t, "view=hidden", ""))
	if len(page.Items) != 1 || !reflect.DeepEqual(page.Items[0].HiddenReasons, []article.HiddenReason{article.HiddenAuthorSuspended}) {
		t.Fatal("candidate/identity barrier lost committed state")
	}
}

func adminPGModerationTransitions(t *testing.T) {
	for _, owner := range []string{"active", "suspended", "held"} {
		t.Run(owner, func(t *testing.T) {
			f := newAdminPG(t)
			r := f.s.base(1)
			if owner == "suspended" {
				r.AuthorID = &f.s.suspended.UserID
			} else if owner == "held" {
				r.AuthorID = nil
				r.OrphanedFromUserID = &f.s.deleted.UserID
				r.PublicationState = article.PublicationHeld
			}
			f.s.insert(r)
			adminPGAssertModerationRead(t, f, r, owner, false)
			category := article.ViolationViolence
			_, _, err := f.s.command().Moderate(f.s.ctx, f.s.admin, r.ID, 1, &category)
			noticeMust(t, err)
			adminPGAssertModerationRead(t, f, r, owner, true)
			_, _, err = f.s.command().Moderate(f.s.ctx, f.s.admin, r.ID, 2, nil)
			noticeMust(t, err)
			adminPGAssertModerationRead(t, f, r, owner, false)
		})
	}
}

func adminPGAssertModerationRead(t *testing.T, f *adminPGFixture, record article.Article, owner string, marked bool) {
	t.Helper()
	reasons := []article.HiddenReason{}
	if owner == "suspended" {
		reasons = append(reasons, article.HiddenAuthorSuspended)
	}
	if marked {
		reasons = append(reasons, article.HiddenViolation)
	}
	if owner == "held" {
		reasons = append(reasons, article.HiddenOrphanHold)
	}
	for _, view := range []string{"all", "violating", "hidden"} {
		page := adminPGPage(t, f.http(t, "view="+view, ""))
		want := view == "all" || view == "violating" && marked || view == "hidden" && len(reasons) > 0
		if len(page.Items) != 0 && !want || len(page.Items) != 1 && want {
			t.Fatal("dynamic moderation view membership mismatch")
		}
		if want && (page.Items[0].ID != record.ID || !reflect.DeepEqual(page.Items[0].HiddenReasons, reasons)) {
			t.Fatal("dynamic moderation reasons/identity mismatch")
		}
	}
	w := f.http(t, "", record.ID)
	s22AssertStatus(t, w, 200)
	got := adminPGAssertPayload(t, f, w.Body.Bytes(), record.ID)
	if !reflect.DeepEqual(got.HiddenReasons, reasons) || (got.ViolationCategory != nil) != marked {
		t.Fatal("dynamic detail reasons/category mismatch")
	}
	adminPGAssertView(t, f, record.ID)
	adminPGAssertOldModerationReads(t, f, record, owner, marked)
}

func adminPGAssertOldModerationReads(t *testing.T, f *adminPGFixture, record article.Article, owner string, marked bool) {
	t.Helper()
	member, err := article.NewQueryHandler(f.query)
	noticeMust(t, err)
	w := f.s.request(member.GetArticle, f.s.other, http.MethodGet, "/api/v1/articles/"+record.ID, nil, "")
	memberStatus := 404
	if owner == "active" && !marked {
		memberStatus = 200
	}
	s22AssertStatus(t, w, memberStatus)
	author, err := article.NewAuthorDetailHandler(f.query)
	noticeMust(t, err)
	p := f.s.author
	if owner == "suspended" {
		p = f.s.suspended
	}
	w = f.s.request(author.GetAuthorArticle, p, http.MethodGet, "/api/v1/users/me/articles/"+record.ID, nil, "")
	if owner != "active" {
		status := 404
		if owner == "suspended" {
			status = 403
		}
		s22AssertStatus(t, w, status)
		return
	}
	s22AssertStatus(t, w, 200)
	if !reflect.DeepEqual(s22Keys(t, w.Body.Bytes()), []string{"body", "editable", "moderation_status", "title"}) {
		t.Fatal("dynamic author projection fields changed")
	}
	var view article.AuthorArticleView
	noticeMust(t, json.Unmarshal(w.Body.Bytes(), &view))
	want := article.ModerationNormal
	if marked {
		want = article.ModerationRestricted
	}
	if view.ModerationStatus != want || !view.Editable || view.Title != record.Title || view.Body != record.Body {
		t.Fatal("dynamic author projection/content mismatch")
	}
}

func adminPGStrictIdentity(t *testing.T) {
	for _, mode := range []string{"nil", "missing", "empty", "invalid", "deleted", "error"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdminPG(t)
			f.seedDense()
			f.reader.mode, f.reader.failAt = mode, 2
			w := f.http(t, "view=violating", "")
			adminPGError(t, w, 503, platform.CodePlatformDependencyUnavailable)
			if len(f.reader.ids) != 2 || strings.Contains(w.Body.String(), "next_cursor") {
				t.Fatal("second-batch identity failure returned half-page/retried")
			}
		})
	}
}

func adminPGIdentityScope(t *testing.T) {
	f := newAdminPG(t)
	author := f.s.author.UserID
	for i := 3; i >= 1; i-- {
		r := f.s.base(i)
		if i != 1 {
			r.AuthorID = nil
		}
		f.s.insert(r)
	}
	adminPGPage(t, f.http(t, "", ""))
	if len(f.reader.ids) != 1 || !reflect.DeepEqual(f.reader.ids[0], []string{author}) {
		t.Fatal("identity scope did not deduplicate current authors")
	}
	other := newAdminPG(t)
	rows := []article.Article{other.s.base(3), other.s.base(2), other.s.base(1)}
	rows[0].DeletionState, rows[0].DeletionOperationID = article.DeletionPending, &[]string{noticeUUID()}[0]
	rows[1].AuthorID = nil
	rows[2].AuthorID = nil
	rows[2].OrphanedFromUserID = &other.s.deleted.UserID
	rows[2].PublicationState = article.PublicationHeld
	for _, r := range rows {
		other.s.insert(r)
	}
	adminPGPage(t, other.http(t, "", ""))
	if len(other.reader.ids) != 0 {
		t.Fatal("pending/anonymous/source caused identity query")
	}
}

func adminPGDetailPriority(t *testing.T) {
	f := newAdminPG(t)
	r := f.s.base(1)
	r.DeletionState, r.DeletionOperationID = article.DeletionPending, &[]string{noticeUUID()}[0]
	f.s.insert(r)
	f.reader.mode = "error"
	for _, id := range []string{"invalid", f.s.base(99).ID, r.ID} {
		t.Run(id, func(t *testing.T) {
			adminPGError(t, f.http(t, "", id), 404, article.CodeArticleUnavailable)
			if len(f.reader.ids) != 0 {
				t.Fatal("pending/absent priority queried identity")
			}
		})
	}
}

func adminPGSQLFaults(t *testing.T) {
	for _, mode := range []string{"query", "scan", "rows", "detail_scan", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdminPG(t)
			f.seedDense()
			f.db.mode = mode
			f.db.failAt = 2
			id := ""
			if mode == "detail_scan" {
				id = f.s.base(15).ID
			}
			if mode == "cancel" {
				f.db.afterSQL = func(_ context.Context, _ int64) error { f.s.h.cancel(); return context.Canceled }
			}
			w := f.http(t, "view=violating", id)
			adminPGError(t, w, 503, platform.CodePlatformDatabaseUnavailable)
			if mode != "detail_scan" && (f.db.calls.Load() != f.db.closed.Load() || f.db.calls.Load() > 2) {
				t.Fatal("SQL failure leaked rows/retried")
			}
			if f.s.h.pool.Stat().AcquiredConns() != 0 {
				t.Fatal("rows/Scan/cancel leaked connection")
			}
		})
	}
}

func adminPGSnapshot(t *testing.T, f *adminPGFixture) [32]byte {
	t.Helper()
	parts := []string{}
	for _, table := range []string{"articles", "users", "audit_events", "transfer_notices", "transfer_notice_recipients", "deletion_operations"} {
		var raw string
		noticeMust(t, f.s.h.pool.QueryRow(f.s.ctx, "SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY to_jsonb(x)::text),'[]'::jsonb)::text FROM "+pgx.Identifier{table}.Sanitize()+" x").Scan(&raw))
		parts = append(parts, raw)
	}
	return sha256.Sum256([]byte(strings.Join(parts, "\n")))
}

func adminPGReadOnly(t *testing.T) {
	f := newAdminPG(t)
	f.seedMatrix()
	before := adminPGSnapshot(t, f)
	for _, view := range []string{"all", "violating", "hidden"} {
		f.traverse(t, view)
	}
	f.http(t, "", f.s.base(3).ID)
	f.http(t, "view=%ZZ", "")
	f.reader.mode = "error"
	f.http(t, "", f.s.base(3).ID)
	if before != adminPGSnapshot(t, f) {
		t.Fatal("GET success/rejection/fault changed business snapshot")
	}
	t.Log("admin2_business_snapshot=unchanged")
}

func adminPGFields(t *testing.T) {
	f := newAdminPG(t)
	rows := f.seedMatrix()
	for _, r := range rows[1:] {
		t.Run(r.ID, func(t *testing.T) { adminPGAssertView(t, f, r.ID) })
	}
	for _, view := range []string{"all", "violating", "hidden"} {
		t.Run("list_"+view, func(t *testing.T) { adminPGAssertListPayloads(t, f, view) })
	}
}

func adminPGAssertView(t *testing.T, f *adminPGFixture, id string) {
	t.Helper()
	w := f.http(t, "", id)
	s22AssertStatus(t, w, 200)
	want := adminPGAssertPayload(t, f, w.Body.Bytes(), id)
	if w.Header().Get("ETag") != fmt.Sprintf(`"article-%s-v%d"`, id, want.Version) {
		t.Fatal("admin detail ETag differs from SQL record version")
	}
}

func adminPGAssertPayload(t *testing.T, f *adminPGFixture, payload []byte, id string) article.AdminArticleView {
	t.Helper()
	keys := s22Keys(t, payload)
	wantKeys := []string{"author", "body", "created_at", "hidden_reasons", "id", "orphaned_from_user_id", "publication_state", "title", "updated_at", "version", "violation_category", "violation_state"}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Fatal("Admin12 keys changed")
	}
	var raw []byte
	noticeMust(t, f.s.h.pool.QueryRow(f.s.ctx, adminPGViewOracleSQL, id).Scan(&raw))
	var want, got article.AdminArticleView
	noticeMust(t, json.Unmarshal(raw, &want))
	noticeMust(t, json.Unmarshal(payload, &got))
	want.CreatedAt, want.UpdatedAt = want.CreatedAt.UTC(), want.UpdatedAt.UTC()
	if !reflect.DeepEqual(got, want) {
		t.Fatal("Admin12 values differ from independent labeled SQL oracle")
	}
	return got
}

func adminPGAssertListPayloads(t *testing.T, f *adminPGFixture, view string) {
	t.Helper()
	cursor := ""
	ids := []string{}
	for index := 0; index < 30; index++ {
		raw := "view=" + view
		if cursor != "" {
			raw += "&cursor=" + cursor
		}
		w := f.http(t, raw, "")
		s22AssertStatus(t, w, 200)
		if !reflect.DeepEqual(s22Keys(t, w.Body.Bytes()), []string{"items", "next_cursor"}) {
			t.Fatal("list envelope keys changed")
		}
		var page struct {
			Items []json.RawMessage `json:"items"`
			Next  *string           `json:"next_cursor"`
		}
		noticeMust(t, json.Unmarshal(w.Body.Bytes(), &page))
		if page.Items == nil {
			t.Fatal("list items null")
		}
		for _, payload := range page.Items {
			var identity struct {
				ID string `json:"id"`
			}
			noticeMust(t, json.Unmarshal(payload, &identity))
			adminPGAssertPayload(t, f, payload, identity.ID)
			ids = append(ids, identity.ID)
		}
		if page.Next == nil {
			s22AssertSame(t, ids, f.oracle(t, view), "list payload sequence differs from independent SQL")
			return
		}
		cursor = *page.Next
	}
	t.Fatal("list payload traversal did not exhaust")
}

const adminPGViewOracleSQL = `SELECT jsonb_build_object(
 'id',a.id,'title',a.title,'body',a.body,
 'author',CASE WHEN a.author_id IS NULL THEN jsonb_build_object('kind','anonymous','label','匿名') ELSE jsonb_build_object('kind','user','id',a.author_id) END,
 'publication_state',a.publication_state,'violation_state',a.violation_state,'violation_category',a.violation_category,
 'hidden_reasons',array_remove(ARRAY[CASE WHEN a.author_id IS NOT NULL AND u.status='suspended' THEN 'author_suspended' END,CASE WHEN a.violation_state='violating' THEN 'violation' END,CASE WHEN a.publication_state='held' THEN 'orphan_hold' END],NULL),
 'orphaned_from_user_id',a.orphaned_from_user_id,'version',a.version,'created_at',a.created_at,'updated_at',a.updated_at)
 FROM articles a LEFT JOIN users u ON u.id=a.author_id WHERE a.id=$1`

func adminPGETag(t *testing.T) {
	f := newAdminPG(t)
	r := f.s.base(1)
	f.s.insert(r)
	f.db.afterRow = func() { f.s.h.exec("UPDATE articles SET body='later body',version=version+1 WHERE id=$1", r.ID) }
	w := f.http(t, "", r.ID)
	s22AssertStatus(t, w, 200)
	var got article.AdminArticleView
	noticeMust(t, json.Unmarshal(w.Body.Bytes(), &got))
	if got.Body != r.Body || got.Version != 1 || f.db.gets.Load() != 1 || w.Header().Get("ETag") != fmt.Sprintf(`"article-%s-v1"`, r.ID) {
		t.Fatal("concurrent writer caused mixed-record ETag/body")
	}
	if f.s.record(r.ID).Version != 2 {
		t.Fatal("ETag barrier writer did not commit")
	}
}

func adminPGCompatibility(t *testing.T) {
	f := newAdminPG(t)
	rows := f.seedMatrix()
	f.reader.mode = "missing"
	member, err := article.NewQueryHandler(f.query)
	noticeMust(t, err)
	w := f.s.request(member.GetArticle, f.s.author, http.MethodGet, "/api/v1/articles/"+rows[1].ID, nil, "")
	s22AssertStatus(t, w, 404)
	author, err := article.NewAuthorDetailHandler(f.query)
	noticeMust(t, err)
	w = f.s.request(author.GetAuthorArticle, f.s.author, http.MethodGet, "/api/v1/users/me/articles/"+rows[1].ID, nil, "")
	s22AssertStatus(t, w, 200)
	if len(s22Keys(t, w.Body.Bytes())) != 4 {
		t.Fatal("author field contract changed")
	}
	orphans, err := article.NewOrphanQueryService(f.s.repo, s22QueryConfig())
	noticeMust(t, err)
	page, err := orphans.ListOrphanArticles(f.s.ctx, f.s.admin, article.ListArticlesRequest{})
	noticeMust(t, err)
	if len(page.Items) != 2 {
		t.Fatal("orphan set changed")
	}
	// A typed invalid-state fault after a real SQL Scan tests N001 without disabling CHECK.
	fault := &adminPGInvalidRepository{real: f.s.repo}
	query, err := article.NewArticleQueryService(fault, f.s.reader, s22QueryConfig())
	noticeMust(t, err)
	handler, err := article.NewAdminQueryHandler(query)
	noticeMust(t, err)
	saved := f.admin
	f.admin = handler
	adminPGError(t, f.http(t, "", rows[1].ID), 500, platform.CodePlatformInternalError)
	f.admin = saved
	w = httptest.NewRecorder()
	noticeMust(t, article.WriteArticleError(w, httptest.NewRequest(http.MethodPost, "/", nil), article.ErrArticleInvalidState))
	s22AssertStatus(t, w, 409)
}

type adminPGInvalidRepository struct {
	real *article.PostgresArticleQueryRepository
}

func (r *adminPGInvalidRepository) ListCandidateBatch(ctx context.Context, after *article.ArticleCursor, limit int) ([]article.Article, error) {
	return r.real.ListCandidateBatch(ctx, after, limit)
}

func (r *adminPGInvalidRepository) GetArticle(ctx context.Context, id string) (article.Article, error) {
	row, err := r.real.GetArticle(ctx, id)
	if err == nil {
		row.Version = 0
	}
	return row, err
}
