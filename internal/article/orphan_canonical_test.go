package article

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"han/internal/audit"
	"han/internal/platform"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	orphanDeletedUserID = "30000000-0000-4000-8000-000000000003"
	orphanOtherUserID   = "40000000-0000-4000-8000-000000000004"
	orphanTraceID       = "01ARZ3NDEKTSV4RRFFQ69G5FAV"
)

var orphanCanonicalNow = time.Date(2026, time.October, 23, 4, 5, 6, 0, time.UTC)

func TestOrphanArticlesCanonicalRepository_LockCandidates_UsesCanonicalRowLock(t *testing.T) {
	category := ViolationViolence
	operationID := "50000000-0000-4000-8000-000000000005"
	want := []Article{
		orphanCanonicalArticle(
			"10000000-0000-4000-8000-000000000001",
			orphanStringPointer(orphanDeletedUserID),
			nil,
			PublicationPublished,
			ViolationClear,
			nil,
			DeletionActive,
			nil,
			3,
		),
		orphanCanonicalArticle(
			"10000000-0000-4000-8000-000000000002",
			nil,
			orphanStringPointer(orphanDeletedUserID),
			PublicationHeld,
			ViolationViolating,
			&category,
			DeletionPending,
			&operationID,
			7,
		),
	}
	tx := newCanonicalOrphanTxStub()
	tx.rows = &canonicalOrphanRowsStub{articles: want}

	got, err := (CanonicalOrphanRepository{}).LockCanonicalOrphanCandidates(
		context.Background(),
		tx,
		orphanDeletedUserID,
	)

	if err != nil {
		t.Fatalf("LockCanonicalOrphanCandidates() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("articles = %#v, want %#v", got, want)
	}
	requireSQLContains(t, tx.query, []string{
		"FROM articles",
		"deletion_state IN ('active', 'deletion_pending')",
		"(author_id = $1 OR orphaned_from_user_id = $1)",
		"ORDER BY id",
		"FOR UPDATE",
		"violation_state",
		"violation_category",
	})
	if !reflect.DeepEqual(tx.queryArgs, []any{orphanDeletedUserID}) {
		t.Fatalf("query args = %#v", tx.queryArgs)
	}
	if tx.commitCalls != 0 || tx.rollbackCalls != 0 {
		t.Fatalf("commit/rollback calls = %d/%d, want 0/0", tx.commitCalls, tx.rollbackCalls)
	}
}

func TestOrphanArticlesCanonicalRepository_SetCanonicalOrphan_UsesConditionalUpdate(t *testing.T) {
	tx := newCanonicalOrphanTxStub()
	tx.commandTag = pgconn.NewCommandTag("UPDATE 1")
	update := CanonicalOrphanUpdate{
		ArticleID:       "10000000-0000-4000-8000-000000000001",
		DeletedUserID:   orphanDeletedUserID,
		ExpectedVersion: 7,
		UpdatedAt:       orphanCanonicalNow,
	}

	err := (CanonicalOrphanRepository{}).SetCanonicalOrphan(
		context.Background(),
		tx,
		update,
	)

	if err != nil {
		t.Fatalf("SetCanonicalOrphan() error = %v", err)
	}
	requireSQLContains(t, tx.execQuery, []string{
		"UPDATE articles",
		"author_id = NULL",
		"publication_state = $1",
		"orphaned_from_user_id = $2",
		"version = version + 1",
		"updated_at = $3",
		"id = $4",
		"deletion_state = $5",
		"author_id = $2",
		"version = $6",
	})
	if strings.Contains(tx.execQuery, "violation_state") ||
		strings.Contains(tx.execQuery, "violation_category") {
		t.Fatalf("conditional update changes violation fields: %s", tx.execQuery)
	}
	wantArgs := []any{
		PublicationHeld,
		orphanDeletedUserID,
		orphanCanonicalNow,
		update.ArticleID,
		DeletionActive,
		update.ExpectedVersion,
	}
	if !reflect.DeepEqual(tx.execArgs, wantArgs) {
		t.Fatalf("exec args = %#v, want %#v", tx.execArgs, wantArgs)
	}
	if tx.commitCalls != 0 || tx.rollbackCalls != 0 {
		t.Fatalf("commit/rollback calls = %d/%d, want 0/0", tx.commitCalls, tx.rollbackCalls)
	}
}

func TestOrphanArticlesCanonicalRepository_SetCanonicalOrphan_RejectsUnexpectedRowsAffected(t *testing.T) {
	tests := []struct {
		name string
		tag  pgconn.CommandTag
	}{
		{name: "missing or stale row", tag: pgconn.NewCommandTag("UPDATE 0")},
		{name: "non unique update", tag: pgconn.NewCommandTag("UPDATE 2")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := newCanonicalOrphanTxStub()
			tx.commandTag = test.tag

			err := (CanonicalOrphanRepository{}).SetCanonicalOrphan(
				context.Background(),
				tx,
				CanonicalOrphanUpdate{
					ArticleID:       "10000000-0000-4000-8000-000000000001",
					DeletedUserID:   orphanDeletedUserID,
					ExpectedVersion: 1,
					UpdatedAt:       orphanCanonicalNow,
				},
			)

			if err == nil {
				t.Fatal("SetCanonicalOrphan() error = nil")
			}
			if tx.commitCalls != 0 || tx.rollbackCalls != 0 {
				t.Fatalf(
					"commit/rollback calls = %d/%d, want 0/0",
					tx.commitCalls,
					tx.rollbackCalls,
				)
			}
		})
	}
}

