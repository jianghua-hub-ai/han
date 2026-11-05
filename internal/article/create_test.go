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
	"strings"
	"testing"
	"time"

	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	createArticleID     = "10000000-0000-4000-8000-000000000001"
	createArticleUserID = "20000000-0000-4000-8000-000000000002"
)

var createArticleNow = time.Date(2026, time.October, 23, 1, 2, 3, 0, time.UTC)

func TestCreateArticle_ActivePrincipalCreatesPublishedArticle(t *testing.T) {
	fixture := newCreateArticleFixture(t)
	title := "  <script>alert(1)</script> # heading  "
	body := "  **literal markdown**  "
	response := fixture.serve(
		t,
		activeCreateArticlePrincipal(),
		[]byte(`{"title":`+quotedJSON(t, title)+`,"body":`+quotedJSON(t, body)+`}`),
	)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body)
	}
	if got := response.Header().Get("Location"); got != "/api/v1/articles/"+createArticleID {
		t.Fatalf("Location = %q", got)
	}
	if got := response.Header().Get("ETag"); got != `"article-`+createArticleID+`-v1"` {
		t.Fatalf("ETag = %q", got)
	}
	if fixture.repository.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", fixture.repository.calls)
	}
	if fixture.transaction.commitCalls != 1 || fixture.transaction.rollbackCalls != 0 {
		t.Fatalf(
			"transaction commit/rollback calls = %d/%d, want 1/0",
			fixture.transaction.commitCalls,
			fixture.transaction.rollbackCalls,
		)
	}

	created := fixture.repository.article
	if created.AuthorID == nil || *created.AuthorID != createArticleUserID {
		t.Fatalf("author = %#v, want Principal user", created.AuthorID)
	}
	if created.Title != title || created.Body != body {
		t.Fatalf("content = %q/%q, want original %q/%q", created.Title, created.Body, title, body)
	}
	if created.PublicationState != PublicationPublished ||
		created.ViolationState != ViolationClear ||
		created.DeletionState != DeletionActive ||
		created.OrphanedFromUserID != nil ||
		created.ViolationCategory != nil ||
		created.DeletionOperationID != nil ||
		created.Version != 1 {
		t.Fatalf("created article state = %#v", created)
	}
	if !created.CreatedAt.Equal(createArticleNow) || !created.UpdatedAt.Equal(createArticleNow) {
		t.Fatalf("created timestamps = %s/%s", created.CreatedAt, created.UpdatedAt)
	}

	var payload MemberArticleDetailView
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != created.ID || payload.Title != title || payload.Body != body {
		t.Fatalf("response = %#v", payload)
	}
}

func TestCreateArticle_RestrictedPrincipalIsRejectedBeforeCreator(t *testing.T) {
	fixture := newCreateArticleFixture(t)
	principal := activeCreateArticlePrincipal()
	principal.MustChangePassword = true

	response := fixture.serve(t, principal, []byte(`{"title":"t","body":"b"}`))

	requireCreateArticleError(t, response, http.StatusForbidden, platform.CodeAuthMustCompleteOnboarding)
	if fixture.repository.calls != 0 || fixture.transactor.beginCalls != 0 {
		t.Fatalf(
			"repository/begin calls = %d/%d, want 0/0",
			fixture.repository.calls,
			fixture.transactor.beginCalls,
		)
	}
}

