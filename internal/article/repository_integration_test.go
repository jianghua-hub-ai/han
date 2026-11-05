//go:build integration

package article

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"han/internal/testenv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sqlStateUniqueViolation         = "23505"
	sqlStateForeignKeyViolation     = "23503"
	sqlStateCheckViolation          = "23514"
	sqlStateObjectStatePrerequisite = "55000"
)

func TestArticleRepository_DeletionDiscoveryRequiresProductionPool(t *testing.T) {
	if _, err := NewPostgresDeletionOperationDiscoverer(nil); err == nil {
		t.Fatal("NewPostgresDeletionOperationDiscoverer(nil) error = nil")
	}
	for _, fragment := range []string{
		"status IN ('pending_journal', 'journaled')",
		"status = 'finalizing' AND locked_until <= $1",
		"ORDER BY started_at ASC, id ASC",
		"LIMIT $2",
	} {
		if !strings.Contains(discoverDeletionOperationsSQL, fragment) {
			t.Fatalf("discovery SQL missing %q", fragment)
		}
	}
}

func TestArticleRepository_SchemaConstraints_EnforceArticleInvariants(t *testing.T) {
	ctx, tx, schema := newArticleTestTransaction(t)
	fixture := seedArticleUsers(t, ctx, tx)
	now := time.Date(2026, time.October, 22, 8, 0, 0, 0, time.UTC)

	t.Run("orthogonal article state", func(t *testing.T) {
		testArticleStateConstraints(t, ctx, tx, fixture, now)
	})
	t.Run("transfer recipient state", func(t *testing.T) {
		testTransferRecipientConstraints(t, ctx, tx, fixture, now)
	})
	t.Run("deletion operation state", func(t *testing.T) {
		testDeletionOperationConstraints(t, ctx, tx, fixture, now)
	})
	t.Run("replay ownership and immutable evidence", func(t *testing.T) {
		testReplayConstraints(t, ctx, tx, now)
	})
	t.Run("indexes and minimal data", func(t *testing.T) {
		testArticleIndexesAndMinimalData(t, ctx, tx, schema)
	})
}

func TestArticleRepository_InsertModels_PersistsMinimalRows(t *testing.T) {
	ctx, tx, _ := newArticleTestTransaction(t)
	fixture := seedArticleUsers(t, ctx, tx)
	now := time.Date(2026, time.October, 22, 9, 0, 0, 0, time.UTC)
	repository := Repository{}

	articleID := insertRepositoryArticle(t, ctx, tx, repository, fixture.authorID, now)
	insertRepositoryNotice(t, ctx, tx, repository, fixture, articleID, now)
	insertRepositoryOperation(t, ctx, tx, repository, fixture.actorID, now)
	insertRepositoryReplay(t, ctx, tx, repository, now)

	assertRowCount(t, ctx, tx, "articles", 1)
	assertRowCount(t, ctx, tx, "transfer_notices", 1)
	assertRowCount(t, ctx, tx, "transfer_notice_recipients", 2)
	assertRowCount(t, ctx, tx, "deletion_operations", 1)
	assertRowCount(t, ctx, tx, "deletion_replay_runs", 1)
	assertRowCount(t, ctx, tx, "deletion_replay_evidence", 1)
}

