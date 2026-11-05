package article

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

func TestAdminQuery(t *testing.T) {
	t.Run("PrincipalBoundary", adminUnitPrincipal)
	t.Run("RawQueryAndPresence", adminUnitRawQuery)
	t.Run("FiveViewIsolation", adminUnitCursors)
	t.Run("ViewStateMatrix", adminUnitViews)
	t.Run("StaticCompositeKeyset", adminUnitKeyset)
	t.Run("ScanAndPublicCursor", adminUnitScanCursor)
	t.Run("LimitsAndExhaustion", adminUnitLimits)
	t.Run("CurrentAuthorTransitions", adminUnitTransitions)
	t.Run("StrictIdentityWholeFailure", adminUnitIdentityFailure)
	t.Run("IdentityScope", adminUnitIdentityScope)
	t.Run("DetailPriority", adminUnitDetail)
	t.Run("CandidateInvariants", adminUnitInvariants)
	t.Run("SQLRowsCancel", adminUnitFaults)
	t.Run("ZeroBusinessWrites", adminUnitReadOnly)
	t.Run("TwelveFieldsAndPrivacy", adminUnitFields)
	t.Run("SameArticleETag", adminUnitETag)
	t.Run("ErrorAndOldConsumerCompatibility", adminUnitCompatibility)
}

type adminUnitRepository struct {
	rows   []Article
	afters []*ArticleCursor
	limits []int
	gets   int
	batch  func(context.Context, *ArticleCursor, int) ([]Article, error)
	get    func(context.Context, string) (Article, error)
}

func (r *adminUnitRepository) ListCandidateBatch(ctx context.Context, after *ArticleCursor, limit int) ([]Article, error) {
	var copied *ArticleCursor
	if after != nil {
		value := *after
		copied = &value
	}
	r.afters, r.limits = append(r.afters, copied), append(r.limits, limit)
	if r.batch != nil {
		return r.batch(ctx, after, limit)
	}
	rows := []Article{}
	for _, row := range r.rows {
		if after == nil || row.CreatedAt.Before(after.CreatedAt) || row.CreatedAt.Equal(after.CreatedAt) && row.ID < after.ID {
			rows = append(rows, row)
		}
		if len(rows) == limit {
			break
		}
	}
	return rows, nil
}

func (r *adminUnitRepository) GetArticle(ctx context.Context, id string) (Article, error) {
	r.gets++
	if r.get != nil {
		return r.get(ctx, id)
	}
	for _, row := range r.rows {
		if row.ID == id {
			return row, nil
		}
	}
	return Article{}, pgx.ErrNoRows
}

func (r *adminUnitRepository) ListOrphanBatch(ctx context.Context, after *ArticleCursor, limit int) ([]Article, error) {
	rows, err := r.ListCandidateBatch(ctx, after, len(r.rows)+1)
	orphans := []Article{}
	for _, row := range rows {
		if isDispositionOrphan(row) && len(orphans) < limit {
			orphans = append(orphans, row)
		}
	}
	return orphans, err
}

type adminUnitIdentity struct {
	states map[string]string
	ids    [][]string
	hook   func(context.Context, []string, int) (map[string]string, error)
}

func (r *adminUnitIdentity) GetAuthorStates(ctx context.Context, ids []string) (map[string]string, error) {
	r.ids = append(r.ids, append([]string(nil), ids...))
	if r.hook != nil {
		return r.hook(ctx, ids, len(r.ids))
	}
	result := map[string]string{}
	for _, id := range ids {
		if state, exists := r.states[id]; exists {
			result[id] = state
		}
	}
	return result, nil
}

type adminUnitQuerier struct {
	service *ArticleQueryService
	lists   int
	details int
	input   ListAdminArticlesRequest
}

func (q *adminUnitQuerier) ListAdminArticles(ctx context.Context, p session.Principal, input ListAdminArticlesRequest) (AdminArticlesPage, error) {
	q.lists++
	q.input = input
	return q.service.ListAdminArticles(ctx, p, input)
}

func (q *adminUnitQuerier) GetAdminArticle(ctx context.Context, p session.Principal, id string) (AdminArticleView, string, error) {
	q.details++
	return q.service.GetAdminArticle(ctx, p, id)
}

type adminUnitFixture struct {
	repo    *adminUnitRepository
	reader  *adminUnitIdentity
	service *ArticleQueryService
	spy     *adminUnitQuerier
	handler *Handler
}