func TestCreateArticle_InvalidJSONNeverCallsCreator(t *testing.T) {
	valid := []byte(`{"title":"t","body":"b"}`)
	invalidUTF8 := append([]byte(`{"title":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`","body":"b"}`)...)
	tests := []struct {
		name string
		body []byte
	}{
		{name: "client author id", body: []byte(`{"title":"t","body":"b","author_id":"attacker"}`)},
		{name: "unknown field", body: []byte(`{"title":"t","body":"b","extra":true}`)},
		{name: "duplicate title", body: []byte(`{"title":"t","title":"other","body":"b"}`)},
		{name: "trailing value", body: []byte(`{"title":"t","body":"b"} {}`)},
		{name: "invalid utf8", body: invalidUTF8},
		{name: "wire size n plus one", body: padCreateArticleBody(t, valid, MaxArticleWriteBodyBytes+1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCreateArticleFixture(t)

			response := fixture.serve(t, activeCreateArticlePrincipal(), test.body)

			requireCreateArticleError(t, response, http.StatusBadRequest, platform.CodeRequestInvalidJSON)
			if fixture.repository.calls != 0 || fixture.transactor.beginCalls != 0 {
				t.Fatalf(
					"repository/begin calls = %d/%d, want 0/0",
					fixture.repository.calls,
					fixture.transactor.beginCalls,
				)
			}
		})
	}
}

func TestCreateArticle_ExactWireLimitContinuesToCreate(t *testing.T) {
	body := padCreateArticleBody(
		t,
		[]byte(`{"title":"t","body":"b"}`),
		MaxArticleWriteBodyBytes,
	)
	fixture := newCreateArticleFixture(t)

	response := fixture.serve(t, activeCreateArticlePrincipal(), body)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", response.Code, response.Body)
	}
	if fixture.repository.calls != 1 {
		t.Fatalf("repository calls = %d, want 1", fixture.repository.calls)
	}
	if fixture.repository.article.Title != "t" || fixture.repository.article.Body != "b" {
		t.Fatalf("repository article = %#v", fixture.repository.article)
	}
}

func TestCreateArticle_ContentValidationBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		body       string
		wantStatus int
		wantCode   platform.Code
	}{
		{name: "missing title", body: "b", wantStatus: http.StatusUnprocessableEntity, wantCode: CodeArticleValidationFailed},
		{name: "missing body", title: "t", wantStatus: http.StatusUnprocessableEntity, wantCode: CodeArticleValidationFailed},
		{name: "unicode whitespace title", title: "\u2003\u3000", body: "b", wantStatus: http.StatusUnprocessableEntity, wantCode: CodeArticleValidationFailed},
		{name: "unicode whitespace body", title: "t", body: "\u2003\u3000", wantStatus: http.StatusUnprocessableEntity, wantCode: CodeArticleValidationFailed},
		{name: "title 200 runes", title: strings.Repeat("界", 200), body: "b", wantStatus: http.StatusCreated},
		{name: "title 201 runes", title: strings.Repeat("界", 201), body: "b", wantStatus: http.StatusUnprocessableEntity, wantCode: CodeArticleValidationFailed},
		{name: "body 20000 runes", title: "t", body: strings.Repeat("界", 20000), wantStatus: http.StatusCreated},
		{name: "body 20001 runes", title: "t", body: strings.Repeat("界", 20001), wantStatus: http.StatusUnprocessableEntity, wantCode: CodeArticleValidationFailed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCreateArticleFixture(t)
			body := []byte(`{"title":` + quotedJSON(t, test.title) + `,"body":` + quotedJSON(t, test.body) + `}`)

			response := fixture.serve(t, activeCreateArticlePrincipal(), body)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body)
			}
			if test.wantCode != "" {
				requireCreateArticleError(t, response, test.wantStatus, test.wantCode)
				if fixture.repository.calls != 0 || fixture.transactor.beginCalls != 0 {
					t.Fatalf(
						"repository/begin calls = %d/%d, want 0/0",
						fixture.repository.calls,
						fixture.transactor.beginCalls,
					)
				}
				return
			}
			if fixture.repository.calls != 1 {
				t.Fatalf("repository calls = %d, want 1", fixture.repository.calls)
			}
			if fixture.repository.article.Title != test.title ||
				fixture.repository.article.Body != test.body {
				t.Fatal("valid content was not preserved")
			}
		})
	}
}

