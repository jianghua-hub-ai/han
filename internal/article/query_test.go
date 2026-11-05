package article

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

var queryTestTime = time.Date(2026, time.October, 25, 1, 2, 3, 0, time.UTC)

func TestVisibleScan_HiddenDenseAndSameTimestamp_NoMissingOrDuplicates(t *testing.T) {
	authorActive := queryTestID(101)
	authorSuspended := queryTestID(102)
	authorMissing := queryTestID(103)
	repository := &articleQueryRepositoryStub{articles: []Article{
		queryTestArticle(9, &authorActive),
		queryViolatingArticle(8, &authorActive),
		queryTestArticle(7, &authorSuspended),
		queryPendingArticle(6, &authorActive),
		queryTestArticle(5, &authorActive),
		queryHeldArticle(4),
		queryTestArticle(3, nil),
		queryTestArticle(2, &authorMissing),
		queryTestArticle(1, &authorActive),
	}}
	reader := &articleAuthorStateReaderStub{states: map[string]queryAuthorStatus{
		authorActive:    queryAuthorActive,
		authorSuspended: queryAuthorSuspended,
	}}
	service := newArticleQueryServiceForTest(t, repository, reader)

	first, err := service.ListArticles(context.Background(), ListArticlesRequest{})
	if err != nil {
		t.Fatalf("ListArticles(first) error = %v", err)
	}
	requireArticleIDs(t, first.Items, []string{queryTestID(9), queryTestID(5)})
	if first.NextCursor == nil {
		t.Fatal("ListArticles(first) next cursor = nil, want cursor")
	}
	if repository.batchCalls != 3 {
		t.Fatalf("ListCandidateBatch(first) calls = %d, want 3", repository.batchCalls)
	}
	position, err := (CursorCodec{}).Decode(*first.NextCursor, CursorViewMember)
	if err != nil {
		t.Fatalf("Decode(first next cursor) error = %v", err)
	}
	if position.ID != queryTestID(5) {
		t.Fatalf("first next cursor id = %q, want %q", position.ID, queryTestID(5))
	}

	second, err := service.ListArticles(context.Background(), ListArticlesRequest{
		Cursor: *first.NextCursor,
	})
	if err != nil {
		t.Fatalf("ListArticles(second) error = %v", err)
	}
	requireArticleIDs(t, second.Items, []string{queryTestID(3), queryTestID(1)})
	if second.NextCursor != nil {
		t.Fatalf("ListArticles(second) next cursor = %q, want nil", *second.NextCursor)
	}
	if repository.batchCalls != 5 {
		t.Fatalf("ListCandidateBatch(total) calls = %d, want 5", repository.batchCalls)
	}

	all := append(append([]MemberArticleSummaryView{}, first.Items...), second.Items...)
	requireArticleIDs(t, all, []string{
		queryTestID(9), queryTestID(5), queryTestID(3), queryTestID(1),
	})
	for _, item := range all {
		data, marshalErr := json.Marshal(item)
		if marshalErr != nil {
			t.Fatalf("json.Marshal(summary) error = %v", marshalErr)
		}
		var fields map[string]json.RawMessage
		if unmarshalErr := json.Unmarshal(data, &fields); unmarshalErr != nil {
			t.Fatalf("json.Unmarshal(summary) error = %v", unmarshalErr)
		}
		if _, leaked := fields["body"]; leaked {
			t.Fatalf("list item leaked body: %s", data)
		}
	}
}