func newAdminUnit(t *testing.T) *adminUnitFixture {
	t.Helper()
	author := queryTestID(900)
	f := &adminUnitFixture{repo: &adminUnitRepository{}, reader: &adminUnitIdentity{states: map[string]string{author: "active", queryTestID(901): "suspended"}}}
	f.repo.rows = []Article{queryTestArticle(3, &author), queryTestArticle(2, &author), queryTestArticle(1, &author)}
	var err error
	f.service, err = NewArticleQueryService(f.repo, f.reader, QueryConfig{DefaultLimit: 2, MaxLimit: 2, CandidateBatchSize: 3})
	if err != nil {
		t.Fatal(err)
	}
	f.spy = &adminUnitQuerier{service: f.service}
	f.handler, err = NewAdminQueryHandler(f.spy)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func adminUnitPrincipalValue() session.Principal {
	return session.Principal{UserID: queryTestID(999), Role: "admin", Status: "active"}
}

func adminUnitHTTP(t *testing.T, fn http.HandlerFunc, principal session.Principal, raw, id string, ctx context.Context) *httptest.ResponseRecorder {
	t.Helper()
	path := "/api/v1/admin/articles"
	if id != "" {
		path += "/" + id
	}
	r := httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx)
	r.URL.RawQuery = raw
	r.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-admin-session"})
	auth, err := platform.NewHTTPAuthorizer(articleQueryPrincipalResolver{principal: principal}, func() time.Time { return queryTestTime })
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	platform.TraceMiddleware(auth.Protect(session.AccessGeneral, fn)).ServeHTTP(w, r)
	return w
}

func (f *adminUnitFixture) list(t *testing.T, raw string) *httptest.ResponseRecorder {
	t.Helper()
	return adminUnitHTTP(t, f.handler.ListAdminArticles, adminUnitPrincipalValue(), raw, "", context.Background())
}

func (f *adminUnitFixture) detail(t *testing.T, id string) *httptest.ResponseRecorder {
	t.Helper()
	return adminUnitHTTP(t, f.handler.GetAdminArticle, adminUnitPrincipalValue(), "", id, context.Background())
}

func adminUnitPage(t *testing.T, w *httptest.ResponseRecorder) AdminArticlesPage {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("HTTP=%d expected=200 (body withheld)", w.Code)
	}
	var page AdminArticlesPage
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	return page
}

func adminUnitError(t *testing.T, w *httptest.ResponseRecorder, status int, code platform.Code) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("HTTP=%d expected=%d (response body withheld)", w.Code, status)
	}
	var envelope platform.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal("invalid public error envelope")
	}
	if envelope.Code != code {
		t.Fatalf("error code=%s expected=%s", envelope.Code, code)
	}
}

func adminUnitPrincipal(t *testing.T) {
	for _, variant := range []string{"admin", "member", "suspended", "onboarding", "no_principal", "direct_service"} {
		t.Run(variant, func(t *testing.T) {
			f := newAdminUnit(t)
			p := adminUnitPrincipalValue()
			if variant == "member" || variant == "direct_service" {
				p.Role = "member"
			}
			if variant == "suspended" {
				p.Status = "suspended"
			}
			if variant == "onboarding" {
				p.MustChangePassword = true
			}
			if variant == "direct_service" {
				_, err := f.service.ListAdminArticles(context.Background(), p, ListAdminArticlesRequest{})
				if !errors.Is(err, ErrArticleForbidden) {
					t.Fatalf("direct service accepted: %v", err)
				}
				_, _, err = f.service.GetAdminArticle(context.Background(), p, queryTestID(3))
				if !errors.Is(err, ErrArticleForbidden) {
					t.Fatalf("direct detail accepted: %v", err)
				}
			} else if variant == "no_principal" {
				w := httptest.NewRecorder()
				f.handler.ListAdminArticles(w, httptest.NewRequest(http.MethodGet, "/", nil))
				adminUnitError(t, w, 401, platform.CodeAuthSessionRequired)
			} else {
				w := adminUnitHTTP(t, f.handler.ListAdminArticles, p, "role=admin", "", context.Background())
				if variant == "admin" {
					adminUnitPage(t, w)
					return
				}
				if w.Code != 403 {
					t.Fatalf("principal rejection HTTP=%d", w.Code)
				}
			}
			if len(f.repo.afters) != 0 || f.repo.gets != 0 || len(f.reader.ids) != 0 {
				t.Fatal("rejection read business data")
			}
		})
	}
	t.Run("dependency_fail_closed", adminUnitUnconfigured)
}

