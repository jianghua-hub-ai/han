package article

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type moderationFixture struct {
	service *CommandService
	handler *Handler
	store   *moderationStore
}

type moderationStore struct {
	stored      Article
	pending     Article
	stagedAudit []any
	audits      [][]any
	events      []string
	failure     string
	status      string
	absent      bool
	transaction *moderationTransaction
}

type moderationTransaction struct {
	pgx.Tx
	store *moderationStore
}

func newModerationFixture(t *testing.T, existing Article) moderationFixture {
	t.Helper()
	store := &moderationStore{stored: existing, status: authorStatusActive}
	service, err := NewCommandService(CommandDependencies{
		Transactions: store, Repository: Repository{},
		IDs: createArticleIDStub{id: deletionAuditID}, Clock: createArticleClockStub{now: updateArticleNow},
		Moderation: &ModerationDependencies{Auditor: audit.Store{}, AuthorStates: store.authorStates},
	})
	if err != nil {
		t.Fatal(err)
	}
	logger, err := NewSlogArticleResponseWriteLogger(slog.New(slog.NewJSONHandler(io.Discard, nil)), "han-server")
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(service, logger)
	if err != nil {
		t.Fatal(err)
	}
	return moderationFixture{service: service, handler: handler, store: store}
}

func (store *moderationStore) Begin(context.Context) (pgx.Tx, error) {
	store.events = append(store.events, "begin")
	if store.failure == "begin" {
		return nil, errors.New("synthetic begin failure")
	}
	store.pending, store.stagedAudit = store.stored, nil
	store.transaction = &moderationTransaction{store: store}
	return store.transaction, nil
}

func (transaction *moderationTransaction) QueryRow(_ context.Context, query string, arguments ...any) pgx.Row {
	transaction.store.events = append(transaction.store.events, "lock")
	if !strings.Contains(query, "FOR UPDATE") || arguments[0] != transaction.store.stored.ID {
		return moderationRow{err: errors.New("article must be locked by ID")}
	}
	if transaction.store.absent {
		return moderationRow{err: pgx.ErrNoRows}
	}
	if transaction.store.failure == "lock" {
		return moderationRow{err: errors.New("synthetic lock failure")}
	}
	return moderationRow{article: transaction.store.stored}
}

type moderationRow struct {
	article Article
	err     error
}

