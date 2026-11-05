package identity

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
)

const (
	dispositionSourceID       = "00000000-0000-4000-8000-000000000002"
	dispositionTargetID       = "00000000-0000-4000-8000-000000000001"
	dispositionArticleID      = "00000000-0000-4000-8000-000000000003"
	dispositionNoticeID       = "00000000-0000-4000-8000-000000000004"
	dispositionOperationID    = "00000000-0000-4000-8000-000000000005"
	dispositionAuditID        = "00000000-0000-4000-8000-000000000006"
	dispositionReleaseAuditID = "00000000-0000-4000-8000-000000000007"
)

func TestArticleDispositionLockParties_ValidContacts_OrderedSQL(t *testing.T) {
	transaction := newDispositionTransaction()
	targetID := dispositionTargetID
	parties, err := (&ArticleIdentityLifecycle{}).LockTransferParties(
		context.Background(), transaction, &targetID, dispositionSourceID)
	if err != nil || parties.SourceEmail == nil || parties.TargetEmail == nil {
		t.Fatalf("lock contacts failed: %v", err)
	}
	want := []string{"user:" + targetID, "user:" + dispositionSourceID,
		"identifiers:" + targetID, "identifiers:" + dispositionSourceID}
	if !reflect.DeepEqual(transaction.calls, want) || transaction.closed {
		t.Fatalf("unexpected lock ordering or transaction ownership: %v", transaction.calls)
	}
}

func TestArticleDispositionLockParties_InvalidIdentity_Rejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*dispositionTransaction)
	}{
		{"source active", func(transaction *dispositionTransaction) {
			user := transaction.users[dispositionSourceID]
			user.Status = UserStatusActive
			transaction.users[dispositionSourceID] = user
		}},
		{"target suspended", func(transaction *dispositionTransaction) {
			user := transaction.users[dispositionTargetID]
			user.Status = UserStatusSuspended
			transaction.users[dispositionTargetID] = user
		}},
		{"source email missing", func(transaction *dispositionTransaction) {
			transaction.identifiers[dispositionSourceID] = transaction.identifiers[dispositionSourceID][:1]
		}},
		{"source binding drift", func(transaction *dispositionTransaction) {
			transaction.identifiers[dispositionSourceID][1].NormalizedValue = "other@example.test"
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			transaction := newDispositionTransaction()
			testCase.mutate(transaction)
			targetID := dispositionTargetID
			_, err := (&ArticleIdentityLifecycle{}).LockTransferParties(
				context.Background(), transaction, &targetID, dispositionSourceID)
			if err == nil || len(transaction.writes) != 0 || transaction.closed {
				t.Fatalf("invalid identity accepted or transaction closed: %v", err)
			}
		})
	}
}

type dispositionTransaction struct {
	pgx.Tx
	users              map[string]User
	identifiers        map[string][]IdentityIdentifier
	calls              []string
	writes             []string
	closed             bool
	queryErr           error
	rowErr             error
	rowsErr            error
	scanErr            error
	execErr            error
	execFailAt         int
	zeroAt             int
	writeArgs          [][]any
	initialUsers       map[string]User
	initialIdentifiers map[string][]IdentityIdentifier
	contexts           []context.Context
}

func newDispositionTransaction() *dispositionTransaction {
	now := time.Date(2026, time.October, 27, 0, 0, 0, 0, time.UTC)
	retainedEmail, targetEmail := "Former@Example.test", "New@Example.test"
	return &dispositionTransaction{
		users: map[string]User{
			dispositionSourceID: {ID: dispositionSourceID, Username: "Former",
				Status: UserStatusDeleted, DeletedAt: &now, RetainedEmail: &retainedEmail},
			dispositionTargetID: {ID: dispositionTargetID, Username: "New",
				Status: UserStatusActive, Email: &targetEmail},
		},
		identifiers: map[string][]IdentityIdentifier{
			dispositionSourceID: {
				{UserID: dispositionSourceID, Kind: IdentifierUsername,
					NormalizedValue: "former", Lifecycle: IdentifierReserved},
				{UserID: dispositionSourceID, Kind: IdentifierEmail,
					NormalizedValue: "former@example.test", Lifecycle: IdentifierReserved},
			},
			dispositionTargetID: {
				{UserID: dispositionTargetID, Kind: IdentifierUsername,
					NormalizedValue: "new", Lifecycle: IdentifierActive},
				{UserID: dispositionTargetID, Kind: IdentifierEmail,
					NormalizedValue: "new@example.test", Lifecycle: IdentifierActive},
			},
		},
	}
}

func (transaction *dispositionTransaction) QueryRow(
	ctx context.Context, query string, arguments ...any,
) pgx.Row {
	if transaction.closed {
		return dispositionRow{err: pgx.ErrTxClosed}
	}
	transaction.contexts = append(transaction.contexts, ctx)
	if transaction.rowErr != nil {
		return dispositionRow{err: transaction.rowErr}
	}
	if !strings.Contains(query, "FROM users") || !strings.Contains(query, "FOR UPDATE NOWAIT") {
		return dispositionRow{err: errors.New("unexpected user SQL")}
	}
	userID := arguments[0].(string)
	transaction.calls = append(transaction.calls, "user:"+userID)
	user, found := transaction.users[userID]
	if !found {
		return dispositionRow{err: pgx.ErrNoRows}
	}
	return dispositionRow{values: []any{user.ID, user.Username, user.Email, user.Phone,
		user.Status, user.RetainedEmail, user.DeletedAt}}
}

func (transaction *dispositionTransaction) Query(
	ctx context.Context, query string, arguments ...any,
) (pgx.Rows, error) {
	if transaction.closed {
		return nil, pgx.ErrTxClosed
	}
	transaction.contexts = append(transaction.contexts, ctx)
	if transaction.queryErr != nil {
		return nil, transaction.queryErr
	}
	if !strings.Contains(query, "FROM identity_identifiers") ||
		!strings.Contains(query, "ORDER BY user_id, kind") || !strings.Contains(query, "FOR UPDATE NOWAIT") {
		return nil, errors.New("unexpected identifier SQL")
	}
	userID := arguments[0].(string)
	transaction.calls = append(transaction.calls, "identifiers:"+userID)
	return &dispositionRows{identifiers: transaction.identifiers[userID],
		err: transaction.rowsErr, scanErr: transaction.scanErr}, nil
}