func adminUnitUnconfigured(t *testing.T) {
	if _, err := NewAdminQueryHandler(nil); err == nil {
		t.Fatal("nil Admin querier accepted")
	}
	h := &Handler{}
	w := adminUnitHTTP(t, h.ListAdminArticles, adminUnitPrincipalValue(), "", "", context.Background())
	adminUnitError(t, w, 500, platform.CodePlatformInternalError)
	w = adminUnitHTTP(t, h.GetAdminArticle, adminUnitPrincipalValue(), "", queryTestID(3), context.Background())
	adminUnitError(t, w, 500, platform.CodePlatformInternalError)
	var service *ArticleQueryService
	_, err := service.ListAdminArticles(context.Background(), adminUnitPrincipalValue(), ListAdminArticlesRequest{})
	if platform.PublicCode(err) != platform.CodePlatformInternalError {
		t.Fatal("nil service returned success")
	}
}

func adminUnitRawQuery(t *testing.T) {
	for index, raw := range []string{"view=%ZZ", "view=hidden&view=%ZZ", "view=all&cursor=%ZZ", "view=hidden&limit=%ZZ", "view=hidden;view=violating", "view=all&other=%ZZ", "view=", "view=all&view=hidden", "v%69ew=all&view=hidden", "view=ALL", "view=%20all", "view=unknown"} {
		t.Run(fmt.Sprintf("invalid_%02d", index), func(t *testing.T) {
			f := newAdminUnit(t)
			w := f.list(t, raw)
			adminUnitError(t, w, 422, CodeArticleValidationFailed)
			if f.spy.lists != 0 || len(f.repo.afters) != 0 || len(f.reader.ids) != 0 {
				t.Fatal("invalid raw query reached downstream")
			}
			if strings.Contains(w.Body.String(), raw) {
				t.Fatal("raw query leaked")
			}
		})
	}
	for index, raw := range []string{"", "view=all", "view=hidden", "view=violating", "limit=01", "limit=%2B1", "other=valid"} {
		t.Run(fmt.Sprintf("valid_%02d", index), func(t *testing.T) {
			f := newAdminUnit(t)
			adminUnitPage(t, f.list(t, raw))
			if f.spy.lists != 1 {
				t.Fatal("valid query not dispatched once")
			}
		})
	}
	t.Run("typed_zero_all", func(t *testing.T) {
		f := newAdminUnit(t)
		page, err := f.service.ListAdminArticles(context.Background(), adminUnitPrincipalValue(), ListAdminArticlesRequest{})
		if err != nil || len(page.Items) != 2 {
			t.Fatalf("typed zero = %d/%v", len(page.Items), err)
		}
	})
}

func adminUnitCursors(t *testing.T) {
	t.Run("repository_views", adminUnitRepositoryViews)
	for index, raw := range []string{"cursor=", "cursor=x&cursor=y", "cursor=%25ZZ"} {
		t.Run(fmt.Sprintf("malformed_%02d", index), func(t *testing.T) {
			f := newAdminUnit(t)
			adminUnitError(t, f.list(t, raw), 400, platform.CodeRequestInvalidCursor)
			if len(f.repo.afters) != 0 || len(f.reader.ids) != 0 {
				t.Fatal("bad cursor reached persistence")
			}
		})
	}
	for _, view := range []CursorView{CursorViewMember, CursorViewAdminAll, CursorViewAdminViolating, CursorViewAdminHidden, CursorViewAdminOrphan} {
		t.Run(string(view), func(t *testing.T) {
			encoded, err := (CursorCodec{}).Encode(ArticleCursor{CreatedAt: queryTestTime, ID: queryTestID(9), View: view})
			if err != nil {
				t.Fatal(err)
			}
			f := newAdminUnit(t)
			w := f.list(t, "view=hidden&cursor="+encoded)
			if view == CursorViewAdminHidden {
				adminUnitPage(t, w)
				return
			}
			adminUnitError(t, w, 400, platform.CodeRequestInvalidCursor)
			if len(f.repo.afters) != 0 {
				t.Fatal("cross view read data")
			}
		})
	}
}

