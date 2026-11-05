package article

import (
	"context"
	"encoding/json"
	"errors"
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

func TestOrphanDispositionFinalizer_ExpiredLease_RejectsBeforeDelete(t *testing.T) {
	fixture := newPermanentDeletionFixture(t)
	operation, tombstone := fixture.prepareJournaledDeletion(t)
	claimed, owned, err := fixture.realFinalizer.claim(context.Background(), operation.ID, tombstone)
	if err != nil || !owned {
		t.Fatalf("claim owned=%v, error=%v", owned, err)
	}
	fixture.realFinalizer.dependencies.Clock = deletionClockStub{now: claimed.LockedUntil.Add(time.Second)}
	err = fixture.realFinalizer.complete(context.Background(), claimed, tombstone, deletionTraceID)
	if !errors.Is(err, ErrDeletionInProgress) || fixture.repository.deleteCalls.Load() != 0 ||
		fixture.repository.completeCalls.Load() != 0 || len(fixture.auditor.events) != 0 {
		t.Fatalf("expired lease reached deletion: error=%v, deletes=%d", err, fixture.repository.deleteCalls.Load())
	}
}

func TestOrphanDispositionCoordinator_NanosecondClock_PersistsCanonicalTimeAcrossRetry(t *testing.T) {
	fixture := newPermanentDeletionFixture(t)
	input := deletionNow.Add(123456789 * time.Nanosecond)
	fixture.coordinator.dependencies.Clock = deletionClockStub{now: input}
	fixture.journal.appendErr = errors.New("controlled first append failure")
	err := fixture.coordinator.DeletePermanent(context.Background(), deletionAuthorPrincipal(), deletionRequest(DeleteOrdinary))
	if !errors.Is(err, ErrDeletionInProgress) {
		t.Fatalf("first append result=%v", err)
	}
	expected := input.UTC().Truncate(time.Microsecond)
	if !fixture.repository.operation.StartedAt.Equal(expected) || fixture.repository.operation.StartedAt.Nanosecond()%1000 != 0 {
		t.Fatal("operation persistence retained submicrosecond clock precision")
	}
	fixture.journal.appendErr = nil
	fixture.coordinator.dependencies.Clock = deletionClockStub{now: input.Add(2 * time.Second)}
	if err := fixture.coordinator.DeletePermanent(context.Background(), deletionAuthorPrincipal(), deletionRequest(DeleteOrdinary)); err != nil {
		t.Fatal(err)
	}
	if len(fixture.journal.records) != 1 || !fixture.journal.records[0].DeletedAt.Equal(expected) ||
		!fixture.repository.operation.StartedAt.Equal(expected) || fixture.repository.operation.Status != DeletionOperationCompleted {
		t.Fatal("retry changed the operation/journal canonical timestamp")
	}
}

func serveOrphanHTTP(t *testing.T, principal session.Principal, handler http.HandlerFunc, request *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	authorizer, err := platform.NewHTTPAuthorizer(createArticlePrincipalResolver{principal: principal}, func() time.Time { return deletionNow })
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: platform.SessionCookieName, Value: "synthetic-session"})
	request.Header.Set(platform.CSRFHeaderName, "synthetic-csrf")
	response := httptest.NewRecorder()
	authorizer.Protect(session.AccessGeneral, handler).ServeHTTP(response, request)
	return response
}

