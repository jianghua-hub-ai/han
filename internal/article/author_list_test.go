package article

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"han/internal/platform"
	"han/internal/session"
)

// authorListRepository is a deterministic in-memory projection of the SQL
// keyset query: it only returns the requested author's rows, newest first.
type authorListRepository struct {
	articles []Article
	calls    int
}

func (r *authorListRepository) ListCandidateBatch(
	context.Context,
	*ArticleCursor,
	int,
) ([]Article, error) {
	return nil, nil
}

func (r *authorListRepository) GetArticle(context.Context, string) (Article, error) {
	return Article{}, nil
}

func (r *authorListRepository) ListAuthorBatch(
	_ context.Context,
	authorID string,
	after *ArticleCursor,
	limit int,
) ([]Article, error) {
	r.calls++
	items := make([]Article, 0, limit)
	for _, article := range r.articles {
		if article.AuthorID == nil || *article.AuthorID != authorID {
			continue
		}
		if after != nil {
			if article.CreatedAt.After(after.CreatedAt) {
				continue
			}
			if article.CreatedAt.Equal(after.CreatedAt) &&
				article.ID >= after.ID {
				continue
			}
		}
		items = append(items, article)
		if len(items) == limit {
			break
		}
	}
	return items, nil
}

func authorListArticle(id string, minutesAgo int) Article {
	record := activeUpdateArticle()
	record.ID = id
	record.CreatedAt = updateArticleNow.Add(-time.Duration(minutesAgo) * time.Minute)
	record.UpdatedAt = record.CreatedAt
	return record
}

func newAuthorListService(
	t *testing.T,
	articles []Article,
) (*ArticleQueryService, *authorListRepository) {
	t.Helper()
	repository := &authorListRepository{articles: articles}
	service, err := NewArticleQueryService(
		repository,
		&articleAuthorStateReaderStub{},
		QueryConfig{DefaultLimit: 2, MaxLimit: 2, CandidateBatchSize: 3},
	)
	if err != nil {
		t.Fatalf("NewArticleQueryService() error = %v", err)
	}
	return service, repository
}

func TestListAuthorArticles_OwnOnlyIncludingViolating(t *testing.T) {
	category := ViolationViolence
	violating := authorListArticle("11111111-1111-4111-8111-111111111111", 10)
	violating.ViolationState = ViolationViolating
	violating.ViolationCategory = &category

	other := authorListArticle("22222222-2222-4222-8222-222222222222", 20)
	otherAuthor := "99999999-9999-4999-8999-999999999999"
	other.AuthorID = &otherAuthor

	deleted := authorListArticle("33333333-3333-4333-8333-333333333333", 30)
	deleted.DeletionState = DeletionPending

	mine := authorListArticle("44444444-4444-4444-8444-444444444444", 40)

	service, _ := newAuthorListService(t, []Article{violating, other, deleted, mine})
	page, err := service.ListAuthorArticles(
		context.Background(),
		activeUpdatePrincipal(),
		ListArticlesRequest{},
	)
	if err != nil {
		t.Fatalf("ListAuthorArticles() error = %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items = %d, want 2 (default limit); page=%#v", len(page.Items), page)
	}
	if page.Items[0].ID != violating.ID {
		t.Fatalf("first item = %s, want the violating own article", page.Items[0].ID)
	}
	for _, item := range page.Items {
		if item.ID == other.ID {
			t.Fatal("author list leaked another user's article")
		}
		if item.ID == deleted.ID {
			t.Fatal("author list included a deleted article")
		}
	}
}

func TestListAuthorArticles_PagesByCursorWithoutDuplicatesOrSkips(t *testing.T) {
	service, _ := newAuthorListService(t, []Article{
		authorListArticle("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", 10),
		authorListArticle("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", 20),
		authorListArticle("cccccccc-cccc-4ccc-8ccc-cccccccccccc", 30),
	})
	principal := activeUpdatePrincipal()

	first, err := service.ListAuthorArticles(
		context.Background(),
		principal,
		ListArticlesRequest{},
	)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if len(first.Items) != 2 || first.NextCursor == nil {
		t.Fatalf("first page = %#v, want 2 items plus a cursor", first)
	}

	second, err := service.ListAuthorArticles(
		context.Background(),
		principal,
		ListArticlesRequest{Cursor: *first.NextCursor},
	)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(second.Items) != 1 || second.NextCursor != nil {
		t.Fatalf("second page = %#v, want the last item and no cursor", second)
	}
	seen := map[string]bool{}
	for _, item := range append(first.Items, second.Items...) {
		if seen[item.ID] {
			t.Fatalf("duplicate article %s across pages", item.ID)
		}
		seen[item.ID] = true
	}
	if len(seen) != 3 {
		t.Fatalf("distinct articles = %d, want 3", len(seen))
	}
}

func TestListMyArticlesHandler_EnvelopeAndLimitValidation(t *testing.T) {
	service, _ := newAuthorListService(t, []Article{
		authorListArticle("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", 10),
		authorListArticle("bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", 20),
	})
	handler, err := NewAuthorArticleListHandler(service)
	if err != nil {
		t.Fatalf("NewAuthorArticleListHandler() error = %v", err)
	}
	authorizer, err := platform.NewHTTPAuthorizer(
		updateArticlePrincipalResolver{principal: activeUpdatePrincipal()},
		func() time.Time { return updateArticleNow },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	protected := authorizer.Protect(
		session.AccessGeneral,
		http.HandlerFunc(handler.ListMyArticles),
	)
	serve := func(target string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		request.AddCookie(&http.Cookie{
			Name:  platform.SessionCookieName,
			Value: "session",
		})
		response := httptest.NewRecorder()
		platform.TraceMiddleware(protected).ServeHTTP(response, request)
		return response
	}

	response := serve("/api/v1/users/me/articles?limit=1")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body)
	}
	var page ListArticlesPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatalf("page = %#v, want one item and a cursor", page)
	}

	if invalid := serve("/api/v1/users/me/articles?limit=99"); invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("oversized limit status = %d, want 422", invalid.Code)
	}
}
