package article

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	updateArticleID     = "30000000-0000-4000-8000-000000000003"
	updateArticleUserID = "40000000-0000-4000-8000-000000000004"
	updateOtherUserID   = "50000000-0000-4000-8000-000000000005"
)

var updateArticleNow = time.Date(2026, time.October, 27, 2, 3, 4, 0, time.UTC)

func TestUpdateArticle_AuthorReplacesContentWithMatchingETag(t *testing.T) {
	fixture := newUpdateArticleFixture(t, activeUpdateArticle())
	title := "  new title  "
	body := "  **new body**  "
	response := fixture.serve(t, activeUpdatePrincipal(), title, body, `"article-`+updateArticleID+`-v2"`)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body)
	}
	if got := response.Header().Get("ETag"); got != `"article-`+updateArticleID+`-v3"` {
		t.Fatalf("ETag = %q, want v3", got)
	}
	updated := fixture.repository.updated
	if updated.Title != title || updated.Body != body {
		t.Fatalf("content = %q/%q, want %q/%q", updated.Title, updated.Body, title, body)
	}
	if updated.Version != 3 {
		t.Fatalf("version = %d, want 3", updated.Version)
	}
	if !updated.UpdatedAt.Equal(updateArticleNow) {
		t.Fatalf("updated_at = %s, want %s", updated.UpdatedAt, updateArticleNow)
	}

	var payload AuthorArticleView
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Title != title || payload.Body != body ||
		payload.ModerationStatus != ModerationNormal || !payload.Editable {
		t.Fatalf("author payload = %#v", payload)
	}
}

func TestUpdateArticle_ViolatingArticleStaysViolatingAndRestricted(t *testing.T) {
	existing := activeUpdateArticle()
	category := ViolationViolence
	existing.ViolationState = ViolationViolating
	existing.ViolationCategory = &category
	fixture := newUpdateArticleFixture(t, existing)

	response := fixture.serve(t, activeUpdatePrincipal(), "t", "b", `"article-`+updateArticleID+`-v2"`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body)
	}
	var payload AuthorArticleView
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ModerationStatus != ModerationRestricted {
		t.Fatalf("moderation_status = %q, want restricted", payload.ModerationStatus)
	}
	// Governance fields must not leak into the author projection.
	if bytes.Contains(response.Body.Bytes(), []byte("violation")) {
		t.Fatalf("author projection leaked governance field: %s", response.Body)
	}
	updated := fixture.repository.updated
	if updated.ViolationState != ViolationViolating ||
		updated.ViolationCategory == nil || *updated.ViolationCategory != category {
		t.Fatalf("violation facts not preserved: %#v", updated)
	}
	if updated.PublicationState != PublicationPublished || updated.DeletionState != DeletionActive {
		t.Fatalf("orthogonal state changed: %#v", updated)
	}
}

func TestUpdateArticle_NonAuthorMemberIsNotFound(t *testing.T) {
	fixture := newUpdateArticleFixture(t, activeUpdateArticle())
	principal := activeUpdatePrincipal()
	principal.UserID = updateOtherUserID
	response := fixture.serve(t, principal, "t", "b", `"article-`+updateArticleID+`-v2"`)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", response.Code, response.Body)
	}
	if fixture.repository.updateCalls != 0 {
		t.Fatalf("repository update calls = %d, want 0", fixture.repository.updateCalls)
	}
}

func TestUpdateArticle_AdminNonAuthorIsForbidden(t *testing.T) {
	fixture := newUpdateArticleFixture(t, activeUpdateArticle())
	principal := activeUpdatePrincipal()
	principal.UserID = updateOtherUserID
	principal.Role = principalRoleAdmin
	response := fixture.serve(t, principal, "t", "b", `"article-`+updateArticleID+`-v2"`)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body = %s", response.Code, response.Body)
	}
	if fixture.repository.updateCalls != 0 {
		t.Fatalf("repository update calls = %d, want 0", fixture.repository.updateCalls)
	}
}

func TestUpdateArticle_AdminAsAuthorIsTreatedAsAuthor(t *testing.T) {
	fixture := newUpdateArticleFixture(t, activeUpdateArticle())
	principal := activeUpdatePrincipal()
	principal.Role = principalRoleAdmin
	response := fixture.serve(t, principal, "t", "b", `"article-`+updateArticleID+`-v2"`)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", response.Code, response.Body)
	}
	if fixture.repository.updateCalls != 1 {
		t.Fatalf("repository update calls = %d, want 1", fixture.repository.updateCalls)
	}
}

