//go:build integration

package articleintegration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/audit"
	"han/internal/identity"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// S2.2 owns these adapters only. The retained notice harness remains unchanged.
type s22Fixture struct {
	t                                        *testing.T
	h                                        *noticeHarness
	ctx                                      context.Context
	now                                      time.Time
	author, other, suspended, deleted, admin session.Principal
	reader                                   *identity.ArticleIdentityReader
	repo                                     *article.PostgresArticleQueryRepository
}

func s22New(t *testing.T) *s22Fixture {
	t.Helper()
	h := newNoticeHarness(t, true)
	ctx, err := platform.ContextWithTraceID(h.ctx, noticeTrace)
	noticeMust(t, err)
	s := &s22Fixture{t: t, h: h, ctx: ctx, now: h.now()}
	s.author = s.user("member", "active")
	s.other = s.user("member", "active")
	s.suspended = s.user("member", "suspended")
	s.deleted = s.user("member", "deleted")
	s.admin = s.user("admin", "active")
	s.reader, err = identity.NewArticleIdentityReader(h.pool)
	noticeMust(t, err)
	s.repo, err = article.NewPostgresArticleQueryRepository(h.pool)
	noticeMust(t, err)
	t.Logf("s22_case=%s schema=%s", t.Name(), h.schema)
	t.Cleanup(func() {
		h.cancel()
		h.group.Wait()
		if acquired := h.pool.Stat().AcquiredConns(); acquired != 0 {
			t.Errorf("S2.2 runtime leaked %d acquired connections", acquired)
		}
		t.Log("s22_resources=workers_joined,transactions_returned; pool_close=registered")
	})
	return s
}
func (s *s22Fixture) user(role, status string) session.Principal {
	s.t.Helper()
	id := noticeUUID()
	email := id + "@example.test"
	if status == "deleted" {
		s.h.exec(`INSERT INTO users(id,username,role,status,credential_mode,retained_email,created_at,updated_at,deleted_at) VALUES($1,$2,$3,'deleted','permanent',$4,$5,$5,$5)`, id, "s22-"+id, role, email, s.now)
	} else {
		s.h.exec(`INSERT INTO users(id,username,email,phone,role,status,credential_mode,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,'permanent',$7,$7)`, id, "s22-"+id, email, "phone-"+id, role, status, s.now)
	}
	return session.Principal{UserID: id, Role: role, Status: status}
}
func (s *s22Fixture) base(index int) article.Article {
	id := fmt.Sprintf("00000000-0000-4000-8000-%012x", index)
	return article.Article{ID: id, AuthorID: &s.author.UserID, Title: fmt.Sprintf("title-%d", index), Body: "literal <b>正文</b> **text**", PublicationState: article.PublicationPublished, ViolationState: article.ViolationClear, DeletionState: article.DeletionActive, Version: 1, CreatedAt: s.now, UpdatedAt: s.now}
}
func (s *s22Fixture) insert(r article.Article) {
	s.t.Helper()
	tx, err := s.h.pool.Begin(s.ctx)
	noticeMust(s.t, err)
	defer tx.Rollback(context.Background())
	if r.DeletionState == article.DeletionPending {
		op := article.DeletionOperation{ID: *r.DeletionOperationID, ArticleID: r.ID, ArticleVersion: r.Version, DeleteKind: article.DeleteOrdinary, Status: article.DeletionOperationStatus("pending_journal"), StartedAt: s.now}
		noticeMust(s.t, (article.Repository{}).InsertDeletionOperation(s.ctx, tx, op))
	}
	noticeMust(s.t, (article.Repository{}).InsertArticle(s.ctx, tx, r))
	noticeMust(s.t, tx.Commit(s.ctx))
}
func (s *s22Fixture) record(id string) article.Article {
	s.t.Helper()
	r, err := s.repo.GetArticle(s.ctx, id)
	noticeMust(s.t, err)
	return r
}
func (s *s22Fixture) snapshot(id string) string {
	s.t.Helper()
	var raw string
	noticeMust(s.t, s.h.pool.QueryRow(s.ctx, `SELECT jsonb_build_object('article',(SELECT to_jsonb(a) FROM articles a WHERE id=$1),'audit',(SELECT coalesce(jsonb_agg(to_jsonb(e) ORDER BY e.id),'[]'::jsonb) FROM audit_events e))::text`, id).Scan(&raw))
	return raw
}
func s22QueryConfig() article.QueryConfig {
	return article.QueryConfig{DefaultLimit: 2, MaxLimit: 2, CandidateBatchSize: 3}
}
func (s *s22Fixture) query() *article.ArticleQueryService {
	s.t.Helper()
	q, err := article.NewArticleQueryService(s.repo, s.reader, s22QueryConfig())
	noticeMust(s.t, err)
	return q
}

