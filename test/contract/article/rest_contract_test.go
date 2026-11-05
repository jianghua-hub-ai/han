//go:build contract

package articlecontract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"han/internal/article"
	"han/internal/platform"
	"han/internal/session"
)

type transportPort struct {
	row           article.Article
	err           error
	calls         int
	ctx           context.Context
	actor         session.Principal
	id, target    string
	version       int64
	input         article.CreateArticleInput
	deleteRequest article.PermanentDeletionRequest
	list          article.ListArticlesRequest
	admin         article.ListAdminArticlesRequest
	writeFailures int
}

func (p *transportPort) mark(ctx context.Context, actor session.Principal, id string) {
	p.calls++
	p.ctx = ctx
	p.actor = actor
	p.id = id
}
func (p *transportPort) Create(ctx context.Context, actor session.Principal, in article.CreateArticleInput) (article.Article, error) {
	p.mark(ctx, actor, "")
	p.input = in
	row := p.row
	row.Title = in.Title
	row.Body = in.Body
	row.Version = 1
	return row, p.err
}
func (p *transportPort) ReplaceContent(ctx context.Context, actor session.Principal, id string, version int64, in article.CreateArticleInput) (article.Article, error) {
	p.mark(ctx, actor, id)
	p.input = in
	p.version = version
	row := p.row
	row.Title = in.Title
	row.Body = in.Body
	row.Version = version + 1
	return row, p.err
}
func (p *transportPort) GetAuthorArticle(ctx context.Context, actor session.Principal, id string) (article.AuthorArticleView, string, error) {
	p.mark(ctx, actor, id)
	view, err := (article.VisibilityPolicy{}).AuthorProjection(p.row, actor)
	return view, etag(p.row), errors.Join(err, p.err)
}
func (p *transportPort) ListArticles(ctx context.Context, in article.ListArticlesRequest) (article.ListArticlesPage, error) {
	p.mark(ctx, session.Principal{}, "")
	p.list = in
	view, err := (article.VisibilityPolicy{}).MemberListProjection(p.row, "active")
	return article.ListArticlesPage{Items: []article.MemberArticleSummaryView{view}}, errors.Join(err, p.err)
}
func (p *transportPort) GetArticle(ctx context.Context, id string) (article.MemberArticleDetailView, string, error) {
	p.mark(ctx, session.Principal{}, id)
	view, err := (article.VisibilityPolicy{}).MemberDetailProjection(p.row, "active")
	return view, etag(p.row), errors.Join(err, p.err)
}
func (p *transportPort) ListAdminArticles(ctx context.Context, actor session.Principal, in article.ListAdminArticlesRequest) (article.AdminArticlesPage, error) {
	p.mark(ctx, actor, "")
	p.admin = in
	view, err := (article.VisibilityPolicy{}).AdminProjection(p.row, actor, "active")
	return article.AdminArticlesPage{Items: []article.AdminArticleView{view}}, errors.Join(err, p.err)
}
func (p *transportPort) GetAdminArticle(ctx context.Context, actor session.Principal, id string) (article.AdminArticleView, string, error) {
	p.mark(ctx, actor, id)
	view, err := (article.VisibilityPolicy{}).AdminProjection(p.row, actor, "active")
	return view, etag(p.row), errors.Join(err, p.err)
}
func (p *transportPort) ListOrphanArticles(ctx context.Context, actor session.Principal, in article.ListArticlesRequest) (article.OrphanArticlesPage, error) {
	p.mark(ctx, actor, "")
	p.list = in
	view, err := (article.VisibilityPolicy{}).AdminProjection(p.row, actor, "active")
	return article.OrphanArticlesPage{Items: []article.AdminArticleView{view}}, errors.Join(err, p.err)
}
func (p *transportPort) GetOrphanArticle(ctx context.Context, actor session.Principal, id string) (article.AdminArticleView, string, error) {
	return p.GetAdminArticle(ctx, actor, id)
}
func (p *transportPort) DeletePermanent(ctx context.Context, actor session.Principal, in article.PermanentDeletionRequest) error {
	p.mark(ctx, actor, in.ArticleID)
	p.deleteRequest = in
	return p.err
}
func (p *transportPort) Moderate(ctx context.Context, actor session.Principal, id string, version int64, category *article.ViolationCategory) (article.AdminArticleView, string, error) {
	p.mark(ctx, actor, id)
	p.version = version
	row := p.row
	row.Version++
	row.ViolationState = article.ViolationClear
	row.ViolationCategory = category
	if category != nil {
		row.ViolationState = article.ViolationViolating
	}
	view, err := (article.VisibilityPolicy{}).AdminProjection(row, actor, "active")
	return view, etag(row), errors.Join(err, p.err)
}
func (p *transportPort) PublishOrphan(ctx context.Context, actor session.Principal, id string, version int64) (article.OrphanDispositionResult, error) {
	p.mark(ctx, actor, id)
	p.version = version
	row := p.row
	row.PublicationState = article.PublicationPublished
	row.OrphanedFromUserID = nil
	row.Version++
	return article.OrphanDispositionResult{Article: row, ETag: etag(row)}, p.err
}
func (p *transportPort) TransferOrphan(ctx context.Context, actor session.Principal, id string, version int64, target string) (article.OrphanDispositionResult, error) {
	p.mark(ctx, actor, id)
	p.version = version
	p.target = target
	row := p.row
	row.AuthorID = &target
	row.PublicationState = article.PublicationPublished
	row.OrphanedFromUserID = nil
	row.Version++
	notice := noticeID
	return article.OrphanDispositionResult{Article: row, ETag: etag(row), NoticeID: &notice}, p.err
}
func (p *transportPort) LogCreateResponseWriteFailure(context.Context, string, error) {
	p.writeFailures++
}