func TestUpdateArticle_WireByteLimitBoundaries(t *testing.T) {
	valid := []byte(`{"title":"t","body":"b"}`)
	invalidUTF8 := append([]byte(`{"title":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`","body":"t"}`)...)
	cases := []struct {
		name       string
		body       []byte
		wantStatus int
		wantCalls  int
	}{
		{
			name:       "exact wire limit",
			body:       padCreateArticleBody(t, valid, MaxArticleWriteBodyBytes),
			wantStatus: http.StatusOK,
			wantCalls:  1,
		},
		{
			name:       "wire limit plus one",
			body:       padCreateArticleBody(t, valid, MaxArticleWriteBodyBytes+1),
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
		},
		{
			name:       "invalid utf8",
			body:       invalidUTF8,
			wantStatus: http.StatusBadRequest,
			wantCalls:  0,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fixture := newUpdateArticleFixture(t, activeUpdateArticle())
			response := serveUpdateArticleBody(
				t, fixture.handler, activeUpdatePrincipal(),
				test.body, `"article-`+updateArticleID+`-v2"`,
			)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body)
			}
			if fixture.repository.updateCalls != test.wantCalls {
				t.Fatalf("repository update calls = %d, want %d", fixture.repository.updateCalls, test.wantCalls)
			}
		})
	}
}

func TestUpdateArticle_StaleETagIsPreconditionFailed(t *testing.T) {
	fixture := newUpdateArticleFixture(t, activeUpdateArticle())
	response := fixture.serve(t, activeUpdatePrincipal(), "t", "b", `"article-`+updateArticleID+`-v1"`)
	if response.Code != http.StatusPreconditionFailed {
		t.Fatalf("status = %d, want 412; body = %s", response.Code, response.Body)
	}
	if fixture.repository.updateCalls != 0 {
		t.Fatalf("repository update calls = %d, want 0", fixture.repository.updateCalls)
	}
}

func TestUpdateArticle_MissingOrMalformedIfMatchFailsBeforeRepository(t *testing.T) {
	cases := []struct {
		name     string
		ifMatch  string
		wantHTTP int
	}{
		{"missing", "", http.StatusPreconditionRequired},
		{"weak", `W/"article-` + updateArticleID + `-v2"`, http.StatusBadRequest},
		{"multiple", `"article-` + updateArticleID + `-v2", "x"`, http.StatusBadRequest},
		{"wildcard", `*`, http.StatusBadRequest},
		{"other resource", `"article-` + updateOtherUserID + `-v2"`, http.StatusBadRequest},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fixture := newUpdateArticleFixture(t, activeUpdateArticle())
			response := fixture.serve(t, activeUpdatePrincipal(), "t", "b", test.ifMatch)
			if response.Code != test.wantHTTP {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantHTTP, response.Body)
			}
			if fixture.repository.updateCalls != 0 {
				t.Fatalf("repository update calls = %d, want 0", fixture.repository.updateCalls)
			}
		})
	}
}

func TestUpdateArticle_InvalidContentFailsBeforeRepository(t *testing.T) {
	cases := []struct {
		name  string
		title string
		body  string
	}{
		{"blank title", "   ", "body"},
		{"blank body", "title", "\t\n"},
		{"title too long", longUpdateString(maxArticleTitleRunes + 1), "body"},
		{"body too long", "title", longUpdateString(maxArticleBodyRunes + 1)},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fixture := newUpdateArticleFixture(t, activeUpdateArticle())
			response := fixture.serve(
				t, activeUpdatePrincipal(), test.title, test.body,
				`"article-`+updateArticleID+`-v2"`,
			)
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422; body = %s", response.Code, response.Body)
			}
			if fixture.repository.updateCalls != 0 {
				t.Fatalf("repository update calls = %d, want 0", fixture.repository.updateCalls)
			}
		})
	}
}

func TestUpdateArticle_DeletedOrPendingArticleIsUnavailable(t *testing.T) {
	existing := activeUpdateArticle()
	existing.DeletionState = DeletionPending
	existing.DeletionOperationID = func() *string { id := updateOtherUserID; return &id }()
	fixture := newUpdateArticleFixture(t, existing)
	response := fixture.serve(t, activeUpdatePrincipal(), "t", "b", `"article-`+updateArticleID+`-v2"`)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body = %s", response.Code, response.Body)
	}
	if fixture.repository.updateCalls != 0 {
		t.Fatalf("repository update calls = %d, want 0", fixture.repository.updateCalls)
	}
}

func TestUpdateArticle_RepositoryFailureRollsBack(t *testing.T) {
	fixture := newUpdateArticleFixture(t, activeUpdateArticle())
	fixture.repository.updateErr = errors.New("injected update failure")
	response := fixture.serve(t, activeUpdatePrincipal(), "t", "b", `"article-`+updateArticleID+`-v2"`)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body = %s", response.Code, response.Body)
	}
	if fixture.transaction.rollbackCalls == 0 {
		t.Fatal("repository failure did not roll back")
	}
}

func longUpdateString(runes int) string {
	buffer := make([]rune, runes)
	for index := range buffer {
		buffer[index] = 'a'
	}
	return string(buffer)
}

type updateArticleFixture struct {
	handler     *Handler
	service     *CommandService
	transactor  *updateArticleTransactorStub
	transaction *updateArticleTxStub
	repository  *updateArticleRepositoryStub
	existing    Article
}