type adminUnitCandidateDatabase struct {
	calls int
	sql   string
	args  []any
}

func (d *adminUnitCandidateDatabase) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	d.calls++
	d.sql, d.args = sql, args
	return nil, errors.New("controlled SQL probe")
}

func (*adminUnitCandidateDatabase) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

func adminUnitRepositoryViews(t *testing.T) {
	for _, view := range []CursorView{CursorViewMember, CursorViewAdminAll, CursorViewAdminViolating, CursorViewAdminHidden, CursorViewAdminOrphan} {
		t.Run(string(view), func(t *testing.T) {
			db := &adminUnitCandidateDatabase{}
			repo, err := NewPostgresArticleQueryRepository(db)
			if err != nil {
				t.Fatal(err)
			}
			cursor := ArticleCursor{ID: queryTestID(9), CreatedAt: queryTestTime, View: view}
			_, err = repo.ListCandidateBatch(context.Background(), &cursor, 3)
			if err == nil {
				t.Fatal("SQL probe unexpectedly returned success")
			}
			if view == CursorViewAdminOrphan {
				if db.calls != 0 {
					t.Fatal("orphan cursor reached ordinary candidate SQL")
				}
				return
			}
			if db.calls != 1 || db.sql != listArticleCandidatesSQL || !reflect.DeepEqual(db.args, []any{queryTestTime, queryTestID(9), 3}) {
				t.Fatal("candidate view/SQL/bindings changed")
			}
		})
	}
}

func adminUnitStateRows() []Article {
	author, suspended, source := queryTestID(900), queryTestID(901), queryTestID(902)
	rows := []Article{}
	for i := 8; i >= 1; i-- {
		rows = append(rows, queryTestArticle(i, &author))
	}
	rows[0].DeletionState, rows[0].DeletionOperationID = DeletionPending, stringPointer(queryTestID(903))
	rows[2].AuthorID = &suspended
	rows[3].AuthorID = nil
	rows[4].AuthorID, rows[4].OrphanedFromUserID, rows[4].PublicationState = nil, &source, PublicationHeld
	for _, index := range []int{5, 6, 7} {
		rows[index].ViolationState, rows[index].ViolationCategory = ViolationViolating, violationCategoryPointer(ViolationViolence)
	}
	rows[6].AuthorID = &suspended
	rows[7].AuthorID, rows[7].OrphanedFromUserID, rows[7].PublicationState = nil, &source, PublicationHeld
	return rows
}

func adminUnitTraverse(t *testing.T, f *adminUnitFixture, view string) []string {
	t.Helper()
	ids, cursor := []string{}, ""
	for page := 0; page < 30; page++ {
		raw := "view=" + view
		if cursor != "" {
			raw += "&cursor=" + cursor
		}
		value := adminUnitPage(t, f.list(t, raw))
		for _, item := range value.Items {
			ids = append(ids, item.ID)
		}
		if value.NextCursor == nil {
			return ids
		}
		cursor = *value.NextCursor
	}
	t.Fatal("admin pagination did not converge")
	return nil
}

func adminUnitViews(t *testing.T) {
	for view, sequence := range map[string][]int{"all": {7, 6, 5, 4, 3, 2, 1}, "violating": {3, 2, 1}, "hidden": {6, 4, 3, 2, 1}} {
		t.Run(view, func(t *testing.T) {
			f := newAdminUnit(t)
			f.repo.rows = adminUnitStateRows()
			want := []string{}
			for _, id := range sequence {
				want = append(want, queryTestID(id))
			}
			if got := adminUnitTraverse(t, f, view); !reflect.DeepEqual(got, want) {
				t.Fatalf("view IDs = %v, want %v", got, want)
			}
		})
	}
}

func adminUnitKeyset(t *testing.T) {
	for _, view := range []string{"all", "violating", "hidden"} {
		t.Run(view, func(t *testing.T) {
			f := newAdminUnit(t)
			f.repo.rows = nil
			want := []string{}
			for i, id := range []int{2, 1, 99, 98, 199, 198} {
				r := queryTestArticle(id, stringPointer(queryTestID(900)))
				r.CreatedAt = queryTestTime.Add(-time.Duration(i/2) * time.Second)
				r.ViolationState, r.ViolationCategory = ViolationViolating, violationCategoryPointer(ViolationSexual)
				f.repo.rows = append(f.repo.rows, r)
				want = append(want, r.ID)
			}
			if got := adminUnitTraverse(t, f, view); !reflect.DeepEqual(got, want) {
				t.Fatalf("composite keyset = %v, want %v", got, want)
			}
		})
	}
}