func etag(row article.Article) string {
	return `"article-` + row.ID + `-v` + intText(row.Version) + `"`
}
func intText(v int64) string { return strconv.FormatInt(v, 10) }

type transportFixture struct {
	port                                *transportPort
	base, member, author, admin, orphan *article.Handler
}

func newTransport(t *testing.T) *transportFixture {
	t.Helper()
	p := &transportPort{row: baseArticle()}
	base, err := article.NewHandler(p, p, p)
	must(t, err)
	base, err = article.NewOrphanCommandHandler(base, p, &permissionGate{})
	must(t, err)
	member, err := article.NewQueryHandler(p)
	must(t, err)
	author, err := article.NewAuthorDetailHandler(p)
	must(t, err)
	admin, err := article.NewAdminQueryHandler(p)
	must(t, err)
	orphan, err := article.NewOrphanQueryHandler(p)
	must(t, err)
	return &transportFixture{p, base, member, author, admin, orphan}
}

type permissionGate struct {
	err   error
	calls int
}

func (g *permissionGate) RequireOpen(context.Context) error { g.calls++; return g.err }

type resolver struct{ actor session.Principal }

type parentMarker struct{}

func (r resolver) ResolvePrincipal(context.Context, session.ResolveInput) (session.Principal, error) {
	return r.actor, nil
}