type s22Clock struct{ now time.Time }

func (c s22Clock) Now() time.Time { return c.now }
func (s *s22Fixture) command(options ...func(*article.CommandDependencies)) *article.CommandService {
	s.t.Helper()
	d := article.CommandDependencies{Transactions: s.h.pool, Repository: article.Repository{}, IDs: noticeIDs{}, Clock: s22Clock{s.now.Add(time.Second)}, Moderation: &article.ModerationDependencies{Auditor: audit.Store{}, AuthorStates: s22AuthorStates}}
	for _, o := range options {
		o(&d)
	}
	c, err := article.NewCommandService(d)
	noticeMust(s.t, err)
	return c
}
func s22AuthorStates(ctx context.Context, tx pgx.Tx, ids []string) (map[string]string, error) {
	r, err := identity.NewArticleIdentityReader(tx)
	if err != nil {
		return nil, err
	}
	states, err := r.GetAuthorStates(ctx, ids)
	out := map[string]string{}
	for id, status := range states {
		out[id] = string(status)
	}
	return out, err
}

type s22Principal struct{ principal session.Principal }

func (p s22Principal) ResolvePrincipal(context.Context, session.ResolveInput) (session.Principal, error) {
	return p.principal, nil
}
func (s *s22Fixture) handler(c *article.CommandService) *article.Handler {
	s.t.Helper()
	logger, err := article.NewSlogArticleResponseWriteLogger(slog.New(slog.NewJSONHandler(io.Discard, nil)), "han-server")
	noticeMust(s.t, err)
	h, err := article.NewHandler(c, logger)
	noticeMust(s.t, err)
	return h
}
func (s *s22Fixture) request(fn http.HandlerFunc, p session.Principal, method, path string, body []byte, etag string) *httptest.ResponseRecorder {
	s.t.Helper()
	auth, err := platform.NewHTTPAuthorizer(s22Principal{p}, func() time.Time { return s.now })
	noticeMust(s.t, err)
	r := httptest.NewRequest(method, path, bytes.NewReader(body)).WithContext(s.ctx)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
	r.Header.Set("If-Match", etag)
	r.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
	w := httptest.NewRecorder()
	auth.Protect(session.AccessGeneral, fn).ServeHTTP(w, r)
	return w
}
func s22Body(input article.CreateArticleInput) []byte {
	b, err := json.Marshal(input)
	if err != nil {
		panic("fixture encoding")
	}
	return b
}
func s22Tag(t *testing.T, r article.Article) string {
	t.Helper()
	tag, err := article.StrongETag(r)
	noticeMust(t, err)
	return tag
}
func s22Keys(t *testing.T, body []byte) []string {
	t.Helper()
	var v map[string]json.RawMessage
	noticeMust(t, json.Unmarshal(body, &v))
	keys := []string{}
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func s22AssertStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("HTTP status=%d expected=%d (raw body withheld)", w.Code, status)
	}
}
func s22AssertSame(t *testing.T, before, after any, message string) {
	t.Helper()
	if !reflect.DeepEqual(before, after) {
		t.Fatal(message)
	}
}

// TestArticleState proves SQL-backed state, projection and rollback invariants.
func TestArticleState(t *testing.T) {
	t.Run("VisibilityMatrix", s22Visibility)
	t.Run("ThreeProjections", s22Projections)
	t.Run("CurrentAuthorState", s22CurrentAuthor)
	t.Run("AuthorStateFailureClosed", s22AuthorFailure)
	t.Run("ContentFieldIsolation", s22ContentIsolation)
	t.Run("InvalidContentRollback", s22InvalidContent)
	t.Run("WireBoundaryPersistence", s22Wire)
	t.Run("ModerationStateAndAuditRollback", s22Moderation)
}

type s22StateCase struct {
	name, owner                         string
	violating, pending, absent, visible bool
}