func TestArticleRepository_InsertTransferNotice_RejectsInvalidRecipientSetBeforeWrite(
	t *testing.T,
) {
	ctx, tx, _ := newArticleTestTransaction(t)
	fixture := seedArticleUsers(t, ctx, tx)
	now := time.Date(2026, time.October, 22, 9, 30, 0, 0, time.UTC)
	repository := Repository{}

	tests := []struct {
		name       string
		recipients func(string) [2]TransferNoticeRecipient
	}{
		{
			name: "recipient belongs to another notice",
			recipients: func(noticeID string) [2]TransferNoticeRecipient {
				return [2]TransferNoticeRecipient{
					{
						NoticeID: randomUUID(t), Kind: RecipientNewAuthor,
						EmailSnapshot: "new-author@example.test", Status: RecipientPending,
						MaxAttempts: 3, CreatedAt: now, UpdatedAt: now,
					},
					{
						NoticeID: noticeID, Kind: RecipientFormerDeletedAuthor,
						EmailSnapshot: "former@example.test", Status: RecipientPending,
						MaxAttempts: 3, CreatedAt: now, UpdatedAt: now,
					},
				}
			},
		},
		{
			name: "recipient kinds do not cover both required kinds",
			recipients: func(noticeID string) [2]TransferNoticeRecipient {
				return [2]TransferNoticeRecipient{
					{
						NoticeID: noticeID, Kind: RecipientNewAuthor,
						EmailSnapshot: "new-author@example.test", Status: RecipientPending,
						MaxAttempts: 3, CreatedAt: now, UpdatedAt: now,
					},
					{
						NoticeID: noticeID, Kind: RecipientNewAuthor,
						EmailSnapshot: "former@example.test", Status: RecipientPending,
						MaxAttempts: 3, CreatedAt: now, UpdatedAt: now,
					},
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			const savepoint = "article_notice_repository_probe"
			if _, err := tx.Exec(ctx, "SAVEPOINT "+savepoint); err != nil {
				t.Fatalf("create repository probe savepoint: %v", err)
			}
			notice := TransferNotice{
				ID: randomUUID(t), ArticleRef: randomUUID(t),
				NewAuthorID:         fixture.newAuthorID,
				FormerDeletedUserID: fixture.deletedUserID,
				CreatedAt:           now, UpdatedAt: now,
			}
			err := repository.InsertTransferNotice(
				ctx, tx, notice, test.recipients(notice.ID),
			)
			var count int
			queryErr := tx.QueryRow(
				ctx,
				"SELECT count(*) FROM transfer_notices WHERE id = $1",
				notice.ID,
			).Scan(&count)
			if _, rollbackErr := tx.Exec(
				ctx, "ROLLBACK TO SAVEPOINT "+savepoint,
			); rollbackErr != nil {
				t.Fatalf("rollback repository probe savepoint: %v", rollbackErr)
			}
			if _, releaseErr := tx.Exec(
				ctx, "RELEASE SAVEPOINT "+savepoint,
			); releaseErr != nil {
				t.Fatalf("release repository probe savepoint: %v", releaseErr)
			}
			if err == nil {
				t.Fatal("insert transfer notice error = nil, want invalid recipient set")
			}
			if queryErr != nil {
				t.Fatalf("query notice after validation failure: %v", queryErr)
			}
			if count != 0 {
				t.Fatalf("notice row count after validation failure = %d, want 0", count)
			}
		})
	}
}

func insertRepositoryArticle(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	repository Repository,
	authorID string,
	now time.Time,
) string {
	t.Helper()
	articleID := randomUUID(t)
	article := Article{
		ID: articleID, AuthorID: &authorID, Title: "Repository article",
		Body: "Synthetic repository body", PublicationState: PublicationPublished,
		ViolationState: ViolationClear, DeletionState: DeletionActive,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := repository.InsertArticle(ctx, tx, article); err != nil {
		t.Fatalf("insert article model: %v", err)
	}
	return articleID
}

func insertRepositoryNotice(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	repository Repository,
	fixture articleUserFixture,
	articleID string,
	now time.Time,
) {
	t.Helper()
	notice := TransferNotice{
		ID: randomUUID(t), ArticleRef: articleID, NewAuthorID: fixture.newAuthorID,
		FormerDeletedUserID: fixture.deletedUserID, CreatedAt: now, UpdatedAt: now,
	}
	recipients := [2]TransferNoticeRecipient{
		{
			NoticeID: notice.ID, Kind: RecipientNewAuthor,
			EmailSnapshot: "new-author@example.test", Status: RecipientPending,
			MaxAttempts: 3, CreatedAt: now, UpdatedAt: now,
		},
		{
			NoticeID: notice.ID, Kind: RecipientFormerDeletedAuthor,
			EmailSnapshot: "former@example.test", Status: RecipientPending,
			MaxAttempts: 3, CreatedAt: now, UpdatedAt: now,
		},
	}
	if err := repository.InsertTransferNotice(ctx, tx, notice, recipients); err != nil {
		t.Fatalf("insert transfer notice models: %v", err)
	}
}

func insertRepositoryOperation(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	repository Repository,
	actorID string,
	now time.Time,
) {
	t.Helper()
	operation := DeletionOperation{
		ID: randomUUID(t), ArticleID: randomUUID(t), ArticleVersion: 1,
		DeleteKind: DeleteOrdinary, ActorUserID: integrationStringPointer(actorID),
		ActorRole: integrationStringPointer("admin"), Status: DeletionOperationPendingJournal,
		StartedAt: now,
	}
	if err := repository.InsertDeletionOperation(ctx, tx, operation); err != nil {
		t.Fatalf("insert deletion operation model: %v", err)
	}
}

func insertRepositoryReplay(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	repository Repository,
	now time.Time,
) {
	t.Helper()
	run := DeletionReplayRun{
		RunID: "repository-run", Namespace: ArticleDeletionNamespace,
		BackupRef: "repository-backup", Status: ReplayRunRunning,
		StartedAt: now, UpdatedAt: now,
	}
	if err := repository.InsertDeletionReplayRun(ctx, tx, run); err != nil {
		t.Fatalf("insert deletion replay run model: %v", err)
	}
	evidence := DeletionReplayEvidence{
		EvidenceVersion: DeletionReplayEvidenceVersion,
		RunID:           run.RunID, BackupRef: run.BackupRef, Namespace: run.Namespace,
		JournalHeadSequence: 7, JournalHeadHash: "sha256:head",
		InventoryScope: "recoverable-backups-before-cutoff", BackupCutoff: now,
		InventoryDigest:          "sha256:inventory",
		DeletionOperationsDigest: "sha256:operations",
		AuditEventsDigest:        "sha256:audit", TombstonesTotal: 3,
		TombstonesReplayed: 2, AlreadyAbsent: 1,
		DeletionOperationsReconciledCount: 3, AuditEventsVerifiedCount: 3,
		EvidenceDigest: "sha256:evidence", EvidenceRef: "repository-evidence",
		CompletedAt: now,
	}
	if err := repository.InsertDeletionReplayEvidence(ctx, tx, evidence); err != nil {
		t.Fatalf("insert deletion replay evidence model: %v", err)
	}
	if err := repository.SetDeletionReplayRunCurrent(
		ctx, tx, run.RunID, run.Namespace, "sha256:input", evidence.EvidenceRef, now,
	); err != nil {
		t.Fatalf("set current deletion replay evidence: %v", err)
	}
}

func TestArticleModels_MinimalEvidence_ExcludeDerivedAndSensitiveFields(t *testing.T) {
	forbiddenFields(t, reflect.TypeFor[Article](),
		"Visibility", "HiddenReasons", "ModerationStatus", "Editable")
	forbiddenFieldFragments(t, reflect.TypeFor[ArticleDeletionTombstone](),
		"Title", "Body", "Content", "Summary", "Author", "Actor",
		"Username", "Email", "Phone")
	forbiddenFieldFragments(t, reflect.TypeFor[DeletionReplayEvidence](),
		"Title", "Body", "Content", "Summary", "Author", "Actor",
		"Username", "Email", "Phone")
}

func testArticleStateConstraints(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	fixture articleUserFixture,
	now time.Time,
) {
	insertArticle(t, ctx, tx, articleRow{
		id: randomUUID(t), authorID: fixture.authorID, title: "published",
		body: "synthetic body", publication: "published", violation: "clear",
		deletion: "active", version: 1, now: now,
	})
	insertArticle(t, ctx, tx, articleRow{
		id: randomUUID(t), title: "anonymous", body: "synthetic body",
		publication: "published", violation: "clear", deletion: "active",
		version: 1, now: now,
	})
	insertArticle(t, ctx, tx, articleRow{
		id: randomUUID(t), orphanedFromUserID: fixture.deletedUserID,
		title: "held", body: "synthetic body", publication: "held",
		violation: "violating", violationCategory: "violence",
		deletion: "active", version: 1, now: now,
	})

	cases := []articleRow{
		{id: randomUUID(t), authorID: fixture.authorID, title: "held", body: "body",
			publication: "held", violation: "clear", deletion: "active", version: 1, now: now},
		{id: randomUUID(t), title: "held", body: "body", publication: "held",
			violation: "clear", deletion: "active", version: 1, now: now},
		{id: randomUUID(t), authorID: fixture.authorID, orphanedFromUserID: fixture.deletedUserID,
			title: "source", body: "body", publication: "published",
			violation: "clear", deletion: "active", version: 1, now: now},
		{id: randomUUID(t), title: "category", body: "body", publication: "published",
			violation: "clear", violationCategory: "sexual", deletion: "active", version: 1, now: now},
		{id: randomUUID(t), title: "missing category", body: "body", publication: "published",
			violation: "violating", deletion: "active", version: 1, now: now},
		{id: randomUUID(t), title: "bad category", body: "body", publication: "published",
			violation: "violating", violationCategory: "other", deletion: "active", version: 1, now: now},
		{id: randomUUID(t), title: "bad version", body: "body", publication: "published",
			violation: "clear", deletion: "active", version: 0, now: now},
		{id: randomUUID(t), title: " ", body: "body", publication: "published",
			violation: "clear", deletion: "active", version: 1, now: now},
		{id: randomUUID(t), title: "title", body: "\t", publication: "published",
			violation: "clear", deletion: "active", version: 1, now: now},
	}
	for _, row := range cases {
		requireSQLState(t, ctx, tx, sqlStateCheckViolation, articleInsertSQL, row.args()...)
	}

	unknown := randomUUID(t)
	requireSQLState(t, ctx, tx, sqlStateForeignKeyViolation, articleInsertSQL,
		articleRow{id: randomUUID(t), authorID: unknown, title: "unknown", body: "body",
			publication: "published", violation: "clear", deletion: "active",
			version: 1, now: now}.args()...)
	testPendingDeletionReference(t, ctx, tx, fixture, now)
}

func testPendingDeletionReference(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	fixture articleUserFixture,
	now time.Time,
) {
	articleID := randomUUID(t)
	operationID := randomUUID(t)
	insertDeletionOperation(t, ctx, tx, deletionOperationRow{
		id: operationID, articleID: articleID, articleVersion: 1,
		deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
		status: "pending_journal", startedAt: now,
	})
	insertArticle(t, ctx, tx, articleRow{
		id: articleID, authorID: fixture.authorID, title: "pending", body: "body",
		publication: "published", violation: "clear", deletion: "deletion_pending",
		deletionOperationID: operationID, version: 2, now: now,
	})
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, articleInsertSQL,
		articleRow{id: randomUUID(t), title: "missing operation", body: "body",
			publication: "published", violation: "clear", deletion: "deletion_pending",
			version: 1, now: now}.args()...)
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, articleInsertSQL,
		articleRow{id: randomUUID(t), title: "active operation", body: "body",
			publication: "published", violation: "clear", deletion: "active",
			deletionOperationID: operationID, version: 1, now: now}.args()...)
	requireSQLState(t, ctx, tx, sqlStateForeignKeyViolation, articleInsertSQL,
		articleRow{id: randomUUID(t), title: "mismatched operation", body: "body",
			publication: "published", violation: "clear", deletion: "deletion_pending",
			deletionOperationID: operationID, version: 1, now: now}.args()...)
}