func request(method, path string, body []byte) *http.Request {
	r := httptest.NewRequest(method, path, bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("If-Match", etag(baseArticle()))
	r.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-token"})
	return r.WithContext(context.WithValue(r.Context(), parentMarker{}, "contract-parent"))
}
func component(t *testing.T, fn http.HandlerFunc, actor *session.Principal) http.Handler {
	t.Helper()
	var next http.Handler = fn
	if actor != nil {
		auth, err := platform.NewHTTPAuthorizer(resolver{*actor}, func() time.Time { return fixtureTime })
		must(t, err)
		next = auth.Protect(session.AccessGeneral, next)
	}
	return platform.TraceMiddleware(next)
}
func serve(t *testing.T, fn http.HandlerFunc, actor *session.Principal, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	component(t, fn, actor).ServeHTTP(w, r)
	return w
}

type endpoint struct {
	name, method, path string
	body               []byte
	fn                 http.HandlerFunc
	actor              session.Principal
	status             int
}

func endpointFor(t *testing.T, f *transportFixture, name string) endpoint {
	t.Helper()
	e, ok := endpointTable(t, f)[name]
	if !ok {
		t.Fatal("unknown endpoint")
	}
	e.name = name
	e.actor = principal("admin")
	if name == "Create" || name == "Replace" || name == "AuthorDetail" {
		e.actor = principal("member")
	}
	if e.method == "" {
		e.method = http.MethodGet
	}
	if e.status == 0 {
		e.status = 200
	}
	if name == "Restore" || name == "DeleteViolating" {
		c := article.ViolationViolence
		f.port.row.ViolationState = article.ViolationViolating
		f.port.row.ViolationCategory = &c
	}
	if strings.Contains(name, "Orphan") {
		source := authorID
		f.port.row.AuthorID = nil
		f.port.row.OrphanedFromUserID = &source
		f.port.row.PublicationState = article.PublicationHeld
	}
	return e
}

func endpointTable(t *testing.T, f *transportFixture) map[string]endpoint {
	t.Helper()
	id := "/" + articleID
	body := marshal(t, article.CreateArticleInput{Title: f.port.row.Title, Body: f.port.row.Body})
	return map[string]endpoint{
		"Create":          {method: http.MethodPost, path: "/api/v1/articles", fn: f.base.Create, status: 201, body: body},
		"MemberList":      {path: "/api/v1/articles?limit=7&cursor=opaque%2Bcursor", fn: f.member.ListArticles},
		"MemberDetail":    {path: "/api/v1/articles" + id, fn: f.member.GetArticle},
		"AuthorDetail":    {path: "/api/v1/users/me/articles" + id, fn: f.author.GetAuthorArticle},
		"Replace":         {method: http.MethodPut, path: "/api/v1/articles" + id, fn: f.base.UpdateArticle, body: body},
		"DeleteOrdinary":  {method: http.MethodDelete, path: "/api/v1/articles" + id, fn: f.base.DeleteOrdinary, status: 204},
		"AdminList":       {path: "/api/v1/admin/articles?limit=7&cursor=opaque%2Bcursor&view=all", fn: f.admin.ListAdminArticles},
		"AdminDetail":     {path: "/api/v1/admin/articles" + id, fn: f.admin.GetAdminArticle},
		"MarkViolation":   {method: http.MethodPost, path: "/api/v1/admin/articles" + id + "/violation", fn: f.base.MarkViolation, body: []byte(`{"category":"violence"}`)},
		"Restore":         {method: http.MethodPost, path: "/api/v1/admin/articles" + id + "/restore", fn: f.base.RestoreArticle},
		"DeleteViolating": {method: http.MethodDelete, path: "/api/v1/admin/articles" + id, fn: f.base.DeleteViolating, status: 204},
		"OrphanList":      {path: "/api/v1/admin/orphan-articles?limit=7&cursor=opaque%2Bcursor", fn: f.orphan.ListOrphanArticles},
		"OrphanDetail":    {path: "/api/v1/admin/orphan-articles" + id, fn: f.orphan.GetOrphanArticle},
		"PublishOrphan":   {method: http.MethodPost, path: "/api/v1/admin/orphan-articles" + id + "/publish", fn: f.base.PublishOrphan},
		"TransferOrphan":  {method: http.MethodPost, path: "/api/v1/admin/orphan-articles" + id + "/transfer", fn: f.base.TransferOrphan, body: []byte(`{"new_author_id":"` + targetID + `"}`)},
		"DeleteOrphan":    {method: http.MethodDelete, path: "/api/v1/admin/orphan-articles" + id, fn: f.base.DeleteOrphan, status: 204},
	}
}
func endpointNames() []string {
	return []string{"Create", "MemberList", "MemberDetail", "AuthorDetail", "Replace", "DeleteOrdinary", "AdminList", "AdminDetail", "MarkViolation", "Restore", "DeleteViolating", "OrphanList", "OrphanDetail", "PublishOrphan", "TransferOrphan", "DeleteOrphan"}
}

func TestArticleRESTContract(t *testing.T) {
	t.Run("Endpoints", restEndpoints)
	t.Run("WireAndStrictJSON", restWire)
	t.Run("Preconditions", restPreconditions)
	t.Run("PublicErrors", restPublicErrors)
	t.Run("AdminRawQueryAndStatePriority", restAdminPriority)
	t.Run("ResponseWriteFailure", restWriteFailure)
}

func restEndpoints(t *testing.T) {
	for _, name := range endpointNames() {
		t.Run(name, func(t *testing.T) {
			t.Run("success", func(t *testing.T) {
				f := newTransport(t)
				e := endpointFor(t, f, name)
				w := serve(t, e.fn, &e.actor, request(e.method, e.path, e.body))
				if w.Code != e.status {
					t.Fatalf("HTTP=%d expected=%d", w.Code, e.status)
				}
				assertEndpoint(t, f, e, w)
			})
			t.Run("error", func(t *testing.T) {
				f := newTransport(t)
				e := endpointFor(t, f, name)
				f.port.err = article.ErrArticleUnavailable
				w := serve(t, e.fn, &e.actor, request(e.method, e.path, e.body))
				assertError(t, w, 404, article.CodeArticleUnavailable)
				if f.port.calls != 1 {
					t.Fatal("error path did not call configured capability exactly once")
				}
				assertEndpointBinding(t, f.port, e, w)
			})
			t.Run("missing_principal", func(t *testing.T) {
				f := newTransport(t)
				e := endpointFor(t, f, name)
				w := serve(t, e.fn, nil, request(e.method, e.path, e.body))
				assertError(t, w, 401, platform.CodeAuthSessionRequired)
				if f.port.calls != 0 {
					t.Fatal("missing Principal reached port")
				}
			})
			if strings.HasPrefix(name, "Delete") {
				t.Run("in_progress", func(t *testing.T) {
					f := newTransport(t)
					e := endpointFor(t, f, name)
					f.port.err = article.ErrDeletionInProgress
					w := serve(t, e.fn, &e.actor, request(e.method, e.path, e.body))
					assertError(t, w, 503, article.CodeArticleDeletionInProgress)
					if f.port.calls != 1 {
						t.Fatal("pending delete did not call permanent-deletion capability")
					}
				})
			}
		})
	}
}

func assertEndpoint(t *testing.T, f *transportFixture, e endpoint, w *httptest.ResponseRecorder) {
	t.Helper()
	p := f.port
	row := p.row
	assertEndpointBinding(t, p, e, w)
	if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get(platform.TraceHeader) == "" {
		t.Fatal("missing response security headers")
	}
	if e.status == 204 {
		assertDeletion(t, p, e, w)
		return
	}
	if e.name == "MemberList" || e.name == "AdminList" || e.name == "OrphanList" {
		assertEndpointPage(t, w, row, e.name)
		return
	}
	row = expectedEndpointRow(e.name, row)
	switch e.name {
	case "Create", "MemberDetail":
		assertMember(t, w.Body.Bytes(), row, true)
	case "AuthorDetail", "Replace":
		assertAuthor(t, w.Body.Bytes(), row, false)
	case "TransferOrphan":
		assertTransfer(t, w, row)
	default:
		reasons := []string{}
		if e.name == "MarkViolation" {
			reasons = []string{"violation"}
		}
		if e.name == "OrphanDetail" {
			reasons = []string{"orphan_hold"}
		}
		assertAdmin(t, w.Body.Bytes(), row, reasons)
	}
	if e.name == "Create" && w.Header().Get("Location") != "/api/v1/articles/"+articleID {
		t.Fatal("wrong creation Location")
	}
	if w.Header().Get("ETag") != etag(row) {
		t.Fatal("response ETag differs from same result")
	}
	if e.method != http.MethodGet && e.name != "Create" && p.version != 2 {
		t.Fatal("If-Match version argument changed")
	}
}

func assertEndpointBinding(t *testing.T, p *transportPort, e endpoint, w *httptest.ResponseRecorder) {
	t.Helper()
	if p.calls != 1 || p.ctx == nil || p.ctx.Value(parentMarker{}) != "contract-parent" {
		t.Fatal("Handler lost parent context or capability call count")
	}
	trace, ok := platform.TraceIDFromContext(p.ctx)
	if !ok || trace != w.Header().Get(platform.TraceHeader) {
		t.Fatal("capability context lost response Trace")
	}
	wantID := articleID
	switch e.name {
	case "Create", "MemberList", "AdminList", "OrphanList":
		wantID = ""
	}
	if p.id != wantID {
		t.Fatal("Handler changed article ID argument")
	}
	if e.name == "TransferOrphan" && p.target != targetID {
		t.Fatal("Handler changed transfer target argument")
	}
	if e.name != "MemberDetail" && e.name != "MemberList" && p.actor != e.actor {
		t.Fatal("Handler changed Principal argument")
	}
	if e.name == "MemberList" || e.name == "OrphanList" {
		if p.list.Cursor != "opaque+cursor" || p.list.Limit == nil || *p.list.Limit != 7 {
			t.Fatal("Handler changed list cursor/limit arguments")
		}
	}
	if e.name == "AdminList" && (p.admin.View != article.AdminListViewAll || p.admin.Cursor != "opaque+cursor" || p.admin.Limit == nil || *p.admin.Limit != 7) {
		t.Fatal("Handler changed admin view/cursor/limit arguments")
	}
}

func expectedEndpointRow(name string, row article.Article) article.Article {
	switch name {
	case "Create":
		row.Version = 1
	case "Replace":
		row.Version++
	case "MarkViolation":
		row.Version++
		c := article.ViolationViolence
		row.ViolationState = article.ViolationViolating
		row.ViolationCategory = &c
	case "Restore":
		row.Version++
		row.ViolationState = article.ViolationClear
		row.ViolationCategory = nil
	case "PublishOrphan", "TransferOrphan":
		row.Version++
		row.PublicationState = article.PublicationPublished
		row.OrphanedFromUserID = nil
		if name == "TransferOrphan" {
			id := targetID
			row.AuthorID = &id
		}
	}
	return row
}

func assertDeletion(t *testing.T, p *transportPort, e endpoint, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Body.Len() != 0 || p.actor != e.actor || p.deleteRequest.ArticleID != articleID || p.deleteRequest.IfMatch != etag(p.row) || p.deleteRequest.TraceID != w.Header().Get(platform.TraceHeader) {
		t.Fatal("DELETE transport binding/body changed")
	}
	kinds := map[string]article.DeleteKind{"DeleteOrdinary": article.DeleteOrdinary, "DeleteViolating": article.DeleteViolating, "DeleteOrphan": article.DeleteOrphan}
	if p.deleteRequest.DeleteKind != kinds[e.name] {
		t.Fatal("wrong deletion capability kind")
	}
}

func assertEndpointPage(t *testing.T, w *httptest.ResponseRecorder, row article.Article, name string) {
	t.Helper()
	obj := jsonObject(t, w.Body.Bytes(), "items", "next_cursor")
	fieldValue(t, obj, "next_cursor", nil)
	var items []json.RawMessage
	must(t, json.Unmarshal(obj["items"], &items))
	if len(items) != 1 {
		t.Fatal("list item count changed")
	}
	if name == "MemberList" {
		assertMember(t, items[0], row, false)
	} else {
		reasons := []string{}
		if name == "OrphanList" {
			reasons = []string{"orphan_hold"}
		}
		assertAdmin(t, items[0], row, reasons)
	}
	if w.Header().Get("ETag") != "" {
		t.Fatal("list gained whole-page ETag")
	}
}
func assertTransfer(t *testing.T, w *httptest.ResponseRecorder, row article.Article) {
	t.Helper()
	obj := jsonObject(t, w.Body.Bytes(), "article", "notification")
	assertAdmin(t, obj["article"], row, []string{})
	n := jsonObject(t, obj["notification"], "id", "delivery_capability", "recipients")
	fieldValue(t, n, "id", noticeID)
	fieldValue(t, n, "delivery_capability", "open")
	fieldValue(t, n, "recipients", []map[string]string{{"kind": string(article.RecipientNewAuthor), "status": "pending"}, {"kind": string(article.RecipientFormerDeletedAuthor), "status": "pending"}})
}

func assertError(t *testing.T, w *httptest.ResponseRecorder, status int, code platform.Code) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("HTTP=%d expected=%d (body withheld)", w.Code, status)
	}
	var e platform.Envelope
	must(t, json.Unmarshal(w.Body.Bytes(), &e))
	var fields map[string]json.RawMessage
	must(t, json.Unmarshal(w.Body.Bytes(), &fields))
	for key := range fields {
		if key != "code" && key != "message" && key != "trace_id" && key != "field_errors" {
			t.Fatalf("public error gained field %s", key)
		}
	}
	if e.Code != code || e.Message != expectedErrorMessage(t, code) || e.TraceID == "" || e.TraceID != w.Header().Get(platform.TraceHeader) || w.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatal("public error contract changed")
	}
	if strings.Contains(w.Body.String(), privateCanary) || w.Header().Get("ETag") != "" {
		t.Fatal("public error leaked private cause/data")
	}
}