func (row moderationRow) Scan(destinations ...any) error {
	if row.err != nil {
		return row.err
	}
	article := row.article
	values := []any{article.ID, article.AuthorID, article.OrphanedFromUserID, article.Title, article.Body,
		article.PublicationState, article.ViolationState, article.ViolationCategory, article.DeletionState,
		article.DeletionOperationID, article.Version, article.CreatedAt, article.UpdatedAt}
	if len(destinations) != len(values) {
		return errors.New("unexpected article scan")
	}
	for index, value := range values {
		reflect.ValueOf(destinations[index]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

func (transaction *moderationTransaction) Exec(_ context.Context, query string, arguments ...any) (pgconn.CommandTag, error) {
	store := transaction.store
	if strings.Contains(query, "UPDATE articles") {
		store.events = append(store.events, "update")
		if store.failure == "update" {
			return pgconn.CommandTag{}, errors.New("synthetic update failure")
		}
		if store.failure == "compare" {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		}
		normalized := strings.Join(strings.Fields(query), " ")
		if !strings.Contains(normalized, "SET violation_state = $2, violation_category = $3, version = $4, updated_at = $5 WHERE") ||
			!strings.Contains(normalized, "version = $4 - 1") || !strings.Contains(normalized, "deletion_state = 'active'") {
			return pgconn.CommandTag{}, errors.New("update exceeded moderation fields or lacked guards")
		}
		store.pending.ViolationState = arguments[1].(ViolationState)
		store.pending.ViolationCategory = arguments[2].(*ViolationCategory)
		store.pending.Version = arguments[3].(int64)
		store.pending.UpdatedAt = arguments[4].(time.Time)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	if strings.Contains(query, "INSERT INTO audit_events") {
		store.events = append(store.events, "audit")
		if store.failure == "audit" {
			return pgconn.CommandTag{}, errors.New("synthetic audit failure")
		}
		store.stagedAudit = arguments
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	return pgconn.CommandTag{}, errors.New("unexpected transaction write")
}

func (transaction *moderationTransaction) Commit(context.Context) error {
	store := transaction.store
	store.events = append(store.events, "commit")
	if store.failure == "commit" {
		return errors.New("synthetic commit failure")
	}
	store.stored = store.pending
	store.audits = append(store.audits, store.stagedAudit)
	return nil
}

func (transaction *moderationTransaction) Rollback(context.Context) error {
	transaction.store.events = append(transaction.store.events, "rollback")
	if transaction.store.failure == "rollback" {
		return errors.New("synthetic rollback failure")
	}
	return nil
}

func (store *moderationStore) authorStates(_ context.Context, transaction pgx.Tx, authorIDs []string) (map[string]string, error) {
	store.events = append(store.events, "reader")
	if transaction != store.transaction {
		return nil, errors.New("author reader must share locked article transaction")
	}
	if store.failure == "reader" {
		return nil, errors.New("synthetic reader failure")
	}
	if len(authorIDs) != 1 || authorIDs[0] != *store.stored.AuthorID {
		return nil, errors.New("wrong author IDs")
	}
	return map[string]string{authorIDs[0]: store.status}, nil
}

func moderationContext() context.Context {
	ctx, _ := platform.ContextWithTraceID(context.Background(), deletionTraceID)
	return ctx
}

func (fixture moderationFixture) serve(t *testing.T, principal session.Principal, action, body string, etags ...string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/articles/"+fixture.store.stored.ID+"/"+action, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	for _, etag := range etags {
		request.Header.Add("If-Match", etag)
	}
	request.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
	authorizer, err := platform.NewHTTPAuthorizer(createArticlePrincipalResolver{principal: principal}, func() time.Time { return updateArticleNow })
	if err != nil {
		t.Fatal(err)
	}
	handler := fixture.handler.MarkViolation
	if action == "restore" {
		handler = fixture.handler.RestoreArticle
	}
	response := httptest.NewRecorder()
	platform.TraceMiddleware(authorizer.Protect(session.AccessGeneral, http.HandlerFunc(handler))).ServeHTTP(response, request)
	return response
}

func TestModeration_Mark_CategoriesAndViews(t *testing.T) {
	for _, category := range []ViolationCategory{ViolationViolence, ViolationSexual} {
		t.Run(string(category), func(t *testing.T) {
			original := activeUpdateArticle()
			fixture := newModerationFixture(t, original)
			etag, _ := StrongETag(original)
			response := fixture.serve(t, deletionAdminPrincipal(), "violation", fmt.Sprintf(`{"category":%q}`, category), etag)
			if response.Code != http.StatusOK {
				t.Fatalf("mark = %d %s", response.Code, response.Body)
			}
			expected := original
			expected.ViolationState, expected.ViolationCategory = ViolationViolating, &category
			expected.Version++
			expected.UpdatedAt = updateArticleNow
			requireModerationResult(t, fixture, response, expected, "article.violation.mark", category)
			if (VisibilityPolicy{}).MemberVisible(fixture.store.stored, authorStatusActive) {
				t.Fatal("member still sees violating article")
			}
			if _, err := (VisibilityPolicy{}).MemberDetailProjection(fixture.store.stored, authorStatusActive); !errors.Is(err, ErrArticleUnavailable) {
				t.Fatalf("member result = %v", err)
			}
			author, err := (VisibilityPolicy{}).AuthorProjection(fixture.store.stored, activeUpdatePrincipal())
			if err != nil || author.ModerationStatus != ModerationRestricted || !author.Editable {
				t.Fatalf("author = %#v, %v", author, err)
			}
			if !reflect.DeepEqual(fixture.store.events, []string{"begin", "lock", "reader", "update", "audit", "commit"}) {
				t.Fatalf("transaction order = %v", fixture.store.events)
			}
		})
	}
}

func requireModerationResult(t *testing.T, fixture moderationFixture, response *httptest.ResponseRecorder, expected Article, action string, category ViolationCategory) {
	t.Helper()
	if !reflect.DeepEqual(fixture.store.stored, expected) {
		t.Fatalf("persisted state = %#v, want %#v", fixture.store.stored, expected)
	}
	etag, _ := StrongETag(expected)
	if response.Header().Get("ETag") != etag {
		t.Fatalf("etag = %q, want %q", response.Header().Get("ETag"), etag)
	}
	var payload AdminArticleView
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	want, err := (VisibilityPolicy{}).AdminProjection(expected, deletionAdminPrincipal(), fixture.store.status)
	if err != nil || !reflect.DeepEqual(payload, want) {
		t.Fatalf("admin projection = %#v, want %#v, %v", payload, want, err)
	}
	if len(fixture.store.audits) != 1 {
		t.Fatalf("audits = %d", len(fixture.store.audits))
	}
	event := fixture.store.audits[0]
	if event[2] != audit.ActorUser || *event[3].(*string) != deletionAdminPrincipal().UserID ||
		*event[5].(*string) != principalRoleAdmin || event[6] != action || event[7] != "article" ||
		event[8] != expected.ID || event[9] != "succeeded" || event[11] == "" {
		t.Fatalf("audit = %#v", event)
	}
	var metadata map[string]any
	if err := json.Unmarshal(event[13].([]byte), &metadata); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(metadata, map[string]any{"category": string(category), "article_version": float64(expected.Version)}) {
		t.Fatalf("audit metadata = %#v", metadata)
	}
}

func violatingModerationArticle() Article {
	article := activeUpdateArticle()
	category := ViolationSexual
	article.ViolationState, article.ViolationCategory = ViolationViolating, &category
	return article
}

func TestModeration_Restore_RetainsOtherConstraints(t *testing.T) {
	for _, variant := range []string{"active", "suspended", "orphan", "anonymous"} {
		t.Run(variant, func(t *testing.T) {
			original := violatingModerationArticle()
			if variant == "orphan" {
				original.AuthorID, original.OrphanedFromUserID = nil, stringPointer(updateArticleUserID)
				original.PublicationState = PublicationHeld
			}
			if variant == "anonymous" {
				original.AuthorID = nil
			}
			fixture := newModerationFixture(t, original)
			if variant == "suspended" {
				fixture.store.status = authorStatusSuspended
			}
			etag, _ := StrongETag(original)
			response := fixture.serve(t, deletionAdminPrincipal(), "restore", "", etag)
			if response.Code != http.StatusOK {
				t.Fatalf("restore = %d %s", response.Code, response.Body)
			}
			expected := original
			expected.ViolationState, expected.ViolationCategory = ViolationClear, nil
			expected.Version++
			expected.UpdatedAt = updateArticleNow
			requireModerationResult(t, fixture, response, expected, "article.violation.clear", ViolationSexual)
			visible := (VisibilityPolicy{}).MemberVisible(fixture.store.stored, fixture.store.status)
			if visible != (variant == "active" || variant == "anonymous") {
				t.Fatalf("visibility = %v for %s", visible, variant)
			}
		})
	}
}

func TestModeration_StateAndVersion_RejectsWithoutWrite(t *testing.T) {
	for _, variant := range []string{"repeat mark", "clear restore", "stale", "stale repeat", "pending", "absent", "invalid state"} {
		t.Run(variant, func(t *testing.T) {
			original := activeUpdateArticle()
			action, status := "violation", http.StatusConflict
			if variant == "repeat mark" || variant == "stale repeat" {
				original = violatingModerationArticle()
			}
			if variant == "clear restore" {
				action = "restore"
			}
			if variant == "pending" {
				original.DeletionState, original.DeletionOperationID = DeletionPending, stringPointer(deletionOperationID)
				status = http.StatusNotFound
			}
			if variant == "invalid state" {
				original.PublicationState = PublicationHeld
			}
			fixture := newModerationFixture(t, original)
			if variant == "absent" {
				fixture.store.absent, status = true, http.StatusNotFound
			}
			etag, _ := StrongETag(activeUpdateArticle())
			if strings.HasPrefix(variant, "stale") {
				etag, status = strings.Replace(etag, "v2", "v1", 1), http.StatusPreconditionFailed
			}
			body := `{"category":"violence"}`
			if action == "restore" {
				body = ""
			}
			response := fixture.serve(t, deletionAdminPrincipal(), action, body, etag)
			if response.Code != status {
				t.Fatalf("response = %d %s, want %d", response.Code, response.Body, status)
			}
			if !reflect.DeepEqual(fixture.store.stored, original) || len(fixture.store.audits) != 0 ||
				!reflect.DeepEqual(fixture.store.events, []string{"begin", "lock", "rollback"}) {
				t.Fatalf("rejection mutated state: %v", fixture.store.events)
			}
		})
	}
}

func TestModeration_Failures_RollBackArticleAndAudit(t *testing.T) {
	for _, failure := range []string{"begin", "lock", "reader", "update", "compare", "audit", "commit", "rollback", "missing author", "audit ID", "clock", "trace"} {
		t.Run(failure, func(t *testing.T) {
			original := activeUpdateArticle()
			fixture := newModerationFixture(t, original)
			fixture.store.failure = failure
			if failure == "missing author" {
				fixture.store.status = ""
			}
			if failure == "audit ID" {
				fixture.service.dependencies.IDs = createArticleIDStub{err: errors.New("synthetic audit ID failure")}
			}
			if failure == "clock" {
				fixture.service.dependencies.Clock = createArticleClockStub{}
			}
			if failure == "rollback" {
				fixture.store.status = ""
			}
			ctx := moderationContext()
			if failure == "trace" {
				ctx = context.Background()
			}
			category := ViolationViolence
			payload, etag, err := fixture.service.Moderate(ctx, deletionAdminPrincipal(), original.ID, original.Version, &category)
			if err == nil || etag != "" || !reflect.DeepEqual(payload, AdminArticleView{}) {
				t.Fatalf("failure reported success: %#v %q %v", payload, etag, err)
			}
			if !reflect.DeepEqual(fixture.store.stored, original) || len(fixture.store.audits) != 0 {
				t.Fatal("failure committed partial article or audit")
			}
			if failure != "begin" && failure != "commit" && fixture.store.events[len(fixture.store.events)-1] != "rollback" {
				t.Fatalf("transaction leaked: %v", fixture.store.events)
			}
			if failure != "commit" && strings.Contains(strings.Join(fixture.store.events, ","), "commit") {
				t.Fatalf("failed command called commit: %v", fixture.store.events)
			}
		})
	}
}

func TestModeration_HTTP_InvalidInputNeverBeginsTransaction(t *testing.T) {
	tests := []struct {
		name, action, body string
		etags              []string
		status             int
	}{
		{"missing etag", "violation", `{"category":"violence"}`, nil, 428},
		{"weak etag", "violation", `{"category":"violence"}`, []string{`W/"article-` + updateArticleID + `-v2"`}, 400},
		{"wildcard", "restore", "", []string{"*"}, 400},
		{"duplicate etag", "restore", "", []string{`"article-` + updateArticleID + `-v2"`, `"article-` + updateArticleID + `-v2"`}, 400},
		{"other resource etag", "restore", "", []string{`"article-` + updateOtherUserID + `-v2"`}, 412},
		{"invalid category", "violation", `{"category":"other"}`, nil, 422},
		{"missing category", "violation", `{}`, nil, 422},
		{"null category", "violation", `{"category":null}`, nil, 422},
		{"unknown field", "violation", `{"category":"violence","extra":true}`, nil, 400},
		{"wrong case field", "violation", `{"Category":"violence"}`, nil, 400},
		{"wrong type", "violation", `{"category":3}`, nil, 400},
		{"duplicate field", "violation", `{"category":"violence","category":"sexual"}`, nil, 400},
		{"trailing object", "violation", `{"category":"violence"}{}`, nil, 400},
		{"not object", "violation", `[]`, nil, 400},
		{"restore object", "restore", `{}`, nil, 400},
		{"restore whitespace", "restore", " ", nil, 400},
		{"invalid utf8", "violation", "{\"category\":\"\xff\"}", nil, 400},
		{"oversize body", "violation", strings.Repeat(" ", MaxArticleWriteBodyBytes+1), nil, 400},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newModerationFixture(t, activeUpdateArticle())
			etags := test.etags
			if etags == nil && test.name != "missing etag" {
				etags = []string{`"article-` + updateArticleID + `-v2"`}
			}
			response := fixture.serve(t, deletionAdminPrincipal(), test.action, test.body, etags...)
			if response.Code != test.status {
				t.Fatalf("status = %d %s, want %d", response.Code, response.Body, test.status)
			}
			if len(fixture.store.events) != 0 {
				t.Fatalf("invalid input reached repository: %v", fixture.store.events)
			}
		})
	}
}

func TestModeration_Command_AuthorizationAndCategoryFailBeforeBegin(t *testing.T) {
	for _, variant := range []string{"member", "suspended", "password restricted", "privacy restricted", "invalid category"} {
		t.Run(variant, func(t *testing.T) {
			fixture := newModerationFixture(t, activeUpdateArticle())
			principal := deletionAdminPrincipal()
			category := ViolationViolence
			want := ErrArticleForbidden
			switch variant {
			case "member":
				principal.Role = principalRoleMember
			case "suspended":
				principal.Status = authorStatusSuspended
			case "password restricted":
				principal.MustChangePassword = true
			case "privacy restricted":
				principal.MustAcknowledgePrivacy = true
			case "invalid category":
				category, want = ViolationCategory("unknown"), ErrArticleValidationFailed
			}
			_, _, err := fixture.service.Moderate(moderationContext(), principal, updateArticleID, 2, &category)
			if !errors.Is(err, want) || len(fixture.store.events) != 0 {
				t.Fatalf("error/events = %v/%v", err, fixture.store.events)
			}
		})
	}
}

func TestModeration_HTTP_CompareFailureIsPreconditionFailed(t *testing.T) {
	fixture := newModerationFixture(t, activeUpdateArticle())
	fixture.store.failure = "compare"
	response := fixture.serve(t, deletionAdminPrincipal(), "violation", `{"category":"violence"}`, `"article-`+updateArticleID+`-v2"`)
	if response.Code != http.StatusPreconditionFailed {
		t.Fatalf("compare response = %d %s, want 412", response.Code, response.Body)
	}
}

func TestModeration_DeleteViolating_UsesSharedFinalizer(t *testing.T) {
	for _, variant := range []string{"success", "stale", "clear", "audit fails"} {
		t.Run(variant, func(t *testing.T) {
			fixture := newPermanentDeletionFixture(t)
			category := ViolationViolence
			if variant != "clear" {
				fixture.repository.article.ViolationState, fixture.repository.article.ViolationCategory = ViolationViolating, &category
			}
			if variant == "audit fails" {
				fixture.realFinalizer.dependencies.Auditor = moderationRejectingAuditor{}
				fixture.realFinalizer.dependencies.Transactions = moderationDeletionTransactor{fixture: fixture}
			}
			etag := deletionRequest(DeleteViolating).IfMatch
			status := http.StatusNoContent
			if variant == "stale" {
				etag, status = strings.Replace(etag, "v3", "v2", 1), http.StatusPreconditionFailed
			}
			if variant == "clear" {
				status = http.StatusConflict
			}
			if variant == "audit fails" {
				status = http.StatusServiceUnavailable
			}
			request := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/articles/"+deletionArticleID, nil)
			request.Header.Set("If-Match", etag)
			response := httptest.NewRecorder()
			fixture.serve(t, fixture.handler(t).DeleteViolating, deletionAdminPrincipal(), response, request)
			if response.Code != status {
				t.Fatalf("delete = %d %s, want %d", response.Code, response.Body, status)
			}
			if variant == "success" {
				if fixture.repository.article != nil || len(fixture.journal.records) != 1 || len(fixture.auditor.events) != 1 || fixture.auditor.events[0].Action != "article.violation.delete" {
					t.Fatal("delete bypassed shared journal/finalizer audit")
				}
			} else if variant == "audit fails" {
				if fixture.repository.article == nil || fixture.repository.article.DeletionState != DeletionPending || fixture.repository.operation.Status == DeletionOperationCompleted || len(fixture.auditor.events) != 0 {
					t.Fatal("audit failure committed physical deletion")
				}
			} else {
				requireAdminDeletionNotStarted(t, fixture)
			}
		})
	}
}

type moderationRejectingAuditor struct{}

func TestModeration_AnonymousMarkThenDelete_SharedFinalizerCompletes(t *testing.T) {
	for _, category := range []ViolationCategory{ViolationViolence, ViolationSexual} {
		t.Run(string(category), func(t *testing.T) {
			moderation, deletion, etag := newAnonymousModerationDeletion(t, category)
			version := moderation.store.stored.Version
			response := serveModerationDeletion(t, deletion, moderation.handler.DeleteViolating, etag)
			if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
				t.Fatalf("anonymous clear -> mark -> DELETE with new ETag = %d %s, want empty 204", response.Code, response.Body)
			}
			requireAnonymousModerationDeletionCompleted(t, deletion, version)
		})
	}
}

func newAnonymousModerationDeletion(t *testing.T, category ViolationCategory) (moderationFixture, permanentDeletionFixture, string) {
	t.Helper()
	original := deletionTestArticle()
	original.AuthorID = nil
	moderation := newModerationFixture(t, original)
	etag, err := StrongETag(original)
	if err != nil {
		t.Fatal(err)
	}
	response := moderation.serve(t, deletionAdminPrincipal(), "violation", fmt.Sprintf(`{"category":%q}`, category), etag)
	if response.Code != http.StatusOK || response.Header().Get("ETag") == etag {
		t.Fatalf("anonymous mark = %d %s, ETag = %q", response.Code, response.Body, response.Header().Get("ETag"))
	}
	marked := moderation.store.stored
	if marked.AuthorID != nil || marked.OrphanedFromUserID != nil || marked.PublicationState != PublicationPublished ||
		marked.ViolationState != ViolationViolating || marked.ViolationCategory == nil || *marked.ViolationCategory != category {
		t.Fatalf("mark changed anonymous ownership/publication: %#v", marked)
	}
	deletion := newPermanentDeletionFixture(t)
	deletion.repository.article = &marked
	moderation.service.dependencies.Deletion = deletion.coordinator
	moderation.handler.deleter = moderation.service
	return moderation, deletion, response.Header().Get("ETag")
}

func serveModerationDeletion(t *testing.T, fixture permanentDeletionFixture, handler http.HandlerFunc, etag string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/articles/"+deletionArticleID, nil)
	request.Header.Set("If-Match", etag)
	response := httptest.NewRecorder()
	fixture.serve(t, handler, deletionAdminPrincipal(), response, request)
	return response
}

func requireAnonymousModerationDeletionCompleted(t *testing.T, fixture permanentDeletionFixture, version int64) {
	t.Helper()
	operation := fixture.repository.operation
	if fixture.repository.article != nil || operation == nil || operation.Status != DeletionOperationCompleted ||
		operation.AuditEventID == nil || len(fixture.journal.records) != 1 || len(fixture.auditor.events) != 1 {
		t.Fatal("204 requires one shared tombstone, physical deletion and completion audit")
	}
	tombstone := fixture.journal.records[0]
	if tombstone.ArticleID != deletionArticleID || tombstone.ArticleVersion != version || tombstone.DeletionOperationID != operation.ID {
		t.Fatalf("tombstone = %#v", tombstone)
	}
	event := fixture.auditor.events[0]
	if event.Action != "article.violation.delete" || event.TargetID != deletionArticleID || event.ID != *operation.AuditEventID ||
		event.ActorUserID == nil || *event.ActorUserID != deletionAdminID || event.ActorRole == nil || *event.ActorRole != principalRoleAdmin ||
		!reflect.DeepEqual(event.Metadata, map[string]any{"article_version": version, "delete_kind": DeleteViolating}) {
		t.Fatalf("shared violation deletion audit = %#v", event)
	}
}

func TestModeration_AnonymousDelete_StaleAndRestoredClearAreRejected(t *testing.T) {
	moderation, deletion, etag := newAnonymousModerationDeletion(t, ViolationViolence)
	response := serveModerationDeletion(t, deletion, moderation.handler.DeleteViolating, deletionRequest(DeleteViolating).IfMatch)
	requireAdminDeleteError(t, response, http.StatusPreconditionFailed, string(CodeArticlePreconditionFailed))
	requireAdminDeletionNotStarted(t, deletion)
	response = moderation.serve(t, deletionAdminPrincipal(), "restore", "", etag)
	if response.Code != http.StatusOK {
		t.Fatalf("restore anonymous article = %d %s", response.Code, response.Body)
	}
	restored := moderation.store.stored
	deletion.repository.article = &restored
	response = serveModerationDeletion(t, deletion, moderation.handler.DeleteViolating, response.Header().Get("ETag"))
	requireAdminDeleteError(t, response, http.StatusConflict, string(CodeArticleInvalidState))
	requireAdminDeletionNotStarted(t, deletion)
}

func TestModeration_AnonymousDelete_PendingHiddenAndRetryConverges(t *testing.T) {
	for _, failure := range []string{"journal", "audit"} {
		t.Run(failure, func(t *testing.T) {
			moderation, deletion, etag := newAnonymousModerationDeletion(t, ViolationSexual)
			version := moderation.store.stored.Version
			if failure == "journal" {
				deletion.journal.appendErr = errors.New("synthetic anonymous journal failure")
			} else {
				deletion.realFinalizer.dependencies.Auditor = moderationRejectingAuditor{}
				deletion.realFinalizer.dependencies.Transactions = moderationDeletionTransactor{fixture: deletion}
			}
			response := serveModerationDeletion(t, deletion, moderation.handler.DeleteViolating, etag)
			requireAdminDeleteError(t, response, http.StatusServiceUnavailable, string(CodeArticleDeletionInProgress))
			requireAnonymousModerationPendingHidden(t, deletion, version)
			pendingETag, err := StrongETag(*deletion.repository.article)
			if err != nil {
				t.Fatal(err)
			}
			response = serveModerationDeletion(t, deletion, moderation.handler.DeleteViolating, pendingETag)
			requireAdminDeleteError(t, response, http.StatusPreconditionFailed, string(CodeArticlePreconditionFailed))
			deletion.journal.appendErr = nil
			deletion.realFinalizer.dependencies.Auditor = deletion.auditor
			deletion.realFinalizer.dependencies.Clock = deletionClockStub{now: deletionNow.Add(2 * time.Minute)}
			response = serveModerationDeletion(t, deletion, moderation.handler.DeleteViolating, etag)
			if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
				t.Fatalf("pending retry with original deletion ETag = %d %s", response.Code, response.Body)
			}
			requireAnonymousModerationDeletionCompleted(t, deletion, version)
		})
	}
}