func testTransferRecipientConstraints(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	fixture articleUserFixture,
	now time.Time,
) {
	noticeID := randomUUID(t)
	if _, err := tx.Exec(ctx, transferNoticeInsertSQL,
		noticeID, randomUUID(t), fixture.newAuthorID, fixture.deletedUserID, now, now,
	); err != nil {
		t.Fatalf("insert transfer notice: %v", err)
	}
	insertRecipient(t, ctx, tx, recipientRow{
		noticeID: noticeID, kind: "new_author", email: "new-author@example.test",
		status: "pending", attemptCount: 0, maxAttempts: 3, now: now,
	})
	insertRecipient(t, ctx, tx, recipientRow{
		noticeID: noticeID, kind: "former_deleted_author", email: "former@example.test",
		status: "processing", claimedBy: "worker-one", lockedUntil: integrationTimePointer(now.Add(time.Minute)),
		attemptCount: 1, maxAttempts: 3, lastAttemptAt: integrationTimePointer(now), now: now,
	})

	requireDeferredSQLState(t, ctx, tx, sqlStateCheckViolation, func() error {
		_, err := tx.Exec(ctx, transferNoticeInsertSQL,
			randomUUID(t), randomUUID(t), fixture.newAuthorID, fixture.deletedUserID,
			now, now,
		)
		return err
	})
	requireDeferredSQLState(t, ctx, tx, sqlStateCheckViolation, func() error {
		oneRecipientNoticeID := randomUUID(t)
		if _, err := tx.Exec(ctx, transferNoticeInsertSQL,
			oneRecipientNoticeID, randomUUID(t), fixture.newAuthorID,
			fixture.deletedUserID, now, now,
		); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, recipientInsertSQL, recipientRow{
			noticeID: oneRecipientNoticeID, kind: "new_author",
			email: "one@example.test", status: "pending", maxAttempts: 3, now: now,
		}.args()...)
		return err
	})
	requireDeferredSQLState(t, ctx, tx, sqlStateCheckViolation, func() error {
		_, err := tx.Exec(
			ctx,
			`DELETE FROM transfer_notice_recipients
			 WHERE notice_id = $1 AND recipient_kind = 'new_author'`,
			noticeID,
		)
		return err
	})

	requireSQLState(t, ctx, tx, sqlStateUniqueViolation, recipientInsertSQL,
		recipientRow{noticeID: noticeID, kind: "new_author", email: "duplicate@example.test",
			status: "pending", maxAttempts: 3, now: now}.args()...)
	cases := []recipientRow{
		{noticeID: noticeID, kind: "other", email: "kind@example.test",
			status: "pending", maxAttempts: 3, now: now},
		{noticeID: noticeID, kind: "former_deleted_author", email: "status@example.test",
			status: "sent", maxAttempts: 3, now: now},
		{noticeID: noticeID, kind: "former_deleted_author", email: "attempt@example.test",
			status: "pending", attemptCount: 4, maxAttempts: 3, now: now},
		{noticeID: noticeID, kind: "former_deleted_author", email: "processing@example.test",
			status: "processing", attemptCount: 1, maxAttempts: 3, now: now},
		{noticeID: noticeID, kind: "former_deleted_author", email: "retry@example.test",
			status: "retry_wait", attemptCount: 1, maxAttempts: 3, now: now},
		{noticeID: noticeID, kind: "former_deleted_author", email: "accepted@example.test",
			status: "accepted", attemptCount: 1, maxAttempts: 3, now: now},
	}
	for _, row := range cases {
		requireSQLState(t, ctx, tx, sqlStateCheckViolation, recipientInsertSQL, row.args()...)
	}

	acceptedAt := now.Add(time.Second)
	nonAcceptedStates := []struct {
		status string
		extra  string
	}{
		{status: "pending"},
		{status: "retry_wait", extra: ", next_attempt_at = now()"},
		{
			status: "processing",
			extra:  ", claimed_by = 'worker-two', locked_until = now(), last_attempt_at = now()",
		},
		{status: "failed_terminal"},
	}
	for _, state := range nonAcceptedStates {
		requireSQLState(t, ctx, tx, sqlStateCheckViolation, `
			UPDATE transfer_notice_recipients
			SET status = $3, accepted_at = $4`+state.extra+`
			WHERE notice_id = $1 AND recipient_kind = $2
		`, noticeID, "new_author", state.status, acceptedAt)
	}
}