func restWire(t *testing.T) {
	const wireContractBytes = 262144 // Design §2.4.4; independent of the implementation limit.
	valid := []byte(`{"title":"t","body":"b"}`)
	for _, name := range []string{"Create", "Replace"} {
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name        string
				body        []byte
				contentType string
				status      int
			}{
				{"N", append(append([]byte{}, valid...), bytes.Repeat([]byte(" "), wireContractBytes-len(valid))...), "application/json", 200},
				{"Nplus1", append(append([]byte{}, valid...), bytes.Repeat([]byte(" "), wireContractBytes+1-len(valid))...), "application/json", 400},
				{"unknown", []byte(`{"title":"t","body":"b","extra":1}`), "application/json", 400},
				{"duplicate", []byte(`{"title":"t","title":"x","body":"b"}`), "application/json", 400},
				{"trailing", append(append([]byte{}, valid...), []byte(` {}`)...), "application/json", 400},
				{"type", []byte(`{"title":1,"body":"b"}`), "application/json", 400},
				{"array", []byte(`[]`), "application/json", 400},
				{"utf8", []byte{'{', '"', 't', 'i', 't', 'l', 'e', '"', ':', '"', 0xff, '"', '}'}, "application/json", 400},
				{"content_type", valid, "text/plain", 400},
				{"unicode", marshal(t, article.CreateArticleInput{Title: " 标题 ", Body: "🙂 <b>raw</b>"}), "application/json", 200},
			} {
				t.Run(tc.name, func(t *testing.T) {
					wireAssertion(t, name, tc.body, tc.contentType, tc.status)
				})
			}
		})
	}
	bodyFreeRejects(t)
}