func s22Visibility(t *testing.T) {
	cases := []s22StateCase{{name: "active", owner: "active", visible: true}, {name: "anonymous", owner: "anonymous", visible: true}, {name: "suspended", owner: "suspended"}, {name: "deleted_author", owner: "deleted"}, {name: "violating", owner: "active", violating: true}, {name: "held", owner: "held"}, {name: "held_violating", owner: "held", violating: true}, {name: "pending", owner: "active", pending: true}, {name: "anonymous_pending", owner: "anonymous", pending: true}, {name: "absent", absent: true}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := s22New(t)
			r := s22StateRecord(s, tc)
			if !tc.absent {
				s.insert(r)
			}
			q := s.query()
			view, _, err := q.GetArticle(s.ctx, r.ID)
			if tc.visible {
				noticeMust(t, err)
				if view.ID != r.ID {
					t.Fatal("member detail lost identity")
				}
			} else if !errors.Is(err, article.ErrArticleUnavailable) {
				t.Fatal("hidden/absent detail did not fail closed")
			}
			h, err := article.NewQueryHandler(q)
			noticeMust(t, err)
			w := s.request(h.GetArticle, s.other, http.MethodGet, "/api/v1/articles/"+r.ID, nil, "")
			if tc.visible {
				s22AssertStatus(t, w, 200)
			} else {
				s22AssertStatus(t, w, 404)
				if strings.Contains(w.Body.String(), r.Title) || strings.Contains(w.Body.String(), "hidden_reasons") {
					t.Fatal("hidden detail disclosed content/reasons")
				}
			}
		})
	}
}
func s22StateRecord(s *s22Fixture, tc s22StateCase) article.Article {
	r := s.base(1)
	switch tc.owner {
	case "anonymous":
		r.AuthorID = nil
	case "suspended":
		r.AuthorID = &s.suspended.UserID
	case "deleted":
		r.AuthorID = &s.deleted.UserID
	case "held":
		r.AuthorID = nil
		r.OrphanedFromUserID = &s.deleted.UserID
		r.PublicationState = article.PublicationHeld
	}
	if tc.violating {
		category := article.ViolationViolence
		r.ViolationState = article.ViolationViolating
		r.ViolationCategory = &category
	}
	if tc.pending {
		op := noticeUUID()
		r.DeletionState = article.DeletionPending
		r.DeletionOperationID = &op
	}
	return r
}
func s22Projections(t *testing.T) {
	for _, violating := range []bool{false, true} {
		t.Run(fmt.Sprintf("violating_%t", violating), func(t *testing.T) {
			s := s22New(t)
			r := s22StateRecord(s, s22StateCase{owner: "active", violating: violating})
			s.insert(r)
			q := s.query()
			h, err := article.NewAuthorDetailHandler(q)
			noticeMust(t, err)
			w := s.request(h.GetAuthorArticle, s.author, http.MethodGet, "/api/v1/users/me/articles/"+r.ID, nil, "")
			s22AssertStatus(t, w, 200)
			s22AssertSame(t, s22Keys(t, w.Body.Bytes()), []string{"body", "editable", "moderation_status", "title"}, "author allowlist leaked governance")
			var author article.AuthorArticleView
			noticeMust(t, json.Unmarshal(w.Body.Bytes(), &author))
			if (author.ModerationStatus == article.ModerationRestricted) != violating || !author.Editable || author.Body != r.Body {
				t.Fatal("author controlled view mismatch")
			}
			states, err := s.reader.GetAuthorStates(s.ctx, []string{s.author.UserID})
			noticeMust(t, err)
			admin, err := (article.VisibilityPolicy{}).AdminProjection(s.record(r.ID), s.admin, string(states[s.author.UserID]))
			noticeMust(t, err)
			if admin.Version != r.Version || (admin.ViolationCategory != nil) != violating || len(admin.HiddenReasons) != map[bool]int{false: 0, true: 1}[violating] {
				t.Fatal("administrator projection lost governance")
			}
			_, _, err = q.GetArticle(s.ctx, r.ID)
			if violating {
				if !errors.Is(err, article.ErrArticleUnavailable) {
					t.Fatal("violating member view visible")
				}
			} else {
				noticeMust(t, err)
			}
			if w.Header().Get("ETag") != s22Tag(t, s.record(r.ID)) {
				t.Fatal("author ETag detached from SQL version")
			}
		})
	}
}
func s22CurrentAuthor(t *testing.T) {
	s := s22New(t)
	r := s.base(1)
	s.insert(r)
	q := s.query()
	_, _, err := q.GetArticle(s.ctx, r.ID)
	noticeMust(t, err)
	for _, status := range []string{"suspended", "active"} {
		s.h.exec("UPDATE users SET status=$2 WHERE id=$1", s.author.UserID, status)
		_, _, err = q.GetArticle(s.ctx, r.ID)
		if status == "active" {
			noticeMust(t, err)
		} else if !errors.Is(err, article.ErrArticleUnavailable) {
			t.Fatal("author status was cached")
		}
	}
}

type s22ReaderFault struct {
	real   *identity.ArticleIdentityReader
	mode   string
	calls  int
	failAt int
}