func (transaction *dispositionTransaction) Exec(
	ctx context.Context, query string, arguments ...any,
) (pgconn.CommandTag, error) {
	if transaction.closed {
		return pgconn.CommandTag{}, pgx.ErrTxClosed
	}
	transaction.contexts = append(transaction.contexts, ctx)
	transaction.writes = append(transaction.writes, query)
	transaction.writeArgs = append(transaction.writeArgs, arguments)
	if transaction.execFailAt == len(transaction.writes) {
		return pgconn.CommandTag{}, transaction.execErr
	}
	if transaction.zeroAt == len(transaction.writes) {
		return pgconn.NewCommandTag("UPDATE 0"), nil
	}
	if strings.Contains(query, "UPDATE users") {
		user := transaction.users[arguments[0].(string)]
		if user.RetainedEmail == nil || *user.RetainedEmail != arguments[1].(string) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		}
		user.RetainedEmail = nil
		transaction.users[user.ID] = user
	} else if strings.Contains(query, "DELETE FROM identity_identifiers") {
		userID := arguments[0].(string)
		remaining := make([]IdentityIdentifier, 0)
		for _, identifier := range transaction.identifiers[userID] {
			if identifier.Kind != IdentifierEmail {
				remaining = append(remaining, identifier)
			}
		}
		transaction.identifiers[userID] = remaining
	} else if !strings.Contains(query, "INSERT INTO audit_events") {
		return pgconn.CommandTag{}, errors.New("unexpected write SQL")
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (transaction *dispositionTransaction) Commit(context.Context) error {
	transaction.closed = true
	return nil
}

func (transaction *dispositionTransaction) Rollback(context.Context) error {
	transaction.closed = true
	transaction.writes = nil
	if transaction.initialUsers != nil {
		transaction.users = transaction.initialUsers
		transaction.identifiers = transaction.initialIdentifiers
	}
	return nil
}

func (transaction *dispositionTransaction) snapshot() {
	transaction.initialUsers = make(map[string]User)
	transaction.initialIdentifiers = make(map[string][]IdentityIdentifier)
	for userID, user := range transaction.users {
		transaction.initialUsers[userID] = user
	}
	for userID, identifiers := range transaction.identifiers {
		transaction.initialIdentifiers[userID] = append([]IdentityIdentifier(nil), identifiers...)
	}
}

type dispositionVerifier struct {
	transaction pgx.Tx
	reference   ArticleCompletionReference
	facts       VerifiedArticleCompletion
	ctx         context.Context
	calls       int
	err         error
}

func (verifier *dispositionVerifier) VerifyCompletion(
	ctx context.Context, transaction pgx.Tx, reference ArticleCompletionReference,
) (VerifiedArticleCompletion, error) {
	verifier.calls++
	if verifier.err != nil {
		return VerifiedArticleCompletion{}, verifier.err
	}
	if ctx != verifier.ctx || transaction != verifier.transaction || !reflect.DeepEqual(reference, verifier.reference) {
		return VerifiedArticleCompletion{}, errors.New("reference does not match captured transaction preimage")
	}
	return verifier.facts, nil
}

type dispositionFixture struct {
	ctx         context.Context
	transaction *dispositionTransaction
	verifier    *dispositionVerifier
	lifecycle   *ArticleIdentityLifecycle
	reference   ArticleCompletionReference
}

func newDispositionFixture(t *testing.T, action ArticleDispositionAction) *dispositionFixture {
	t.Helper()
	ctx := newRegistrationContext(t)
	transaction := newDispositionTransaction()
	reference, facts := dispositionCompletion(transaction, action)
	facts.Audit.TraceID, _ = platform.TraceIDFromContext(ctx)
	verifier := &dispositionVerifier{transaction: transaction, reference: reference, facts: facts, ctx: ctx}
	lifecycle, err := NewArticleIdentityLifecycle(verifier, audit.Store{},
		&registrationIDStub{ids: []string{dispositionReleaseAuditID}},
		registrationClockStub{now: time.Date(2026, time.October, 27, 1, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	return &dispositionFixture{ctx: ctx, transaction: transaction, verifier: verifier,
		lifecycle: lifecycle, reference: reference}
}

func dispositionCompletion(
	transaction pgx.Tx, action ArticleDispositionAction,
) (ArticleCompletionReference, VerifiedArticleCompletion) {
	sourceID := dispositionSourceID
	before := ArticleDispositionState{Exists: true, ID: dispositionArticleID, Version: 9,
		SourceID: &sourceID, Publication: "held", Violation: "clear", Deletion: "active"}
	reference := ArticleCompletionReference{Transaction: transaction, ExpectedArticleID: dispositionArticleID,
		SourceID: sourceID, BeforeVersion: 9, Action: action, AuditEventID: dispositionAuditID, BeforeArticle: before}
	facts := VerifiedArticleCompletion{AfterArticle: ArticleDispositionState{Exists: true,
		ID: dispositionArticleID, Version: 10, Publication: "published", Violation: "clear", Deletion: "active"},
		Audit: ArticleDispositionAudit{ID: dispositionAuditID, ArticleID: dispositionArticleID,
			Action: "article.orphan.publish", Outcome: "succeeded"}}
	if action == ArticleDispositionTransfer {
		noticeID, targetID := dispositionNoticeID, dispositionTargetID
		reference.NoticeID, reference.NewAuthorID = &noticeID, &targetID
		facts.AfterArticle.AuthorID = &targetID
		facts.Audit.Action = "article.orphan.transfer"
		facts.Notice = &ArticleDispositionNotice{ID: noticeID, ArticleID: dispositionArticleID,
			NewAuthorID: targetID, FormerDeletedUserID: sourceID}
		facts.Recipients = []ArticleDispositionRecipient{
			{Kind: "new_author", EmailSnapshot: "New@Example.test", Status: "pending"},
			{Kind: "former_deleted_author", EmailSnapshot: "Former@Example.test", Status: "pending"},
		}
	}
	if action == ArticleDispositionDeleteComplete {
		operationID := dispositionOperationID
		reference.OperationID = &operationID
		reference.BeforeArticle.Version = 10
		reference.BeforeArticle.Deletion = "deletion_pending"
		reference.BeforeArticle.OperationID = &operationID
		facts.AfterArticle = ArticleDispositionState{}
		facts.Audit.Action = "article.orphan.delete"
		facts.CompletedOperation = &ArticleDispositionOperation{ID: operationID, ArticleID: dispositionArticleID,
			ArticleVersion: 9, DeleteKind: "orphan", SourceID: &sourceID, Status: "completed", AuditEventID: dispositionAuditID}
	}
	return reference, facts
}

func (fixture *dispositionFixture) clear() (bool, error) {
	return fixture.lifecycle.ClearRetainedEmailIfEligible(fixture.ctx, fixture.transaction,
		dispositionSourceID, fixture.reference)
}

func TestArticleDispositionClear_ValidCompletion_ReleasesAtomically(t *testing.T) {
	for _, action := range []ArticleDispositionAction{
		ArticleDispositionPublish, ArticleDispositionTransfer, ArticleDispositionDeleteComplete,
	} {
		t.Run(string(action), func(t *testing.T) {
			fixture := newDispositionFixture(t, action)
			phone := IdentityIdentifier{UserID: dispositionSourceID, Kind: IdentifierPhone,
				NormalizedValue: "+15550000002", Lifecycle: IdentifierReserved}
			fixture.transaction.identifiers[dispositionSourceID] = append(
				fixture.transaction.identifiers[dispositionSourceID], phone)
			cleared, err := fixture.clear()
			if err != nil || !cleared || fixture.verifier.calls != 1 || fixture.transaction.closed {
				t.Fatalf("clear result=%v, error=%v, verifier calls=%d", cleared, err, fixture.verifier.calls)
			}
			assertDispositionReleaseSQL(t, fixture)
			if fixture.transaction.users[dispositionSourceID].RetainedEmail != nil ||
				len(fixture.transaction.identifiers[dispositionSourceID]) != 2 ||
				fixture.transaction.identifiers[dispositionSourceID][1] != phone {
				t.Fatal("contact not cleared or username/phone tombstones changed")
			}
			for _, ctx := range fixture.transaction.contexts {
				if ctx != fixture.ctx {
					t.Fatal("SQL did not use caller context")
				}
			}
		})
	}
}

func assertDispositionReleaseSQL(t *testing.T, fixture *dispositionFixture) {
	t.Helper()
	transaction := fixture.transaction
	if len(transaction.writes) != 3 ||
		!strings.Contains(transaction.writes[0], "retained_email = $2") ||
		!strings.Contains(transaction.writes[0], "status = 'deleted'") ||
		!strings.Contains(transaction.writes[1], "user_id = $1 AND kind = 'email'") ||
		!strings.Contains(transaction.writes[1], "normalized_value = $2") ||
		!strings.Contains(transaction.writes[1], "lifecycle = 'reserved' AND released_at IS NULL") ||
		!strings.Contains(transaction.writes[2], "INSERT INTO audit_events") {
		t.Fatal("release did not execute conditional identity SQL and production audit store")
	}
	if transaction.writeArgs[0][0] != dispositionSourceID ||
		transaction.writeArgs[0][1] != "Former@Example.test" ||
		!reflect.DeepEqual(transaction.writeArgs[1], []any{dispositionSourceID, "former@example.test"}) {
		t.Fatal("release SQL argument binding mismatch")
	}
	auditArguments := transaction.writeArgs[2]
	if auditArguments[0] != dispositionReleaseAuditID || auditArguments[2] != audit.ActorSystem ||
		auditArguments[6] != "identity.identifier.email.release" || auditArguments[7] != "identity_identifier" ||
		auditArguments[8] != dispositionSourceID || auditArguments[9] != "succeeded" ||
		auditArguments[11] != fixture.verifier.facts.Audit.TraceID ||
		string(auditArguments[13].([]byte)) != `{"kind":"email"}` {
		t.Fatal("release audit binding or minimal metadata mismatch")
	}
}

func TestArticleDispositionClear_RemainingAssociations_Unchanged(t *testing.T) {
	for _, counts := range [][2]int64{{1, 0}, {0, 1}, {3, 4}} {
		fixture := newDispositionFixture(t, ArticleDispositionPublish)
		fixture.verifier.facts.RemainingActiveCount = counts[0]
		fixture.verifier.facts.RemainingPendingCount = counts[1]
		cleared, err := fixture.clear()
		if err != nil || cleared || fixture.verifier.calls != 1 || len(fixture.transaction.writes) != 0 ||
			fixture.transaction.users[dispositionSourceID].RetainedEmail == nil {
			t.Fatalf("nonzero counts cleanup result=%v, error=%v", cleared, err)
		}
	}
}

func TestArticleDispositionClear_Reentry_UnchangedWithoutDuplicateAudit(t *testing.T) {
	for _, action := range []ArticleDispositionAction{
		ArticleDispositionPublish, ArticleDispositionTransfer, ArticleDispositionDeleteComplete,
	} {
		t.Run(string(action), func(t *testing.T) {
			fixture := newDispositionFixture(t, action)
			if cleared, err := fixture.clear(); err != nil || !cleared {
				t.Fatalf("first clear=%v, error=%v", cleared, err)
			}
			cleared, err := fixture.clear()
			if err != nil || cleared || len(fixture.transaction.writes) != 3 || fixture.verifier.calls != 2 {
				t.Fatalf("reentry clear=%v, error=%v, verifier calls=%d", cleared, err, fixture.verifier.calls)
			}
		})
	}
}

func TestArticleDispositionClear_EmptyPublishAndDelete_VerifiesWithoutContact(t *testing.T) {
	for _, action := range []ArticleDispositionAction{ArticleDispositionPublish, ArticleDispositionDeleteComplete} {
		fixture := newDispositionFixture(t, action)
		clearDispositionContact(fixture.transaction)
		if cleared, err := fixture.clear(); err != nil || cleared || fixture.verifier.calls != 1 ||
			len(fixture.transaction.writes) != 0 {
			t.Fatalf("empty contact clear=%v, error=%v", cleared, err)
		}
	}
}

func newAbsentDispositionFixture(t *testing.T) *dispositionFixture {
	t.Helper()
	fixture := newDispositionFixture(t, ArticleDispositionDeleteComplete)
	fixture.reference.BeforeArticle = ArticleDispositionState{}
	fixture.verifier.reference = fixture.reference
	return fixture
}

func TestArticleDispositionClear_AbsentOrphan_VerifiesAndReleasesOnce(t *testing.T) {
	for _, action := range []string{"article.orphan.delete", "article.deletion.replay"} {
		t.Run(action, func(t *testing.T) {
			fixture := newAbsentDispositionFixture(t)
			fixture.verifier.facts.Audit.Action = action
			cleared, err := fixture.clear()
			if err != nil || !cleared || fixture.verifier.calls != 1 || fixture.transaction.closed {
				t.Fatalf("absent orphan clear=%v, error=%v, verifier calls=%d", cleared, err, fixture.verifier.calls)
			}
			assertDispositionReleaseSQL(t, fixture)
			cleared, err = fixture.clear()
			if err != nil || cleared || fixture.verifier.calls != 2 || len(fixture.transaction.writes) != 3 {
				t.Fatalf("absent orphan reentry clear=%v, error=%v", cleared, err)
			}
		})
	}
}

func TestArticleDispositionClear_AbsentOrphanRemainingOrCleared_VerifiesAndPreserves(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		active  int64
		pending int64
		empty   bool
	}{
		{"active remains", 1, 0, false},
		{"pending remains", 0, 1, false},
		{"already cleared", 0, 0, true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newAbsentDispositionFixture(t)
			fixture.verifier.facts.RemainingActiveCount = testCase.active
			fixture.verifier.facts.RemainingPendingCount = testCase.pending
			if testCase.empty {
				clearDispositionContact(fixture.transaction)
			}
			cleared, err := fixture.clear()
			if err != nil || cleared || fixture.verifier.calls != 1 || len(fixture.transaction.writes) != 0 ||
				(fixture.transaction.users[dispositionSourceID].RetainedEmail == nil) != testCase.empty {
				t.Fatalf("absent orphan preservation clear=%v, error=%v", cleared, err)
			}
		})
	}
}

func TestArticleDispositionClear_AbsentUntrustedCompletion_Rejects(t *testing.T) {
	mutations := append(dispositionCommonFactMutations(), dispositionDeleteFactMutations()...)
	mutations = append(mutations,
		dispositionMutation{"ordinary without source", func(fixture *dispositionFixture) {
			setDispositionAuthoredDelete(fixture, "ordinary")
		}},
		dispositionMutation{"violating without source", func(fixture *dispositionFixture) {
			setDispositionAuthoredDelete(fixture, "violating")
		}},
		dispositionMutation{"cross transaction", func(fixture *dispositionFixture) {
			fixture.reference.Transaction = newDispositionTransaction()
		}},
		dispositionMutation{"wrong bound operation", func(fixture *dispositionFixture) {
			fixture.reference.OperationID = stringDispositionPointer(dispositionNoticeID)
		}},
		dispositionMutation{"source not deleted", func(fixture *dispositionFixture) {
			user := fixture.transaction.users[dispositionSourceID]
			user.Status, user.DeletedAt = UserStatusActive, nil
			fixture.transaction.users[dispositionSourceID] = user
		}},
		dispositionMutation{"verifier rejects tombstone or provenance", func(fixture *dispositionFixture) {
			fixture.verifier.err = errors.New("tombstone or operation provenance mismatch")
		}},
		dispositionMutation{"publish absent", func(fixture *dispositionFixture) {
			fixture.reference.Action = ArticleDispositionPublish
			fixture.reference.OperationID = nil
			fixture.verifier.reference = fixture.reference
		}},
	)
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			fixture := newAbsentDispositionFixture(t)
			mutation.mutate(fixture)
			if cleared, err := fixture.clear(); err == nil || cleared || fixture.transaction.closed ||
				len(fixture.transaction.writes) != 0 {
				t.Fatalf("untrusted absent completion clear=%v, error=%v", cleared, err)
			}
		})
	}
}