func adminUnitDense(f *adminUnitFixture) {
	f.repo.rows = nil
	for i := 15; i >= 1; i-- {
		r := queryTestArticle(i, stringPointer(queryTestID(900)))
		if i == 8 || i == 4 || i == 1 {
			r.ViolationState, r.ViolationCategory = ViolationViolating, violationCategoryPointer(ViolationSexual)
		}
		f.repo.rows = append(f.repo.rows, r)
	}
}

func adminUnitScanCursor(t *testing.T) {
	f := newAdminUnit(t)
	adminUnitDense(f)
	page := adminUnitPage(t, f.list(t, "view=violating"))
	if len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatal("lookahead page missing")
	}
	cursor, err := (CursorCodec{}).Decode(*page.NextCursor, CursorViewAdminViolating)
	if err != nil || cursor.ID != queryTestID(4) {
		t.Fatalf("public cursor = %+v/%v, want last returned 4", cursor, err)
	}
	if len(f.repo.afters) < 4 {
		t.Fatal("fixture did not scan multiple filtered batches")
	}
	for i, after := range f.repo.afters {
		if f.repo.limits[i] != 3 || i > 0 && (after == nil || after.View != CursorViewAdminViolating || after.ID != queryTestID(15-i*3+1)) {
			t.Fatalf("batch %d after/limit = %+v/%d", i, after, f.repo.limits[i])
		}
	}
	last := adminUnitPage(t, f.list(t, "view=violating&cursor="+*page.NextCursor))
	if len(last.Items) != 1 || last.Items[0].ID != queryTestID(1) || last.NextCursor != nil {
		t.Fatal("lookahead was consumed or duplicated")
	}
}

func adminUnitLimits(t *testing.T) {
	for index, raw := range []string{"limit=", "limit=1&limit=2", "limit=x", "limit=0", "limit=-1", "limit=3"} {
		t.Run(fmt.Sprintf("invalid_limit_%02d", index), func(t *testing.T) {
			f := newAdminUnit(t)
			adminUnitError(t, f.list(t, raw), 422, CodeArticleValidationFailed)
			if len(f.repo.afters) != 0 {
				t.Fatal("invalid limit read data")
			}
		})
	}
	for _, mode := range []string{"empty", "all_filtered", "last_page"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdminUnit(t)
			raw := "view=hidden"
			if mode == "empty" {
				f.repo.rows = nil
			}
			if mode == "last_page" {
				f.repo.rows = f.repo.rows[:1]
				raw = "view=all"
			}
			w := f.list(t, raw)
			page := adminUnitPage(t, w)
			if page.Items == nil || page.NextCursor != nil || w.Header().Get("ETag") != "" {
				t.Fatal("exhausted envelope or list ETag changed")
			}
		})
	}
}

func adminUnitTransitions(t *testing.T) {
	f := newAdminUnit(t)
	for _, state := range []string{"suspended", "active"} {
		f.reader.states[queryTestID(900)] = state
		page := adminUnitPage(t, f.list(t, "view=hidden"))
		if (len(page.Items) > 0) != (state == "suspended") {
			t.Fatal("author state was cached")
		}
	}
	f.repo.batch = func(_ context.Context, _ *ArticleCursor, _ int) ([]Article, error) {
		f.reader.states[queryTestID(900)] = "suspended"
		return f.repo.rows[:1], nil
	}
	page := adminUnitPage(t, f.list(t, "view=hidden"))
	if len(page.Items) != 1 || !reflect.DeepEqual(page.Items[0].HiddenReasons, []HiddenReason{HiddenAuthorSuspended}) {
		t.Fatal("candidate-to-identity transition lost current fact")
	}
}