func (r *s22ReaderFault) GetAuthorStates(ctx context.Context, ids []string) (map[string]identity.UserStatus, error) {
	states, err := r.real.GetAuthorStates(ctx, ids)
	if err != nil {
		return nil, err
	}
	r.calls++
	if r.failAt > 0 && r.calls != r.failAt {
		return states, nil
	}
	if r.mode == "missing" {
		for id := range states {
			delete(states, id)
		}
	} else {
		return nil, errors.New("synthetic failure after real identity SQL")
	}
	return states, nil
}
func s22AuthorFailure(t *testing.T) {
	for _, mode := range []string{"missing", "fault"} {
		t.Run(mode, func(t *testing.T) {
			s := s22New(t)
			r := s.base(1)
			s.insert(r)
			reader := &s22ReaderFault{real: s.reader, mode: mode}
			q, err := article.NewArticleQueryService(s.repo, reader, s22QueryConfig())
			noticeMust(t, err)
			page, err := q.ListArticles(s.ctx, article.ListArticlesRequest{})
			if mode == "missing" {
				noticeMust(t, err)
				if len(page.Items) != 0 {
					t.Fatal("missing author state treated as active")
				}
			} else if err == nil || len(page.Items) != 0 || page.NextCursor != nil {
				t.Fatal("identity failure returned partial page")
			}
		})
	}
}
func s22ContentIsolation(t *testing.T) {
	for _, violating := range []bool{false, true} {
		for _, size := range []string{"literal", "rune_boundary"} {
			t.Run(fmt.Sprintf("violating_%t_%s", violating, size), func(t *testing.T) {
				s := s22New(t)
				r := s22StateRecord(s, s22StateCase{owner: "active", violating: violating})
				s.insert(r)
				before := s.record(r.ID)
				input := article.CreateArticleInput{Title: "  标题 <x> **literal**  ", Body: "\n正文🙂 <script>literal</script>\n"}
				if size == "rune_boundary" {
					input.Title = strings.Repeat("界", 200)
					input.Body = strings.Repeat("🙂", 20000)
				}
				updated, err := s.command().ReplaceContent(s.ctx, s.author, r.ID, 1, input)
				noticeMust(t, err)
				after := s.record(r.ID)
				expected := before
				expected.Title = input.Title
				expected.Body = input.Body
				expected.Version++
				expected.UpdatedAt = updated.UpdatedAt
				s22AssertSame(t, after, expected, "content update changed governance/ownership/sort fields")
				if s.h.scalar("SELECT count(*) FROM audit_events") != 0 {
					t.Fatal("pure content replacement created a fictitious audit")
				}
				if violating {
					_, _, err = s.query().GetArticle(s.ctx, r.ID)
					if !errors.Is(err, article.ErrArticleUnavailable) {
						t.Fatal("edit cleared member restriction")
					}
				}
				q := s.query()
				author, _, err := q.GetAuthorArticle(s.ctx, s.author, r.ID)
				noticeMust(t, err)
				if author.Title != input.Title || author.Body != input.Body || (author.ModerationStatus == article.ModerationRestricted) != violating {
					t.Fatal("edited author view lost literal content/restriction")
				}
				states, err := s.reader.GetAuthorStates(s.ctx, []string{s.author.UserID})
				noticeMust(t, err)
				admin, err := (article.VisibilityPolicy{}).AdminProjection(after, s.admin, string(states[s.author.UserID]))
				noticeMust(t, err)
				if admin.Title != input.Title || admin.Body != input.Body || (admin.ViolationCategory != nil) != violating {
					t.Fatal("edited administrator view lost content/category")
				}
				ah, err := article.NewAuthorDetailHandler(q)
				noticeMust(t, err)
				w := s.request(ah.GetAuthorArticle, s.author, http.MethodGet, "/api/v1/users/me/articles/"+r.ID, nil, "")
				s22AssertStatus(t, w, 200)
				var wire article.AuthorArticleView
				noticeMust(t, json.Unmarshal(w.Body.Bytes(), &wire))
				s22AssertSame(t, wire, author, "JSON output did not preserve exact literal content")
			})
		}
	}
}
func s22InvalidContent(t *testing.T) {
	cases := []article.CreateArticleInput{{Title: "", Body: "valid"}, {Title: "\u2003\u3000", Body: "valid"}, {Title: strings.Repeat("界", 201), Body: "valid"}, {Title: "valid", Body: strings.Repeat("🙂", 20001)}, {Title: "valid", Body: ""}, {Title: "valid", Body: "\u2003\u3000"}}
	for _, violating := range []bool{false, true} {
		for i, input := range cases {
			t.Run(fmt.Sprintf("violating_%t_input_%d", violating, i), func(t *testing.T) {
				s := s22New(t)
				r := s22StateRecord(s, s22StateCase{owner: "active", violating: violating})
				s.insert(r)
				before := s.snapshot(r.ID)
				h := s.handler(s.command())
				w := s.request(h.UpdateArticle, s.author, http.MethodPut, "/api/v1/articles/"+r.ID, s22Body(input), s22Tag(t, r))
				s22AssertStatus(t, w, 422)
				s22AssertSame(t, s.snapshot(r.ID), before, "invalid content persisted a mutation")
			})
		}
	}
}