func testDeletionOperationConstraints(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	fixture articleUserFixture,
	now time.Time,
) {
	articleID := randomUUID(t)
	pendingOperationID := randomUUID(t)
	insertDeletionOperation(t, ctx, tx, deletionOperationRow{
		id: pendingOperationID, articleID: articleID, articleVersion: 1,
		deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
		status: "pending_journal", startedAt: now,
	})
	insertDeletionOperation(t, ctx, tx, deletionOperationRow{
		id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
		deleteKind: "ordinary", status: "journaled",
		tombstoneSequence: integrationInt64Pointer(1), journalKeyID: "key-one",
		journalKeyVersion: "v1", startedAt: now,
	})
	requireSQLState(t, ctx, tx, sqlStateUniqueViolation, deletionOperationInsertSQL,
		deletionOperationRow{id: randomUUID(t), articleID: articleID, articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "pending_journal", startedAt: now}.args()...)

	cases := []deletionOperationRow{
		{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 0,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "pending_journal", startedAt: now},
		{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "soft", actorUserID: fixture.actorID, actorRole: "admin",
			status: "pending_journal", startedAt: now},
		{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "orphan", actorUserID: fixture.actorID, actorRole: "admin",
			status: "pending_journal", startedAt: now},
		{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			orphanedFromUserID: fixture.deletedUserID, status: "pending_journal", startedAt: now},
		{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "journaled", tombstoneSequence: integrationInt64Pointer(1), startedAt: now},
		{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "finalizing", tombstoneSequence: integrationInt64Pointer(1),
			journalKeyID: "key-one", journalKeyVersion: "v1", startedAt: now},
		{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "completed", tombstoneSequence: integrationInt64Pointer(1),
			journalKeyID: "key-one", journalKeyVersion: "v1", startedAt: now},
	}
	for _, row := range cases {
		requireSQLState(t, ctx, tx, sqlStateCheckViolation, deletionOperationInsertSQL, row.args()...)
	}

	requireSQLState(t, ctx, tx, sqlStateCheckViolation, deletionOperationInsertSQL,
		deletionOperationRow{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "pending_journal", tombstoneSequence: integrationInt64Pointer(1),
			journalKeyID: "key-one", journalKeyVersion: "v1", startedAt: now}.args()...)
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, deletionOperationInsertSQL,
		deletionOperationRow{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "journaled", tombstoneSequence: integrationInt64Pointer(1),
			journalKeyID: "key-one", journalKeyVersion: "v1",
			claimedBy: "worker-one", lockedUntil: integrationTimePointer(now.Add(time.Minute)),
			startedAt: now}.args()...)
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, deletionOperationInsertSQL,
		deletionOperationRow{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "journaled", tombstoneSequence: integrationInt64Pointer(1),
			journalKeyID: "key-one", journalKeyVersion: "v1",
			claimedBy: "worker-one", startedAt: now}.args()...)
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, deletionOperationInsertSQL,
		deletionOperationRow{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "journaled", tombstoneSequence: integrationInt64Pointer(1),
			journalKeyID: "key-one", journalKeyVersion: "v1",
			auditEventID: randomUUID(t), startedAt: now,
			completedAt: integrationTimePointer(now.Add(time.Minute))}.args()...)
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, deletionOperationInsertSQL,
		deletionOperationRow{id: randomUUID(t), articleID: randomUUID(t), articleVersion: 1,
			deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
			status: "journaled", tombstoneSequence: integrationInt64Pointer(1),
			journalKeyID: "key-one", journalKeyVersion: "v1",
			auditEventID: randomUUID(t), startedAt: now}.args()...)

	requireSQLState(t, ctx, tx, sqlStateObjectStatePrerequisite,
		"UPDATE deletion_operations SET article_version = 2 WHERE id = $1",
		pendingOperationID)
	requireSQLState(t, ctx, tx, sqlStateObjectStatePrerequisite,
		"DELETE FROM deletion_operations WHERE id = $1", pendingOperationID)

	auditEventID := insertDeletionAuditEvent(t, ctx, tx, fixture.actorID, now)
	completedOperationID := randomUUID(t)
	insertDeletionOperation(t, ctx, tx, deletionOperationRow{
		id: completedOperationID, articleID: randomUUID(t), articleVersion: 1,
		deleteKind: "ordinary", actorUserID: fixture.actorID, actorRole: "admin",
		status: "completed", tombstoneSequence: integrationInt64Pointer(1),
		journalKeyID: "key-one", journalKeyVersion: "v1",
		auditEventID: auditEventID, startedAt: now,
		completedAt: integrationTimePointer(now.Add(time.Minute)),
	})
	requireSQLState(t, ctx, tx, sqlStateObjectStatePrerequisite, `
		UPDATE deletion_operations
		SET status = 'finalizing', claimed_by = 'worker-two',
			locked_until = $2, audit_event_id = NULL, completed_at = NULL
		WHERE id = $1
	`, completedOperationID, now.Add(2*time.Minute))
}