func TestCreateArticle_DatabaseFailuresRespectTransactionOutcome(t *testing.T) {
	injected := errors.New("database unavailable")
	tests := []struct {
		name              string
		insertErr         error
		commitErr         error
		wantCommitCalls   int
		wantRollbackCalls int
	}{
		{
			name: "insert", insertErr: injected,
			wantRollbackCalls: 1,
		},
		{
			name: "commit", commitErr: injected,
			wantCommitCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newCreateArticleFixture(t)
			fixture.repository.err = test.insertErr
			fixture.transaction.commitErr = test.commitErr

			response := fixture.serve(
				t,
				activeCreateArticlePrincipal(),
				[]byte(`{"title":"t","body":"b"}`),
			)

			requireCreateArticleError(
				t,
				response,
				http.StatusServiceUnavailable,
				platform.CodePlatformDatabaseUnavailable,
			)
			if fixture.transaction.commitCalls != test.wantCommitCalls {
				t.Fatalf(
					"commit calls = %d, want %d",
					fixture.transaction.commitCalls,
					test.wantCommitCalls,
				)
			}
			if fixture.transaction.rollbackCalls != test.wantRollbackCalls {
				t.Fatalf(
					"rollback calls = %d, want %d",
					fixture.transaction.rollbackCalls,
					test.wantRollbackCalls,
				)
			}
		})
	}
}

func TestCreateArticle_CommitFailurePreservesUnknownOutcome(t *testing.T) {
	fixture := newCreateArticleFixture(t)
	injected := errors.New("commit connection lost")
	fixture.transaction.commitErr = injected

	_, err := fixture.service.Create(
		context.Background(),
		activeCreateArticlePrincipal(),
		CreateArticleInput{Title: "t", Body: "b"},
	)

	if !errors.Is(err, errCreateArticleCommitOutcomeUnknown) {
		t.Fatalf("Create() error = %v, want commit outcome unknown", err)
	}
	if !errors.Is(err, injected) {
		t.Fatalf("Create() error = %v, want injected cause", err)
	}
	if got := platform.PublicCode(err); got != platform.CodePlatformDatabaseUnavailable {
		t.Fatalf("public code = %q, want %q", got, platform.CodePlatformDatabaseUnavailable)
	}
	if fixture.transaction.commitCalls != 1 || fixture.transaction.rollbackCalls != 0 {
		t.Fatalf(
			"transaction commit/rollback calls = %d/%d, want 1/0",
			fixture.transaction.commitCalls,
			fixture.transaction.rollbackCalls,
		)
	}
	if !fixture.transaction.closed {
		t.Fatal("transaction remains open after Commit returned")
	}
}