func newUpdateArticleFixture(t *testing.T, existing Article) updateArticleFixture {
	t.Helper()
	responseLogger, err := NewSlogArticleResponseWriteLogger(
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		"han-server",
	)
	if err != nil {
		t.Fatalf("NewSlogArticleResponseWriteLogger() error = %v", err)
	}
	transaction := &updateArticleTxStub{}
	transactor := &updateArticleTransactorStub{transaction: transaction}
	repository := &updateArticleRepositoryStub{existing: existing}
	service, err := NewCommandService(CommandDependencies{
		Transactions: transactor,
		Repository:   repository,
		IDs:          createArticleIDStub{id: updateArticleID},
		Clock:        updateArticleClockStub{now: updateArticleNow},
	})
	if err != nil {
		t.Fatalf("NewCommandService() error = %v", err)
	}
	handler, err := NewHandler(service, responseLogger)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return updateArticleFixture{
		handler: handler, service: service, transactor: transactor,
		transaction: transaction, repository: repository, existing: existing,
	}
}

func (f updateArticleFixture) serve(
	t *testing.T,
	principal session.Principal,
	title string,
	body string,
	ifMatch string,
) *httptest.ResponseRecorder {
	t.Helper()
	return serveUpdateArticle(t, f.handler, principal, title, body, ifMatch)
}

func serveUpdateArticle(
	t *testing.T,
	handler *Handler,
	principal session.Principal,
	title string,
	body string,
	ifMatch string,
) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(CreateArticleInput{Title: title, Body: body})
	if err != nil {
		t.Fatalf("encode update payload: %v", err)
	}
	return serveUpdateArticleBody(t, handler, principal, payload, ifMatch)
}

func serveUpdateArticleBody(
	t *testing.T,
	handler *Handler,
	principal session.Principal,
	payload []byte,
	ifMatch string,
) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	authorizer, err := platform.NewHTTPAuthorizer(
		updateArticlePrincipalResolver{principal: principal},
		func() time.Time { return updateArticleNow },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	protected := authorizer.Protect(session.AccessGeneral, http.HandlerFunc(handler.UpdateArticle))
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/articles/"+updateArticleID,
		bytes.NewReader(payload),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(platform.CSRFHeaderName, "csrf")
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "session"})

	platform.TraceMiddleware(protected).ServeHTTP(response, request)
	return response
}

func activeUpdatePrincipal() session.Principal {
	return session.Principal{
		SessionID: "session",
		UserID:    updateArticleUserID,
		Role:      principalRoleMember,
		Status:    principalStatusActive,
	}
}

func activeUpdateArticle() Article {
	authorID := updateArticleUserID
	return Article{
		ID:               updateArticleID,
		AuthorID:         &authorID,
		Title:            "old title",
		Body:             "old body",
		PublicationState: PublicationPublished,
		ViolationState:   ViolationClear,
		DeletionState:    DeletionActive,
		Version:          2,
		CreatedAt:        updateArticleNow.Add(-time.Hour),
		UpdatedAt:        updateArticleNow.Add(-time.Hour),
	}
}

type updateArticlePrincipalResolver struct {
	principal session.Principal
}

func (r updateArticlePrincipalResolver) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return r.principal, nil
}

type updateArticleClockStub struct {
	now time.Time
}

func (c updateArticleClockStub) Now() time.Time { return c.now }

type updateArticleTransactorStub struct {
	transaction *updateArticleTxStub
	beginCalls  int
}

func (s *updateArticleTransactorStub) Begin(context.Context) (pgx.Tx, error) {
	s.beginCalls++
	return s.transaction, nil
}

type updateArticleTxStub struct {
	commitCalls   int
	rollbackCalls int
}

func (tx *updateArticleTxStub) Begin(context.Context) (pgx.Tx, error) { return tx, nil }
func (tx *updateArticleTxStub) Commit(context.Context) error {
	tx.commitCalls++
	return nil
}
func (tx *updateArticleTxStub) Rollback(context.Context) error {
	tx.rollbackCalls++
	return nil
}
func (tx *updateArticleTxStub) CopyFrom(
	context.Context, pgx.Identifier, []string, pgx.CopyFromSource,
) (int64, error) {
	return 0, nil
}
func (tx *updateArticleTxStub) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *updateArticleTxStub) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *updateArticleTxStub) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (tx *updateArticleTxStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (tx *updateArticleTxStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}
func (tx *updateArticleTxStub) QueryRow(context.Context, string, ...any) pgx.Row { return nil }
func (tx *updateArticleTxStub) Conn() *pgx.Conn                                  { return nil }

type updateArticleRepositoryStub struct {
	existing    Article
	updated     Article
	updateCalls int
	updateErr   error
}

func (s *updateArticleRepositoryStub) InsertArticle(context.Context, pgx.Tx, Article) error {
	return nil
}

func (s *updateArticleRepositoryStub) LockArticleForUpdate(
	context.Context, pgx.Tx, string,
) (Article, error) {
	if s.existing.ID == "" {
		return Article{}, pgx.ErrNoRows
	}
	return s.existing, nil
}

func (s *updateArticleRepositoryStub) UpdateArticleContent(
	_ context.Context, _ pgx.Tx, updated Article,
) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updateCalls++
	s.updated = updated
	return nil
}