func testReplayConstraints(t *testing.T, ctx context.Context, tx pgx.Tx, now time.Time) {
	runID := "run-" + randomHex(t, 8)
	insertReplayRun(t, ctx, tx, runID, "backup-one", "running", now)
	requireSQLState(t, ctx, tx, sqlStateUniqueViolation, replayRunInsertSQL,
		runID, "article-deletion/v1", "backup-one", "running", now, now)
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, replayRunInsertSQL,
		"wrong-namespace", "recovery/v1", "backup-one", "running", now, now)
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, replayRunInsertSQL,
		"wrong-status", "article-deletion/v1", "backup-one", "complete", now, now)
	requireSQLState(t, ctx, tx, sqlStateObjectStatePrerequisite,
		"UPDATE deletion_replay_runs SET backup_ref = 'backup-two' WHERE run_id = $1", runID)
	requireSQLState(t, ctx, tx, sqlStateObjectStatePrerequisite,
		"DELETE FROM deletion_replay_runs WHERE run_id = $1", runID)

	evidenceRef := "evidence-" + randomHex(t, 8)
	insertReplayEvidence(t, ctx, tx, replayEvidenceRow{
		evidenceRef: evidenceRef, runID: runID, backupRef: "backup-one",
		digest: "sha256:one", completedAt: now,
	})
	requireSQLState(t, ctx, tx, sqlStateUniqueViolation, replayEvidenceInsertSQL,
		replayEvidenceRow{evidenceRef: "duplicate-" + randomHex(t, 8), runID: runID,
			backupRef: "backup-one", digest: "sha256:one", completedAt: now}.args()...)
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, replayEvidenceInsertSQL,
		replayEvidenceRow{evidenceRef: "bad-count-" + randomHex(t, 8), runID: runID,
			backupRef: "backup-one", digest: "sha256:bad-count",
			tombstonesTotal: 3, tombstonesReplayed: 1, alreadyAbsent: 1,
			completedAt: now}.args()...)
	requireSQLState(t, ctx, tx, sqlStateForeignKeyViolation, replayEvidenceInsertSQL,
		replayEvidenceRow{evidenceRef: "bad-backup-" + randomHex(t, 8), runID: runID,
			backupRef: "backup-two", digest: "sha256:bad-backup", completedAt: now}.args()...)
	requireSQLState(t, ctx, tx, sqlStateObjectStatePrerequisite,
		"UPDATE deletion_replay_evidence SET evidence_digest = 'sha256:changed' WHERE evidence_ref = $1",
		evidenceRef)
	requireSQLState(t, ctx, tx, sqlStateObjectStatePrerequisite,
		"DELETE FROM deletion_replay_evidence WHERE evidence_ref = $1", evidenceRef)

	if _, err := tx.Exec(ctx, `
		UPDATE deletion_replay_runs
		SET input_digest = 'sha256:input', current_evidence_ref = $2,
			status = 'ready', updated_at = $3
		WHERE run_id = $1 AND namespace = 'article-deletion/v1'
	`, runID, evidenceRef, now); err != nil {
		t.Fatalf("bind current replay evidence: %v", err)
	}
	testReplayCurrentOwnership(t, ctx, tx, evidenceRef, now)
}