func TestVisibleScan_AuthorStateMissingAndCurrentChanges_FailsClosed(t *testing.T) {
	authorID := queryTestID(201)
	repository := &articleQueryRepositoryStub{articles: []Article{
		queryTestArticle(3, &authorID),
		queryTestArticle(2, nil),
	}}
	reader := &articleAuthorStateReaderStub{stateSequence: []map[string]queryAuthorStatus{
		{},
		{authorID: queryAuthorSuspended},
		{authorID: queryAuthorActive},
	}}
	service := newArticleQueryServiceForTest(t, repository, reader)
	limit := 1

	missing, err := service.ListArticles(context.Background(), ListArticlesRequest{Limit: &limit})
	if err != nil {
		t.Fatalf("ListArticles(missing state) error = %v", err)
	}
	requireArticleIDs(t, missing.Items, []string{queryTestID(2)})

	suspended, err := service.ListArticles(context.Background(), ListArticlesRequest{Limit: &limit})
	if err != nil {
		t.Fatalf("ListArticles(suspended state) error = %v", err)
	}
	requireArticleIDs(t, suspended.Items, []string{queryTestID(2)})

	active, err := service.ListArticles(context.Background(), ListArticlesRequest{Limit: &limit})
	if err != nil {
		t.Fatalf("ListArticles(active state) error = %v", err)
	}
	requireArticleIDs(t, active.Items, []string{queryTestID(3)})
	if reader.calls != 3 {
		t.Fatalf("GetAuthorStates() calls = %d, want 3 current reads", reader.calls)
	}

	reader.err = errors.New("identity unavailable")
	_, err = service.ListArticles(context.Background(), ListArticlesRequest{Limit: &limit})
	if platform.PublicCode(err) != platform.CodePlatformDependencyUnavailable {
		t.Fatalf("ListArticles(identity failure) code = %q, want %q", platform.PublicCode(err), platform.CodePlatformDependencyUnavailable)
	}
}

func TestListArticles_DependencyFailuresAndLocalDefectsKeepStableBoundaries(t *testing.T) {
	authorID := queryTestID(250)
	tests := []struct {
		name       string
		repository *articleQueryRepositoryStub
		reader     *articleAuthorStateReaderStub
		wantStatus int
		wantCode   platform.Code
		canary     string
	}{
		{
			name: "repository unavailable",
			repository: &articleQueryRepositoryStub{
				err: errors.New("repository-list-cause-canary"),
			},
			reader:     &articleAuthorStateReaderStub{},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   platform.CodePlatformDatabaseUnavailable,
			canary:     "repository-list-cause-canary",
		},
		{
			name: "identity unavailable",
			repository: &articleQueryRepositoryStub{
				articles: []Article{queryTestArticle(2, &authorID)},
			},
			reader: &articleAuthorStateReaderStub{
				err: errors.New("identity-list-cause-canary"),
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   platform.CodePlatformDependencyUnavailable,
			canary:     "identity-list-cause-canary",
		},
		{
			name: "local candidate order defect",
			repository: &articleQueryRepositoryStub{
				articles: []Article{
					queryTestArticle(1, nil),
					queryTestArticle(2, nil),
				},
			},
			reader:     &articleAuthorStateReaderStub{},
			wantStatus: http.StatusInternalServerError,
			wantCode:   platform.CodePlatformInternalError,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newArticleQueryServiceForTest(
				t,
				test.repository,
				test.reader,
			)
			handler, err := NewQueryHandler(service)
			if err != nil {
				t.Fatalf("NewQueryHandler() error = %v", err)
			}

			response := serveArticleQuery(
				t,
				handler.ListArticles,
				"/api/v1/articles",
			)

			requireArticleQueryError(t, response, test.wantStatus, test.wantCode)
			if test.canary != "" && strings.Contains(response.Body.String(), test.canary) {
				t.Fatalf("error envelope leaked cause: %s", response.Body)
			}
		})
	}
}