func TestOrphanArticlesCanonical_MixedActiveAndPendingRows_OrphansAndCounts(t *testing.T) {
	category := ViolationViolence
	operationID := "50000000-0000-4000-8000-000000000005"
	repository := &orphanCanonicalRepositoryStub{
		rows: []Article{
			orphanCanonicalArticle("article-clear", orphanStringPointer(orphanDeletedUserID), nil, PublicationPublished, ViolationClear, nil, DeletionActive, nil, 3),
			orphanCanonicalArticle("article-violating", orphanStringPointer(orphanDeletedUserID), nil, PublicationPublished, ViolationViolating, &category, DeletionActive, nil, 7),
			orphanCanonicalArticle("article-already", nil, orphanStringPointer(orphanDeletedUserID), PublicationHeld, ViolationClear, nil, DeletionActive, nil, 4),
			orphanCanonicalArticle("article-pending-owned", orphanStringPointer(orphanDeletedUserID), nil, PublicationPublished, ViolationClear, nil, DeletionPending, &operationID, 9),
			orphanCanonicalArticle("article-pending-orphan", nil, orphanStringPointer(orphanDeletedUserID), PublicationHeld, ViolationClear, nil, DeletionPending, &operationID, 10),
		},
	}
	fixture := newOrphanCanonicalFixture(t, repository)

	result, err := fixture.service.OrphanArticlesCanonical(
		orphanCanonicalContext(t),
		fixture.transaction,
		orphanDeletedUserID,
	)

	if err != nil {
		t.Fatalf("OrphanArticlesCanonical() error = %v", err)
	}
	wantResult := OrphanArticlesResult{
		NewlyOrphanedCount: 2, AlreadyOrphanedCount: 1, PendingOrphanCount: 5,
	}
	if result != wantResult {
		t.Fatalf("result = %#v, want %#v", result, wantResult)
	}
	if len(repository.updates) != 2 {
		t.Fatalf("updates = %#v, want 2", repository.updates)
	}
	for index, want := range []CanonicalOrphanUpdate{
		{ArticleID: "article-clear", DeletedUserID: orphanDeletedUserID, ExpectedVersion: 3, UpdatedAt: orphanCanonicalNow},
		{ArticleID: "article-violating", DeletedUserID: orphanDeletedUserID, ExpectedVersion: 7, UpdatedAt: orphanCanonicalNow},
	} {
		if repository.updates[index] != want {
			t.Errorf("update[%d] = %#v, want %#v", index, repository.updates[index], want)
		}
	}
	if got := repository.row("article-violating"); got.ViolationState != ViolationViolating ||
		got.ViolationCategory == nil || *got.ViolationCategory != category {
		t.Fatalf("violating article governance changed: %#v", got)
	}
	requireCanonicalOrphanAudits(t, fixture.auditor.events, []string{"article-clear", "article-violating"})
	requireCallerOwnsOrphanTransaction(t, fixture)
}