func insertDeletionAuditEvent(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	actorID string,
	now time.Time,
) string {
	t.Helper()
	eventID := randomUUID(t)
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			id, occurred_at, actor_kind, actor_user_id, actor_role, action,
			target_type, target_id, outcome, trace_id
		) VALUES ($1, $2, 'user', $3, 'admin', 'article.delete',
			'article', $4, 'completed', $5)
	`, eventID, now, actorID, randomUUID(t), "trace-"+randomHex(t, 8)); err != nil {
		t.Fatalf("insert deletion audit event: %v", err)
	}
	return eventID
}

func testReplayCurrentOwnership(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	evidenceRef string,
	now time.Time,
) {
	t.Helper()
	otherRunID := "run-" + randomHex(t, 8)
	insertReplayRun(t, ctx, tx, otherRunID, "backup-one", "running", now)
	requireSQLState(t, ctx, tx, sqlStateForeignKeyViolation, `
		UPDATE deletion_replay_runs
		SET input_digest = 'sha256:input', current_evidence_ref = $2,
			status = 'ready', updated_at = $3
		WHERE run_id = $1 AND namespace = 'article-deletion/v1'
	`, otherRunID, evidenceRef, now)
}

func testArticleIndexesAndMinimalData(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	schema string,
) {
	indexes := []struct {
		table    string
		name     string
		fragment string
	}{
		{"articles", "articles_created_at_id_idx", "(created_at DESC, id DESC)"},
		{"articles", "articles_author_id_idx", "(author_id)"},
		{"articles", "articles_orphan_source_idx", "(orphaned_from_user_id, created_at DESC, id DESC)"},
		{"articles", "articles_violation_state_idx", "(violation_state, created_at DESC, id DESC)"},
		{"articles", "articles_deletion_state_idx", "(deletion_state, created_at DESC, id DESC)"},
		{"transfer_notice_recipients", "transfer_notice_recipients_claim_idx", "(status, next_attempt_at, locked_until)"},
		{"deletion_operations", "deletion_operations_finalizer_idx", "(status, locked_until, started_at, id)"},
		{"deletion_replay_evidence", "deletion_replay_evidence_run_idx", "(run_id, namespace, completed_at DESC)"},
	}
	for _, index := range indexes {
		requireIndex(t, ctx, tx, schema, index.table, index.name, index.fragment)
	}

	requireColumns(t, ctx, tx, schema, "articles", []string{
		"author_id", "body", "created_at", "deletion_operation_id", "deletion_state",
		"id", "orphaned_from_user_id", "publication_state", "title", "updated_at",
		"version", "violation_category", "violation_state",
	})
	requireColumns(t, ctx, tx, schema, "deletion_replay_runs", []string{
		"attempt_generation", "attempt_token", "backup_ref", "current_evidence_ref",
		"input_digest", "last_error_bucket", "namespace", "run_id", "started_at",
		"status", "updated_at",
	})
	requireColumns(t, ctx, tx, schema, "deletion_replay_evidence", []string{
		"already_absent", "audit_events_digest", "backup_cutoff", "backup_ref",
		"completed_at", "deletion_operations_digest",
		"deletion_operations_reconciled_count", "evidence_digest", "evidence_ref",
		"evidence_version", "inventory_digest", "inventory_scope",
		"journal_head_hash", "journal_head_sequence", "namespace", "run_id",
		"audit_events_verified_count", "tombstones_replayed", "tombstones_total",
	})
	var tombstoneTable *string
	if err := tx.QueryRow(ctx, "SELECT to_regclass('article_deletion_tombstones')::text").
		Scan(&tombstoneTable); err != nil {
		t.Fatalf("query external tombstone relation: %v", err)
	}
	if tombstoneTable != nil {
		t.Fatalf("external tombstone unexpectedly persisted in PostgreSQL: %s", *tombstoneTable)
	}
}

type articleUserFixture struct {
	authorID      string
	newAuthorID   string
	deletedUserID string
	actorID       string
}

func seedArticleUsers(t *testing.T, ctx context.Context, tx pgx.Tx) articleUserFixture {
	t.Helper()
	now := time.Date(2026, time.October, 22, 7, 0, 0, 0, time.UTC)
	fixture := articleUserFixture{
		authorID: randomUUID(t), newAuthorID: randomUUID(t),
		deletedUserID: randomUUID(t), actorID: randomUUID(t),
	}
	insertTestUser(t, ctx, tx, fixture.authorID, "article-author", "member", "active", now)
	insertTestUser(t, ctx, tx, fixture.newAuthorID, "new-author", "member", "active", now)
	insertTestUser(t, ctx, tx, fixture.actorID, "article-admin", "admin", "active", now)
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (
			id, username, role, is_initial_admin, status, credential_mode,
			recovery_required, retained_email, created_at, updated_at, deleted_at
		) VALUES ($1, 'deleted-author', 'member', false, 'deleted', 'permanent',
			false, 'former@example.test', $2, $2, $2)
	`, fixture.deletedUserID, now); err != nil {
		t.Fatalf("insert deleted user tombstone: %v", err)
	}
	return fixture
}