func TestListArticles_HandlerValidatesPaginationAndWritesEnvelope(t *testing.T) {
	repository := &articleQueryRepositoryStub{articles: []Article{
		queryTestArticle(2, nil),
		queryTestArticle(1, nil),
	}}
	service := newArticleQueryServiceForTest(t, repository, &articleAuthorStateReaderStub{})
	handler, err := NewQueryHandler(service)
	if err != nil {
		t.Fatalf("NewQueryHandler() error = %v", err)
	}

	response := serveArticleQuery(t, handler.ListArticles, "/api/v1/articles?limit=1")
	if response.Code != http.StatusOK {
		t.Fatalf("ListArticles status = %d, want 200; body=%s", response.Code, response.Body)
	}
	var page ListArticlesPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode list page: %v", err)
	}
	requireArticleIDs(t, page.Items, []string{queryTestID(2)})
	if page.NextCursor == nil {
		t.Fatal("ListArticles next_cursor = nil, want cursor")
	}

	tests := []struct {
		name       string
		target     string
		wantStatus int
		wantCode   platform.Code
	}{
		{
			name: "invalid cursor", target: "/api/v1/articles?cursor=not-base64",
			wantStatus: http.StatusBadRequest, wantCode: platform.CodeRequestInvalidCursor,
		},
		{
			name: "empty limit", target: "/api/v1/articles?limit=",
			wantStatus: http.StatusUnprocessableEntity, wantCode: CodeArticleValidationFailed,
		},
		{
			name: "limit above configured maximum", target: "/api/v1/articles?limit=3",
			wantStatus: http.StatusUnprocessableEntity, wantCode: CodeArticleValidationFailed,
		},
		{
			name: "duplicate cursor", target: "/api/v1/articles?cursor=a&cursor=b",
			wantStatus: http.StatusBadRequest, wantCode: platform.CodeRequestInvalidCursor,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serveArticleQuery(t, handler.ListArticles, test.target)
			requireArticleQueryError(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func TestGetArticle_DependencyFailuresKeepStableBoundaries(t *testing.T) {
	authorID := queryTestID(350)
	articleID := queryTestID(351)
	visible := queryTestArticle(351, &authorID)
	tests := []struct {
		name       string
		repository *articleQueryRepositoryStub
		reader     *articleAuthorStateReaderStub
		wantCode   platform.Code
		canary     string
	}{
		{
			name: "repository unavailable",
			repository: &articleQueryRepositoryStub{
				err: errors.New("repository-detail-cause-canary"),
			},
			reader:   &articleAuthorStateReaderStub{},
			wantCode: platform.CodePlatformDatabaseUnavailable,
			canary:   "repository-detail-cause-canary",
		},
		{
			name: "identity unavailable",
			repository: &articleQueryRepositoryStub{
				byID: map[string]Article{articleID: visible},
			},
			reader: &articleAuthorStateReaderStub{
				err: errors.New("identity-detail-cause-canary"),
			},
			wantCode: platform.CodePlatformDependencyUnavailable,
			canary:   "identity-detail-cause-canary",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newArticleQueryServiceForTest(
				t,
				test.repository,
				test.reader,
			)
			handler, err := NewQueryHandler(service)
			if err != nil {
				t.Fatalf("NewQueryHandler() error = %v", err)
			}

			response := serveArticleQuery(
				t,
				handler.GetArticle,
				"/api/v1/articles/"+articleID,
			)

			requireArticleQueryError(
				t,
				response,
				http.StatusServiceUnavailable,
				test.wantCode,
			)
			if strings.Contains(response.Body.String(), test.canary) {
				t.Fatalf("error envelope leaked cause: %s", response.Body)
			}
		})
	}
}

func TestGetArticle_VisibleDetailReturnsETagAndInvisibleStatesShare404(t *testing.T) {
	authorID := queryTestID(301)
	visible := queryTestArticle(9, &authorID)
	visible.Body = "visible body"
	repository := &articleQueryRepositoryStub{
		articles: []Article{visible},
		byID: map[string]Article{
			visible.ID:     visible,
			queryTestID(8): queryViolatingArticle(8, &authorID),
			queryTestID(7): queryPendingArticle(7, &authorID),
			queryTestID(6): queryHeldArticle(6),
			queryTestID(5): queryTestArticle(5, &authorID),
		},
	}
	reader := &articleAuthorStateReaderStub{states: map[string]queryAuthorStatus{
		authorID: queryAuthorActive,
	}}
	service := newArticleQueryServiceForTest(t, repository, reader)
	handler, err := NewQueryHandler(service)
	if err != nil {
		t.Fatalf("NewQueryHandler() error = %v", err)
	}

	response := serveArticleQuery(
		t,
		handler.GetArticle,
		"/api/v1/articles/"+visible.ID,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("GetArticle status = %d, want 200; body=%s", response.Code, response.Body)
	}
	if got := response.Header().Get("ETag"); got != `"article-`+visible.ID+`-v1"` {
		t.Fatalf("GetArticle ETag = %q", got)
	}
	var detail MemberArticleDetailView
	if err := json.Unmarshal(response.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode article detail: %v", err)
	}
	if detail.ID != visible.ID || detail.Body != visible.Body {
		t.Fatalf("GetArticle detail = %#v", detail)
	}

	tests := []struct {
		name      string
		articleID string
		before    func()
	}{
		{name: "not found", articleID: queryTestID(404)},
		{name: "violating", articleID: queryTestID(8)},
		{name: "deletion pending", articleID: queryTestID(7)},
		{name: "held orphan", articleID: queryTestID(6)},
		{
			name: "author suspended", articleID: queryTestID(5),
			before: func() {
				reader.states[authorID] = queryAuthorSuspended
			},
		},
		{
			name: "author state missing", articleID: queryTestID(5),
			before: func() {
				delete(reader.states, authorID)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader.states[authorID] = queryAuthorActive
			if test.before != nil {
				test.before()
			}
			response := serveArticleQuery(
				t,
				handler.GetArticle,
				"/api/v1/articles/"+test.articleID,
			)
			requireArticleQueryError(
				t,
				response,
				http.StatusNotFound,
				CodeArticleUnavailable,
			)
			if response.Header().Get("ETag") != "" {
				t.Fatalf("invisible response exposed ETag %q", response.Header().Get("ETag"))
			}
			if containsAny(response.Body.String(), "visible body", "violation", "deletion") {
				t.Fatalf("404 response leaked article state/content: %s", response.Body)
			}
		})
	}
}

func newArticleQueryServiceForTest(
	t *testing.T,
	repository ArticleQueryRepository,
	reader ArticleAuthorStateReader[queryAuthorStatus],
) *ArticleQueryService {
	t.Helper()
	service, err := NewArticleQueryService(repository, reader, QueryConfig{
		DefaultLimit:       2,
		MaxLimit:           2,
		CandidateBatchSize: 3,
	})
	if err != nil {
		t.Fatalf("NewArticleQueryService() error = %v", err)
	}
	return service
}

func queryTestArticle(sequence int, authorID *string) Article {
	return Article{
		ID:               queryTestID(sequence),
		AuthorID:         authorID,
		Title:            fmt.Sprintf("title-%d", sequence),
		Body:             fmt.Sprintf("body-%d", sequence),
		PublicationState: PublicationPublished,
		ViolationState:   ViolationClear,
		DeletionState:    DeletionActive,
		Version:          1,
		CreatedAt:        queryTestTime,
		UpdatedAt:        queryTestTime,
	}
}

func queryViolatingArticle(sequence int, authorID *string) Article {
	article := queryTestArticle(sequence, authorID)
	category := ViolationViolence
	article.ViolationState = ViolationViolating
	article.ViolationCategory = &category
	return article
}

func queryPendingArticle(sequence int, authorID *string) Article {
	article := queryTestArticle(sequence, authorID)
	operationID := queryTestID(sequence + 500)
	article.DeletionState = DeletionPending
	article.DeletionOperationID = &operationID
	return article
}

func queryHeldArticle(sequence int) Article {
	article := queryTestArticle(sequence, nil)
	sourceID := queryTestID(sequence + 600)
	article.OrphanedFromUserID = &sourceID
	article.PublicationState = PublicationHeld
	return article
}

func queryTestID(sequence int) string {
	return fmt.Sprintf("10000000-0000-4000-8000-%012d", sequence)
}

func requireArticleIDs(
	t *testing.T,
	items []MemberArticleSummaryView,
	want []string,
) {
	t.Helper()
	got := make([]string, 0, len(items))
	for _, item := range items {
		got = append(got, item.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("article IDs = %v, want %v", got, want)
	}
}

func requireArticleQueryError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	status int,
	code platform.Code,
) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, status, response.Body)
	}
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Code != code {
		t.Fatalf("error code = %q, want %q", envelope.Code, code)
	}
}