func TestCreateArticle_ResponseWriteFailureIsObservedWithoutSecondResponse(t *testing.T) {
	const (
		title = "title-must-not-appear-in-log"
		body  = "body-must-not-appear-in-log"
	)
	var logs bytes.Buffer
	responseLogger, err := NewSlogArticleResponseWriteLogger(
		slog.New(slog.NewJSONHandler(&logs, nil)),
		"han-server",
	)
	if err != nil {
		t.Fatalf("NewSlogArticleResponseWriteLogger() error = %v", err)
	}
	fixture := newCreateArticleFixtureWithResponseLogger(t, responseLogger)
	response := &createArticlePartialResponseWriter{
		err: errors.New("simulated partial response write"),
	}

	serveCreateArticleTo(
		t,
		fixture.handler,
		activeCreateArticlePrincipal(),
		[]byte(`{"title":`+quotedJSON(t, title)+`,"body":`+quotedJSON(t, body)+`}`),
		response,
	)

	if len(response.statuses) != 1 || response.statuses[0] != http.StatusCreated {
		t.Fatalf("response statuses = %v, want [201]", response.statuses)
	}
	if response.writeCalls != 1 {
		t.Fatalf("response body write calls = %d, want 1", response.writeCalls)
	}
	if strings.Contains(response.body.String(), `"code"`) {
		t.Fatalf("response contains a second error envelope: %q", response.body.String())
	}

	var record map[string]any
	decoder := json.NewDecoder(&logs)
	if err := decoder.Decode(&record); err != nil {
		t.Fatalf("decode response write log: %v; log = %q", err, logs.String())
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Fatalf("response write log count exceeds one: %q", logs.String())
	}
	traceID := response.Header().Get(platform.TraceHeader)
	for field, want := range map[string]any{
		"level":         "ERROR",
		"service":       "han-server",
		"module":        "article",
		"trace_id":      traceID,
		"event":         createArticleResponseWriteFailureEvent,
		"endpoint":      createArticleEndpoint,
		"status":        float64(http.StatusCreated),
		"reason_bucket": createArticleResponseWriteFailureReason,
		"error_code":    string(platform.CodePlatformInternalError),
	} {
		if got := record[field]; got != want {
			t.Errorf("log field %q = %#v, want %#v; log = %q", field, got, want, logs.String())
		}
	}
	if traceID == "" {
		t.Fatal("response write failure log has no request trace")
	}
	if got, _ := record["error"].(string); !strings.Contains(got, response.err.Error()) {
		t.Errorf("log error = %q, want original write error", got)
	}
	if strings.Contains(logs.String(), title) || strings.Contains(logs.String(), body) {
		t.Fatalf("response write log contains article content: %q", logs.String())
	}
}

type createArticleFixture struct {
	handler     *Handler
	service     *CommandService
	transactor  *createArticleTransactorStub
	transaction *createArticleTxStub
	repository  *createArticleRepositoryStub
}

func newCreateArticleFixture(t *testing.T) createArticleFixture {
	t.Helper()
	responseLogger, err := NewSlogArticleResponseWriteLogger(
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		"han-server",
	)
	if err != nil {
		t.Fatalf("NewSlogArticleResponseWriteLogger() error = %v", err)
	}
	return newCreateArticleFixtureWithResponseLogger(t, responseLogger)
}

func newCreateArticleFixtureWithResponseLogger(
	t *testing.T,
	responseLogger ArticleResponseWriteLogger,
) createArticleFixture {
	t.Helper()
	transaction := &createArticleTxStub{}
	transactor := &createArticleTransactorStub{transaction: transaction}
	repository := &createArticleRepositoryStub{}
	service, err := NewCommandService(CommandDependencies{
		Transactions: transactor,
		Repository:   repository,
		IDs:          createArticleIDStub{id: createArticleID},
		Clock:        createArticleClockStub{now: createArticleNow},
	})
	if err != nil {
		t.Fatalf("NewCommandService() error = %v", err)
	}
	handler, err := NewHandler(service, responseLogger)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return createArticleFixture{
		handler: handler, service: service, transactor: transactor,
		transaction: transaction, repository: repository,
	}
}

func (f createArticleFixture) serve(
	t *testing.T,
	principal session.Principal,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	return serveCreateArticle(t, f.handler, principal, body)
}

func serveCreateArticle(
	t *testing.T,
	handler *Handler,
	principal session.Principal,
	body []byte,
) *httptest.ResponseRecorder {
	t.Helper()
	response := httptest.NewRecorder()
	serveCreateArticleTo(t, handler, principal, body, response)
	return response
}

func serveCreateArticleTo(
	t *testing.T,
	handler *Handler,
	principal session.Principal,
	body []byte,
	response http.ResponseWriter,
) {
	t.Helper()
	authorizer, err := platform.NewHTTPAuthorizer(
		createArticlePrincipalResolver{principal: principal},
		func() time.Time { return createArticleNow },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	protected := authorizer.Protect(session.AccessGeneral, http.HandlerFunc(handler.Create))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/articles", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(platform.CSRFHeaderName, "csrf")
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "session"})

	platform.TraceMiddleware(protected).ServeHTTP(response, request)
}

