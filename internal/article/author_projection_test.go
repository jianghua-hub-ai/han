package article

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"han/internal/platform"
	"han/internal/session"
)

func TestAuthorArticle_ReturnsMinimalProjectionAndETag(t *testing.T) {
	existing := activeUpdateArticle()
	fixture := newAuthorProjectionFixture(t, existing)
	response := fixture.serve(t, activeUpdatePrincipal())

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body)
	}
	if got := response.Header().Get("ETag"); got != `"article-`+updateArticleID+`-v2"` {
		t.Fatalf("ETag = %q, want v2", got)
	}
	payload := decodeAuthorProjection(t, response)
	if payload.Title != existing.Title || payload.Body != existing.Body ||
		payload.ModerationStatus != ModerationNormal || !payload.Editable {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestAuthorArticle_ViolatingArticleIsRestrictedWithoutLeakingCategory(t *testing.T) {
	existing := activeUpdateArticle()
	category := ViolationViolence
	existing.ViolationState = ViolationViolating
	existing.ViolationCategory = &category
	fixture := newAuthorProjectionFixture(t, existing)
	response := fixture.serve(t, activeUpdatePrincipal())

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body)
	}
	payload := decodeAuthorProjection(t, response)
	if payload.ModerationStatus != ModerationRestricted {
		t.Fatalf("moderation_status = %q, want restricted", payload.ModerationStatus)
	}
	body := response.Body.String()
	for _, forbidden := range []string{
		"violation_category", "violation_state", "hidden_reasons",
		string(category), "violation",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("author projection leaked %q: %s", forbidden, body)
		}
	}
}

func TestAuthorArticle_NonAuthorMemberIsUnavailable(t *testing.T) {
	fixture := newAuthorProjectionFixture(t, activeUpdateArticle())
	principal := activeUpdatePrincipal()
	principal.UserID = updateOtherUserID
	response := fixture.serve(t, principal)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", response.Code, response.Body)
	}
}

func TestAuthorArticle_NotFoundForUnknownArticle(t *testing.T) {
	fixture := newAuthorProjectionFixture(t, Article{})
	response := fixture.serve(t, activeUpdatePrincipal())
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", response.Code, response.Body)
	}
}

func TestAuthorArticle_DeletedArticleIsUnavailable(t *testing.T) {
	existing := activeUpdateArticle()
	existing.DeletionState = DeletionPending
	fixture := newAuthorProjectionFixture(t, existing)
	response := fixture.serve(t, activeUpdatePrincipal())
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", response.Code, response.Body)
	}
}

func decodeAuthorProjection(
	t *testing.T,
	response *httptest.ResponseRecorder,
) AuthorArticleView {
	t.Helper()
	var payload AuthorArticleView
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode author projection: %v", err)
	}
	return payload
}

type authorProjectionFixture struct {
	handler *Handler
	reader  *articleQueryRepositoryStub
}

func newAuthorProjectionFixture(t *testing.T, existing Article) authorProjectionFixture {
	t.Helper()
	reader := &articleQueryRepositoryStub{articles: []Article{existing}}
	service, err := NewArticleQueryService(
		reader,
		&articleAuthorStateReaderStub{},
		QueryConfig{DefaultLimit: 2, MaxLimit: 2, CandidateBatchSize: 3},
	)
	if err != nil {
		t.Fatalf("NewArticleQueryService() error = %v", err)
	}
	handler, err := NewAuthorDetailHandler(service)
	if err != nil {
		t.Fatalf("NewAuthorDetailHandler() error = %v", err)
	}
	return authorProjectionFixture{handler: handler, reader: reader}
}

func (f authorProjectionFixture) serve(
	t *testing.T,
	principal session.Principal,
) *httptest.ResponseRecorder {
	t.Helper()
	return serveAuthorProjection(t, f.handler, principal)
}

func serveAuthorProjection(
	t *testing.T,
	handler *Handler,
	principal session.Principal,
) *httptest.ResponseRecorder {
	t.Helper()
	authorizer, err := platform.NewHTTPAuthorizer(
		updateArticlePrincipalResolver{principal: principal},
		func() time.Time { return updateArticleNow },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	protected := authorizer.Protect(
		session.AccessGeneral,
		http.HandlerFunc(handler.GetAuthorArticle),
	)
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/users/me/articles/"+updateArticleID,
		nil,
	)
	request.AddCookie(&http.Cookie{
		Name:  platform.SessionCookieName,
		Value: "session",
	})
	response := httptest.NewRecorder()
	platform.TraceMiddleware(protected).ServeHTTP(response, request)
	return response
}