func adminUnitIdentityFailure(t *testing.T) {
	for _, mode := range []string{"nil", "missing", "empty", "invalid", "deleted", "error"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdminUnit(t)
			adminUnitDense(f)
			// First batch has a match, so a later failure must discard accumulated data.
			f.repo.rows[0].ViolationState, f.repo.rows[0].ViolationCategory = ViolationViolating, violationCategoryPointer(ViolationSexual)
			f.reader.hook = func(_ context.Context, ids []string, call int) (map[string]string, error) {
				if call != 2 {
					return map[string]string{queryTestID(900): "active"}, nil
				}
				switch mode {
				case "nil":
					return nil, nil
				case "missing":
					return map[string]string{}, nil
				case "error":
					return nil, errors.New("synthetic identity unavailable")
				case "empty":
					return map[string]string{ids[0]: ""}, nil
				default:
					return map[string]string{ids[0]: mode}, nil
				}
			}
			w := f.list(t, "view=violating")
			adminUnitError(t, w, 503, platform.CodePlatformDependencyUnavailable)
			if len(f.reader.ids) != 2 || strings.Contains(w.Body.String(), "items") || strings.Contains(w.Body.String(), "next_cursor") {
				t.Fatal("strict identity returned a partial page or continued scanning")
			}
		})
	}
}

func adminUnitIdentityScope(t *testing.T) {
	f := newAdminUnit(t)
	rows := adminUnitStateRows()
	f.repo.rows = []Article{rows[0], rows[3], rows[4]}
	adminUnitPage(t, f.list(t, ""))
	if len(f.reader.ids) != 0 {
		t.Fatal("pending/anonymous/orphan source caused identity SQL")
	}
	f = newAdminUnit(t)
	adminUnitPage(t, f.list(t, ""))
	if len(f.reader.ids) != 1 || !reflect.DeepEqual(f.reader.ids[0], []string{queryTestID(900)}) {
		t.Fatalf("author deduplication = %v", f.reader.ids)
	}
}

func adminUnitDetail(t *testing.T) {
	for _, mode := range []string{"invalid_id", "absent", "pending", "invalid_state", "identity_missing"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdminUnit(t)
			id, status, code := queryTestID(3), 404, CodeArticleUnavailable
			switch mode {
			case "invalid_id":
				id = "invalid"
			case "absent":
				id = queryTestID(100)
			case "pending":
				f.repo.rows[0].DeletionState = DeletionPending
				f.reader.states = nil
			case "invalid_state":
				f.repo.rows[0].Version = 0
				status, code = 500, platform.CodePlatformInternalError
			case "identity_missing":
				f.reader.states = nil
				status, code = 503, platform.CodePlatformDependencyUnavailable
			}
			adminUnitError(t, f.detail(t, id), status, code)
			if mode != "identity_missing" && len(f.reader.ids) != 0 {
				t.Fatal("detail priority called identity")
			}
		})
	}
}

func adminUnitInvariants(t *testing.T) {
	for _, mode := range []string{"oversized", "duplicate", "unordered", "stalled", "uuid", "zero_time", "non_utc", "invalid_state"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdminUnit(t)
			f.repo.batch = func(_ context.Context, _ *ArticleCursor, _ int) ([]Article, error) {
				rows := append([]Article(nil), f.repo.rows...)
				switch mode {
				case "oversized":
					rows = append(rows, rows[2])
				case "duplicate":
					rows[1] = rows[0]
				case "unordered":
					rows[0], rows[1] = rows[1], rows[0]
				case "stalled":
					if len(f.repo.afters) > 1 {
						return rows, nil
					}
				case "uuid":
					rows[0].ID = "bad"
				case "zero_time":
					rows[0].CreatedAt = time.Time{}
				case "non_utc":
					rows[0].CreatedAt = rows[0].CreatedAt.In(time.FixedZone("offset", 3600))
				case "invalid_state":
					rows[0].ViolationState = ViolationViolating
				}
				return rows, nil
			}
			raw := "view=all"
			if mode == "stalled" {
				raw = "view=hidden"
			}
			adminUnitError(t, f.list(t, raw), 500, platform.CodePlatformInternalError)
			if len(f.repo.afters) > 2 {
				t.Fatal("invalid candidates caused a scan loop")
			}
		})
	}
}