func TestArticleDispositionClear_AbsentOrphanWriteFailure_CallerOwnsRollback(t *testing.T) {
	for _, failureAt := range []int{1, 2, 3} {
		fixture := newAbsentDispositionFixture(t)
		fixture.transaction.snapshot()
		cause := errors.New("injected absent orphan release SQL failure")
		fixture.transaction.execFailAt, fixture.transaction.execErr = failureAt, cause
		cleared, err := fixture.clear()
		if !errors.Is(err, cause) || cleared || fixture.verifier.calls != 1 || fixture.transaction.closed ||
			len(fixture.transaction.writes) != failureAt {
			t.Fatalf("absent write %d clear=%v, error=%v", failureAt, cleared, err)
		}
		if err := fixture.transaction.Rollback(fixture.ctx); err != nil ||
			fixture.transaction.users[dispositionSourceID].RetainedEmail == nil ||
			len(fixture.transaction.identifiers[dispositionSourceID]) != 2 || len(fixture.transaction.writes) != 0 {
			t.Fatal("caller rollback did not restore absent orphan release effects")
		}
	}
}

func clearDispositionContact(transaction *dispositionTransaction) {
	user := transaction.users[dispositionSourceID]
	user.RetainedEmail = nil
	transaction.users[dispositionSourceID] = user
	transaction.identifiers[dispositionSourceID] = transaction.identifiers[dispositionSourceID][:1]
}