func serveArticleQuery(
	t *testing.T,
	method http.HandlerFunc,
	target string,
) *httptest.ResponseRecorder {
	t.Helper()
	authorizer, err := platform.NewHTTPAuthorizer(
		articleQueryPrincipalResolver{principal: session.Principal{
			SessionID: "query-session",
			UserID:    queryTestID(900),
			Role:      principalRoleMember,
			Status:    principalStatusActive,
		}},
		func() time.Time { return queryTestTime },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	protected := authorizer.Protect(session.AccessGeneral, method)
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.AddCookie(&http.Cookie{
		Name:  platform.SessionCookieName,
		Value: "query-session",
	})
	response := httptest.NewRecorder()
	platform.TraceMiddleware(protected).ServeHTTP(response, request)
	return response
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if candidate != "" && len(value) >= len(candidate) {
			for index := 0; index+len(candidate) <= len(value); index++ {
				if value[index:index+len(candidate)] == candidate {
					return true
				}
			}
		}
	}
	return false
}

type articleQueryPrincipalResolver struct {
	principal session.Principal
}

func (r articleQueryPrincipalResolver) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return r.principal, nil
}

type articleQueryRepositoryStub struct {
	articles   []Article
	byID       map[string]Article
	batchCalls int
	getCalls   int
	err        error
}

