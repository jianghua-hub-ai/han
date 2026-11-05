package article

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"han/internal/session"
)

func TestAdminDeleteOrdinaryArticle_DualCompletionAndAudit(t *testing.T) {
	for _, principal := range []session.Principal{deletionAdminPrincipal(), deletionAuthorPrincipal()} {
		t.Run(principal.Role, func(t *testing.T) {
			fixture := newPermanentDeletionFixture(t)
			response := serveAdminOrdinaryDelete(t, fixture, principal, deletionRequest(DeleteOrdinary).IfMatch)
			if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
				t.Fatalf("response = %d %s, want empty 204", response.Code, response.Body)
			}
			requireAdminDeletionCompleted(t, fixture, principal)
			response = serveAdminOrdinaryDelete(t, fixture, principal, deletionRequest(DeleteOrdinary).IfMatch)
			if response.Code != http.StatusNotFound || len(fixture.auditor.events) != 1 {
				t.Fatalf("completed retry = %d, audits = %d", response.Code, len(fixture.auditor.events))
			}
		})
	}
}

func TestAdminDeleteOrdinaryArticle_PendingRetry(t *testing.T) {
	for _, failure := range []string{"journal", "finalizer"} {
		t.Run(failure, func(t *testing.T) {
			fixture := newPermanentDeletionFixture(t)
			if failure == "journal" {
				fixture.journal.appendErr = errors.New("synthetic journal failure")
			} else {
				fixture.finalizer.err = errors.New("synthetic finalizer failure")
			}
			principal := deletionAdminPrincipal()
			etag := deletionRequest(DeleteOrdinary).IfMatch
			response := serveAdminOrdinaryDelete(t, fixture, principal, etag)
			requireAdminDeleteError(t, response, http.StatusServiceUnavailable, string(CodeArticleDeletionInProgress))
			if fixture.repository.article == nil || fixture.repository.article.DeletionState != DeletionPending ||
				fixture.repository.operation == nil || len(fixture.auditor.events) != 0 {
				t.Fatal("incomplete deletion must remain pending without completion audit")
			}
			fixture.journal.appendErr, fixture.finalizer.err = nil, nil
			response = serveAdminOrdinaryDelete(t, fixture, principal, etag)
			if response.Code != http.StatusNoContent {
				t.Fatalf("retry = %d %s", response.Code, response.Body)
			}
			requireAdminDeletionCompleted(t, fixture, principal)
		})
	}
}

func TestAdminDeleteOrdinaryArticle_StateAndAuthorization(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*Article, *session.Principal)
		status  int
		code    string
	}{
		{"violating", func(article *Article, _ *session.Principal) {
			category := ViolationViolence
			article.ViolationState = ViolationViolating
			article.ViolationCategory = &category
		}, http.StatusConflict, string(CodeArticleInvalidState)},
		{"orphan", func(article *Article, _ *session.Principal) {
			article.AuthorID = nil
			article.OrphanedFromUserID = stringPointer(deletionAuthorID)
			article.PublicationState = PublicationHeld
		}, http.StatusConflict, string(CodeArticleInvalidState)},
		{"held", func(article *Article, _ *session.Principal) { article.PublicationState = PublicationHeld }, http.StatusConflict, string(CodeArticleInvalidState)},
		{"other member", func(_ *Article, principal *session.Principal) { principal.Role = "member" }, http.StatusNotFound, string(CodeArticleUnavailable)},
		{"password restricted", func(_ *Article, principal *session.Principal) { principal.MustChangePassword = true }, http.StatusForbidden, ""},
		{"privacy restricted", func(_ *Article, principal *session.Principal) { principal.MustAcknowledgePrivacy = true }, http.StatusForbidden, ""},
		{"suspended", func(_ *Article, principal *session.Principal) { principal.Status = "suspended" }, http.StatusForbidden, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPermanentDeletionFixture(t)
			principal := deletionAdminPrincipal()
			test.prepare(fixture.repository.article, &principal)
			response := serveAdminOrdinaryDelete(t, fixture, principal, deletionRequest(DeleteOrdinary).IfMatch)
			requireAdminDeleteError(t, response, test.status, test.code)
			requireAdminDeletionNotStarted(t, fixture)
		})
	}
}