func wireAssertion(t *testing.T, name string, body []byte, contentType string, status int) {
	t.Helper()
	f := newTransport(t)
	e := endpointFor(t, f, name)
	r := request(e.method, e.path, body)
	r.Header.Set("Content-Type", contentType)
	w := serve(t, e.fn, &e.actor, r)
	if status == 400 {
		assertError(t, w, 400, platform.CodeRequestInvalidJSON)
		if f.port.calls != 0 {
			t.Fatal("bad wire reached command")
		}
		return
	}
	if w.Code != e.status || f.port.calls != 1 {
		t.Fatal("valid boundary body rejected")
	}
	var in article.CreateArticleInput
	must(t, json.Unmarshal(body, &in))
	if f.port.input != in {
		t.Fatal("decoder repaired/trimmed literal content")
	}
}

func bodyFreeRejects(t *testing.T) {
	for _, name := range []string{"Restore", "PublishOrphan"} {
		f := newTransport(t)
		e := endpointFor(t, f, name)
		w := serve(t, e.fn, &e.actor, request(e.method, e.path, []byte(`{}`)))
		assertError(t, w, 400, platform.CodeRequestInvalidJSON)
		if f.port.calls != 0 {
			t.Fatal("body-free command reached service")
		}
	}
}

func restPreconditions(t *testing.T) {
	for _, name := range []string{"Replace", "DeleteOrdinary", "DeleteViolating", "DeleteOrphan", "MarkViolation", "Restore", "PublishOrphan", "TransferOrphan"} {
		t.Run(name, func(t *testing.T) {
			for _, tc := range []struct {
				name   string
				values []string
				status int
			}{
				{"missing", nil, 428}, {"weak", []string{`W/` + etag(baseArticle())}, 400}, {"multiple", []string{etag(baseArticle()), etag(baseArticle())}, 400}, {"star", []string{"*"}, 400}, {"malformed", []string{"bad"}, 400},
			} {
				t.Run(tc.name, func(t *testing.T) {
					f := newTransport(t)
					e := endpointFor(t, f, name)
					r := request(e.method, e.path, e.body)
					r.Header.Del("If-Match")
					for _, v := range tc.values {
						r.Header.Add("If-Match", v)
					}
					w := serve(t, e.fn, &e.actor, r)
					code := article.CodeRequestInvalidPrecondition
					if tc.status == 428 {
						code = article.CodeRequestPreconditionRequired
					}
					assertError(t, w, tc.status, code)
					if f.port.calls != 0 {
						t.Fatal("invalid condition reached port")
					}
				})
			}
			f := newTransport(t)
			e := endpointFor(t, f, name)
			f.port.err = article.ErrPreconditionFailed
			w := serve(t, e.fn, &e.actor, request(e.method, e.path, e.body))
			assertError(t, w, 412, article.CodeArticlePreconditionFailed)
		})
	}
	row := baseArticle()
	for _, value := range []string{etag(row), `"article-` + articleID + `-v1"`} {
		err := article.RequireMatchingETag(value, row)
		if (err == nil) != (value == etag(row)) {
			t.Fatal("actual ETag matcher stale semantics changed")
		}
	}
}