type s22CountingTx struct {
	pool   *pgxpool.Pool
	begins atomic.Int64
}

func (c *s22CountingTx) Begin(ctx context.Context) (pgx.Tx, error) {
	c.begins.Add(1)
	return c.pool.Begin(ctx)
}
func s22Wire(t *testing.T) {
	for _, violating := range []bool{false, true} {
		for _, kind := range []string{"N", "Nplus1", "invalid_utf8"} {
			t.Run(fmt.Sprintf("violating_%t_%s", violating, kind), func(t *testing.T) {
				s := s22New(t)
				r := s22StateRecord(s, s22StateCase{owner: "active", violating: violating})
				s.insert(r)
				before := s.snapshot(r.ID)
				counter := &s22CountingTx{pool: s.h.pool}
				cmd := s.command(func(d *article.CommandDependencies) { d.Transactions = counter })
				h := s.handler(cmd)
				input := article.CreateArticleInput{Title: "  界  ", Body: " raw <b>🙂</b> "}
				raw := s22Body(input)
				size := article.MaxArticleWriteBodyBytes
				if kind == "Nplus1" {
					size++
				}
				raw = append(raw, bytes.Repeat([]byte(" "), size-len(raw))...)
				status := 200
				if kind == "invalid_utf8" {
					raw = []byte{'{', '"', 't', 'i', 't', 'l', 'e', '"', ':', '"', 0xff, '"', ',', '"', 'b', 'o', 'd', 'y', '"', ':', '"', 'x', '"', '}'}
					status = 400
				}
				if kind == "Nplus1" {
					status = 400
				}
				w := s.request(h.UpdateArticle, s.author, http.MethodPut, "/api/v1/articles/"+r.ID, raw, s22Tag(t, r))
				s22AssertStatus(t, w, status)
				if status == 400 {
					s22AssertSame(t, s.snapshot(r.ID), before, "wire rejection changed SQL snapshot")
					if counter.begins.Load() != 0 {
						t.Fatal("wire rejection entered persistence")
					}
				} else {
					after := s.record(r.ID)
					if after.Version != 2 || after.Title != input.Title || after.Body != input.Body || counter.begins.Load() != 1 {
						t.Fatal("N-byte request did not persist exactly once")
					}
					expected := r
					expected.Title = input.Title
					expected.Body = input.Body
					expected.Version = 2
					expected.UpdatedAt = after.UpdatedAt
					s22AssertSame(t, after, expected, "N-byte edit changed non-content fields")
				}
			})
		}
	}
}

type s22AuditFault struct{ real audit.Store }

func (a s22AuditFault) Append(ctx context.Context, tx pgx.Tx, event audit.Event) error {
	if err := a.real.Append(ctx, tx, event); err != nil {
		return err
	}
	return errors.New("synthetic failure after real audit insertion")
}
func s22Moderation(t *testing.T) {
	t.Run("MarkRestoreAndConstraints", s22ModerationStates)
	t.Run("AuditFailureRollsBackAndUnlocks", s22AuditRollback)
	t.Run("UpdateFailureRollsBack", s22UpdateRollback)
}
func s22ModerationStates(t *testing.T) {
	for _, owner := range []string{"active", "suspended", "held"} {
		t.Run(owner, func(t *testing.T) {
			s := s22New(t)
			r := s22StateRecord(s, s22StateCase{owner: owner})
			s.insert(r)
			cmd := s.command()
			category := article.ViolationSexual
			_, _, err := cmd.Moderate(s.ctx, s.admin, r.ID, 1, &category)
			noticeMust(t, err)
			marked := s.record(r.ID)
			expected := r
			expected.ViolationState = article.ViolationViolating
			expected.ViolationCategory = &category
			expected.Version = 2
			expected.UpdatedAt = marked.UpdatedAt
			s22AssertSame(t, marked, expected, "mark changed non-governance fields")
			_, _, err = cmd.Moderate(s.ctx, s.admin, r.ID, 2, nil)
			noticeMust(t, err)
			restored := s.record(r.ID)
			expected = r
			expected.Version = 3
			expected.UpdatedAt = restored.UpdatedAt
			s22AssertSame(t, restored, expected, "restore changed original visibility constraints")
			_, _, err = s.query().GetArticle(s.ctx, r.ID)
			if owner == "active" {
				noticeMust(t, err)
			} else if !errors.Is(err, article.ErrArticleUnavailable) {
				t.Fatal("restore removed author/orphan restriction")
			}
			s22AssertModerationAudit(s, r.ID, map[int64]s22ModerationAuditExpectation{
				2: {action: "article.violation.mark", category: article.ViolationSexual},
				3: {action: "article.violation.clear", category: article.ViolationSexual},
			})
			before := s.snapshot(r.ID)
			_, _, err = cmd.Moderate(s.ctx, s.admin, r.ID, 3, nil)
			if !errors.Is(err, article.ErrArticleInvalidState) {
				t.Fatal("restore on clear was not invalid_state")
			}
			s22AssertSame(t, s.snapshot(r.ID), before, "invalid restore mutated state")
		})
	}
}