func TestOrphanDispositionHTTP_TransferResponse_PreservesNotificationContract(t *testing.T) {
	fixture := newOrphanCommandFixture(t)
	category := ViolationViolence
	fixture.store.article.ViolationState, fixture.store.article.ViolationCategory = ViolationViolating, &category
	handler, err := NewOrphanCommandHandler(&Handler{}, fixture.service, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orphan-articles/"+deletionArticleID+"/transfer",
		strings.NewReader(`{"new_author_id":"`+deletionAdminID+`"}`)).WithContext(fixture.ctx)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("If-Match", `"article-`+deletionArticleID+`-v3"`)
	response := serveOrphanHTTP(t, deletionAdminPrincipal(), handler.TransferOrphan, request)
	var payload struct {
		Article      AdminArticleView `json:"article"`
		Notification struct {
			ID                 string `json:"id"`
			DeliveryCapability string `json:"delivery_capability"`
			Recipients         []struct {
				Kind   RecipientKind   `json:"kind"`
				Status RecipientStatus `json:"status"`
			} `json:"recipients"`
		} `json:"notification"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || payload.Notification.ID != *fixture.identity.proof.Reference().NoticeID ||
		payload.Notification.DeliveryCapability != "closed" || len(payload.Notification.Recipients) != 2 ||
		payload.Article.ViolationState != ViolationViolating || !reflect.DeepEqual(payload.Article.HiddenReasons, []HiddenReason{HiddenViolation}) {
		t.Fatalf("transfer contract mismatch: status=%d", response.Code)
	}
	kinds := make(map[RecipientKind]bool)
	for _, recipient := range payload.Notification.Recipients {
		if recipient.Status != RecipientPending {
			t.Fatal("response claimed delivery")
		}
		kinds[recipient.Kind] = true
	}
	if !kinds[RecipientNewAuthor] || !kinds[RecipientFormerDeletedAuthor] || strings.Contains(response.Body.String(), "@") {
		t.Fatal("response lost recipient identity or leaked a contact")
	}
}

type orphanQueryRepositoryStub struct {
	records   []Article
	listCalls int
	getCalls  int
	lastAfter *ArticleCursor
}

func (repository *orphanQueryRepositoryStub) ListOrphanBatch(ctx context.Context, after *ArticleCursor, limit int) ([]Article, error) {
	repository.listCalls++
	repository.lastAfter = after
	var result []Article
	for _, record := range repository.records {
		if after == nil || cursorStrictlyAfter(*after, articleCursorValue(record)) {
			result = append(result, record)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
func (repository *orphanQueryRepositoryStub) GetArticle(ctx context.Context, articleID string) (Article, error) {
	repository.getCalls++
	for _, record := range repository.records {
		if record.ID == articleID {
			return record, nil
		}
	}
	return Article{}, pgx.ErrNoRows
}

func newOrphanQueryFixture(t *testing.T) (*OrphanQueryService, *orphanQueryRepositoryStub) {
	t.Helper()
	record := deletionTestArticle()
	record.AuthorID, record.OrphanedFromUserID, record.PublicationState = nil, stringPointer(deletionAuthorID), PublicationHeld
	record.ID = "10000000-0000-4000-8000-000000000105"
	second := record
	second.ID = "10000000-0000-4000-8000-000000000104"
	category := ViolationViolence
	second.ViolationState, second.ViolationCategory = ViolationViolating, &category
	repository := &orphanQueryRepositoryStub{records: []Article{record, second}}
	service, err := NewOrphanQueryService(repository, QueryConfig{DefaultLimit: 1, MaxLimit: 2, CandidateBatchSize: 3})
	if err != nil {
		t.Fatal(err)
	}
	return service, repository
}

func TestOrphanDispositionGET_CursorDetailAndAuthorization(t *testing.T) {
	service, repository := newOrphanQueryFixture(t)
	page, err := service.ListOrphanArticles(context.Background(), deletionAdminPrincipal(), ListArticlesRequest{})
	if err != nil || len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatalf("first page: %v", err)
	}
	cursor, err := (CursorCodec{}).Decode(*page.NextCursor, CursorViewAdminOrphan)
	if err != nil || cursor.ID != repository.records[0].ID {
		t.Fatalf("wrong cursor: %v", err)
	}
	last, err := service.ListOrphanArticles(context.Background(), deletionAdminPrincipal(), ListArticlesRequest{Cursor: *page.NextCursor})
	if err != nil || len(last.Items) != 1 || last.Items[0].ID != repository.records[1].ID || last.NextCursor != nil || repository.lastAfter.View != CursorViewAdminOrphan {
		t.Fatalf("second page: %v", err)
	}
	view, etag, err := service.GetOrphanArticle(context.Background(), deletionAdminPrincipal(), repository.records[1].ID)
	if err != nil || view.ViolationState != ViolationViolating || len(view.HiddenReasons) != 2 || etag == "" {
		t.Fatalf("detail: %v", err)
	}
	reads := repository.listCalls + repository.getCalls
	if _, err := service.ListOrphanArticles(context.Background(), deletionAuthorPrincipal(), ListArticlesRequest{Cursor: "invalid"}); !errors.Is(err, ErrArticleForbidden) {
		t.Fatal("member list was not rejected first")
	}
	if _, _, err := service.GetOrphanArticle(context.Background(), deletionAuthorPrincipal(), "invalid"); !errors.Is(err, ErrArticleForbidden) || reads != repository.listCalls+repository.getCalls {
		t.Fatal("member detail reached persistence")
	}
	memberCursor, err := (CursorCodec{}).Encode(ArticleCursor{ID: cursor.ID, CreatedAt: cursor.CreatedAt, View: CursorViewMember})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ListOrphanArticles(context.Background(), deletionAdminPrincipal(), ListArticlesRequest{Cursor: memberCursor}); !errors.Is(err, ErrInvalidCursor) {
		t.Fatal("member cursor crossed views")
	}
}

func TestOrphanDispositionGET_PendingDisposedAndMissing_AreUnavailable(t *testing.T) {
	for _, state := range []string{"pending", "transferred", "published", "missing"} {
		t.Run(state, func(t *testing.T) {
			service, repository := newOrphanQueryFixture(t)
			articleID := repository.records[0].ID
			switch state {
			case "pending":
				repository.records[0].DeletionState, repository.records[0].DeletionOperationID = DeletionPending, stringPointer(deletionOperationID)
			case "transferred":
				repository.records[0].AuthorID, repository.records[0].OrphanedFromUserID, repository.records[0].PublicationState = stringPointer(deletionAuthorID), nil, PublicationPublished
			case "published":
				repository.records[0].OrphanedFromUserID, repository.records[0].PublicationState = nil, PublicationPublished
			case "missing":
				repository.records = nil
			}
			if _, _, err := service.GetOrphanArticle(context.Background(), deletionAdminPrincipal(), articleID); !errors.Is(err, ErrArticleUnavailable) {
				t.Fatalf("non-orphan detail=%v", err)
			}
		})
	}
}

func TestOrphanDispositionGET_HTTPBoundsCursorAndProjection(t *testing.T) {
	service, repository := newOrphanQueryFixture(t)
	handler, err := NewOrphanQueryHandler(service)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		query  string
		status int
	}{
		{"", http.StatusOK}, {"?cursor=", http.StatusBadRequest}, {"?cursor=bad", http.StatusBadRequest},
		{"?cursor=one&cursor=two", http.StatusBadRequest}, {"?limit=0", http.StatusUnprocessableEntity},
		{"?limit=3", http.StatusUnprocessableEntity}, {"?limit=1&limit=2", http.StatusUnprocessableEntity},
	} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orphan-articles"+testCase.query, nil)
		response := serveOrphanHTTP(t, deletionAdminPrincipal(), handler.ListOrphanArticles, request)
		if response.Code != testCase.status {
			t.Fatalf("query=%s status=%d", testCase.query, response.Code)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orphan-articles/"+repository.records[1].ID, nil)
	response := serveOrphanHTTP(t, deletionAdminPrincipal(), handler.GetOrphanArticle, request)
	if response.Code != http.StatusOK || response.Header().Get("ETag") == "" || strings.Contains(response.Body.String(), "@") ||
		!strings.Contains(response.Body.String(), `"violation"`) || !strings.Contains(response.Body.String(), `"orphan_hold"`) {
		t.Fatal("GET projection lost governance or leaked contacts")
	}
}

type orphanQuerySQLDatabase struct {
	ArticleQueryDatabase
	query     string
	arguments []any
	records   []Article
	rows      *orphanQuerySQLRows
}

func (database *orphanQuerySQLDatabase) Query(ctx context.Context, query string, arguments ...any) (pgx.Rows, error) {
	database.query, database.arguments = query, arguments
	database.rows = &orphanQuerySQLRows{records: database.records}
	return database.rows, nil
}

type orphanQuerySQLRows struct {
	pgx.Rows
	records []Article
	index   int
	closed  bool
}

func (rows *orphanQuerySQLRows) Next() bool { rows.index++; return rows.index <= len(rows.records) }
func (rows *orphanQuerySQLRows) Scan(destinations ...any) error {
	return (orphanSQLRow{values: orphanArticleValues(rows.records[rows.index-1])}).Scan(destinations...)
}
func (rows *orphanQuerySQLRows) Err() error { return nil }
func (rows *orphanQuerySQLRows) Close()     { rows.closed = true }

func TestOrphanDispositionGET_ProductionRepository_FiltersBeforeLimit(t *testing.T) {
	_, fixture := newOrphanQueryFixture(t)
	database := &orphanQuerySQLDatabase{records: fixture.records}
	repository, err := NewPostgresArticleQueryRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	after := &ArticleCursor{ID: fixture.records[0].ID, CreatedAt: fixture.records[0].CreatedAt, View: CursorViewAdminOrphan}
	records, err := repository.ListOrphanBatch(context.Background(), after, 2)
	if err != nil || len(records) != 2 || !database.rows.closed || !reflect.DeepEqual(database.arguments, []any{after.CreatedAt, after.ID, 2}) {
		t.Fatalf("orphan SQL arguments/resources=%v", err)
	}
	for _, clause := range []string{"author_id IS NULL", "orphaned_from_user_id IS NOT NULL", "publication_state = 'held'", "deletion_state = 'active'", "ORDER BY created_at DESC, id DESC LIMIT $3"} {
		if !strings.Contains(database.query, clause) {
			t.Fatalf("missing orphan SQL clause %s", clause)
		}
	}
}

type orphanHTTPGate struct {
	fixture *orphanCommandFixture
	open    bool
	checked bool
}

func (gate *orphanHTTPGate) RequireOpen(ctx context.Context) error {
	gate.checked = true
	if len(gate.fixture.transactor.transactions) != 1 || gate.fixture.transactor.transactions[0].commitCalls != 1 {
		return errors.New("gate evaluated before commit")
	}
	if !gate.open {
		return errors.New("processing permission is closed")
	}
	return nil
}

func TestOrphanDispositionHTTP_GatePermission_IsNotDelivery(t *testing.T) {
	for _, open := range []bool{false, true} {
		fixture := newOrphanCommandFixture(t)
		gate := &orphanHTTPGate{fixture: fixture, open: open}
		handler, err := NewOrphanCommandHandler(&Handler{}, fixture.service, gate)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orphan-articles/"+deletionArticleID+"/transfer", strings.NewReader(`{"new_author_id":"`+deletionAdminID+`"}`)).WithContext(fixture.ctx)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("If-Match", `"article-`+deletionArticleID+`-v3"`)
		response := serveOrphanHTTP(t, deletionAdminPrincipal(), handler.TransferOrphan, request)
		var payload orphanTransferResponse
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		expected := orphanDeliveryClosed
		if open {
			expected = orphanDeliveryOpen
		}
		if response.Code != http.StatusOK || !gate.checked || payload.Notification.DeliveryCapability != expected {
			t.Fatal("gate permission was not reported honestly")
		}
		for _, recipient := range payload.Notification.Recipients {
			if recipient.Status != RecipientPending {
				t.Fatal("permission was reported as SMTP acceptance")
			}
		}
		for _, recipient := range fixture.store.recipients {
			if recipient.Status != RecipientPending {
				t.Fatal("HTTP path dispatched SMTP")
			}
		}
	}
}

func TestOrphanDispositionHTTP_StrictBodyAndETag_RejectBeforeWrites(t *testing.T) {
	cases := []struct {
		name, action, body, etag string
		status                   int
	}{
		{"missing etag", "publish", "", "", http.StatusPreconditionRequired},
		{"weak etag", "publish", "", `W/"article-` + deletionArticleID + `-v3"`, http.StatusBadRequest},
		{"wrong article", "publish", "", `"article-` + deletionOperationID + `-v3"`, http.StatusPreconditionFailed},
		{"stale etag", "publish", "", `"article-` + deletionArticleID + `-v2"`, http.StatusPreconditionFailed},
		{"publish object", "publish", `{}`, `"article-` + deletionArticleID + `-v3"`, http.StatusBadRequest},
		{"unknown field", "transfer", `{"new_author_id":"` + deletionAdminID + `","extra":true}`, `"article-` + deletionArticleID + `-v3"`, http.StatusBadRequest},
		{"duplicate field", "transfer", `{"new_author_id":"` + deletionAdminID + `","new_author_id":"` + deletionAdminID + `"}`, `"article-` + deletionArticleID + `-v3"`, http.StatusBadRequest},
		{"trailing json", "transfer", `{"new_author_id":"` + deletionAdminID + `"} {}`, `"article-` + deletionArticleID + `-v3"`, http.StatusBadRequest},
		{"invalid target", "transfer", `{"new_author_id":"invalid"}`, `"article-` + deletionArticleID + `-v3"`, http.StatusUnprocessableEntity},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newOrphanCommandFixture(t)
			handler, err := NewOrphanCommandHandler(&Handler{}, fixture.service, nil)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/orphan-articles/"+deletionArticleID+"/"+testCase.action, strings.NewReader(testCase.body)).WithContext(fixture.ctx)
			request.Header.Set("Content-Type", "application/json")
			if testCase.etag != "" {
				request.Header.Set("If-Match", testCase.etag)
			}
			consumer := handler.PublishOrphan
			if testCase.action == "transfer" {
				consumer = handler.TransferOrphan
			}
			response := serveOrphanHTTP(t, deletionAdminPrincipal(), consumer, request)
			if response.Code != testCase.status || fixture.store.notice != nil || len(fixture.store.audits) != 0 || fixture.store.article.Version != 3 {
				t.Fatalf("strict request status=%d", response.Code)
			}
		})
	}
}

type orphanSQLRow struct {
	values []any
	err    error
}

func (row orphanSQLRow) Scan(destinations ...any) error {
	if row.err != nil {
		return row.err
	}
	if len(destinations) != len(row.values) {
		return errors.New("unexpected disposition SQL scan shape")
	}
	for index, value := range row.values {
		if value == nil {
			reflect.ValueOf(destinations[index]).Elem().SetZero()
			continue
		}
		reflect.ValueOf(destinations[index]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

func orphanOperationValues(operation DeletionOperation) []any {
	return []any{operation.ID, operation.ArticleID, operation.ArticleVersion, operation.DeleteKind,
		operation.ActorUserID, operation.ActorRole, operation.OrphanedFromUserID, operation.Status,
		operation.TombstoneSequence, operation.JournalKeyID, operation.JournalKeyVersion, operation.ClaimedBy,
		operation.LockedUntil, operation.AuditEventID, operation.StartedAt, operation.CompletedAt}
}

func orphanAuditValues(event audit.Event) []any {
	metadata, _ := json.Marshal(event.Metadata)
	return []any{event.ID, event.OccurredAt, event.ActorKind, event.ActorUserID, event.ActorSystem, event.ActorRole,
		event.Action, event.TargetType, event.TargetID, event.Outcome, event.ErrorCode, event.TraceID,
		event.SourceFingerprint, metadata}
}

func readDeletionFixtureRow(repository *deletionRepositoryStub, auditor *deletionAuditorStub, query string, arguments ...any) pgx.Row {
	if query == readDispositionStateSQL {
		return orphanSQLRow{err: pgx.ErrNoRows}
	}
	if query == readDispositionOperationSQL {
		repository.mu.Lock()
		defer repository.mu.Unlock()
		if repository.operation == nil {
			return orphanSQLRow{err: pgx.ErrNoRows}
		}
		return orphanSQLRow{values: orphanOperationValues(*repository.operation)}
	}
	if query == countDispositionAssociationsSQL {
		return orphanSQLRow{values: []any{int64(0), int64(0)}}
	}
	if strings.Contains(query, "FROM audit_events") {
		auditor.mu.Lock()
		defer auditor.mu.Unlock()
		for _, event := range auditor.events {
			if event.ID == arguments[0].(string) {
				return orphanSQLRow{values: orphanAuditValues(event)}
			}
		}
		return orphanSQLRow{err: pgx.ErrNoRows}
	}
	return orphanSQLRow{err: errors.New("unexpected deletion proof SQL")}
}

type orphanSQLStore struct {
	article    *Article
	operation  *DeletionOperation
	notice     *TransferNotice
	recipients []TransferNoticeRecipient
	audits     map[string]audit.Event
	active     int64
	pending    int64
	calls      []string
	failStage  string
	zeroStage  string
	failure    error
	onLock     func(*orphanSQLStore)
	onRead     func(*orphanSQLStore)
}

func (store *orphanSQLStore) copy() orphanSQLStore {
	image := *store
	if store.article != nil {
		article := *store.article
		image.article = &article
	}
	if store.operation != nil {
		operation := *store.operation
		image.operation = &operation
	}
	if store.notice != nil {
		notice := *store.notice
		image.notice = &notice
	}
	image.recipients = append([]TransferNoticeRecipient(nil), store.recipients...)
	image.audits = make(map[string]audit.Event)
	for eventID, event := range store.audits {
		image.audits[eventID] = event
	}
	return image
}

type orphanSQLTx struct {
	*createArticleTxStub
	store     *orphanSQLStore
	image     orphanSQLStore
	isolation pgx.TxIsoLevel
}

func (transaction *orphanSQLTx) Rollback(ctx context.Context) error {
	if !transaction.closed {
		calls := append(transaction.store.calls, "rollback")
		*transaction.store = transaction.image
		transaction.store.calls = calls
	}
	return transaction.createArticleTxStub.Rollback(ctx)
}

func (transaction *orphanSQLTx) Commit(ctx context.Context) error {
	transaction.store.calls = append(transaction.store.calls, "commit")
	return transaction.createArticleTxStub.Commit(ctx)
}

func (transaction *orphanSQLTx) QueryRow(ctx context.Context, query string, arguments ...any) pgx.Row {
	if transaction.closed {
		return orphanSQLRow{err: pgx.ErrTxClosed}
	}
	store := transaction.store
	switch query {
	case locateDispositionArticleSQL:
		store.calls = append(store.calls, "locate")
		if store.article == nil {
			return orphanSQLRow{err: pgx.ErrNoRows}
		}
		return orphanSQLRow{values: []any{store.article.ID, store.article.OrphanedFromUserID, store.article.Version}}
	case lockDispositionArticleSQL:
		store.calls = append(store.calls, "article NOWAIT")
		if store.failStage == "article NOWAIT" {
			return orphanSQLRow{err: store.failure}
		}
		if store.onLock != nil {
			store.onLock(store)
		}
		if store.article == nil {
			return orphanSQLRow{err: pgx.ErrNoRows}
		}
		return orphanSQLRow{values: orphanArticleValues(*store.article)}
	case readDispositionStateSQL:
		store.calls = append(store.calls, "read article")
		if store.onRead != nil {
			store.onRead(store)
		}
		if store.article == nil {
			return orphanSQLRow{err: pgx.ErrNoRows}
		}
		state := dispositionState(*store.article)
		return orphanSQLRow{values: []any{state.ID, state.Version, state.AuthorID, state.SourceID,
			state.Publication, state.Violation, state.Category, state.Deletion, state.OperationID}}
	case readDispositionNoticeSQL:
		if store.notice == nil {
			return orphanSQLRow{err: pgx.ErrNoRows}
		}
		return orphanSQLRow{values: []any{store.notice.ID, store.notice.ArticleRef, store.notice.NewAuthorID, store.notice.FormerDeletedUserID}}
	case countDispositionAssociationsSQL:
		return transaction.associationCountRow(arguments...)
	case loadDeletionOperationForUpdateSQL, readDispositionOperationSQL:
		if store.operation == nil {
			return orphanSQLRow{err: pgx.ErrNoRows}
		}
		return orphanSQLRow{values: orphanOperationValues(*store.operation)}
	}
	return transaction.additionalQueryRow(query, arguments...)
}

func (transaction *orphanSQLTx) associationCountRow(arguments ...any) pgx.Row {
	if transaction.isolation != pgx.ReadCommitted {
		return orphanSQLRow{err: errors.New("counts used inherited stale isolation")}
	}
	transaction.store.calls = append(transaction.store.calls, "same snapshot counts")
	if transaction.store.failStage == "counts" {
		return orphanSQLRow{err: transaction.store.failure}
	}
	if len(arguments) != 1 || arguments[0] != deletionAuthorID {
		return orphanSQLRow{err: errors.New("count source binding mismatch")}
	}
	return orphanSQLRow{values: []any{transaction.store.active, transaction.store.pending}}
}

func (transaction *orphanSQLTx) additionalQueryRow(query string, arguments ...any) pgx.Row {
	store := transaction.store
	if strings.Contains(query, "UPDATE deletion_operations") && strings.Contains(query, "RETURNING") {
		deadline := arguments[2].(time.Time).UTC().Truncate(time.Microsecond)
		store.operation.Status, store.operation.ClaimedBy, store.operation.LockedUntil =
			DeletionOperationFinalizing, stringPointer(arguments[1].(string)), &deadline
		return orphanSQLRow{values: orphanOperationValues(*store.operation)}
	}
	if strings.Contains(query, "FROM audit_events") {
		event, exists := store.audits[arguments[0].(string)]
		if !exists {
			return orphanSQLRow{err: pgx.ErrNoRows}
		}
		return orphanSQLRow{values: orphanAuditValues(event)}
	}
	if strings.Contains(query, "SELECT EXISTS") {
		return orphanSQLRow{values: []any{store.article != nil}}
	}
	return orphanSQLRow{err: errors.New("unexpected disposition SQL query")}
}

func orphanArticleValues(article Article) []any {
	return []any{article.ID, article.AuthorID, article.OrphanedFromUserID, article.Title, article.Body,
		article.PublicationState, article.ViolationState, article.ViolationCategory, article.DeletionState,
		article.DeletionOperationID, article.Version, article.CreatedAt, article.UpdatedAt}
}

func (transaction *orphanSQLTx) Query(ctx context.Context, query string, arguments ...any) (pgx.Rows, error) {
	if transaction.closed {
		return nil, pgx.ErrTxClosed
	}
	if query != readDispositionRecipientsSQL {
		return nil, errors.New("unexpected disposition rows SQL")
	}
	return &orphanRecipientRows{recipients: transaction.store.recipients}, nil
}

func (transaction *orphanSQLTx) Exec(ctx context.Context, query string, arguments ...any) (pgconn.CommandTag, error) {
	if transaction.closed {
		return pgconn.CommandTag{}, pgx.ErrTxClosed
	}
	stage := orphanWriteStage(query)
	transaction.store.calls = append(transaction.store.calls, stage)
	if transaction.store.failStage == stage {
		return pgconn.CommandTag{}, transaction.store.failure
	}
	if transaction.store.zeroStage == stage {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	return transaction.applySQLWrite(stage, arguments...)
}

func orphanWriteStage(query string) string {
	switch {
	case query == insertDeletionOperationRecordSQL:
		return "rebuild operation"
	case query == dispositionReadCommittedSQL:
		return "read committed"
	case query == insertTransferNoticeRecordSQL:
		return "notice"
	case query == insertTransferNoticeRecipientRecordSQL:
		return "recipient"
	case query == updateOrphanDispositionSQL:
		return "disposition update"
	case strings.Contains(query, "INSERT INTO audit_events"):
		return "audit"
	case strings.Contains(query, "DELETE FROM articles"):
		return "physical delete"
	case strings.Contains(query, "UPDATE deletion_operations"):
		return "complete operation"
	default:
		return "unknown"
	}
}

func (transaction *orphanSQLTx) applySQLWrite(stage string, arguments ...any) (pgconn.CommandTag, error) {
	store := transaction.store
	switch stage {
	case "rebuild operation":
		return transaction.rebuildOperation(arguments...)
	case "read committed":
		transaction.isolation = pgx.ReadCommitted
	case "notice":
		store.notice = &TransferNotice{ID: arguments[0].(string), ArticleRef: arguments[1].(string),
			NewAuthorID: arguments[2].(string), FormerDeletedUserID: arguments[3].(string)}
	case "recipient":
		store.recipients = append(store.recipients, TransferNoticeRecipient{NoticeID: arguments[0].(string),
			Kind: arguments[1].(RecipientKind), EmailSnapshot: arguments[2].(string), Status: arguments[3].(RecipientStatus)})
	case "disposition update":
		if store.article.Version != arguments[1].(int64) ||
			!sameDispositionString(store.article.OrphanedFromUserID, arguments[2].(*string)) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		}
		store.article.AuthorID, store.article.OrphanedFromUserID = arguments[3].(*string), nil
		store.article.PublicationState, store.article.UpdatedAt = PublicationPublished, arguments[4].(time.Time)
		store.article.Version++
	case "audit":
		store.audits[arguments[0].(string)] = audit.Event{ID: arguments[0].(string), OccurredAt: arguments[1].(time.Time),
			ActorKind: arguments[2].(audit.ActorKind), ActorUserID: arguments[3].(*string), ActorSystem: arguments[4].(*string), ActorRole: arguments[5].(*string),
			Action: arguments[6].(string), TargetType: arguments[7].(string), TargetID: arguments[8].(string),
			Outcome: arguments[9].(string), TraceID: arguments[11].(string)}
	case "physical delete":
		if store.article == nil {
			return pgconn.NewCommandTag("DELETE 0"), nil
		}
		if store.article.Version != arguments[1].(int64) ||
			!sameDispositionString(store.article.DeletionOperationID, stringPointer(arguments[2].(string))) {
			return pgconn.NewCommandTag("DELETE 0"), nil
		}
		store.article = nil
	case "complete operation":
		store.operation.Status, store.operation.ClaimedBy, store.operation.LockedUntil = DeletionOperationCompleted, nil, nil
		store.operation.AuditEventID = stringPointer(arguments[2].(string))
		store.operation.CompletedAt = timePointer(arguments[3].(time.Time))
	default:
		return pgconn.CommandTag{}, errors.New("unexpected disposition SQL mutation")
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (transaction *orphanSQLTx) rebuildOperation(arguments ...any) (pgconn.CommandTag, error) {
	operation := DeletionOperation{}
	destinations := []any{&operation.ID, &operation.ArticleID, &operation.ArticleVersion, &operation.DeleteKind,
		&operation.ActorUserID, &operation.ActorRole, &operation.OrphanedFromUserID, &operation.Status,
		&operation.TombstoneSequence, &operation.JournalKeyID, &operation.JournalKeyVersion,
		&operation.ClaimedBy, &operation.LockedUntil, &operation.AuditEventID, &operation.StartedAt, &operation.CompletedAt}
	if err := (orphanSQLRow{values: arguments}).Scan(destinations...); err != nil {
		return pgconn.CommandTag{}, err
	}
	transaction.store.operation = &operation
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

type orphanRecipientRows struct {
	pgx.Rows
	recipients []TransferNoticeRecipient
	index      int
}

func (rows *orphanRecipientRows) Next() bool { rows.index++; return rows.index <= len(rows.recipients) }
func (rows *orphanRecipientRows) Close()     {}
func (rows *orphanRecipientRows) Err() error { return nil }
func (rows *orphanRecipientRows) Scan(destinations ...any) error {
	recipient := rows.recipients[rows.index-1]
	return (orphanSQLRow{values: []any{recipient.NoticeID, recipient.Kind, recipient.EmailSnapshot, recipient.Status}}).Scan(destinations...)
}

type orphanTransactor struct {
	store        *orphanSQLStore
	transactions []*orphanSQLTx
}

func (transactor *orphanTransactor) Begin(context.Context) (pgx.Tx, error) {
	transaction := &orphanSQLTx{createArticleTxStub: &createArticleTxStub{}, store: transactor.store, image: transactor.store.copy(), isolation: pgx.RepeatableRead}
	transactor.transactions = append(transactor.transactions, transaction)
	return transaction, nil
}

type orphanIdentity struct {
	store         *orphanSQLStore
	proof         *DispositionCompletionProof
	deleted       bool
	clearCalls    int
	lockErr       error
	clearErr      error
	beforeClear   func(context.Context, pgx.Tx, *DispositionCompletionProof) error
	emptyContacts bool
}

func (identity *orphanIdentity) LockTransferParties(
	ctx context.Context, transaction pgx.Tx, targetID *string, sourceID string,
) (DispositionParties, error) {
	identity.store.calls = append(identity.store.calls, "identity NOWAIT")
	if identity.lockErr != nil {
		return DispositionParties{}, identity.lockErr
	}
	parties := DispositionParties{SourceID: sourceID, TargetID: copyDispositionString(targetID),
		SourceEmail: stringPointer("Former@Example.test"), TargetEmail: stringPointer("New@Example.test")}
	if identity.emptyContacts {
		parties.SourceEmail, parties.TargetEmail = nil, nil
	}
	return parties, nil
}

func (identity *orphanIdentity) LockDeletionCandidate(
	ctx context.Context, transaction pgx.Tx, authorID string,
) (DispositionDeletionCandidate, error) {
	identity.store.calls = append(identity.store.calls, "candidate NOWAIT")
	return DispositionDeletionCandidate{UserID: authorID, Deleted: identity.deleted}, identity.lockErr
}

func (identity *orphanIdentity) ClearRetainedEmailIfEligible(
	ctx context.Context, transaction pgx.Tx, sourceID string, proof *DispositionCompletionProof,
) (bool, error) {
	identity.clearCalls++
	identity.proof = proof
	identity.store.calls = append(identity.store.calls, "identity clear")
	if _, err := proof.VerifyCompletion(ctx, transaction); err != nil {
		return false, err
	}
	if identity.beforeClear != nil {
		if err := identity.beforeClear(ctx, transaction, proof); err != nil {
			return false, err
		}
	}
	return false, identity.clearErr
}

type orphanCommandFixture struct {
	ctx        context.Context
	store      *orphanSQLStore
	transactor *orphanTransactor
	identity   *orphanIdentity
	service    *CommandService
}

func newOrphanCommandFixture(t *testing.T) *orphanCommandFixture {
	t.Helper()
	article := deletionTestArticle()
	article.AuthorID, article.OrphanedFromUserID, article.PublicationState = nil, stringPointer(deletionAuthorID), PublicationHeld
	store := &orphanSQLStore{article: &article, audits: make(map[string]audit.Event), failure: errors.New("injected SQL failure")}
	transactor := &orphanTransactor{store: store}
	identity := &orphanIdentity{store: store}
	service, err := NewCommandService(CommandDependencies{Transactions: transactor, Repository: Repository{},
		IDs: deletionIDStub{id: deletionAuditID}, Clock: deletionClockStub{now: deletionNow.Add(time.Second)},
		Disposition: &OrphanDispositionDependencies{Identity: identity, Auditor: audit.Store{}, RecipientMaxAttempts: 3}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := platform.ContextWithTraceID(context.Background(), deletionTraceID)
	if err != nil {
		t.Fatal(err)
	}
	return &orphanCommandFixture{ctx: ctx, store: store, transactor: transactor, identity: identity, service: service}
}

func TestOrphanDispositionCommand_PublishTransfer_ProductionSQLAndProof(t *testing.T) {
	for _, transfer := range []bool{false, true} {
		fixture := newOrphanCommandFixture(t)
		var result OrphanDispositionResult
		var err error
		if transfer {
			category := ViolationViolence
			fixture.store.article.ViolationState, fixture.store.article.ViolationCategory = ViolationViolating, &category
			result, err = fixture.service.TransferOrphan(fixture.ctx, deletionAdminPrincipal(), deletionArticleID, 3, deletionAdminID)
		} else {
			result, err = fixture.service.PublishOrphan(fixture.ctx, deletionAdminPrincipal(), deletionArticleID, 3)
		}
		if err != nil || result.Article.Version != 4 || result.Article.OrphanedFromUserID != nil ||
			result.Article.PublicationState != PublicationPublished || fixture.identity.clearCalls != 1 ||
			fixture.transactor.transactions[0].commitCalls != 1 {
			t.Fatalf("disposition failed: %v", err)
		}
		if transfer && (len(fixture.store.recipients) != 2 || result.Article.ViolationState != ViolationViolating || result.NoticeID == nil) {
			t.Fatal("transfer did not retain violation and double pending snapshots")
		}
		for _, recipient := range fixture.store.recipients {
			if recipient.Status != RecipientPending {
				t.Fatal("transfer attempted delivery")
			}
		}
		if !reflect.DeepEqual(fixture.store.calls[:4], []string{"read committed", "locate", "identity NOWAIT", "article NOWAIT"}) {
			t.Fatalf("wrong production locking order: %v", fixture.store.calls)
		}
		proof := fixture.identity.proof
		if _, err := proof.VerifyCompletion(fixture.ctx, fixture.transactor.transactions[0]); !errors.Is(err, pgx.ErrTxClosed) {
			t.Fatalf("proof survived completed transaction: %v", err)
		}
	}
}

func TestOrphanDispositionCommand_Failure_RollsBackAllSQL(t *testing.T) {
	for _, stage := range []string{"read committed", "article NOWAIT", "notice", "recipient", "disposition update", "audit", "counts", "identity clear"} {
		t.Run(stage, func(t *testing.T) {
			fixture := newOrphanCommandFixture(t)
			fixture.store.failStage = stage
			if stage == "identity clear" {
				fixture.identity.clearErr = fixture.store.failure
			}
			_, err := fixture.service.TransferOrphan(fixture.ctx, deletionAdminPrincipal(), deletionArticleID, 3, deletionAdminID)
			if !errors.Is(err, fixture.store.failure) || fixture.transactor.transactions[0].commitCalls != 0 ||
				fixture.store.article.Version != 3 || fixture.store.article.PublicationState != PublicationHeld ||
				fixture.store.article.OrphanedFromUserID == nil || fixture.store.notice != nil ||
				len(fixture.store.recipients) != 0 || len(fixture.store.audits) != 0 {
				t.Fatalf("failed transfer left partial SQL changes: %v", err)
			}
		})
	}
}

func TestOrphanDispositionCommand_DriftAndZeroRows_Rejects(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(*orphanCommandFixture)
	}{
		{"source drift", func(fixture *orphanCommandFixture) {
			fixture.store.onLock = func(store *orphanSQLStore) {
				store.article.OrphanedFromUserID = stringPointer(deletionAdminID)
			}
		}},
		{"version drift", func(fixture *orphanCommandFixture) {
			fixture.store.onLock = func(store *orphanSQLStore) { store.article.Version++ }
		}},
		{"pending", func(fixture *orphanCommandFixture) {
			fixture.store.onLock = func(store *orphanSQLStore) { store.article.DeletionState = DeletionPending }
		}},
		{"zero notice", func(fixture *orphanCommandFixture) { fixture.store.zeroStage = "notice" }},
		{"zero recipient", func(fixture *orphanCommandFixture) { fixture.store.zeroStage = "recipient" }},
		{"zero article update", func(fixture *orphanCommandFixture) { fixture.store.zeroStage = "disposition update" }},
		{"missing notice readback", func(fixture *orphanCommandFixture) {
			fixture.store.onRead = func(store *orphanSQLStore) { store.notice = nil }
		}},
		{"wrong notice association", func(fixture *orphanCommandFixture) {
			fixture.store.onRead = func(store *orphanSQLStore) { store.notice.ArticleRef = deletionAdminID }
		}},
		{"duplicate recipient kind", func(fixture *orphanCommandFixture) {
			fixture.store.onRead = func(store *orphanSQLStore) { store.recipients[1].Kind = store.recipients[0].Kind }
		}},
		{"snapshot drift", func(fixture *orphanCommandFixture) {
			fixture.store.onRead = func(store *orphanSQLStore) { store.recipients[0].EmailSnapshot = "wrong@example.test" }
		}},
		{"recipient not pending", func(fixture *orphanCommandFixture) {
			fixture.store.onRead = func(store *orphanSQLStore) { store.recipients[0].Status = RecipientAccepted }
		}},
		{"negative counts", func(fixture *orphanCommandFixture) { fixture.store.pending = -1 }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newOrphanCommandFixture(t)
			testCase.prepare(fixture)
			_, err := fixture.service.TransferOrphan(fixture.ctx, deletionAdminPrincipal(), deletionArticleID, 3, deletionAdminID)
			if err == nil || fixture.transactor.transactions[0].commitCalls != 0 || len(fixture.store.audits) != 0 {
				t.Fatalf("invalid production readback committed: %v", err)
			}
		})
	}
}

func TestOrphanDispositionProof_PublicProjectionAndCrossTx_DoNotGrantAuthority(t *testing.T) {
	fixture := newOrphanCommandFixture(t)
	fixture.identity.beforeClear = func(ctx context.Context, transaction pgx.Tx, proof *DispositionCompletionProof) error {
		reference := proof.Reference()
		*reference.BeforeArticle.SourceID = deletionAdminID
		*reference.NewAuthorID = deletionAuthorID
		*reference.NoticeID = deletionArticleID
		if proof.Reference().SourceID != deletionAuthorID || *proof.Reference().BeforeArticle.SourceID != deletionAuthorID ||
			*proof.Reference().NewAuthorID != deletionAdminID {
			return errors.New("public projection mutated private proof")
		}
		if _, err := proof.VerifyCompletion(ctx, &createArticleTxStub{}); err == nil {
			return errors.New("proof accepted another transaction")
		}
		if _, err := (&DispositionCompletionProof{}).VerifyCompletion(ctx, transaction); err == nil {
			return errors.New("public empty proof accepted")
		}
		return nil
	}
	_, err := fixture.service.TransferOrphan(fixture.ctx, deletionAdminPrincipal(), deletionArticleID, 3, deletionAdminID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOrphanDispositionCommand_CountsUseSingleReadCommittedSnapshot(t *testing.T) {
	fixture := newOrphanCommandFixture(t)
	fixture.store.active, fixture.store.pending = 2, 3
	fixture.identity.beforeClear = func(ctx context.Context, transaction pgx.Tx, proof *DispositionCompletionProof) error {
		facts, err := proof.VerifyCompletion(ctx, transaction)
		if err != nil {
			return err
		}
		if facts.RemainingActiveCount != 2 || facts.RemainingPendingCount != 3 {
			return errors.New("wrong source counts")
		}
		return nil
	}
	_, err := fixture.service.PublishOrphan(fixture.ctx, deletionAdminPrincipal(), deletionArticleID, 3)
	if err != nil || fixture.transactor.transactions[0].isolation != pgx.ReadCommitted {
		t.Fatalf("read committed guarantee failed: %v", err)
	}
	if !strings.Contains(countDispositionAssociationsSQL, "count(*) FILTER (WHERE deletion_state = 'active')") ||
		!strings.Contains(countDispositionAssociationsSQL, "count(*) FILTER (WHERE deletion_state = 'deletion_pending')") ||
		!strings.Contains(countDispositionAssociationsSQL, "WHERE author_id = $1::uuid OR orphaned_from_user_id = $1::uuid") ||
		strings.Contains(countDispositionAssociationsSQL, "FOR UPDATE") {
		t.Fatal("counting changed snapshot or association contract")
	}
}

func newOrphanFinalizerFixture(t *testing.T, kind DeleteKind, deleted, absent bool) (*orphanCommandFixture, *DeletionFinalizer, ArticleDeletionTombstone) {
	t.Helper()
	fixture := newOrphanCommandFixture(t)
	journal := &deletionJournalStub{}
	tombstone, err := journal.Append(context.Background(), DeletionTombstoneRequest{ArticleID: deletionArticleID,
		ArticleVersion: 3, DeletionOperationID: deletionOperationID, DeletedAt: deletionNow})
	if err != nil {
		t.Fatal(err)
	}
	operation := DeletionOperation{ID: deletionOperationID, ArticleID: deletionArticleID, ArticleVersion: 3,
		DeleteKind: kind, Status: DeletionOperationJournaled, TombstoneSequence: int64Pointer(tombstone.Sequence),
		JournalKeyID: stringPointer(tombstone.KeyID), JournalKeyVersion: stringPointer(tombstone.KeyVersion),
		ActorUserID: stringPointer(deletionAdminID), ActorRole: stringPointer("admin"), StartedAt: deletionNow}
	if kind == DeleteOrphan {
		operation.OrphanedFromUserID = stringPointer(deletionAuthorID)
	} else {
		fixture.store.article.AuthorID, fixture.store.article.OrphanedFromUserID = stringPointer(deletionAuthorID), nil
		fixture.store.article.PublicationState = PublicationPublished
	}
	fixture.store.article.Version, fixture.store.article.DeletionState = 4, DeletionPending
	fixture.store.article.DeletionOperationID = stringPointer(operation.ID)
	if absent {
		fixture.store.article = nil
	}
	fixture.store.operation, fixture.identity.deleted = &operation, deleted
	finalizer, err := NewDeletionFinalizer(DeletionFinalizerDependencies{Transactions: fixture.transactor,
		Repository: Repository{}, Journal: journal, Auditor: audit.Store{}, AuditIDs: deletionIDStub{id: deletionAuditID},
		Clock: deletionClockStub{now: deletionNow.Add(time.Second)}, WorkerID: "test-finalizer", LeaseDuration: time.Minute, Identity: fixture.identity})
	if err != nil {
		t.Fatal(err)
	}
	return fixture, finalizer, tombstone
}

func TestOrphanDispositionFinalizer_SourceBranches_ProductionSQL(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		kind    DeleteKind
		deleted bool
		absent  bool
		cleanup bool
	}{
		{"orphan", DeleteOrphan, true, false, true}, {"orphan absent", DeleteOrphan, true, true, true},
		{"ordinary deleted", DeleteOrdinary, true, false, true}, {"violating deleted", DeleteViolating, true, false, true},
		{"active author", DeleteOrdinary, false, false, false}, {"absent without source", DeleteOrdinary, true, true, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fixture, finalizer, _ := newOrphanFinalizerFixture(t, testCase.kind, testCase.deleted, testCase.absent)
			if err := finalizer.Finalize(context.Background(), deletionOperationID, deletionTraceID); err != nil {
				t.Fatal(err)
			}
			if fixture.store.article != nil || fixture.store.operation.Status != DeletionOperationCompleted ||
				fixture.store.operation.DeleteKind != testCase.kind || (fixture.identity.clearCalls == 1) != testCase.cleanup {
				t.Fatal("finalization source branch did not complete correctly")
			}
			if testCase.kind != DeleteOrphan && fixture.store.operation.OrphanedFromUserID != nil {
				t.Fatal("operation source history changed")
			}
			event := fixture.store.audits[*fixture.store.operation.AuditEventID]
			if event.ActorKind != audit.ActorUser || !sameDispositionString(event.ActorUserID, stringPointer(deletionAdminID)) ||
				event.ActorSystem != nil || event.Action != deletionAuditAction(testCase.kind) {
				t.Fatal("user deletion audit changed to recovery semantics")
			}
			if testCase.cleanup && fixture.identity.proof.Reference().SourceID != deletionAuthorID {
				t.Fatal("source derived from actor")
			}
			if fixture.transactor.transactions[1].isolation != pgx.ReadCommitted {
				t.Fatal("T2 inherited stale snapshot isolation")
			}
		})
	}
}

func TestOrphanDispositionFinalizer_SourceFreeReplay_SystemAuditAndIdempotence(t *testing.T) {
	fixture, finalizer, tombstone := newOrphanFinalizerFixture(t, DeleteOrdinary, true, true)
	fixture.store.operation = nil
	result, err := finalizer.ReplayAllTombstones(fixture.ctx, []ArticleDeletionTombstone{tombstone}, deletionTraceID)
	if err != nil || result.AlreadyAbsent != 1 || result.Replayed != 0 {
		t.Fatalf("source-free replay result=%+v error=%v", result, err)
	}
	operation := fixture.store.operation
	if operation.Status != DeletionOperationCompleted || operation.AuditEventID == nil || operation.CompletedAt == nil ||
		operation.DeleteKind != DeleteOrdinary || operation.ActorUserID != nil || operation.ActorRole != nil || operation.OrphanedFromUserID != nil {
		t.Fatal("reconstructed operation lost completion or invented historical identity")
	}
	event := fixture.store.audits[*operation.AuditEventID]
	if event.ActorKind != audit.ActorSystem || event.ActorSystem == nil || *event.ActorSystem != "article-deletion-replay" ||
		event.ActorUserID != nil || event.ActorRole != nil || event.Action != "article.deletion.replay" ||
		event.TargetType != "article" || event.TargetID != tombstone.ArticleID || event.TraceID != deletionTraceID ||
		event.Outcome != "succeeded" || !event.OccurredAt.Equal(*operation.CompletedAt) {
		t.Fatalf("source-free recovery did not write a bound system replay audit: kind=%s action=%s", event.ActorKind, event.Action)
	}
	before := fixture.store.copy()
	result, err = finalizer.ReplayAllTombstones(fixture.ctx, []ArticleDeletionTombstone{tombstone}, deletionTraceID)
	if err != nil || result.AlreadyAbsent != 1 || !reflect.DeepEqual(before.operation, fixture.store.operation) ||
		!reflect.DeepEqual(before.audits, fixture.store.audits) || fixture.identity.clearCalls != 0 {
		t.Fatalf("reentry duplicated completion or guessed cleanup provenance: %v", err)
	}
	if strings.Contains(strings.Join(fixture.store.calls, ","), "identity") || len(fixture.store.audits) != 1 {
		t.Fatal("source-free replay invoked identity or duplicated audit")
	}
}

func TestOrphanDispositionFinalizer_SourceFreeReplay_FailureRollsBackT2(t *testing.T) {
	for _, stage := range []string{"audit", "complete operation"} {
		t.Run(stage, func(t *testing.T) {
			fixture, finalizer, tombstone := newOrphanFinalizerFixture(t, DeleteOrdinary, true, true)
			fixture.store.operation, fixture.store.failStage = nil, stage
			_, err := finalizer.ReplayAllTombstones(fixture.ctx, []ArticleDeletionTombstone{tombstone}, deletionTraceID)
			operation := fixture.store.operation
			if !errors.Is(err, fixture.store.failure) || operation == nil || operation.Status != DeletionOperationFinalizing ||
				operation.ClaimedBy == nil || operation.LockedUntil == nil || operation.CompletedAt != nil || operation.AuditEventID != nil ||
				operation.ActorUserID != nil || operation.ActorRole != nil || operation.OrphanedFromUserID != nil ||
				len(fixture.store.audits) != 0 || fixture.identity.clearCalls != 0 {
				t.Fatalf("failed recovery lost committed claim or retained T2 effects: %v", err)
			}
			fixture.store.failStage = ""
			finalizer.dependencies.Clock = deletionClockStub{now: deletionNow.Add(2 * time.Minute)}
			if _, err := finalizer.ReplayAllTombstones(fixture.ctx, []ArticleDeletionTombstone{tombstone}, deletionTraceID); err != nil {
				t.Fatal(err)
			}
			if fixture.store.operation.Status != DeletionOperationCompleted || len(fixture.store.audits) != 1 || fixture.identity.clearCalls != 0 {
				t.Fatal("recovery retry did not converge exactly once")
			}
		})
	}
}

func TestOrphanDispositionFinalizer_Failure_PreservesCommittedClaim(t *testing.T) {
	for _, stage := range []string{"read committed", "physical delete", "audit", "complete operation", "counts", "identity clear"} {
		t.Run(stage, func(t *testing.T) {
			fixture, finalizer, _ := newOrphanFinalizerFixture(t, DeleteOrdinary, true, false)
			fixture.store.failStage = stage
			if stage == "identity clear" {
				fixture.identity.clearErr = fixture.store.failure
			}
			err := finalizer.Finalize(context.Background(), deletionOperationID, deletionTraceID)
			if !errors.Is(err, fixture.store.failure) || fixture.store.article == nil ||
				fixture.store.operation.Status != DeletionOperationFinalizing || fixture.store.operation.LockedUntil == nil ||
				fixture.store.operation.AuditEventID != nil || len(fixture.store.audits) != 0 ||
				fixture.transactor.transactions[0].commitCalls != 1 || fixture.transactor.transactions[1].commitCalls != 0 {
				t.Fatalf("T2 failure lost committed claim or retained effects: %v", err)
			}
		})
	}
}

func TestOrphanDispositionFinalizer_NOWAIT_RemainsWorkerTransient(t *testing.T) {
	for _, identityLock := range []bool{false, true} {
		fixture, finalizer, _ := newOrphanFinalizerFixture(t, DeleteOrdinary, true, false)
		cause := &pgconn.PgError{Code: "55P03", Message: "lock unavailable"}
		if identityLock {
			fixture.identity.lockErr = cause
		} else {
			fixture.store.failStage, fixture.store.failure = "article NOWAIT", cause
		}
		err := finalizer.Finalize(context.Background(), deletionOperationID, deletionTraceID)
		var postgresError *pgconn.PgError
		if !errors.Is(err, cause) || !errors.As(err, &postgresError) || errors.Is(err, ErrDeletionInProgress) ||
			fixture.store.article == nil || fixture.store.operation.Status != DeletionOperationFinalizing {
			t.Fatalf("NOWAIT error was swallowed or claim revoked: %v", err)
		}
	}
}

func TestOrphanDispositionFinalizer_MissingIdentity_FailsClosed(t *testing.T) {
	for _, kind := range []DeleteKind{DeleteOrdinary, DeleteOrphan} {
		fixture, finalizer, _ := newOrphanFinalizerFixture(t, kind, true, false)
		finalizer.dependencies.Identity = nil
		if err := finalizer.Finalize(context.Background(), deletionOperationID, deletionTraceID); err == nil ||
			fixture.store.article == nil || len(fixture.store.audits) != 0 {
			t.Fatalf("missing identity was bypassed: %v", err)
		}
	}
}

func TestOrphanDispositionFinalizer_ClaimUsesStoredMicrosecondsAndLeaseFence(t *testing.T) {
	for _, mutation := range []string{"valid", "deadline", "owner", "status", "expired"} {
		t.Run(mutation, func(t *testing.T) {
			fixture, finalizer, tombstone := newOrphanFinalizerFixture(t, DeleteOrdinary, false, false)
			finalizer.dependencies.Clock = deletionClockStub{now: deletionNow.Add(123 * time.Nanosecond)}
			claimed, owned, err := finalizer.claim(context.Background(), deletionOperationID, tombstone)
			if err != nil || !owned || claimed.LockedUntil.Nanosecond()%int(time.Microsecond) != 0 ||
				!claimed.LockedUntil.Equal(*fixture.store.operation.LockedUntil) {
				t.Fatalf("non-authoritative lease: %v", err)
			}
			switch mutation {
			case "deadline":
				fixture.store.operation.LockedUntil = timePointer(claimed.LockedUntil.Add(time.Microsecond))
			case "owner":
				fixture.store.operation.ClaimedBy = stringPointer("other-owner")
			case "status":
				fixture.store.operation.Status = DeletionOperationJournaled
			case "expired":
				finalizer.dependencies.Clock = deletionClockStub{now: claimed.LockedUntil.Add(time.Second)}
			}
			err = finalizer.complete(context.Background(), claimed, tombstone, deletionTraceID)
			if mutation == "valid" {
				if err != nil || fixture.store.article != nil {
					t.Fatalf("valid lease rejected: %v", err)
				}
			} else if !errors.Is(err, ErrDeletionInProgress) || fixture.store.article == nil || len(fixture.store.audits) != 0 {
				t.Fatalf("lost lease reached physical deletion: %v", err)
			}
		})
	}
}

func TestOrphanDispositionFinalizer_LockAcquiredAfterExpiry_UsesFreshClock(t *testing.T) {
	fixture := newPermanentDeletionFixture(t)
	operation, tombstone := fixture.prepareJournaledDeletion(t)
	claimed, owned, err := fixture.realFinalizer.claim(context.Background(), operation.ID, tombstone)
	if err != nil || !owned {
		t.Fatalf("claim owned=%v, error=%v", owned, err)
	}
	fixture.repository.onLoadOperation = func() {
		fixture.realFinalizer.dependencies.Clock = deletionClockStub{now: claimed.LockedUntil.Add(time.Second)}
	}
	err = fixture.realFinalizer.complete(context.Background(), claimed, tombstone, deletionTraceID)
	if !errors.Is(err, ErrDeletionInProgress) || fixture.repository.deleteCalls.Load() != 0 {
		t.Fatalf("lease was checked before acquiring operation lock: %v", err)
	}
}

func TestOrphanDispositionCommand_ContactsAreTransferOnly(t *testing.T) {
	fixture := newOrphanCommandFixture(t)
	fixture.identity.emptyContacts = true
	if _, err := fixture.service.PublishOrphan(fixture.ctx, deletionAdminPrincipal(), deletionArticleID, 3); err != nil {
		t.Fatalf("publish required transfer contacts: %v", err)
	}
	fixture = newOrphanCommandFixture(t)
	fixture.identity.emptyContacts = true
	if _, err := fixture.service.TransferOrphan(fixture.ctx, deletionAdminPrincipal(), deletionArticleID, 3, deletionAdminID); err == nil ||
		len(fixture.store.audits) != 0 || fixture.store.notice != nil {
		t.Fatal("transfer accepted empty contacts")
	}
}

func TestOrphanDispositionCommand_AuthorizationAndState_RejectBeforeWrites(t *testing.T) {
	fixture := newOrphanCommandFixture(t)
	if _, err := fixture.service.TransferOrphan(fixture.ctx, deletionAuthorPrincipal(), deletionArticleID, 3, "invalid"); !errors.Is(err, ErrArticleForbidden) || len(fixture.transactor.transactions) != 0 {
		t.Fatal("authorization did not precede input/SQL")
	}
	category := ViolationSexual
	fixture.store.article.ViolationState, fixture.store.article.ViolationCategory = ViolationViolating, &category
	if _, err := fixture.service.PublishOrphan(fixture.ctx, deletionAdminPrincipal(), deletionArticleID, 3); !errors.Is(err, ErrArticleInvalidState) || len(fixture.store.audits) != 0 {
		t.Fatal("violating orphan was published")
	}
}