func adminUnitFaults(t *testing.T) {
	for _, mode := range []string{"list_error", "detail_error", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			f := newAdminUnit(t)
			if mode == "detail_error" {
				f.repo.get = func(context.Context, string) (Article, error) { return Article{}, errors.New("query/scan failure") }
				adminUnitError(t, f.detail(t, queryTestID(3)), 503, platform.CodePlatformDatabaseUnavailable)
				return
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f.repo.batch = func(incoming context.Context, _ *ArticleCursor, _ int) ([]Article, error) {
				if incoming.Err() != nil {
					t.Fatal("context unexpectedly lost before query")
				}
				if mode == "cancel" {
					cancel()
					return f.repo.rows, nil
				}
				return nil, errors.New("query/scan/rows failure")
			}
			w := adminUnitHTTP(t, f.handler.ListAdminArticles, adminUnitPrincipalValue(), "", "", ctx)
			adminUnitError(t, w, 503, platform.CodePlatformDatabaseUnavailable)
			if len(f.repo.afters) != 1 || len(f.reader.ids) != 0 {
				t.Fatal("failure retried/read identity after cancellation")
			}
		})
	}
}

func adminUnitReadOnly(t *testing.T) {
	f := newAdminUnit(t)
	beforeRows := append([]Article(nil), f.repo.rows...)
	beforeStates := map[string]string{}
	for k, v := range f.reader.states {
		beforeStates[k] = v
	}
	adminUnitPage(t, f.list(t, ""))
	f.detail(t, queryTestID(3))
	if !reflect.DeepEqual(beforeRows, f.repo.rows) || !reflect.DeepEqual(beforeStates, f.reader.states) {
		t.Fatal("query mutated controlled business inputs")
	}
}

func adminUnitFields(t *testing.T) {
	f := newAdminUnit(t)
	f.repo.rows = adminUnitStateRows()
	reasons := map[int][]HiddenReason{1: {}, 2: {HiddenAuthorSuspended}, 3: {}, 4: {HiddenOrphanHold}, 5: {HiddenViolation}, 6: {HiddenAuthorSuspended, HiddenViolation}, 7: {HiddenViolation, HiddenOrphanHold}}
	for index := range f.repo.rows {
		f.repo.rows[index].Title = "  标题 <b>literal</b>  "
		f.repo.rows[index].Body = " **正文** \n🙂 "
		f.repo.rows[index].CreatedAt = queryTestTime.Add(-time.Duration(index) * time.Microsecond)
		f.repo.rows[index].UpdatedAt = f.repo.rows[index].CreatedAt
	}
	for index, expected := range reasons {
		t.Run(fmt.Sprintf("record_%d", index), func(t *testing.T) {
			r := f.repo.rows[index]
			w := f.detail(t, r.ID)
			if w.Code != 200 {
				t.Fatalf("detail HTTP=%d", w.Code)
			}
			adminUnitAssertPayload(t, w.Body.Bytes(), r, expected)
		})
	}
	for _, view := range []string{"all", "violating", "hidden"} {
		t.Run("list_"+view, func(t *testing.T) { adminUnitListFields(t, f, view, reasons) })
	}
}

func adminUnitAssertPayload(t *testing.T, payload []byte, r Article, reasons []HiddenReason) {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	wantKeys := []string{"author", "body", "created_at", "hidden_reasons", "id", "orphaned_from_user_id", "publication_state", "title", "updated_at", "version", "violation_category", "violation_state"}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Fatalf("admin fields=%v", keys)
	}
	var got AdminArticleView
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != r.ID || got.Title != r.Title || got.Body != r.Body || got.Version != r.Version || !reflect.DeepEqual(got.HiddenReasons, reasons) || !reflect.DeepEqual(got.ViolationCategory, r.ViolationCategory) || !reflect.DeepEqual(got.OrphanedFromUserID, r.OrphanedFromUserID) || got.PublicationState != r.PublicationState || got.ViolationState != r.ViolationState || !got.CreatedAt.Equal(r.CreatedAt) || !got.UpdatedAt.Equal(r.UpdatedAt) {
		t.Fatal("admin twelve-field values differ from independent fixture")
	}
	if r.AuthorID == nil && (got.Author.Kind != "anonymous" || got.Author.Label != "匿名" || got.Author.ID != nil) || r.AuthorID != nil && (got.Author.Kind != "user" || got.Author.ID == nil || *got.Author.ID != *r.AuthorID || got.Author.Label != "") {
		t.Fatal("nested author values changed")
	}
}