func TestOrphanArticlesCanonical_RepeatedCall_IsIdempotent(t *testing.T) {
	category := ViolationSexual
	repository := &orphanCanonicalRepositoryStub{
		rows: []Article{
			orphanCanonicalArticle("article-clear", orphanStringPointer(orphanDeletedUserID), nil, PublicationPublished, ViolationClear, nil, DeletionActive, nil, 2),
			orphanCanonicalArticle("article-violating", orphanStringPointer(orphanDeletedUserID), nil, PublicationPublished, ViolationViolating, &category, DeletionActive, nil, 5),
		},
	}
	fixture := newOrphanCanonicalFixture(t, repository)
	ctx := orphanCanonicalContext(t)

	first, err := fixture.service.OrphanArticlesCanonical(ctx, fixture.transaction, orphanDeletedUserID)
	if err != nil {
		t.Fatalf("first OrphanArticlesCanonical() error = %v", err)
	}
	second, err := fixture.service.OrphanArticlesCanonical(ctx, fixture.transaction, orphanDeletedUserID)
	if err != nil {
		t.Fatalf("second OrphanArticlesCanonical() error = %v", err)
	}

	if first != (OrphanArticlesResult{NewlyOrphanedCount: 2, PendingOrphanCount: 2}) {
		t.Fatalf("first result = %#v", first)
	}
	if second != (OrphanArticlesResult{AlreadyOrphanedCount: 2, PendingOrphanCount: 2}) {
		t.Fatalf("second result = %#v", second)
	}
	if len(repository.updates) != 2 || len(fixture.auditor.events) != 2 {
		t.Fatalf("updates/audits = %d/%d, want 2/2", len(repository.updates), len(fixture.auditor.events))
	}
	if got := repository.row("article-clear"); got.Version != 3 {
		t.Fatalf("clear version = %d, want 3", got.Version)
	}
	if got := repository.row("article-violating"); got.Version != 6 {
		t.Fatalf("violating version = %d, want 6", got.Version)
	}
	requireCallerOwnsOrphanTransaction(t, fixture)
}

func TestOrphanArticlesCanonical_DisposedMatrix_DoesNotRegress(t *testing.T) {
	tests := []struct {
		name string
		rows []Article
	}{
		{name: "zero articles"},
		{
			name: "anonymous published",
			rows: []Article{
				orphanCanonicalArticle("anonymous", nil, nil, PublicationPublished, ViolationClear, nil, DeletionActive, nil, 4),
			},
		},
		{
			name: "transferred",
			rows: []Article{
				orphanCanonicalArticle("transferred", orphanStringPointer(orphanOtherUserID), nil, PublicationPublished, ViolationClear, nil, DeletionActive, nil, 5),
			},
		},
		{name: "permanently deleted row absent"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &orphanCanonicalRepositoryStub{rows: test.rows}
			fixture := newOrphanCanonicalFixture(t, repository)

			result, err := fixture.service.OrphanArticlesCanonical(
				orphanCanonicalContext(t),
				fixture.transaction,
				orphanDeletedUserID,
			)

			if err != nil {
				t.Fatalf("OrphanArticlesCanonical() error = %v", err)
			}
			if result != (OrphanArticlesResult{}) {
				t.Fatalf("result = %#v, want zero", result)
			}
			if len(repository.updates) != 0 || len(fixture.auditor.events) != 0 {
				t.Fatalf("updates/audits = %d/%d, want 0/0", len(repository.updates), len(fixture.auditor.events))
			}
			requireCallerOwnsOrphanTransaction(t, fixture)
		})
	}
}