func (s *articleQueryRepositoryStub) ListCandidateBatch(
	_ context.Context,
	after *ArticleCursor,
	limit int,
) ([]Article, error) {
	s.batchCalls++
	if s.err != nil {
		return nil, s.err
	}
	start := 0
	if after != nil {
		start = len(s.articles)
		for index, article := range s.articles {
			if article.CreatedAt.Before(after.CreatedAt) ||
				article.CreatedAt.Equal(after.CreatedAt) && article.ID < after.ID {
				start = index
				break
			}
		}
	}
	end := start + limit
	if end > len(s.articles) {
		end = len(s.articles)
	}
	return append([]Article(nil), s.articles[start:end]...), nil
}

func (s *articleQueryRepositoryStub) GetArticle(
	_ context.Context,
	articleID string,
) (Article, error) {
	s.getCalls++
	if s.err != nil {
		return Article{}, s.err
	}
	if s.byID != nil {
		if article, exists := s.byID[articleID]; exists {
			return article, nil
		}
	}
	for _, article := range s.articles {
		if article.ID == articleID {
			return article, nil
		}
	}
	return Article{}, pgx.ErrNoRows
}

type articleAuthorStateReaderStub struct {
	states        map[string]queryAuthorStatus
	stateSequence []map[string]queryAuthorStatus
	calls         int
	err           error
}

func (s *articleAuthorStateReaderStub) GetAuthorStates(
	_ context.Context,
	_ []string,
) (map[string]queryAuthorStatus, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	if len(s.stateSequence) >= s.calls {
		return cloneAuthorStates(s.stateSequence[s.calls-1]), nil
	}
	return cloneAuthorStates(s.states), nil
}

func cloneAuthorStates(
	states map[string]queryAuthorStatus,
) map[string]queryAuthorStatus {
	cloned := make(map[string]queryAuthorStatus, len(states))
	for id, status := range states {
		cloned[id] = status
	}
	return cloned
}

type queryAuthorStatus string

const (
	queryAuthorActive    queryAuthorStatus = "active"
	queryAuthorSuspended queryAuthorStatus = "suspended"
)