func adminUnitListFields(t *testing.T, f *adminUnitFixture, view string, reasons map[int][]HiddenReason) {
	t.Helper()
	cursor := ""
	seen := []string{}
	for pageIndex := 0; pageIndex < 20; pageIndex++ {
		raw := "view=" + view
		if cursor != "" {
			raw += "&cursor=" + cursor
		}
		w := f.list(t, raw)
		if w.Code != 200 {
			t.Fatalf("list HTTP=%d", w.Code)
		}
		var page struct {
			Items []json.RawMessage `json:"items"`
			Next  *string           `json:"next_cursor"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if page.Items == nil {
			t.Fatal("list items is null")
		}
		for _, payload := range page.Items {
			seen = append(seen, adminUnitAssertListItem(t, f, payload, reasons))
		}
		if page.Next == nil {
			adminUnitFieldSequence(t, f, view, seen)
			return
		}
		cursor = *page.Next
	}
	t.Fatal("list field traversal did not exhaust")
}

func adminUnitAssertListItem(t *testing.T, f *adminUnitFixture, payload []byte, reasons map[int][]HiddenReason) string {
	t.Helper()
	var identity struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload, &identity); err != nil {
		t.Fatal(err)
	}
	for index, row := range f.repo.rows {
		if row.ID == identity.ID {
			adminUnitAssertPayload(t, payload, row, reasons[index])
			return identity.ID
		}
	}
	t.Fatal("list returned unexpected fixture identity")
	return ""
}

func adminUnitFieldSequence(t *testing.T, f *adminUnitFixture, view string, seen []string) {
	t.Helper()
	sequence := map[string][]int{"all": {1, 2, 3, 4, 5, 6, 7}, "violating": {5, 6, 7}, "hidden": {2, 4, 5, 6, 7}}[view]
	want := []string{}
	for _, index := range sequence {
		want = append(want, f.repo.rows[index].ID)
	}
	if !reflect.DeepEqual(seen, want) {
		t.Fatal("list values/sequence differ from independent fixture")
	}
}

func adminUnitETag(t *testing.T) {
	f := newAdminUnit(t)
	original := f.repo.rows[0]
	f.repo.get = func(context.Context, string) (Article, error) {
		f.repo.rows[0].Version++
		f.repo.rows[0].Body = "later body"
		return original, nil
	}
	w := f.detail(t, original.ID)
	var view AdminArticleView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || f.repo.gets != 1 || view.Body != original.Body || view.Version != original.Version || w.Header().Get("ETag") != fmt.Sprintf(`"article-%s-v%d"`, original.ID, original.Version) {
		t.Fatal("ETag and body were assembled from different Article reads")
	}
}

func adminUnitCompatibility(t *testing.T) {
	t.Run("new_500_old_409", func(t *testing.T) {
		f := newAdminUnit(t)
		f.repo.rows[0].Version = 0
		adminUnitError(t, f.detail(t, f.repo.rows[0].ID), 500, platform.CodePlatformInternalError)
		w := httptest.NewRecorder()
		_ = WriteArticleError(w, httptest.NewRequest(http.MethodPost, "/", nil), ErrArticleInvalidState)
		adminUnitError(t, w, 409, CodeArticleInvalidState)
	})
	t.Run("old_member_author_orphan", func(t *testing.T) {
		f := newAdminUnit(t)
		f.reader.states = nil
		page, err := f.service.ListArticles(context.Background(), ListArticlesRequest{})
		if err != nil || len(page.Items) != 0 {
			t.Fatalf("member missing state semantics changed: %v", err)
		}
		_, _, err = f.service.GetArticle(context.Background(), f.repo.rows[0].ID)
		if !errors.Is(err, ErrArticleUnavailable) {
			t.Fatalf("member detail missing state = %v", err)
		}
		p := adminUnitPrincipalValue()
		p.UserID = *f.repo.rows[0].AuthorID
		view, _, err := f.service.GetAuthorArticle(context.Background(), p, f.repo.rows[0].ID)
		if err != nil || !view.Editable {
			t.Fatalf("author projection changed: %v", err)
		}
		f.repo.rows = adminUnitStateRows()
		q, err := NewOrphanQueryService(f.repo, f.service.config)
		if err != nil {
			t.Fatal(err)
		}
		orphans, err := q.ListOrphanArticles(context.Background(), adminUnitPrincipalValue(), ListArticlesRequest{})
		if err != nil || len(orphans.Items) != 2 {
			t.Fatalf("orphan semantics changed: %d/%v", len(orphans.Items), err)
		}
	})
}