func TestAdminDeleteOrdinaryArticle_ETag(t *testing.T) {
	current := deletionRequest(DeleteOrdinary).IfMatch
	tests := []struct {
		name, etag string
		status     int
		code       string
	}{
		{"missing", "", http.StatusPreconditionRequired, string(CodeRequestPreconditionRequired)},
		{"weak", "W/" + current, http.StatusBadRequest, string(CodeRequestInvalidPrecondition)},
		{"wildcard", "*", http.StatusBadRequest, string(CodeRequestInvalidPrecondition)},
		{"multiple", current + "," + current, http.StatusBadRequest, string(CodeRequestInvalidPrecondition)},
		{"stale", strings.Replace(current, "v3", "v2", 1), http.StatusPreconditionFailed, string(CodeArticlePreconditionFailed)},
		{"other article", strings.Replace(current, deletionArticleID, updateArticleID, 1), http.StatusPreconditionFailed, string(CodeArticlePreconditionFailed)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPermanentDeletionFixture(t)
			response := serveAdminOrdinaryDelete(t, fixture, deletionAdminPrincipal(), test.etag)
			requireAdminDeleteError(t, response, test.status, test.code)
			requireAdminDeletionNotStarted(t, fixture)
		})
	}
}

func TestAdminDeleteOrdinaryArticle_CannotEditOtherAuthor(t *testing.T) {
	fixture := newUpdateArticleFixture(t, activeUpdateArticle())
	response := fixture.serve(t, deletionAdminPrincipal(), "replacement", "replacement body", `"article-`+updateArticleID+`-v2"`)
	if response.Code != http.StatusForbidden || fixture.repository.updateCalls != 0 {
		t.Fatalf("edit response = %d, updates = %d", response.Code, fixture.repository.updateCalls)
	}
}

func serveAdminOrdinaryDelete(t *testing.T, fixture permanentDeletionFixture, principal session.Principal, etag string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/articles/"+deletionArticleID, nil)
	if etag != "" {
		request.Header.Set("If-Match", etag)
	}
	response := httptest.NewRecorder()
	fixture.serve(t, fixture.handler(t).DeleteOrdinary, principal, response, request)
	return response
}

func requireAdminDeletionCompleted(t *testing.T, fixture permanentDeletionFixture, principal session.Principal) {
	t.Helper()
	operation := fixture.repository.operation
	if fixture.repository.article != nil || operation == nil || operation.Status != DeletionOperationCompleted ||
		len(fixture.journal.records) != 1 || len(fixture.auditor.events) != 1 {
		t.Fatal("204 requires physical deletion, one durable tombstone and one completion audit")
	}
	event := fixture.auditor.events[0]
	if event.ActorUserID == nil || *event.ActorUserID != principal.UserID || event.ActorRole == nil || *event.ActorRole != principal.Role ||
		event.Action != "article.delete" || event.TargetID != deletionArticleID || event.TraceID == "" ||
		event.Metadata["article_version"] != int64(3) || event.Metadata["delete_kind"] != DeleteOrdinary {
		t.Fatalf("unexpected audit = %#v", event)
	}
	if principal.Role == principalRoleAdmin {
		if event.Metadata["authority"] != "administrator" || len(event.Metadata) != 3 {
			t.Fatalf("admin audit metadata = %#v, want authority=administrator", event.Metadata)
		}
	} else if _, exists := event.Metadata["authority"]; exists || len(event.Metadata) != 2 {
		t.Fatalf("author audit semantics changed: %#v", event.Metadata)
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private title") || strings.Contains(string(encoded), "private body") {
		t.Fatal("audit contains article content")
	}
}

func requireAdminDeletionNotStarted(t *testing.T, fixture permanentDeletionFixture) {
	t.Helper()
	if fixture.repository.operation != nil || fixture.repository.article == nil || fixture.repository.article.DeletionState != DeletionActive ||
		len(fixture.journal.records) != 0 || len(fixture.auditor.events) != 0 || fixture.finalizer.calls.Load() != 0 {
		t.Fatal("rejected request reached irreversible deletion")
	}
}

func requireAdminDeleteError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, status, response.Body)
	}
	if code != "" && !strings.Contains(response.Body.String(), `"code":"`+code+`"`) {
		t.Fatalf("body = %s, want code %s", response.Body, code)
	}
}