func activeCreateArticlePrincipal() session.Principal {
	return session.Principal{
		SessionID: "session",
		UserID:    createArticleUserID,
		Role:      "member",
		Status:    "active",
	}
}

func quotedJSON(t *testing.T, value string) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode JSON string: %v", err)
	}
	return string(encoded)
}

func padCreateArticleBody(t *testing.T, body []byte, size int) []byte {
	t.Helper()
	if len(body) > size {
		t.Fatalf("fixture body length = %d, exceeds %d", len(body), size)
	}
	padded := make([]byte, size)
	copy(padded, body)
	for index := len(body); index < len(padded); index++ {
		padded[index] = ' '
	}
	return padded
}

func requireCreateArticleError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	status int,
	code platform.Code,
) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body)
	}
	var envelope platform.Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Code != code {
		t.Fatalf("error code = %q, want %q", envelope.Code, code)
	}
}

type createArticlePrincipalResolver struct {
	principal session.Principal
}

func (r createArticlePrincipalResolver) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return r.principal, nil
}

type createArticleTransactorStub struct {
	transaction *createArticleTxStub
	err         error
	beginCalls  int
}

func (s *createArticleTransactorStub) Begin(context.Context) (pgx.Tx, error) {
	s.beginCalls++
	if s.err != nil {
		return nil, s.err
	}
	return s.transaction, nil
}

type createArticleRepositoryStub struct {
	article Article
	err     error
	calls   int
}

func (s *createArticleRepositoryStub) InsertArticle(
	_ context.Context,
	_ pgx.Tx,
	article Article,
) error {
	s.calls++
	s.article = article
	return s.err
}

type createArticleIDStub struct {
	id  string
	err error
}

func (s createArticleIDStub) NewID() (string, error) {
	return s.id, s.err
}

type createArticleClockStub struct {
	now time.Time
}

func (s createArticleClockStub) Now() time.Time {
	return s.now
}

type createArticleTxStub struct {
	commitErr     error
	rollbackErr   error
	commitCalls   int
	rollbackCalls int
	closed        bool
}

func (tx *createArticleTxStub) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("nested transaction is not supported")
}

func (tx *createArticleTxStub) Commit(context.Context) error {
	tx.commitCalls++
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.closed = true
	return tx.commitErr
}

func (tx *createArticleTxStub) Rollback(context.Context) error {
	tx.rollbackCalls++
	if tx.closed {
		return pgx.ErrTxClosed
	}
	tx.closed = true
	return tx.rollbackErr
}

func (tx *createArticleTxStub) CopyFrom(
	context.Context,
	pgx.Identifier,
	[]string,
	pgx.CopyFromSource,
) (int64, error) {
	return 0, errors.New("not implemented")
}

func (tx *createArticleTxStub) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *createArticleTxStub) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *createArticleTxStub) Prepare(
	context.Context,
	string,
	string,
) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}

func (tx *createArticleTxStub) Exec(
	context.Context,
	string,
	...any,
) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("not implemented")
}

func (tx *createArticleTxStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func (tx *createArticleTxStub) QueryRow(context.Context, string, ...any) pgx.Row {
	return createArticleRowStub{}
}

func (tx *createArticleTxStub) Conn() *pgx.Conn {
	return nil
}

type createArticleRowStub struct{}

func (createArticleRowStub) Scan(...any) error {
	return errors.New("not implemented")
}

type createArticlePartialResponseWriter struct {
	header     http.Header
	statuses   []int
	body       bytes.Buffer
	err        error
	writeCalls int
}

func (w *createArticlePartialResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *createArticlePartialResponseWriter) WriteHeader(status int) {
	w.statuses = append(w.statuses, status)
}

func (w *createArticlePartialResponseWriter) Write(body []byte) (int, error) {
	w.writeCalls++
	written := len(body) / 2
	_, _ = w.body.Write(body[:written])
	return written, w.err
}