func insertTestUser(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	id string,
	username string,
	role string,
	status string,
	now time.Time,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (
			id, username, email, phone, role, is_initial_admin, status,
			credential_mode, recovery_required, created_at, updated_at
		) VALUES ($1, $2, $3, '+15550000001', $4, false, $5, 'permanent',
			false, $6, $6)
	`, id, username, username+"@example.test", role, status, now); err != nil {
		t.Fatalf("insert article test user: %v", err)
	}
}

type articleRow struct {
	id                  string
	authorID            any
	orphanedFromUserID  any
	title               string
	body                string
	publication         string
	violation           string
	violationCategory   any
	deletion            string
	deletionOperationID any
	version             int64
	now                 time.Time
}

func (row articleRow) args() []any {
	return []any{
		row.id, row.authorID, row.orphanedFromUserID, row.title, row.body,
		row.publication, row.violation, row.violationCategory, row.deletion,
		row.deletionOperationID, row.version, row.now, row.now,
	}
}

func insertArticle(t *testing.T, ctx context.Context, tx pgx.Tx, row articleRow) {
	t.Helper()
	if _, err := tx.Exec(ctx, articleInsertSQL, row.args()...); err != nil {
		t.Fatalf("insert legal article: %v", err)
	}
}

type recipientRow struct {
	noticeID      string
	kind          string
	email         string
	status        string
	claimedBy     any
	lockedUntil   any
	nextAttemptAt any
	attemptCount  int64
	maxAttempts   int64
	lastAttemptAt any
	acceptedAt    any
	failureBucket any
	now           time.Time
}

func (row recipientRow) args() []any {
	return []any{
		row.noticeID, row.kind, row.email, row.status, row.claimedBy,
		row.lockedUntil, row.nextAttemptAt, row.attemptCount, row.maxAttempts,
		row.lastAttemptAt, row.acceptedAt, row.failureBucket, row.now, row.now,
	}
}

func insertRecipient(t *testing.T, ctx context.Context, tx pgx.Tx, row recipientRow) {
	t.Helper()
	if _, err := tx.Exec(ctx, recipientInsertSQL, row.args()...); err != nil {
		t.Fatalf("insert legal transfer recipient: %v", err)
	}
}

type deletionOperationRow struct {
	id                 string
	articleID          string
	articleVersion     int64
	deleteKind         string
	actorUserID        any
	actorRole          any
	orphanedFromUserID any
	status             string
	tombstoneSequence  any
	journalKeyID       any
	journalKeyVersion  any
	claimedBy          any
	lockedUntil        any
	auditEventID       any
	startedAt          time.Time
	completedAt        any
}

func (row deletionOperationRow) args() []any {
	return []any{
		row.id, row.articleID, row.articleVersion, row.deleteKind,
		row.actorUserID, row.actorRole, row.orphanedFromUserID, row.status,
		row.tombstoneSequence, row.journalKeyID, row.journalKeyVersion,
		row.claimedBy, row.lockedUntil, row.auditEventID, row.startedAt,
		row.completedAt,
	}
}

func insertDeletionOperation(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	row deletionOperationRow,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, deletionOperationInsertSQL, row.args()...); err != nil {
		t.Fatalf("insert legal deletion operation: %v", err)
	}
}

type replayEvidenceRow struct {
	evidenceRef        string
	runID              string
	backupRef          string
	digest             string
	tombstonesTotal    int64
	tombstonesReplayed int64
	alreadyAbsent      int64
	completedAt        time.Time
}

func (row replayEvidenceRow) args() []any {
	total := row.tombstonesTotal
	replayed := row.tombstonesReplayed
	absent := row.alreadyAbsent
	if total == 0 && replayed == 0 && absent == 0 {
		total, replayed, absent = 3, 2, 1
	}
	return []any{
		"article-deletion-replay/v1", row.runID, row.backupRef,
		"article-deletion/v1", 7, "sha256:head", "recoverable-backups-before-cutoff",
		row.completedAt, "sha256:inventory", "sha256:operations", "sha256:audit",
		total, replayed, absent, total, total, row.digest, row.evidenceRef,
		row.completedAt,
	}
}

func insertReplayRun(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	runID string,
	backupRef string,
	status string,
	now time.Time,
) {
	t.Helper()
	if _, err := tx.Exec(
		ctx, replayRunInsertSQL,
		runID, "article-deletion/v1", backupRef, status, now, now,
	); err != nil {
		t.Fatalf("insert legal deletion replay run: %v", err)
	}
}

func insertReplayEvidence(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	row replayEvidenceRow,
) {
	t.Helper()
	if _, err := tx.Exec(ctx, replayEvidenceInsertSQL, row.args()...); err != nil {
		t.Fatalf("insert legal deletion replay evidence: %v", err)
	}
}

func newArticleTestTransaction(t *testing.T) (context.Context, pgx.Tx, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	pool := newArticleTestPool(t, ctx)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin article schema test transaction: %v", err)
	}
	schema := "article_" + randomHex(t, 8)
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	registerArticleSchemaRollback(t, pool, tx, schema)
	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		t.Fatalf("create transaction-scoped article schema: %v", err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+quotedSchema); err != nil {
		t.Fatalf("set transaction-scoped article search path: %v", err)
	}
	for _, path := range []string{
		"../../migrations/0001_shared.sql",
		"../../migrations/0002_identity.sql",
		"../../migrations/0003_article.sql",
		"../../migrations/0009_article_replay_attempt.sql",
	} {
		if _, err := tx.Exec(ctx, readMigration(t, path)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
	return ctx, tx, schema
}

func newArticleTestPool(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	dsn := testenv.RequirePostgresDSN(t)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal("configure PostgreSQL article test database")
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal("open PostgreSQL article test database")
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal("connect PostgreSQL article test database")
	}
	var version int
	if err := pool.QueryRow(ctx, "SELECT current_setting('server_version_num')::integer").
		Scan(&version); err != nil {
		t.Fatalf("read PostgreSQL article test server version: %v", err)
	}
	t.Logf("PostgreSQL article constraint baseline: server_version_num=%d", version)
	if version/10000 != 18 {
		t.Fatalf("PostgreSQL article test major = %d, want 18", version/10000)
	}
	return pool
}

func registerArticleSchemaRollback(
	t *testing.T,
	pool *pgxpool.Pool,
	tx pgx.Tx,
	schema string,
) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := tx.Rollback(cleanupCtx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("rollback article schema test transaction: %v", err)
			return
		}
		var remaining *string
		if err := pool.QueryRow(
			cleanupCtx, "SELECT to_regnamespace($1)::text", schema,
		).Scan(&remaining); err != nil {
			t.Errorf("verify article schema rollback: %v", err)
			return
		}
		if remaining != nil {
			t.Errorf("article schema remains after rollback: %s", *remaining)
		}
	})
}

func requireSQLState(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	wantCode string,
	query string,
	args ...any,
) {
	t.Helper()
	const savepoint = "article_constraint_probe"
	if _, err := tx.Exec(ctx, "SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("create constraint probe savepoint: %v", err)
	}
	_, execErr := tx.Exec(ctx, query, args...)
	var postgresError *pgconn.PgError
	matched := errors.As(execErr, &postgresError) && postgresError.Code == wantCode
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("rollback constraint probe savepoint: %v", err)
	}
	if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("release constraint probe savepoint: %v", err)
	}
	if execErr == nil {
		t.Fatalf("constraint probe succeeded, want SQLSTATE %s", wantCode)
	}
	if !matched {
		t.Fatalf("constraint probe SQLSTATE = %q, want %q", postgresErrorCode(execErr), wantCode)
	}
}

func requireDeferredSQLState(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	wantCode string,
	mutate func() error,
) {
	t.Helper()
	const savepoint = "article_deferred_constraint_probe"
	if _, err := tx.Exec(ctx, "SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("create deferred constraint probe savepoint: %v", err)
	}
	execErr := mutate()
	if execErr == nil {
		_, execErr = tx.Exec(ctx, "SET CONSTRAINTS ALL IMMEDIATE")
	}
	var postgresError *pgconn.PgError
	matched := errors.As(execErr, &postgresError) && postgresError.Code == wantCode
	if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("rollback deferred constraint probe savepoint: %v", err)
	}
	if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT "+savepoint); err != nil {
		t.Fatalf("release deferred constraint probe savepoint: %v", err)
	}
	if execErr == nil {
		t.Fatalf("deferred constraint probe succeeded, want SQLSTATE %s", wantCode)
	}
	if !matched {
		t.Fatalf(
			"deferred constraint probe SQLSTATE = %q, want %q",
			postgresErrorCode(execErr), wantCode,
		)
	}
}

func requireIndex(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	schema string,
	table string,
	index string,
	wantFragment string,
) {
	t.Helper()
	var definition string
	if err := tx.QueryRow(ctx, `
		SELECT indexdef
		FROM pg_indexes
		WHERE schemaname = $1 AND tablename = $2 AND indexname = $3
	`, schema, table, index).Scan(&definition); err != nil {
		t.Fatalf("query index %s: %v", index, err)
	}
	if !strings.Contains(definition, wantFragment) {
		t.Fatalf("index %s definition = %q, want fragment %q", index, definition, wantFragment)
	}
}

func requireColumns(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	schema string,
	table string,
	want []string,
) {
	t.Helper()
	rows, err := tx.Query(ctx, `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY column_name
	`, schema, table)
	if err != nil {
		t.Fatalf("query columns for %s: %v", table, err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatalf("scan column for %s: %v", table, err)
		}
		got = append(got, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns for %s: %v", table, err)
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("%s columns = %v, want %v", table, got, want)
	}
}

func assertRowCount(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	table string,
	want int,
) {
	t.Helper()
	var got int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()).
		Scan(&got); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s row count = %d, want %d", table, got, want)
	}
}

func forbiddenFields(t *testing.T, model reflect.Type, names ...string) {
	t.Helper()
	for _, name := range names {
		if _, exists := model.FieldByName(name); exists {
			t.Fatalf("%s persists forbidden field %s", model.Name(), name)
		}
	}
}

func forbiddenFieldFragments(t *testing.T, model reflect.Type, fragments ...string) {
	t.Helper()
	for fieldIndex := 0; fieldIndex < model.NumField(); fieldIndex++ {
		fieldName := strings.ToLower(model.Field(fieldIndex).Name)
		for _, fragment := range fragments {
			if strings.Contains(fieldName, strings.ToLower(fragment)) {
				t.Fatalf("%s persists forbidden field %s", model.Name(), model.Field(fieldIndex).Name)
			}
		}
	}
}

func readMigration(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration %s: %v", path, err)
	}
	return string(content)
}

func postgresErrorCode(err error) string {
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return ""
	}
	return postgresError.Code
}

func randomUUID(t *testing.T) string {
	t.Helper()
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatalf("generate synthetic UUID: %v", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value[:])
	return strings.Join([]string{
		encoded[0:8], encoded[8:12], encoded[12:16],
		encoded[16:20], encoded[20:32],
	}, "-")
}

func randomHex(t *testing.T, size int) string {
	t.Helper()
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		t.Fatalf("generate article schema suffix: %v", err)
	}
	return hex.EncodeToString(value)
}

func integrationTimePointer(value time.Time) *time.Time {
	return &value
}

func integrationInt64Pointer(value int64) *int64 {
	return &value
}

func integrationStringPointer(value string) *string {
	return &value
}

const articleInsertSQL = `
	INSERT INTO articles (
		id, author_id, orphaned_from_user_id, title, body, publication_state,
		violation_state, violation_category, deletion_state,
		deletion_operation_id, version, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

const transferNoticeInsertSQL = `
	INSERT INTO transfer_notices (
		id, article_ref, new_author_id, former_deleted_user_id, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6)
`

const recipientInsertSQL = `
	INSERT INTO transfer_notice_recipients (
		notice_id, recipient_kind, email_snapshot, status, claimed_by,
		locked_until, next_attempt_at, attempt_count, max_attempts,
		last_attempt_at, accepted_at, failure_bucket, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`

const deletionOperationInsertSQL = `
	INSERT INTO deletion_operations (
		id, article_id, article_version, delete_kind, actor_user_id, actor_role,
		orphaned_from_user_id, status, tombstone_sequence, journal_key_id,
		journal_key_version, claimed_by, locked_until, audit_event_id,
		started_at, completed_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8,
		$9, $10, $11, $12, $13, $14, $15, $16
	)
`

const replayRunInsertSQL = `
	INSERT INTO deletion_replay_runs (
		run_id, namespace, backup_ref, status, started_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6)
`

const replayEvidenceInsertSQL = `
	INSERT INTO deletion_replay_evidence (
		evidence_version, run_id, backup_ref, namespace, journal_head_sequence,
		journal_head_hash, inventory_scope, backup_cutoff, inventory_digest,
		deletion_operations_digest, audit_events_digest, tombstones_total,
		tombstones_replayed, already_absent,
		deletion_operations_reconciled_count, audit_events_verified_count,
		evidence_digest, evidence_ref, completed_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19
	)
`

func TestArticleRepository_ReplayAttemptConstraints(t *testing.T) {
	ctx, tx, _ := newArticleTestTransaction(t)
	now := time.Date(2026, time.October, 22, 7, 0, 0, 0, time.UTC)
	runID := randomUUID(t)
	insert := `
		INSERT INTO deletion_replay_runs (
			run_id, namespace, backup_ref, status, started_at, updated_at,
			attempt_generation, attempt_token
		) VALUES ($1, $2, $3, 'running', $4, $4, $5, $6)
	`
	// Valid: legacy generation=0 with a null token.
	if _, err := tx.Exec(ctx, insert, runID, "article-deletion/v1", "backup-1", now, 0, nil); err != nil {
		t.Fatalf("legacy attempt insert: %v", err)
	}
	// Valid: generation>0 with a non-empty token.
	if _, err := tx.Exec(ctx, insert, randomUUID(t), "article-deletion/v1", "backup-1", now, 3, "token-3"); err != nil {
		t.Fatalf("owned attempt insert: %v", err)
	}
	// Invalid: generation>0 with a null token.
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, insert,
		randomUUID(t), "article-deletion/v1", "backup-1", now, 1, nil)
	// Invalid: generation>0 with a blank token.
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, insert,
		randomUUID(t), "article-deletion/v1", "backup-1", now, 1, "  ")
	// Invalid: generation=0 with a token.
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, insert,
		randomUUID(t), "article-deletion/v1", "backup-1", now, 0, "token-1")
	// Invalid: negative generation.
	requireSQLState(t, ctx, tx, sqlStateCheckViolation, insert,
		randomUUID(t), "article-deletion/v1", "backup-1", now, -1, "token-1")
}