func TestOrphanArticlesCanonical_InvalidInput_RejectsBeforeRepository(t *testing.T) {
	tests := []struct {
		name          string
		tx            pgx.Tx
		deletedUserID string
	}{
		{name: "missing transaction", deletedUserID: orphanDeletedUserID},
		{name: "missing deleted user", tx: &createArticleTxStub{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &orphanCanonicalRepositoryStub{}
			fixture := newOrphanCanonicalFixture(t, repository)

			_, err := fixture.service.OrphanArticlesCanonical(
				orphanCanonicalContext(t),
				test.tx,
				test.deletedUserID,
			)

			if err == nil {
				t.Fatal("OrphanArticlesCanonical() error = nil")
			}
			if repository.lockCalls != 0 {
				t.Fatalf("repository lock calls = %d, want 0", repository.lockCalls)
			}
			requireCallerOwnsOrphanTransaction(t, fixture)
		})
	}
}

func TestOrphanArticlesCanonical_DependencyFailures_ReturnCauseWithoutOwningTransaction(t *testing.T) {
	injected := errors.New("injected failure")
	tests := []struct {
		name          string
		configure     func(*orphanCanonicalFixture)
		wantUpdates   int
		wantAudits    int
		wantLockCalls int
	}{
		{
			name: "lock",
			configure: func(fixture *orphanCanonicalFixture) {
				fixture.repository.lockErr = injected
			},
			wantLockCalls: 1,
		},
		{
			name: "audit id",
			configure: func(fixture *orphanCanonicalFixture) {
				fixture.ids.errAt = 1
				fixture.ids.err = injected
			},
			wantLockCalls: 1,
		},
		{
			name: "update",
			configure: func(fixture *orphanCanonicalFixture) {
				fixture.repository.updateErrAt = 1
				fixture.repository.updateErr = injected
			},
			wantLockCalls: 1,
		},
		{
			name: "audit append",
			configure: func(fixture *orphanCanonicalFixture) {
				fixture.auditor.errAt = 1
				fixture.auditor.err = injected
			},
			wantUpdates: 1, wantAudits: 1, wantLockCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &orphanCanonicalRepositoryStub{
				rows: []Article{
					orphanCanonicalArticle("article", orphanStringPointer(orphanDeletedUserID), nil, PublicationPublished, ViolationClear, nil, DeletionActive, nil, 1),
				},
			}
			fixture := newOrphanCanonicalFixture(t, repository)
			test.configure(&fixture)

			_, err := fixture.service.OrphanArticlesCanonical(
				orphanCanonicalContext(t),
				fixture.transaction,
				orphanDeletedUserID,
			)

			if !errors.Is(err, injected) {
				t.Fatalf("OrphanArticlesCanonical() error = %v, want injected cause", err)
			}
			if repository.lockCalls != test.wantLockCalls ||
				len(repository.updates) != test.wantUpdates ||
				len(fixture.auditor.events) != test.wantAudits {
				t.Fatalf(
					"lock/updates/audits = %d/%d/%d, want %d/%d/%d",
					repository.lockCalls,
					len(repository.updates),
					len(fixture.auditor.events),
					test.wantLockCalls,
					test.wantUpdates,
					test.wantAudits,
				)
			}
			requireCallerOwnsOrphanTransaction(t, fixture)
		})
	}
}