func TestArticleDispositionClear_AuthoredDeletion_RequiresDeletedAuthorAndNullOperationSource(t *testing.T) {
	for _, kind := range []string{"ordinary", "violating"} {
		t.Run(kind, func(t *testing.T) {
			fixture := newDispositionFixture(t, ArticleDispositionDeleteComplete)
			setDispositionAuthoredDelete(fixture, kind)
			candidate, err := fixture.lifecycle.LockDeletionCandidate(fixture.ctx, fixture.transaction, dispositionSourceID)
			if err != nil || !candidate.Deleted || candidate.UserID != dispositionSourceID {
				t.Fatalf("candidate result error=%v", err)
			}
			cleared, err := fixture.clear()
			if err != nil || !cleared || fixture.verifier.facts.CompletedOperation.SourceID != nil {
				t.Fatalf("authored delete clear=%v, error=%v", cleared, err)
			}
		})
	}
}

func setDispositionAuthoredDelete(fixture *dispositionFixture, kind string) {
	sourceID := dispositionSourceID
	fixture.reference.BeforeArticle.AuthorID = &sourceID
	fixture.reference.BeforeArticle.SourceID = nil
	fixture.reference.BeforeArticle.Publication = "published"
	fixture.verifier.reference = fixture.reference
	fixture.verifier.facts.CompletedOperation.DeleteKind = kind
	fixture.verifier.facts.CompletedOperation.SourceID = nil
	fixture.verifier.facts.Audit.Action = "article.delete"
	if kind == "violating" {
		fixture.verifier.facts.Audit.Action = "article.violation.delete"
	}
}

func TestArticleDispositionLockCandidate_Nondeleted_HoldsUntilCallerCommit(t *testing.T) {
	for _, status := range []UserStatus{UserStatusActive, UserStatusSuspended} {
		transaction := newDispositionTransaction()
		user := transaction.users[dispositionTargetID]
		user.Status = status
		transaction.users[dispositionTargetID] = user
		candidate, err := (&ArticleIdentityLifecycle{}).LockDeletionCandidate(context.Background(), transaction, user.ID)
		if err != nil || candidate.Deleted || candidate.UserID != user.ID || transaction.closed ||
			!reflect.DeepEqual(transaction.calls, []string{"user:" + user.ID, "identifiers:" + user.ID}) ||
			len(transaction.writes) != 0 {
			t.Fatalf("nondeleted candidate result=%v, error=%v", candidate, err)
		}
		if err := transaction.Commit(context.Background()); err != nil || !transaction.closed {
			t.Fatal("caller did not retain transaction ownership")
		}
	}
}

type dispositionMutation struct {
	name   string
	mutate func(*dispositionFixture)
}