type s22ModerationAuditExpectation struct {
	action   string
	category article.ViolationCategory
}

func s22AssertModerationAudit(s *s22Fixture, id string, expected map[int64]s22ModerationAuditExpectation) {
	s.t.Helper()
	if s.h.scalar("SELECT count(*) FROM audit_events WHERE target_id=$1", id) != int64(len(expected)) {
		s.t.Fatal("moderation audit count mismatch")
	}
	rows, err := s.h.pool.Query(s.ctx, "SELECT id::text FROM audit_events WHERE target_id=$1", id)
	noticeMust(s.t, err)
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var eid string
		noticeMust(s.t, rows.Scan(&eid))
		ids = append(ids, eid)
	}
	err = rows.Err()
	rows.Close()
	noticeMust(s.t, err)
	versions := map[int64]bool{}
	for _, eid := range ids {
		e, err := (audit.Store{}).Get(s.ctx, s.h.pool, eid)
		noticeMust(s.t, err)
		version, ok := e.Metadata["article_version"].(float64)
		if e.TargetID != id || e.TargetType != "article" || e.ActorKind != audit.ActorUser || e.ActorUserID == nil || *e.ActorUserID != s.admin.UserID || e.ActorRole == nil || *e.ActorRole != "admin" || e.TraceID != noticeTrace || e.Outcome != "succeeded" || !ok || version != float64(int64(version)) || versions[int64(version)] {
			s.t.Fatal("moderation audit actor/target/trace/version binding mismatch")
		}
		want, exists := expected[int64(version)]
		category, categoryOK := e.Metadata["category"].(string)
		if !exists || e.Action != want.action || !categoryOK || category != string(want.category) {
			s.t.Fatalf("moderation audit action/category mismatch at version %d", int64(version))
		}
		versions[int64(version)] = true
	}
	for version := range expected {
		if !versions[version] {
			s.t.Fatal("moderation audit missing committed version")
		}
	}
}
func s22AuditRollback(t *testing.T) {
	s := s22New(t)
	r := s.base(1)
	s.insert(r)
	before := s.snapshot(r.ID)
	cmd := s.command(func(d *article.CommandDependencies) { d.Moderation.Auditor = s22AuditFault{} })
	category := article.ViolationViolence
	_, _, err := cmd.Moderate(s.ctx, s.admin, r.ID, 1, &category)
	if err == nil {
		t.Fatal("audit fault was swallowed")
	}
	s22AssertSame(t, s.snapshot(r.ID), before, "real update/audit transaction did not roll back")
	ctx, cancel := context.WithTimeout(s.ctx, time.Second)
	defer cancel()
	tx, err := s.h.pool.Begin(ctx)
	noticeMust(t, err)
	_, err = (article.Repository{}).LockArticleForUpdate(ctx, tx, r.ID)
	noticeMust(t, err)
	noticeMust(t, tx.Rollback(ctx))
	_, _, err = s.command().Moderate(s.ctx, s.admin, r.ID, 1, &category)
	noticeMust(t, err)
	s22AssertModerationAudit(s, r.ID, map[int64]s22ModerationAuditExpectation{
		2: {action: "article.violation.mark", category: article.ViolationViolence},
	})
}

type s22ContentFault struct{ article.Repository }

func (r s22ContentFault) UpdateArticleContent(ctx context.Context, tx pgx.Tx, a article.Article) error {
	if err := r.Repository.UpdateArticleContent(ctx, tx, a); err != nil {
		return err
	}
	return errors.New("synthetic fault after real content SQL")
}
func s22UpdateRollback(t *testing.T) {
	s := s22New(t)
	r := s.base(1)
	s.insert(r)
	before := s.snapshot(r.ID)
	cmd := s.command(func(d *article.CommandDependencies) { d.Repository = s22ContentFault{} })
	_, err := cmd.ReplaceContent(s.ctx, s.author, r.ID, 1, article.CreateArticleInput{Title: "new", Body: "new"})
	if err == nil {
		t.Fatal("post-write fault was swallowed")
	}
	s22AssertSame(t, s.snapshot(r.ID), before, "content SQL fault failed atomic rollback")
}