func TestOrphanArticlesCanonical_MissingTrace_FailsBeforeMutation(t *testing.T) {
	repository := &orphanCanonicalRepositoryStub{
		rows: []Article{
			orphanCanonicalArticle("article", orphanStringPointer(orphanDeletedUserID), nil, PublicationPublished, ViolationClear, nil, DeletionActive, nil, 1),
		},
	}
	fixture := newOrphanCanonicalFixture(t, repository)

	_, err := fixture.service.OrphanArticlesCanonical(
		context.Background(),
		fixture.transaction,
		orphanDeletedUserID,
	)

	if err == nil {
		t.Fatal("OrphanArticlesCanonical() error = nil")
	}
	if len(repository.updates) != 0 || len(fixture.auditor.events) != 0 {
		t.Fatalf("updates/audits = %d/%d, want 0/0", len(repository.updates), len(fixture.auditor.events))
	}
	requireCallerOwnsOrphanTransaction(t, fixture)
}

type orphanCanonicalFixture struct {
	service     *CommandService
	transaction *createArticleTxStub
	repository  *orphanCanonicalRepositoryStub
	auditor     *orphanCanonicalAuditorStub
	ids         *orphanCanonicalIDStub
}

func newOrphanCanonicalFixture(
	t *testing.T,
	repository *orphanCanonicalRepositoryStub,
) orphanCanonicalFixture {
	t.Helper()
	transaction := &createArticleTxStub{}
	auditor := &orphanCanonicalAuditorStub{}
	ids := &orphanCanonicalIDStub{
		ids: []string{
			"60000000-0000-4000-8000-000000000006",
			"70000000-0000-4000-8000-000000000007",
		},
	}
	service, err := NewCommandService(CommandDependencies{
		Transactions: &createArticleTransactorStub{transaction: transaction},
		Repository:   &createArticleRepositoryStub{},
		IDs:          createArticleIDStub{id: createArticleID},
		Clock:        createArticleClockStub{now: createArticleNow},
		Orphaning: &OrphanArticlesCanonicalDependencies{
			Repository: repository,
			Auditor:    auditor,
			AuditIDs:   ids,
			Clock:      createArticleClockStub{now: orphanCanonicalNow},
		},
	})
	if err != nil {
		t.Fatalf("NewCommandService() error = %v", err)
	}
	return orphanCanonicalFixture{
		service: service, transaction: transaction, repository: repository,
		auditor: auditor, ids: ids,
	}
}

func orphanCanonicalContext(t *testing.T) context.Context {
	t.Helper()
	ctx, err := platform.ContextWithTraceID(context.Background(), orphanTraceID)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	return ctx
}

func orphanCanonicalArticle(
	id string,
	authorID *string,
	sourceID *string,
	publication PublicationState,
	violation ViolationState,
	category *ViolationCategory,
	deletion DeletionState,
	operationID *string,
	version int64,
) Article {
	return Article{
		ID: id, AuthorID: authorID, OrphanedFromUserID: sourceID,
		Title: "must-not-be-audited", Body: "must-not-be-audited",
		PublicationState: publication, ViolationState: violation,
		ViolationCategory: category, DeletionState: deletion,
		DeletionOperationID: operationID, Version: version,
		CreatedAt: orphanCanonicalNow.Add(-time.Hour), UpdatedAt: orphanCanonicalNow.Add(-time.Minute),
	}
}

func orphanStringPointer(value string) *string {
	return &value
}