func TestArticleDispositionClear_InvalidFacts_RejectsBeforeWrites(t *testing.T) {
	mutations := append(dispositionCommonFactMutations(), dispositionPublishFactMutations()...)
	assertDispositionMutations(t, ArticleDispositionPublish, mutations)
	assertDispositionMutations(t, ArticleDispositionTransfer, dispositionTransferFactMutations())
	assertDispositionMutations(t, ArticleDispositionDeleteComplete, dispositionDeleteFactMutations())
}

func assertDispositionMutations(t *testing.T, action ArticleDispositionAction, mutations []dispositionMutation) {
	t.Helper()
	for _, mutation := range mutations {
		t.Run(string(action)+"/"+mutation.name, func(t *testing.T) {
			fixture := newDispositionFixture(t, action)
			mutation.mutate(fixture)
			cleared, err := fixture.clear()
			if err == nil || cleared || len(fixture.transaction.writes) != 0 || fixture.transaction.closed {
				t.Fatalf("invalid completion clear=%v, error=%v", cleared, err)
			}
		})
	}
}

func dispositionCommonFactMutations() []dispositionMutation {
	return []dispositionMutation{
		{"negative active", func(fixture *dispositionFixture) { fixture.verifier.facts.RemainingActiveCount = -1 }},
		{"negative pending", func(fixture *dispositionFixture) { fixture.verifier.facts.RemainingPendingCount = -1 }},
		{"audit id", func(fixture *dispositionFixture) { fixture.verifier.facts.Audit.ID = dispositionNoticeID }},
		{"audit article", func(fixture *dispositionFixture) { fixture.verifier.facts.Audit.ArticleID = dispositionNoticeID }},
		{"audit action", func(fixture *dispositionFixture) { fixture.verifier.facts.Audit.Action = "article.delete" }},
		{"audit outcome", func(fixture *dispositionFixture) { fixture.verifier.facts.Audit.Outcome = "failed" }},
		{"audit trace", func(fixture *dispositionFixture) { fixture.verifier.facts.Audit.TraceID = "different-trace" }},
		{"missing trace", func(fixture *dispositionFixture) {
			fixture.ctx = context.Background()
			fixture.verifier.ctx = fixture.ctx
		}},
	}
}

func dispositionPublishFactMutations() []dispositionMutation {
	return []dispositionMutation{
		{"after absent", func(fixture *dispositionFixture) { fixture.verifier.facts.AfterArticle.Exists = false }},
		{"after id", func(fixture *dispositionFixture) { fixture.verifier.facts.AfterArticle.ID = dispositionNoticeID }},
		{"after version", func(fixture *dispositionFixture) { fixture.verifier.facts.AfterArticle.Version++ }},
		{"after author", func(fixture *dispositionFixture) {
			fixture.verifier.facts.AfterArticle.AuthorID = &fixture.reference.SourceID
		}},
		{"after source", func(fixture *dispositionFixture) {
			fixture.verifier.facts.AfterArticle.SourceID = &fixture.reference.SourceID
		}},
		{"after held", func(fixture *dispositionFixture) { fixture.verifier.facts.AfterArticle.Publication = "held" }},
		{"after pending", func(fixture *dispositionFixture) { fixture.verifier.facts.AfterArticle.Deletion = "deletion_pending" }},
		{"after operation", func(fixture *dispositionFixture) {
			fixture.verifier.facts.AfterArticle.OperationID = &fixture.reference.SourceID
		}},
		{"after category", func(fixture *dispositionFixture) {
			fixture.verifier.facts.AfterArticle.Category = &fixture.reference.SourceID
		}},
		{"extra notice", func(fixture *dispositionFixture) { fixture.verifier.facts.Notice = &ArticleDispositionNotice{} }},
		{"extra recipient", func(fixture *dispositionFixture) {
			fixture.verifier.facts.Recipients = []ArticleDispositionRecipient{{}}
		}},
		{"extra operation", func(fixture *dispositionFixture) {
			fixture.verifier.facts.CompletedOperation = &ArticleDispositionOperation{}
		}},
	}
}

func dispositionTransferFactMutations() []dispositionMutation {
	return []dispositionMutation{
		{"notice absent", func(fixture *dispositionFixture) { fixture.verifier.facts.Notice = nil }},
		{"notice id", func(fixture *dispositionFixture) { fixture.verifier.facts.Notice.ID = dispositionArticleID }},
		{"notice article", func(fixture *dispositionFixture) { fixture.verifier.facts.Notice.ArticleID = dispositionNoticeID }},
		{"notice author", func(fixture *dispositionFixture) { fixture.verifier.facts.Notice.NewAuthorID = dispositionSourceID }},
		{"notice former", func(fixture *dispositionFixture) {
			fixture.verifier.facts.Notice.FormerDeletedUserID = dispositionTargetID
		}},
		{"recipient missing", func(fixture *dispositionFixture) {
			fixture.verifier.facts.Recipients = fixture.verifier.facts.Recipients[:1]
		}},
		{"recipient duplicate", func(fixture *dispositionFixture) {
			fixture.verifier.facts.Recipients[1].Kind = fixture.verifier.facts.Recipients[0].Kind
		}},
		{"recipient unknown", func(fixture *dispositionFixture) { fixture.verifier.facts.Recipients[1].Kind = "unknown" }},
		{"recipient not pending", func(fixture *dispositionFixture) { fixture.verifier.facts.Recipients[1].Status = "sent" }},
		{"recipient empty", func(fixture *dispositionFixture) { fixture.verifier.facts.Recipients[1].EmailSnapshot = "" }},
		{"source address wrong", func(fixture *dispositionFixture) {
			fixture.verifier.facts.Recipients[1].EmailSnapshot = "other@example.test"
		}},
		{"target address wrong", func(fixture *dispositionFixture) {
			fixture.verifier.facts.Recipients[0].EmailSnapshot = "other@example.test"
		}},
		{"after author wrong", func(fixture *dispositionFixture) {
			fixture.verifier.facts.AfterArticle.AuthorID = &fixture.reference.SourceID
		}},
		{"violation drift", func(fixture *dispositionFixture) {
			category := "sexual"
			fixture.verifier.facts.AfterArticle.Violation = "violating"
			fixture.verifier.facts.AfterArticle.Category = &category
		}},
	}
}