func restPublicErrors(t *testing.T) {
	for _, tc := range publicErrorCases() {
		t.Run(string(tc.code), func(t *testing.T) {
			fn := func(w http.ResponseWriter, r *http.Request) { _ = article.WriteArticleError(w, r, tc.cause) }
			w := serve(t, fn, nil, request(http.MethodGet, "/component", nil))
			assertError(t, w, tc.status, tc.code)
		})
	}
	t.Run("field_errors", func(t *testing.T) {
		cause := platform.NewPublicError(platform.CodeRequestInvalidJSON, errors.New(privateCanary), platform.FieldError{Field: "title", Code: "invalid"})
		w := serve(t, func(w http.ResponseWriter, r *http.Request) { _ = article.WriteArticleError(w, r, cause) }, nil, request(http.MethodGet, "/component", nil))
		assertError(t, w, 400, platform.CodeRequestInvalidJSON)
		var e platform.Envelope
		must(t, json.Unmarshal(w.Body.Bytes(), &e))
		if len(e.FieldErrors) != 1 || e.FieldErrors[0].Field != "title" || e.FieldErrors[0].Code != "invalid" {
			t.Fatal("field error contract changed")
		}
	})
}

type publicErrorCase struct {
	code    platform.Code
	status  int
	retry   platform.RetryPolicy
	cause   error
	message string
}

func expectedErrorMessage(t *testing.T, code platform.Code) string {
	t.Helper()
	for _, tc := range publicErrorCases() {
		if tc.code == code {
			return tc.message
		}
	}
	t.Fatalf("no independent message expectation for code %s", code)
	return ""
}