func requireAnonymousModerationPendingHidden(t *testing.T, fixture permanentDeletionFixture, version int64) {
	t.Helper()
	article := fixture.repository.article
	operation := fixture.repository.operation
	if article == nil || article.DeletionState != DeletionPending || article.Version != version+1 ||
		operation == nil || operation.Status == DeletionOperationCompleted || operation.AuditEventID != nil || len(fixture.auditor.events) != 0 {
		t.Fatal("incomplete deletion committed physical deletion or a completion audit")
	}
	if _, err := (VisibilityPolicy{}).MemberDetailProjection(*article, ""); !errors.Is(err, ErrArticleUnavailable) {
		t.Fatalf("pending member projection = %v", err)
	}
	if _, err := (VisibilityPolicy{}).AuthorProjection(*article, deletionAdminPrincipal()); !errors.Is(err, ErrArticleUnavailable) {
		t.Fatalf("pending author projection = %v", err)
	}
	if _, err := (VisibilityPolicy{}).AdminProjection(*article, deletionAdminPrincipal(), ""); !errors.Is(err, ErrArticleUnavailable) {
		t.Fatalf("pending admin projection = %v", err)
	}
}

func TestModeration_AnonymousDelete_HeldSourceRequiresOrphanPath(t *testing.T) {
	for _, violating := range []bool{false, true} {
		t.Run(fmt.Sprintf("violating=%v", violating), func(t *testing.T) {
			deletion := newPermanentDeletionFixture(t)
			article := deletion.repository.article
			article.AuthorID, article.OrphanedFromUserID = nil, stringPointer(deletionAuthorID)
			article.PublicationState = PublicationHeld
			if violating {
				category := ViolationViolence
				article.ViolationState, article.ViolationCategory = ViolationViolating, &category
			}
			etag, err := StrongETag(*article)
			if err != nil {
				t.Fatal(err)
			}
			handler := deletion.handler(t)
			response := serveModerationDeletion(t, deletion, handler.DeleteViolating, etag)
			requireAdminDeleteError(t, response, http.StatusConflict, string(CodeArticleInvalidState))
			requireAdminDeletionNotStarted(t, deletion)
			response = serveModerationDeletion(t, deletion, handler.DeleteOrphan, etag)
			if response.Code != http.StatusNoContent || deletion.repository.article != nil ||
				deletion.repository.operation.Status != DeletionOperationCompleted || len(deletion.journal.records) != 1 ||
				len(deletion.auditor.events) != 1 || deletion.auditor.events[0].Action != "article.orphan.delete" {
				t.Fatalf("held source orphan deletion = %d %s", response.Code, response.Body)
			}
		})
	}
}