// Concurrency adapters keep both pgx transactions and every target query real.
type s22Transactor struct {
	pool     *pgxpool.Pool
	lockHook func(context.Context, pgx.Tx) error
	began    chan uint32
}

func (x *s22Transactor) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := x.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if x.began != nil {
		x.began <- tx.Conn().PgConn().PID()
	}
	return &s22Tx{Tx: tx, hook: x.lockHook}, nil
}

type s22Tx struct {
	pgx.Tx
	hook func(context.Context, pgx.Tx) error
}

func (tx *s22Tx) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	r := tx.Tx.QueryRow(ctx, q, args...)
	if tx.hook != nil && strings.Contains(q, "FOR UPDATE") {
		return s22LockedRow{row: r, after: func() error { return tx.hook(ctx, tx.Tx) }}
	}
	return r
}

type s22LockedRow struct {
	row   pgx.Row
	after func() error
}

func (r s22LockedRow) Scan(dest ...any) error {
	if err := r.row.Scan(dest...); err != nil {
		return err
	}
	return r.after()
}

type s22QueryDB struct {
	real   article.ArticleQueryDatabase
	calls  atomic.Int64
	failAt int64
}

func (d *s22QueryDB) Query(ctx context.Context, q string, args ...any) (pgx.Rows, error) {
	rows, err := d.real.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if d.calls.Add(1) == d.failAt {
		rows.Close()
		return nil, errors.New("synthetic failure after real candidate SQL")
	}
	return rows, nil
}
func (d *s22QueryDB) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	return d.real.QueryRow(ctx, q, args...)
}
func s22QueryWith(s *s22Fixture, db article.ArticleQueryDatabase, reader article.ArticleAuthorStateReader[identity.UserStatus]) *article.ArticleQueryService {
	s.t.Helper()
	repo, err := article.NewPostgresArticleQueryRepository(db)
	noticeMust(s.t, err)
	q, err := article.NewArticleQueryService(repo, reader, s22QueryConfig())
	noticeMust(s.t, err)
	return q
}
func s22PageIDs(page article.ListArticlesPage) []string {
	ids := []string{}
	for _, r := range page.Items {
		ids = append(ids, r.ID)
	}
	return ids
}
func s22Oracle(s *s22Fixture, after *article.ArticleCursor) []string {
	s.t.Helper()
	var stamp, id any
	if after != nil {
		stamp = after.CreatedAt
		id = after.ID
	}
	rows, err := s.h.pool.Query(s.ctx, `SELECT a.id::text FROM articles a LEFT JOIN users u ON u.id=a.author_id WHERE a.deletion_state='active' AND a.publication_state='published' AND a.violation_state='clear' AND (a.author_id IS NULL OR u.status='active') AND ($1::timestamptz IS NULL OR (a.created_at,a.id)<($1::timestamptz,$2::uuid)) ORDER BY a.created_at DESC,a.id DESC`, stamp, id)
	noticeMust(s.t, err)
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		noticeMust(s.t, rows.Scan(&id))
		ids = append(ids, id)
	}
	noticeMust(s.t, rows.Err())
	return ids
}
func s22Traverse(s *s22Fixture, q *article.ArticleQueryService, cursor string) []string {
	s.t.Helper()
	ids := []string{}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		page, err := q.ListArticles(s.ctx, article.ListArticlesRequest{Cursor: cursor})
		noticeMust(s.t, err)
		for _, id := range s22PageIDs(page) {
			if seen[id] {
				s.t.Fatal("visible article duplicated across pages")
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if page.NextCursor == nil {
			return ids
		}
		if len(page.Items) == 0 {
			s.t.Fatal("empty page skipped stable visible tail")
		}
		decoded, err := (article.CursorCodec{}).Decode(*page.NextCursor, article.CursorViewMember)
		noticeMust(s.t, err)
		last := page.Items[len(page.Items)-1]
		if decoded.ID != last.ID || !decoded.CreatedAt.Equal(last.CreatedAt) {
			s.t.Fatal("cursor consumed lookahead/internal scan instead of last returned")
		}
		cursor = *page.NextCursor
	}
	s.t.Fatal("pagination failed to terminate")
	return nil
}
func s22Dense(s *s22Fixture) {
	for i := 1; i <= 20; i++ {
		r := s.base(i)
		if i > 3 && i != 20 {
			switch i % 4 {
			case 0:
				r.AuthorID = &s.suspended.UserID
			case 1:
				r.AuthorID = nil
				r.OrphanedFromUserID = &s.deleted.UserID
				r.PublicationState = article.PublicationHeld
			case 2:
				op := noticeUUID()
				r.DeletionState = article.DeletionPending
				r.DeletionOperationID = &op
			default:
				category := article.ViolationViolence
				r.ViolationState = article.ViolationViolating
				r.ViolationCategory = &category
			}
		}
		s.insert(r)
	}
}

type s22Operation struct{ kind, title string }

func s22OperationCall(s *s22Fixture, h *article.Handler, r article.Article, op s22Operation) *httptest.ResponseRecorder {
	if op.kind == "edit" {
		return s.request(h.UpdateArticle, s.author, http.MethodPut, "/api/v1/articles/"+r.ID, s22Body(article.CreateArticleInput{Title: op.title, Body: "writer-body"}), s22Tag(s.t, r))
	}
	if op.kind == "restore" {
		return s.request(h.RestoreArticle, s.admin, http.MethodPost, "/api/v1/admin/articles/"+r.ID+"/restore", nil, s22Tag(s.t, r))
	}
	return s.request(h.MarkViolation, s.admin, http.MethodPost, "/api/v1/admin/articles/"+r.ID+"/violation", []byte(`{"category":"violence"}`), s22Tag(s.t, r))
}
func s22Concurrent(s *s22Fixture, r article.Article, a, b s22Operation) {
	s.t.Helper()
	locked := make(chan uint32, 1)
	beganB := make(chan uint32, 1)
	release := make(chan struct{})
	var once sync.Once
	s.t.Cleanup(func() { once.Do(func() { close(release) }) })
	txA := &s22Transactor{pool: s.h.pool, lockHook: func(ctx context.Context, tx pgx.Tx) error {
		locked <- tx.Conn().PgConn().PID()
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	txB := &s22Transactor{pool: s.h.pool, began: beganB}
	ha := s.handler(s.command(func(d *article.CommandDependencies) { d.Transactions = txA }))
	hb := s.handler(s.command(func(d *article.CommandDependencies) { d.Transactions = txB }))
	outA, outB := make(chan *httptest.ResponseRecorder, 1), make(chan *httptest.ResponseRecorder, 1)
	s.h.group.Add(1)
	go func() { defer s.h.group.Done(); outA <- s22OperationCall(s, ha, r, a) }()
	var pid uint32
	select {
	case pid = <-locked:
	case <-s.ctx.Done():
		s.t.Fatal("writer A did not acquire row lock")
	}
	s.h.group.Add(1)
	go func() { defer s.h.group.Done(); outB <- s22OperationCall(s, hb, r, b) }()
	select {
	case other := <-beganB:
		if other == pid {
			s.t.Fatal("writers reused same connection")
		}
	case <-s.ctx.Done():
		s.t.Fatal("writer B did not begin")
	}
	s22WaitBlocked(s, pid)
	once.Do(func() { close(release) })
	ra, rb := <-outA, <-outB
	s22AssertStatus(s.t, ra, 200)
	s22AssertStatus(s.t, rb, 412)
	after := s.record(r.ID)
	if after.Version != r.Version+1 || ra.Header().Get("ETag") != s22Tag(s.t, after) {
		s.t.Fatal("concurrent version/response mismatch")
	}
	expected := r
	expected.Version++
	expected.UpdatedAt = after.UpdatedAt
	switch a.kind {
	case "edit":
		expected.Title, expected.Body = a.title, "writer-body"
	case "mark":
		category := article.ViolationViolence
		expected.ViolationState, expected.ViolationCategory = article.ViolationViolating, &category
	case "restore":
		expected.ViolationState, expected.ViolationCategory = article.ViolationClear, nil
	}
	s22AssertSame(s.t, after, expected, "losing writer altered committed fields or winner did not apply")
	if a.kind == "edit" {
		if after.Title != a.title || after.ViolationState != r.ViolationState {
			s.t.Fatal("losing governance writer changed state")
		}
		if s.h.scalar("SELECT count(*) FROM audit_events") != 0 {
			s.t.Fatal("losing governance left audit")
		}
	} else {
		want := s22ModerationAuditExpectation{action: "article.violation.mark", category: article.ViolationViolence}
		if a.kind == "restore" {
			want = s22ModerationAuditExpectation{action: "article.violation.clear", category: *r.ViolationCategory}
		}
		s22AssertModerationAudit(s, r.ID, map[int64]s22ModerationAuditExpectation{r.Version + 1: want})
	}
}
func s22WaitBlocked(s *s22Fixture, pid uint32) {
	s.t.Helper()
	ctx, cancel := context.WithTimeout(s.ctx, 3*time.Second)
	defer cancel()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count int64
		noticeMust(s.t, s.h.pool.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE $1::int=ANY(pg_blocking_pids(pid))", int(pid)).Scan(&count))
		if count > 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			s.t.Fatal("actual PostgreSQL lock wait not observed")
		}
	}
}