func dispositionDeleteFactMutations() []dispositionMutation {
	return []dispositionMutation{
		{"article present", func(fixture *dispositionFixture) { fixture.verifier.facts.AfterArticle.Exists = true }},
		{"operation absent", func(fixture *dispositionFixture) { fixture.verifier.facts.CompletedOperation = nil }},
		{"operation id", func(fixture *dispositionFixture) { fixture.verifier.facts.CompletedOperation.ID = dispositionArticleID }},
		{"operation article", func(fixture *dispositionFixture) {
			fixture.verifier.facts.CompletedOperation.ArticleID = dispositionNoticeID
		}},
		{"operation version", func(fixture *dispositionFixture) { fixture.verifier.facts.CompletedOperation.ArticleVersion++ }},
		{"operation incomplete", func(fixture *dispositionFixture) { fixture.verifier.facts.CompletedOperation.Status = "finalizing" }},
		{"operation audit", func(fixture *dispositionFixture) {
			fixture.verifier.facts.CompletedOperation.AuditEventID = dispositionNoticeID
		}},
		{"orphan no source", func(fixture *dispositionFixture) { fixture.verifier.facts.CompletedOperation.SourceID = nil }},
		{"orphan wrong source", func(fixture *dispositionFixture) {
			sourceID := dispositionTargetID
			fixture.verifier.facts.CompletedOperation.SourceID = &sourceID
		}},
		{"kind rewritten", func(fixture *dispositionFixture) { fixture.verifier.facts.CompletedOperation.DeleteKind = "ordinary" }},
		{"authored operation source", func(fixture *dispositionFixture) {
			setDispositionAuthoredDelete(fixture, "ordinary")
			fixture.verifier.facts.CompletedOperation.SourceID = &fixture.reference.SourceID
		}},
	}
}

func TestArticleDispositionClear_TamperedReference_Rejects(t *testing.T) {
	assertDispositionMutations(t, ArticleDispositionPublish, dispositionReferenceMutations())
	assertDispositionMutations(t, ArticleDispositionTransfer, []dispositionMutation{
		{"before absent", func(fixture *dispositionFixture) {
			fixture.reference.BeforeArticle = ArticleDispositionState{}
			fixture.verifier.reference = fixture.reference
		}},
		{"missing notice", func(fixture *dispositionFixture) { fixture.reference.NoticeID = nil }},
		{"missing author", func(fixture *dispositionFixture) { fixture.reference.NewAuthorID = nil }},
		{"rebound notice", func(fixture *dispositionFixture) { fixture.reference.NoticeID = &fixture.reference.SourceID }},
		{"rebound author", func(fixture *dispositionFixture) { fixture.reference.NewAuthorID = &fixture.reference.SourceID }},
	})
	assertDispositionMutations(t, ArticleDispositionDeleteComplete, []dispositionMutation{
		{"missing operation", func(fixture *dispositionFixture) { fixture.reference.OperationID = nil }},
		{"wrong pending version", func(fixture *dispositionFixture) { fixture.reference.BeforeArticle.Version++ }},
		{"wrong pending operation", func(fixture *dispositionFixture) {
			fixture.reference.BeforeArticle.OperationID = &fixture.reference.SourceID
		}},
		{"absent with no source", func(fixture *dispositionFixture) {
			fixture.reference.BeforeArticle = ArticleDispositionState{}
			fixture.verifier.facts.CompletedOperation.SourceID = nil
		}},
		{"authored unbound source", func(fixture *dispositionFixture) {
			setDispositionAuthoredDelete(fixture, "ordinary")
			authorID := dispositionTargetID
			fixture.reference.BeforeArticle.AuthorID = &authorID
		}},
	})
}

func dispositionReferenceMutations() []dispositionMutation {
	return []dispositionMutation{
		{"transaction", func(fixture *dispositionFixture) { fixture.reference.Transaction = newDispositionTransaction() }},
		{"nil transaction", func(fixture *dispositionFixture) { fixture.reference.Transaction = nil }},
		{"source", func(fixture *dispositionFixture) { fixture.reference.SourceID = dispositionTargetID }},
		{"article", func(fixture *dispositionFixture) { fixture.reference.ExpectedArticleID = dispositionNoticeID }},
		{"audit", func(fixture *dispositionFixture) { fixture.reference.AuditEventID = dispositionNoticeID }},
		{"zero version", func(fixture *dispositionFixture) { fixture.reference.BeforeVersion = 0 }},
		{"negative version", func(fixture *dispositionFixture) { fixture.reference.BeforeVersion = -1 }},
		{"overflow version", func(fixture *dispositionFixture) { fixture.reference.BeforeVersion = 1<<63 - 1 }},
		{"unknown action", func(fixture *dispositionFixture) { fixture.reference.Action = "unknown" }},
		{"rebound action", func(fixture *dispositionFixture) { fixture.reference.Action = ArticleDispositionDeleteComplete }},
		{"before absent", func(fixture *dispositionFixture) { fixture.reference.BeforeArticle.Exists = false }},
		{"before version", func(fixture *dispositionFixture) { fixture.reference.BeforeArticle.Version++ }},
		{"before author", func(fixture *dispositionFixture) {
			fixture.reference.BeforeArticle.AuthorID = &fixture.reference.SourceID
		}},
		{"before source", func(fixture *dispositionFixture) { fixture.reference.BeforeArticle.SourceID = nil }},
		{"before published", func(fixture *dispositionFixture) { fixture.reference.BeforeArticle.Publication = "published" }},
		{"before pending", func(fixture *dispositionFixture) { fixture.reference.BeforeArticle.Deletion = "deletion_pending" }},
		{"before operation", func(fixture *dispositionFixture) {
			fixture.reference.BeforeArticle.OperationID = &fixture.reference.SourceID
		}},
		{"extra target", func(fixture *dispositionFixture) { fixture.reference.NewAuthorID = &fixture.reference.SourceID }},
		{"extra notice", func(fixture *dispositionFixture) { fixture.reference.NoticeID = &fixture.reference.SourceID }},
		{"extra operation", func(fixture *dispositionFixture) { fixture.reference.OperationID = &fixture.reference.SourceID }},
		{"invalid uuid", func(fixture *dispositionFixture) { fixture.reference.AuditEventID = "invalid" }},
	}
}

func TestArticleDispositionClear_VerifierFailure_PreservesCause(t *testing.T) {
	fixture := newDispositionFixture(t, ArticleDispositionPublish)
	cause := errors.New("completion readback failed")
	fixture.verifier.err = cause
	cleared, err := fixture.clear()
	if !errors.Is(err, cause) || cleared || fixture.verifier.calls != 1 || len(fixture.transaction.writes) != 0 {
		t.Fatalf("verifier failure clear=%v, error=%v", cleared, err)
	}
}