func publicErrorCases() []publicErrorCase {
	// Literal wire messages bind design §3.3.1 and the accepted shared-platform
	// golden in internal/platform/errors_test.go, not the runtime registry.
	cases := []publicErrorCase{
		{platform.CodeRequestInvalidJSON, 400, platform.RetryAfterCorrection, nil, "请求格式不正确"},
		{platform.CodeRequestInvalidCursor, 400, platform.RetryAfterCorrection, article.ErrInvalidCursor, "分页游标无效"},
		{article.CodeRequestInvalidPrecondition, 400, platform.RetryAfterCorrection, article.ErrInvalidPrecondition, "资源版本条件格式无效"},
		{platform.CodeAuthSessionRequired, 401, platform.RetryAllowed, nil, "需要重新登录"},
		{platform.CodeAuthCSRFFailed, 403, platform.RetryNever, nil, "请求安全校验失败"},
		{platform.CodeAuthForbidden, 403, platform.RetryNever, article.ErrArticleForbidden, "无权执行该操作"},
		{article.CodeArticleUnavailable, 404, platform.RetryNever, article.ErrArticleUnavailable, "文章不存在或不可用"},
		{article.CodeArticleInvalidState, 409, platform.RetryAfterStateChange, article.ErrArticleInvalidState, "当前文章状态不允许该操作"},
		{article.CodeArticlePreconditionFailed, 412, platform.RetryAllowed, article.ErrPreconditionFailed, "文章已变化，请刷新后重试"},
		{article.CodeArticleValidationFailed, 422, platform.RetryAfterCorrection, article.ErrArticleValidationFailed, "标题或正文不符合要求"},
		{article.CodeArticleTransferTargetInvalid, 422, platform.RetryAfterCorrection, article.ErrTransferTargetInvalid, "目标用户不能成为新作者"},
		{article.CodeRequestPreconditionRequired, 428, platform.RetryAllowed, article.ErrPreconditionRequired, "必须提供当前资源版本"},
		{platform.CodeRequestRateLimited, 429, platform.RetryAllowed, nil, "请求过于频繁，请稍后重试"},
		{platform.CodePlatformProcessingGateClosed, 503, platform.RetryAfterConfiguration, nil, "该能力尚未开放"},
		{article.CodeArticleDeletionInProgress, 503, platform.RetryAllowed, article.ErrDeletionInProgress, "删除正在完成，请稍后核对"},
		{platform.CodePlatformDatabaseUnavailable, 503, platform.RetryAllowed, nil, "服务暂时不可用"},
		{platform.CodePlatformRecoveryInProgress, 503, platform.RetryAllowed, nil, "服务正在恢复，暂不可用"},
		{platform.CodePlatformDependencyUnavailable, 503, platform.RetryAllowed, nil, "依赖服务暂时不可用"},
		{platform.CodePlatformInternalError, 500, platform.RetryConditional, errors.New(privateCanary), "服务内部错误"},
	}
	for i := range cases {
		if cases[i].cause == nil {
			cases[i].cause = platform.NewPublicError(cases[i].code, errors.New(privateCanary))
		}
	}
	return cases
}

type queryFacts struct {
	rows              []article.Article
	calls, identities int
	status            string
	err               error
}

func (q *queryFacts) ListCandidateBatch(context.Context, *article.ArticleCursor, int) ([]article.Article, error) {
	q.calls++
	return append([]article.Article{}, q.rows...), q.err
}
func (q *queryFacts) GetArticle(context.Context, string) (article.Article, error) {
	q.calls++
	return q.rows[0], q.err
}
func (q *queryFacts) GetAuthorStates(context.Context, []string) (map[string]string, error) {
	q.identities++
	if q.status == "" {
		return nil, nil
	}
	return map[string]string{authorID: q.status}, nil
}
func actualQuery(t *testing.T, q *queryFacts) *article.ArticleQueryService {
	t.Helper()
	s, err := article.NewArticleQueryService(q, q, article.QueryConfig{DefaultLimit: 2, MaxLimit: 2, CandidateBatchSize: 3})
	must(t, err)
	return s
}

func restAdminPriority(t *testing.T) {
	for index, raw := range []string{"view=%ZZ", "view=hidden&view=%ZZ", "view=all&cursor=%ZZ", "view=hidden&limit=%ZZ", "view=hidden;view=violating", "view=all&other=%ZZ", "view=", "view=all&view=hidden"} {
		t.Run("raw_"+intText(int64(index)), func(t *testing.T) {
			f := newTransport(t)
			r := request(http.MethodGet, "/api/v1/admin/articles", nil)
			r.URL.RawQuery = raw
			p := principal("admin")
			w := serve(t, f.admin.ListAdminArticles, &p, r)
			assertError(t, w, 422, article.CodeArticleValidationFailed)
			if f.port.calls != 0 {
				t.Fatal("RawQuery error reached service")
			}
		})
	}
	for _, raw := range []string{"", "view=all", "view=violating", "view=hidden"} {
		f := newTransport(t)
		p := principal("admin")
		w := serve(t, f.admin.ListAdminArticles, &p, request(http.MethodGet, "/api/v1/admin/articles?"+raw, nil))
		if w.Code != 200 || f.port.calls != 1 {
			t.Fatal("valid view not passed")
		}
		want := article.AdminListViewAll
		if strings.Contains(raw, "violating") {
			want = article.AdminListViewViolating
		}
		if strings.Contains(raw, "hidden") {
			want = article.AdminListViewHidden
		}
		if f.port.admin.View != want {
			t.Fatal("typed admin view argument changed")
		}
	}
	t.Run("actual_three_views", restActualAdminViews)
	for _, mode := range []string{"invalid_state", "pending", "missing_identity", "cursor", "limit", "typed_zero"} {
		t.Run(mode, func(t *testing.T) { adminStateCase(t, mode) })
	}
	f := newTransport(t)
	p := principal("member")
	r := request(http.MethodGet, "/api/v1/admin/articles?view=%ZZ", nil)
	assertError(t, serve(t, f.admin.ListAdminArticles, &p, r), 403, platform.CodeAuthForbidden)
	if f.port.calls != 0 {
		t.Fatal("nonadmin reached service")
	}
}