func requireCanonicalOrphanAudits(
	t *testing.T,
	events []audit.Event,
	wantArticleIDs []string,
) {
	t.Helper()
	if len(events) != len(wantArticleIDs) {
		t.Fatalf("audit events = %d, want %d", len(events), len(wantArticleIDs))
	}
	system := canonicalOrphanAuditActorSystem
	for index, event := range events {
		if event.ID == "" ||
			!event.OccurredAt.Equal(orphanCanonicalNow) ||
			event.ActorKind != audit.ActorSystem ||
			event.ActorSystem == nil ||
			*event.ActorSystem != system ||
			event.ActorUserID != nil ||
			event.Action != canonicalOrphanAuditAction ||
			event.TargetType != canonicalOrphanAuditTargetType ||
			event.TargetID != wantArticleIDs[index] ||
			event.Outcome != canonicalOrphanAuditOutcome ||
			event.TraceID != orphanTraceID {
			t.Errorf("audit[%d] = %#v", index, event)
		}
		if !reflect.DeepEqual(event.Metadata, map[string]any{
			canonicalOrphanAuditDeletedUserIDKey: orphanDeletedUserID,
			canonicalOrphanAuditVersionKey:       int64(4 + index*4),
		}) {
			t.Errorf("audit[%d] metadata = %#v", index, event.Metadata)
		}
		serialized := event.Action + event.TargetType + event.TargetID + event.Outcome
		for key, value := range event.Metadata {
			serialized += key
			if text, ok := value.(string); ok {
				serialized += text
			}
		}
		if strings.Contains(serialized, "must-not-be-audited") {
			t.Errorf("audit[%d] contains article content", index)
		}
	}
}

func requireCallerOwnsOrphanTransaction(t *testing.T, fixture orphanCanonicalFixture) {
	t.Helper()
	if fixture.transaction.commitCalls != 0 || fixture.transaction.rollbackCalls != 0 {
		t.Fatalf(
			"transaction commit/rollback calls = %d/%d, want 0/0",
			fixture.transaction.commitCalls,
			fixture.transaction.rollbackCalls,
		)
	}
	if fixture.repository.lockCalls > 0 && fixture.repository.lockTx != fixture.transaction {
		t.Fatal("repository lock did not receive the caller-owned transaction")
	}
	for index, tx := range fixture.repository.updateTxs {
		if tx != fixture.transaction {
			t.Fatalf("repository update[%d] did not receive the caller-owned transaction", index)
		}
	}
	for index, tx := range fixture.auditor.txs {
		if tx != fixture.transaction {
			t.Fatalf("audit append[%d] did not receive the caller-owned transaction", index)
		}
	}
}

type orphanCanonicalRepositoryStub struct {
	rows        []Article
	updates     []CanonicalOrphanUpdate
	updateTxs   []pgx.Tx
	lockErr     error
	updateErr   error
	updateErrAt int
	lockCalls   int
	lockTx      pgx.Tx
}

func (s *orphanCanonicalRepositoryStub) LockCanonicalOrphanCandidates(
	_ context.Context,
	tx pgx.Tx,
	_ string,
) ([]Article, error) {
	s.lockCalls++
	s.lockTx = tx
	if s.lockErr != nil {
		return nil, s.lockErr
	}
	return append([]Article(nil), s.rows...), nil
}

func (s *orphanCanonicalRepositoryStub) SetCanonicalOrphan(
	_ context.Context,
	tx pgx.Tx,
	update CanonicalOrphanUpdate,
) error {
	if s.updateErrAt == len(s.updates)+1 {
		return s.updateErr
	}
	s.updates = append(s.updates, update)
	s.updateTxs = append(s.updateTxs, tx)
	for index := range s.rows {
		if s.rows[index].ID != update.ArticleID {
			continue
		}
		source := update.DeletedUserID
		s.rows[index].AuthorID = nil
		s.rows[index].OrphanedFromUserID = &source
		s.rows[index].PublicationState = PublicationHeld
		s.rows[index].Version++
		s.rows[index].UpdatedAt = update.UpdatedAt
		return nil
	}
	return errors.New("article not found")
}

func (s *orphanCanonicalRepositoryStub) row(id string) Article {
	for _, article := range s.rows {
		if article.ID == id {
			return article
		}
	}
	return Article{}
}

type orphanCanonicalAuditorStub struct {
	events []audit.Event
	txs    []pgx.Tx
	err    error
	errAt  int
}