func TestArticleDispositionClear_WriteFailure_CallerOwnsRollback(t *testing.T) {
	for _, failureAt := range []int{1, 2, 3} {
		fixture := newDispositionFixture(t, ArticleDispositionTransfer)
		fixture.transaction.snapshot()
		cause := errors.New("injected SQL failure")
		fixture.transaction.execFailAt, fixture.transaction.execErr = failureAt, cause
		cleared, err := fixture.clear()
		if !errors.Is(err, cause) || cleared || fixture.transaction.closed ||
			len(fixture.transaction.writes) != failureAt {
			t.Fatalf("write %d clear=%v, error=%v", failureAt, cleared, err)
		}
		if failureAt > 1 && fixture.transaction.users[dispositionSourceID].RetainedEmail != nil {
			t.Fatal("failure fixture did not exercise pending identity writes")
		}
		if err := fixture.transaction.Rollback(fixture.ctx); err != nil ||
			fixture.transaction.users[dispositionSourceID].RetainedEmail == nil ||
			len(fixture.transaction.identifiers[dispositionSourceID]) != 2 || len(fixture.transaction.writes) != 0 {
			t.Fatal("caller rollback did not restore pending identity changes")
		}
	}
}

func TestArticleDispositionClear_ConditionalZeroRows_Rejects(t *testing.T) {
	for _, zeroAt := range []int{1, 2} {
		fixture := newDispositionFixture(t, ArticleDispositionPublish)
		fixture.transaction.zeroAt = zeroAt
		cleared, err := fixture.clear()
		if err == nil || cleared || fixture.transaction.closed || len(fixture.transaction.writes) != zeroAt {
			t.Fatalf("zero row %d clear=%v, error=%v", zeroAt, cleared, err)
		}
	}
}

func TestArticleDispositionLocks_SQLFailures_PreserveRootCause(t *testing.T) {
	cause := &pgconn.PgError{Code: "55P03", Message: "lock not available"}
	cases := []struct {
		name   string
		mutate func(*dispositionTransaction)
	}{
		{"user NOWAIT", func(transaction *dispositionTransaction) { transaction.rowErr = cause }},
		{"identifier NOWAIT", func(transaction *dispositionTransaction) { transaction.queryErr = cause }},
		{"scan", func(transaction *dispositionTransaction) { transaction.scanErr = cause }},
		{"rows", func(transaction *dispositionTransaction) { transaction.rowsErr = cause }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			transaction := newDispositionTransaction()
			testCase.mutate(transaction)
			_, err := (&ArticleIdentityLifecycle{}).LockDeletionCandidate(context.Background(), transaction, dispositionSourceID)
			var postgresError *pgconn.PgError
			if !errors.Is(err, cause) || !errors.As(err, &postgresError) || postgresError.Code != "55P03" || transaction.closed {
				t.Fatalf("lock cause lost or transaction owned by callee: %v", err)
			}
		})
	}
}

func TestArticleDispositionLocks_MissingOrInvalidUsers_Rejects(t *testing.T) {
	for _, userID := range []string{"", "invalid", dispositionArticleID} {
		transaction := newDispositionTransaction()
		_, err := (&ArticleIdentityLifecycle{}).LockDeletionCandidate(context.Background(), transaction, userID)
		if err == nil || len(transaction.writes) != 0 || transaction.closed {
			t.Fatal("invalid or missing identity accepted")
		}
	}
	transaction := newDispositionTransaction()
	_, err := (&ArticleIdentityLifecycle{}).LockTransferParties(
		context.Background(), transaction, stringDispositionPointer(dispositionSourceID), dispositionSourceID)
	if !errors.Is(err, ErrArticleTransferTargetInvalid) || !reflect.DeepEqual(transaction.calls,
		[]string{"user:" + dispositionSourceID}) {
		t.Fatal("same party was not deduplicated before rejecting deleted target")
	}
}

func TestArticleDispositionClear_EndedOrUnboundTransaction_Rejects(t *testing.T) {
	fixture := newDispositionFixture(t, ArticleDispositionPublish)
	if err := fixture.transaction.Commit(fixture.ctx); err != nil {
		t.Fatal(err)
	}
	if cleared, err := fixture.clear(); !errors.Is(err, pgx.ErrTxClosed) || cleared || fixture.verifier.calls != 0 {
		t.Fatalf("ended tx clear=%v, error=%v", cleared, err)
	}
	other := newDispositionTransaction()
	_, err := fixture.lifecycle.ClearRetainedEmailIfEligible(fixture.ctx, other, dispositionSourceID, fixture.reference)
	if err == nil || len(other.calls) != 0 {
		t.Fatal("reference reused in different transaction")
	}
}

func stringDispositionPointer(value string) *string { return &value }

func TestArticleDispositionClear_UnconfiguredDependencies_FailsClosed(t *testing.T) {
	fixture := newDispositionFixture(t, ArticleDispositionPublish)
	cases := []struct {
		verifier ArticleCompletionVerifier
		appender CanonicalDeleteAuditAppender
		ids      RegistrationIDGenerator
		clock    RegistrationClock
	}{
		{nil, audit.Store{}, fixture.lifecycle.ids, fixture.lifecycle.clock},
		{fixture.verifier, nil, fixture.lifecycle.ids, fixture.lifecycle.clock},
		{fixture.verifier, audit.Store{}, nil, fixture.lifecycle.clock},
		{fixture.verifier, audit.Store{}, fixture.lifecycle.ids, nil},
	}
	for _, testCase := range cases {
		if _, err := NewArticleIdentityLifecycle(testCase.verifier, testCase.appender, testCase.ids, testCase.clock); err == nil {
			t.Fatal("constructor accepted absent dependency")
		}
	}
	fixture.lifecycle = &ArticleIdentityLifecycle{}
	if cleared, err := fixture.clear(); err == nil || cleared || len(fixture.transaction.calls) != 0 {
		t.Fatal("zero-value lifecycle allowed cleanup")
	}
}

func TestArticleDispositionClear_ViolatingTransfer_PreservesClassification(t *testing.T) {
	fixture := newDispositionFixture(t, ArticleDispositionTransfer)
	category := "violence"
	fixture.reference.BeforeArticle.Violation, fixture.reference.BeforeArticle.Category = "violating", &category
	fixture.verifier.reference = fixture.reference
	fixture.verifier.facts.AfterArticle.Violation, fixture.verifier.facts.AfterArticle.Category = "violating", &category
	if cleared, err := fixture.clear(); err != nil || !cleared {
		t.Fatalf("valid violating transfer clear=%v, error=%v", cleared, err)
	}
}

func TestArticleDispositionLockParties_TargetClassification_DistinctFromInvariantAndDependency(t *testing.T) {
	for _, testCase := range dispositionTargetErrorCases() {
		t.Run(testCase.name, func(t *testing.T) {
			transaction := newDispositionTransaction()
			testCase.mutate(transaction)
			_, err := (&ArticleIdentityLifecycle{}).LockTransferParties(context.Background(),
				transaction, stringDispositionPointer(dispositionTargetID), dispositionSourceID)
			if err == nil || errors.Is(err, ErrArticleTransferTargetInvalid) != testCase.targetInvalid ||
				len(transaction.writes) != 0 || transaction.closed {
				t.Fatalf("target-invalid classification=%v, error=%v",
					errors.Is(err, ErrArticleTransferTargetInvalid), err)
			}
		})
	}
}