func restActualAdminViews(t *testing.T) {
	for _, view := range []string{"all", "violating", "hidden"} {
		t.Run(view, func(t *testing.T) {
			clear := baseArticle()
			hidden := clear
			hidden.ID = "10000000-0000-4000-8000-000000000002"
			hidden.ViolationState = article.ViolationViolating
			c := article.ViolationViolence
			hidden.ViolationCategory = &c
			q := &queryFacts{rows: []article.Article{hidden, clear}, status: "active"}
			h, err := article.NewAdminQueryHandler(actualQuery(t, q))
			must(t, err)
			p := principal("admin")
			w := serve(t, h.ListAdminArticles, &p, request(http.MethodGet, "/api/v1/admin/articles?view="+view, nil))
			if w.Code != 200 {
				t.Fatalf("actual query HTTP=%d", w.Code)
			}
			obj := jsonObject(t, w.Body.Bytes(), "items", "next_cursor")
			var items []json.RawMessage
			must(t, json.Unmarshal(obj["items"], &items))
			want := 1
			if view == "all" {
				want = 2
			}
			if len(items) != want {
				t.Fatal("actual Admin filtering changed")
			}
			assertAdmin(t, items[0], hidden, []string{"violation"})
			if want == 2 {
				assertAdmin(t, items[1], clear, []string{})
			}
			fieldValue(t, obj, "next_cursor", nil)
		})
	}
}

func adminStateCase(t *testing.T, mode string) {
	q := &queryFacts{rows: []article.Article{baseArticle()}, status: "active"}
	p := principal("admin")
	service := actualQuery(t, q)
	h, err := article.NewAdminQueryHandler(service)
	must(t, err)
	if mode == "typed_zero" {
		page, err := service.ListAdminArticles(context.Background(), p, article.ListAdminArticlesRequest{})
		must(t, err)
		if len(page.Items) != 1 {
			t.Fatal("typed zero did not mean all")
		}
		return
	}
	path, status, code := adminStateExpectation(mode, q)
	fn := h.GetAdminArticle
	if mode == "cursor" || mode == "limit" {
		fn = h.ListAdminArticles
	}
	w := serve(t, fn, &p, request(http.MethodGet, path, nil))
	assertError(t, w, status, code)
	if mode == "pending" && q.identities != 0 {
		t.Fatal("pending queried identity")
	}
}

func adminStateExpectation(mode string, q *queryFacts) (string, int, platform.Code) {
	path := "/api/v1/admin/articles/" + articleID
	switch mode {
	case "invalid_state":
		q.rows[0].Version = 0
		return path, 500, platform.CodePlatformInternalError
	case "pending":
		op := noticeID
		q.rows[0].DeletionState = article.DeletionPending
		q.rows[0].DeletionOperationID = &op
		q.status = ""
		return path, 404, article.CodeArticleUnavailable
	case "missing_identity":
		q.status = ""
		return path, 503, platform.CodePlatformDependencyUnavailable
	case "cursor":
		return "/api/v1/admin/articles?cursor=", 400, platform.CodeRequestInvalidCursor
	default:
		return "/api/v1/admin/articles?limit=0", 422, article.CodeArticleValidationFailed
	}
}

type failedWriter struct {
	header http.Header
	status int
}

func (w *failedWriter) Header() http.Header       { return w.header }
func (w *failedWriter) WriteHeader(status int)    { w.status = status }
func (w *failedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func restWriteFailure(t *testing.T) {
	f := newTransport(t)
	e := endpointFor(t, f, "Create")
	w := &failedWriter{header: http.Header{}}
	component(t, e.fn, &e.actor).ServeHTTP(w, request(e.method, e.path, e.body))
	if w.status != 201 || f.port.calls != 1 || f.port.writeFailures != 1 {
		t.Fatal("committed response failure retried command or was not recorded")
	}
}