func (s *orphanCanonicalAuditorStub) Append(
	_ context.Context,
	tx pgx.Tx,
	event audit.Event,
) error {
	s.events = append(s.events, event)
	s.txs = append(s.txs, tx)
	if s.errAt == len(s.events) {
		return s.err
	}
	return nil
}

type orphanCanonicalIDStub struct {
	ids   []string
	err   error
	errAt int
	calls int
}

func (s *orphanCanonicalIDStub) NewID() (string, error) {
	s.calls++
	if s.errAt == s.calls {
		return "", s.err
	}
	if s.calls > len(s.ids) {
		return "", errors.New("no audit ID configured")
	}
	return s.ids[s.calls-1], nil
}

func requireSQLContains(t *testing.T, query string, fragments []string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(query, fragment) {
			t.Errorf("SQL does not contain %q: %s", fragment, query)
		}
	}
}

type canonicalOrphanTxStub struct {
	*createArticleTxStub
	rows       pgx.Rows
	queryErr   error
	query      string
	queryArgs  []any
	commandTag pgconn.CommandTag
	execErr    error
	execQuery  string
	execArgs   []any
}

func newCanonicalOrphanTxStub() *canonicalOrphanTxStub {
	return &canonicalOrphanTxStub{createArticleTxStub: &createArticleTxStub{}}
}

func (s *canonicalOrphanTxStub) Query(
	_ context.Context,
	query string,
	args ...any,
) (pgx.Rows, error) {
	s.query = query
	s.queryArgs = append([]any(nil), args...)
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	return s.rows, nil
}

func (s *canonicalOrphanTxStub) Exec(
	_ context.Context,
	query string,
	args ...any,
) (pgconn.CommandTag, error) {
	s.execQuery = query
	s.execArgs = append([]any(nil), args...)
	return s.commandTag, s.execErr
}

type canonicalOrphanRowsStub struct {
	articles []Article
	index    int
	closed   bool
	err      error
}

var _ pgx.Rows = (*canonicalOrphanRowsStub)(nil)

func (s *canonicalOrphanRowsStub) Close() {
	s.closed = true
}

func (s *canonicalOrphanRowsStub) Err() error {
	return s.err
}

func (s *canonicalOrphanRowsStub) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 0")
}

func (s *canonicalOrphanRowsStub) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (s *canonicalOrphanRowsStub) Next() bool {
	if s.index >= len(s.articles) {
		s.closed = true
		return false
	}
	s.index++
	return true
}

func (s *canonicalOrphanRowsStub) Scan(dest ...any) error {
	if s.index == 0 || s.index > len(s.articles) {
		return errors.New("Scan called without current row")
	}
	article := s.articles[s.index-1]
	values := []any{
		article.ID,
		article.AuthorID,
		article.OrphanedFromUserID,
		article.Title,
		article.Body,
		article.PublicationState,
		article.ViolationState,
		article.ViolationCategory,
		article.DeletionState,
		article.DeletionOperationID,
		article.Version,
		article.CreatedAt,
		article.UpdatedAt,
	}
	return assignCanonicalOrphanScanValues(dest, values)
}

func (s *canonicalOrphanRowsStub) Values() ([]any, error) {
	return nil, errors.New("not implemented")
}

func (s *canonicalOrphanRowsStub) RawValues() [][]byte {
	return nil
}

func (s *canonicalOrphanRowsStub) Conn() *pgx.Conn {
	return nil
}

func (s *canonicalOrphanRowsStub) TypeMap() *pgtype.Map {
	return nil
}

func assignCanonicalOrphanScanValues(dest []any, values []any) error {
	if len(dest) != len(values) {
		return errors.New("unexpected scan destination count")
	}
	for index, value := range values {
		target := reflect.ValueOf(dest[index])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return errors.New("scan destination is not a pointer")
		}
		source := reflect.ValueOf(value)
		if source.Type().AssignableTo(target.Elem().Type()) {
			target.Elem().Set(source)
			continue
		}
		return errors.New("scan destination type mismatch")
	}
	return nil
}