func (moderationRejectingAuditor) Append(context.Context, pgx.Tx, audit.Event) error {
	return errors.New("synthetic finalizer audit failure")
}

type moderationDeletionTransactor struct{ fixture permanentDeletionFixture }

func (transactor moderationDeletionTransactor) Begin(ctx context.Context) (pgx.Tx, error) {
	transaction, err := transactor.fixture.transactor.Begin(ctx)
	if err != nil {
		return nil, err
	}
	snapshot := &moderationDeletionTransaction{Tx: transaction, repository: transactor.fixture.repository}
	if snapshot.repository.article != nil {
		article := *snapshot.repository.article
		snapshot.article = &article
	}
	if snapshot.repository.operation != nil {
		operation := *snapshot.repository.operation
		snapshot.operation = &operation
	}
	return snapshot, nil
}

type moderationDeletionTransaction struct {
	pgx.Tx
	repository *deletionRepositoryStub
	article    *Article
	operation  *DeletionOperation
	committed  bool
}

func (transaction *moderationDeletionTransaction) Commit(ctx context.Context) error {
	err := transaction.Tx.Commit(ctx)
	transaction.committed = err == nil
	return err
}

func (transaction *moderationDeletionTransaction) Rollback(ctx context.Context) error {
	if !transaction.committed {
		transaction.repository.article, transaction.repository.operation = transaction.article, transaction.operation
	}
	return transaction.Tx.Rollback(ctx)
}