func dispositionTargetErrorCases() []struct {
	name          string
	mutate        func(*dispositionTransaction)
	targetInvalid bool
} {
	return []struct {
		name          string
		mutate        func(*dispositionTransaction)
		targetInvalid bool
	}{
		{"missing target", func(transaction *dispositionTransaction) { delete(transaction.users, dispositionTargetID) }, true},
		{"suspended target", func(transaction *dispositionTransaction) {
			user := transaction.users[dispositionTargetID]
			user.Status = UserStatusSuspended
			transaction.users[dispositionTargetID] = user
		}, true},
		{"deleted target", func(transaction *dispositionTransaction) {
			user := transaction.users[dispositionTargetID]
			user.Status = UserStatusDeleted
			transaction.users[dispositionTargetID] = user
		}, true},
		{"missing source", func(transaction *dispositionTransaction) { delete(transaction.users, dispositionSourceID) }, false},
		{"target email absent", func(transaction *dispositionTransaction) {
			user := transaction.users[dispositionTargetID]
			user.Email = nil
			transaction.users[dispositionTargetID] = user
		}, false},
		{"source email absent", clearDispositionContact, false},
		{"target binding drift", func(transaction *dispositionTransaction) {
			transaction.identifiers[dispositionTargetID][1].NormalizedValue = "other@example.test"
		}, false},
		{"database unavailable", func(transaction *dispositionTransaction) {
			transaction.rowErr = &pgconn.PgError{Code: "55P03"}
		}, false},
	}
}

func TestArticleDispositionLocks_IdentifierDrift_RejectsEvenWithRemainingArticles(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*dispositionTransaction)
	}{
		{"unpaired retained email", func(transaction *dispositionTransaction) {
			user := transaction.users[dispositionSourceID]
			user.RetainedEmail = nil
			transaction.users[dispositionSourceID] = user
		}},
		{"missing username", func(transaction *dispositionTransaction) {
			transaction.identifiers[dispositionSourceID] = transaction.identifiers[dispositionSourceID][1:]
		}},
		{"wrong identifier owner", func(transaction *dispositionTransaction) {
			transaction.identifiers[dispositionSourceID][0].UserID = dispositionTargetID
		}},
		{"released identifier", func(transaction *dispositionTransaction) {
			transaction.identifiers[dispositionSourceID][0].ReleasedAt = transaction.users[dispositionSourceID].DeletedAt
		}},
		{"active tombstone identifier", func(transaction *dispositionTransaction) {
			transaction.identifiers[dispositionSourceID][1].Lifecycle = IdentifierActive
		}},
		{"duplicate kind", func(transaction *dispositionTransaction) {
			transaction.identifiers[dispositionSourceID][1] = transaction.identifiers[dispositionSourceID][0]
		}},
		{"unknown kind", func(transaction *dispositionTransaction) {
			transaction.identifiers[dispositionSourceID][1].Kind = "unknown"
		}},
		{"invalid contact", func(transaction *dispositionTransaction) {
			user := transaction.users[dispositionSourceID]
			user.RetainedEmail = stringDispositionPointer("not-an-email")
			transaction.users[dispositionSourceID] = user
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newDispositionFixture(t, ArticleDispositionPublish)
			testCase.mutate(fixture.transaction)
			fixture.verifier.facts.RemainingPendingCount = 1
			if cleared, err := fixture.clear(); err == nil || cleared || len(fixture.transaction.writes) != 0 {
				t.Fatalf("drift bypassed validation: cleared=%v, error=%v", cleared, err)
			}
		})
	}
}

func TestArticleDispositionLocks_NilTransaction_Rejects(t *testing.T) {
	lifecycle := &ArticleIdentityLifecycle{}
	if _, err := lifecycle.LockTransferParties(context.Background(), nil, nil, dispositionSourceID); err == nil {
		t.Fatal("nil transfer transaction accepted")
	}
	if _, err := lifecycle.LockDeletionCandidate(context.Background(), nil, dispositionSourceID); err == nil {
		t.Fatal("nil candidate transaction accepted")
	}
}

func TestArticleDispositionClear_AuditIdentityFailure_RejectsBeforeWrites(t *testing.T) {
	cases := []struct {
		name  string
		ids   RegistrationIDGenerator
		clock RegistrationClock
	}{
		{"generator failure", &registrationIDStub{}, SystemClock{}},
		{"invalid ID", &registrationIDStub{ids: []string{"invalid"}}, SystemClock{}},
		{"zero time", &registrationIDStub{ids: []string{dispositionReleaseAuditID}}, registrationClockStub{}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newDispositionFixture(t, ArticleDispositionPublish)
			lifecycle, err := NewArticleIdentityLifecycle(fixture.verifier, audit.Store{}, testCase.ids, testCase.clock)
			if err != nil {
				t.Fatal(err)
			}
			fixture.lifecycle = lifecycle
			if cleared, err := fixture.clear(); err == nil || cleared || len(fixture.transaction.writes) != 0 {
				t.Fatalf("invalid audit identity clear=%v, error=%v", cleared, err)
			}
		})
	}
}

type dispositionRow struct {
	values []any
	err    error
}

func (row dispositionRow) Scan(destinations ...any) error {
	if row.err != nil {
		return row.err
	}
	if len(destinations) != len(row.values) {
		return errors.New("unexpected scan shape")
	}
	for index, value := range row.values {
		reflect.ValueOf(destinations[index]).Elem().Set(reflect.ValueOf(value))
	}
	return nil
}

type dispositionRows struct {
	pgx.Rows
	identifiers []IdentityIdentifier
	index       int
	err         error
	scanErr     error
	closed      bool
}

func (rows *dispositionRows) Next() bool {
	if rows.index >= len(rows.identifiers) {
		return false
	}
	rows.index++
	return true
}

func (rows *dispositionRows) Scan(destinations ...any) error {
	if rows.scanErr != nil {
		return rows.scanErr
	}
	identifier := rows.identifiers[rows.index-1]
	return (dispositionRow{values: []any{identifier.UserID, identifier.Kind,
		identifier.NormalizedValue, identifier.Lifecycle, identifier.ReleasedAt}}).Scan(destinations...)
}

func (rows *dispositionRows) Close()     { rows.closed = true }
func (rows *dispositionRows) Err() error { return rows.err }
