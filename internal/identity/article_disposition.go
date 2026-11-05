package identity

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"han/internal/audit"
	"han/internal/platform"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrArticleTransferTargetInvalid identifies an absent or nonactive target.
// Contact/identifier drift and database failures do not carry this sentinel.
var ErrArticleTransferTargetInvalid = errors.New("article transfer target is unavailable")

// LockedArticleParties contains contacts read under caller-owned identity locks.
type LockedArticleParties struct {
	SourceID    string
	SourceEmail *string
	TargetID    *string
	TargetEmail *string
}

// ArticleIdentityLifecycle coordinates identity state in an article transaction.
// The caller owns commit and must roll back the entire transaction on any error.
type ArticleIdentityLifecycle struct {
	verifier ArticleCompletionVerifier
	audit    CanonicalDeleteAuditAppender
	ids      RegistrationIDGenerator
	clock    RegistrationClock
}

// NewArticleIdentityLifecycle fixes the trusted provider at the composition root.
// The provider must prove reference provenance, locked preimages and current-tx
// readbacks; no caller can replace it on an individual cleanup call.
func NewArticleIdentityLifecycle(
	verifier ArticleCompletionVerifier, appender CanonicalDeleteAuditAppender,
	ids RegistrationIDGenerator, clock RegistrationClock,
) (*ArticleIdentityLifecycle, error) {
	if verifier == nil || appender == nil || ids == nil || clock == nil {
		return nil, errors.New("article identity dependencies are required")
	}
	return &ArticleIdentityLifecycle{verifier: verifier, audit: appender, ids: ids, clock: clock}, nil
}

// LockTransferParties locks all users before their identifiers, in UUID order.
// A nil target selects publish or orphan deletion without a notification target.
func (lifecycle *ArticleIdentityLifecycle) LockTransferParties(
	ctx context.Context, transaction pgx.Tx, targetID *string, sourceID string,
) (LockedArticleParties, error) {
	parties, err := lockArticleIdentityParties(ctx, transaction, targetID, sourceID)
	if err != nil {
		return LockedArticleParties{}, err
	}
	if targetID != nil && (parties.SourceEmail == nil || parties.TargetEmail == nil) {
		return LockedArticleParties{}, errors.New("article transfer requires both contacts")
	}
	return parties, nil
}

func lockArticleIdentityParties(
	ctx context.Context, transaction pgx.Tx, targetID *string, sourceID string,
) (LockedArticleParties, error) {
	if transaction == nil {
		return LockedArticleParties{}, errors.New("article identity transaction is required")
	}
	userIDs, err := articleIdentityIDs(sourceID, targetID)
	if err != nil {
		return LockedArticleParties{}, err
	}
	users := make(map[string]User, len(userIDs))
	for _, userID := range userIDs {
		user, err := lockArticleIdentityUser(ctx, transaction, userID)
		if err != nil {
			if targetID != nil && userID == *targetID && userID != sourceID && errors.Is(err, pgx.ErrNoRows) {
				return LockedArticleParties{}, fmt.Errorf("lock article transfer target: %w",
					errors.Join(ErrArticleTransferTargetInvalid, err))
			}
			return LockedArticleParties{}, err
		}
		users[userID] = user
	}
	if users[sourceID].Status != UserStatusDeleted {
		return LockedArticleParties{}, errors.New("article disposition source must be deleted")
	}
	if targetID != nil && users[*targetID].Status != UserStatusActive {
		return LockedArticleParties{}, ErrArticleTransferTargetInvalid
	}
	for _, userID := range userIDs {
		identifiers, err := lockArticleIdentityIdentifiers(ctx, transaction, userID)
		if err != nil {
			return LockedArticleParties{}, err
		}
		if err := validateArticleIdentity(users[userID], identifiers); err != nil {
			return LockedArticleParties{}, err
		}
	}
	parties := LockedArticleParties{SourceID: sourceID, SourceEmail: users[sourceID].RetainedEmail}
	if targetID != nil {
		if users[*targetID].Email == nil {
			return LockedArticleParties{}, errors.New("article transfer target contact is missing")
		}
		parties.TargetID, parties.TargetEmail = targetID, users[*targetID].Email
	}
	return parties, nil
}

func articleIdentityIDs(sourceID string, targetID *string) ([]string, error) {
	userIDs := []string{sourceID}
	if targetID != nil && *targetID != sourceID {
		userIDs = append(userIDs, *targetID)
	}
	for _, userID := range userIDs {
		var parsed pgtype.UUID
		if err := parsed.Scan(userID); err != nil || !parsed.Valid || parsed.String() != userID {
			return nil, errors.New("article identity requires canonical UUIDs")
		}
	}
	sort.Strings(userIDs)
	return userIDs, nil
}

func lockArticleIdentityUser(ctx context.Context, transaction pgx.Tx, userID string) (User, error) {
	var user User
	err := transaction.QueryRow(ctx, lockArticleIdentityUserSQL, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Phone,
		&user.Status, &user.RetainedEmail, &user.DeletedAt,
	)
	if err != nil {
		return User{}, fmt.Errorf("lock article identity user: %w", err)
	}
	if user.ID != userID {
		return User{}, errors.New("article identity user binding mismatch")
	}
	return user, nil
}

// LockedArticleDeletionCandidate reports deletion eligibility under retained locks.
// Nondeleted candidates must skip cleanup; all locks remain until caller commit.
type LockedArticleDeletionCandidate struct {
	UserID  string
	Deleted bool
}

// LockDeletionCandidate locks the caller's locked-article author candidate NOWAIT.
func (lifecycle *ArticleIdentityLifecycle) LockDeletionCandidate(
	ctx context.Context, transaction pgx.Tx, candidateID string,
) (LockedArticleDeletionCandidate, error) {
	if transaction == nil {
		return LockedArticleDeletionCandidate{}, errors.New("article identity transaction is required")
	}
	if _, err := articleIdentityIDs(candidateID, nil); err != nil {
		return LockedArticleDeletionCandidate{}, err
	}
	user, err := lockArticleIdentityUser(ctx, transaction, candidateID)
	if err != nil {
		return LockedArticleDeletionCandidate{}, err
	}
	identifiers, err := lockArticleIdentityIdentifiers(ctx, transaction, candidateID)
	if err != nil {
		return LockedArticleDeletionCandidate{}, err
	}
	if err := validateArticleIdentity(user, identifiers); err != nil {
		return LockedArticleDeletionCandidate{}, err
	}
	return LockedArticleDeletionCandidate{UserID: candidateID, Deleted: user.Status == UserStatusDeleted}, nil
}

func validateArticleIdentity(user User, identifiers []IdentityIdentifier) error {
	if !validAuthorStatus(user.Status) || (user.Status == UserStatusDeleted) != (user.DeletedAt != nil) {
		return errors.New("article identity lifecycle mismatch")
	}
	byKind := make(map[IdentifierKind]IdentityIdentifier, len(identifiers))
	for _, identifier := range identifiers {
		if identifier.UserID != user.ID || identifier.NormalizedValue == "" ||
			identifier.ReleasedAt != nil || !articleIdentityIdentifierLifecycle(user, identifier) {
			return errors.New("article identity identifier lifecycle mismatch")
		}
		if _, duplicate := byKind[identifier.Kind]; duplicate {
			return errors.New("article identity duplicate identifier kind")
		}
		byKind[identifier.Kind] = identifier
	}
	if err := articleIdentityBinding(byKind, IdentifierUsername, &user.Username, true); err != nil {
		return err
	}
	if user.Status == UserStatusDeleted {
		if user.Email != nil || user.Phone != nil {
			return errors.New("article identity tombstone retains live contacts")
		}
		return articleIdentityEmailBinding(byKind, user.RetainedEmail)
	}
	if user.RetainedEmail != nil {
		return errors.New("article identity live user has retained email")
	}
	if err := articleIdentityEmailBinding(byKind, user.Email); err != nil {
		return err
	}
	return articleIdentityBinding(byKind, IdentifierPhone, user.Phone, false)
}

func articleIdentityIdentifierLifecycle(user User, identifier IdentityIdentifier) bool {
	if identifier.Kind != IdentifierUsername && identifier.Kind != IdentifierEmail && identifier.Kind != IdentifierPhone {
		return false
	}
	if user.Status == UserStatusDeleted {
		return identifier.Lifecycle == IdentifierReserved
	}
	return canonicalDeleteAdvanceableLifecycle(identifier.Lifecycle)
}

func articleIdentityEmailBinding(identifiers map[IdentifierKind]IdentityIdentifier, email *string) error {
	if email != nil && (!validEmail(*email) || strings.TrimSpace(*email) != *email) {
		return errors.New("article identity contact is invalid")
	}
	return articleIdentityBinding(identifiers, IdentifierEmail, email, true)
}

func articleIdentityBinding(
	identifiers map[IdentifierKind]IdentityIdentifier, kind IdentifierKind, value *string, fold bool,
) error {
	identifier, found := identifiers[kind]
	if value == nil {
		if found {
			return errors.New("article identity unpaired identifier")
		}
		return nil
	}
	expected := *value
	if fold {
		expected = strings.ToLower(expected)
	}
	if !found || expected == "" || identifier.NormalizedValue != expected {
		return errors.New("article identity identifier value mismatch")
	}
	return nil
}

// ArticleDispositionAction selects the completed article transition.
type ArticleDispositionAction string

const (
	ArticleDispositionPublish        ArticleDispositionAction = "publish"
	ArticleDispositionTransfer       ArticleDispositionAction = "transfer"
	ArticleDispositionDeleteComplete ArticleDispositionAction = "delete_complete"
)

// ArticleDispositionState is the content-free locked preimage or current readback.
type ArticleDispositionState struct {
	Exists      bool
	ID          string
	Version     int64
	AuthorID    *string
	SourceID    *string
	Publication string
	Violation   string
	Category    *string
	Deletion    string
	OperationID *string
}

// ArticleCompletionReference locates one completion in its creating transaction.
// The fixed verifier must reject forged preimages, another transaction or an ended
// transaction. This value is internal and must never be decoded from HTTP/JSON.
type ArticleCompletionReference struct {
	Transaction       pgx.Tx
	ExpectedArticleID string
	SourceID          string
	BeforeVersion     int64
	Action            ArticleDispositionAction
	NoticeID          *string
	NewAuthorID       *string
	OperationID       *string
	AuditEventID      string
	BeforeArticle     ArticleDispositionState
}

// ArticleDispositionNotice contains the four transfer association fields.
type ArticleDispositionNotice struct {
	ID                  string
	ArticleID           string
	NewAuthorID         string
	FormerDeletedUserID string
}

// ArticleDispositionRecipient is a persisted notification contact snapshot.
type ArticleDispositionRecipient struct {
	Kind          string
	EmailSnapshot string
	Status        string
}

// ArticleDispositionOperation is the completed deletion authority readback.
type ArticleDispositionOperation struct {
	ID             string
	ArticleID      string
	ArticleVersion int64
	DeleteKind     string
	SourceID       *string
	Status         string
	AuditEventID   string
}

// ArticleDispositionAudit binds the inserted success audit to the current trace.
type ArticleDispositionAudit struct {
	ID        string
	ArticleID string
	Action    string
	Outcome   string
	TraceID   string
}

// VerifiedArticleCompletion contains current-transaction article-owned facts.
// Counts must come from one statement and snapshot after the action, including
// every active/pending author or orphan-source association, without extra locks.
type VerifiedArticleCompletion struct {
	AfterArticle          ArticleDispositionState
	RemainingActiveCount  int64
	RemainingPendingCount int64
	Notice                *ArticleDispositionNotice
	Recipients            []ArticleDispositionRecipient
	CompletedOperation    *ArticleDispositionOperation
	Audit                 ArticleDispositionAudit
}

// ArticleCompletionVerifier proves provenance and reads facts using only caller tx.
// The composition root supplies the sole article-owned implementation; it verifies
// locked preimage, tombstone/operation and audit existence before returning facts.
// It retains the original locked transfer contacts for same-tx reentry after the
// identity contact has been cleared; such reentry never grants a second release.
type ArticleCompletionVerifier interface {
	VerifyCompletion(context.Context, pgx.Tx, ArticleCompletionReference) (VerifiedArticleCompletion, error)
}

// ClearRetainedEmailIfEligible verifies completion before releasing identity data.
// False means unchanged. Errors require caller rollback, including audit failure.
func (lifecycle *ArticleIdentityLifecycle) ClearRetainedEmailIfEligible(
	ctx context.Context, transaction pgx.Tx, sourceID string, reference ArticleCompletionReference,
) (bool, error) {
	if lifecycle == nil || lifecycle.verifier == nil || lifecycle.audit == nil ||
		lifecycle.ids == nil || lifecycle.clock == nil {
		return false, errors.New("article identity lifecycle is not configured")
	}
	if err := validateArticleCompletionReference(transaction, sourceID, reference); err != nil {
		return false, err
	}
	parties, err := lockArticleIdentityParties(ctx, transaction, reference.NewAuthorID, sourceID)
	if err != nil {
		return false, err
	}
	facts, err := lifecycle.verifier.VerifyCompletion(ctx, transaction, reference)
	if err != nil {
		return false, fmt.Errorf("verify article completion: %w", err)
	}
	if err := validateArticleCompletion(ctx, reference, facts, parties); err != nil {
		return false, err
	}
	if facts.RemainingActiveCount != 0 || facts.RemainingPendingCount != 0 || parties.SourceEmail == nil {
		return false, nil
	}
	return lifecycle.releaseArticleIdentityEmail(ctx, transaction, parties, facts.Audit.TraceID)
}

func validateArticleCompletionReference(
	transaction pgx.Tx, sourceID string, reference ArticleCompletionReference,
) error {
	if transaction == nil || reference.Transaction != transaction || reference.SourceID != sourceID ||
		reference.BeforeVersion <= 0 || reference.BeforeVersion == math.MaxInt64 {
		return errors.New("article completion transaction or version mismatch")
	}
	for _, identifier := range []string{sourceID, reference.ExpectedArticleID, reference.AuditEventID} {
		if _, err := articleIdentityIDs(identifier, nil); err != nil {
			return err
		}
	}
	transfer := reference.Action == ArticleDispositionTransfer
	deletion := reference.Action == ArticleDispositionDeleteComplete
	if reference.Action != ArticleDispositionPublish && !transfer && !deletion {
		return errors.New("article completion action is invalid")
	}
	if transfer != (reference.NoticeID != nil) || transfer != (reference.NewAuthorID != nil) ||
		deletion != (reference.OperationID != nil) {
		return errors.New("article completion action fields mismatch")
	}
	for _, identifier := range []*string{reference.NoticeID, reference.NewAuthorID, reference.OperationID} {
		if identifier != nil {
			if _, err := articleIdentityIDs(*identifier, nil); err != nil {
				return err
			}
		}
	}
	return validateArticleCompletionBefore(reference)
}

func validateArticleCompletionBefore(reference ArticleCompletionReference) error {
	before := reference.BeforeArticle
	if !before.Exists && reference.Action == ArticleDispositionDeleteComplete {
		return nil
	}
	if !before.Exists || before.ID != reference.ExpectedArticleID || !validArticleDispositionViolation(before) {
		return errors.New("article completion preimage mismatch")
	}
	if reference.Action != ArticleDispositionDeleteComplete {
		if before.Version != reference.BeforeVersion || before.AuthorID != nil ||
			!articleDispositionValue(before.SourceID, reference.SourceID) || before.Publication != dispositionHeld ||
			before.Deletion != dispositionActive || before.OperationID != nil {
			return errors.New("article disposition preimage binding mismatch")
		}
		return nil
	}
	if before.Version != reference.BeforeVersion+1 || before.Deletion != dispositionPendingDeletion ||
		!articleDispositionSameValue(before.OperationID, reference.OperationID) {
		return errors.New("article deletion preimage binding mismatch")
	}
	orphan := before.AuthorID == nil && articleDispositionValue(before.SourceID, reference.SourceID) &&
		before.Publication == dispositionHeld
	authored := articleDispositionValue(before.AuthorID, reference.SourceID) && before.SourceID == nil &&
		before.Publication == dispositionPublished
	if !orphan && !authored {
		return errors.New("article deletion source has no locked preimage binding")
	}
	return nil
}

func validateArticleCompletion(
	ctx context.Context, reference ArticleCompletionReference,
	facts VerifiedArticleCompletion, parties LockedArticleParties,
) error {
	traceID, found := platform.TraceIDFromContext(ctx)
	if !found || traceID == "" || facts.Audit.TraceID != traceID ||
		facts.Audit.ID != reference.AuditEventID || facts.Audit.ArticleID != reference.ExpectedArticleID ||
		facts.Audit.Outcome != canonicalDeleteAuditOutcomeSucceeded ||
		facts.RemainingActiveCount < 0 || facts.RemainingPendingCount < 0 {
		return errors.New("article completion audit or counters mismatch")
	}
	if reference.Action == ArticleDispositionDeleteComplete {
		return validateArticleDeletionCompletion(reference, facts)
	}
	if facts.CompletedOperation != nil || !validArticleDispositionAfter(reference, facts.AfterArticle) {
		return errors.New("article disposition readback mismatch")
	}
	if reference.Action == ArticleDispositionTransfer {
		return validateArticleTransferCompletion(reference, facts, parties)
	}
	if facts.Notice != nil || len(facts.Recipients) != 0 || facts.Audit.Action != dispositionPublishAudit {
		return errors.New("article publish completion fields mismatch")
	}
	return nil
}

func validArticleDispositionAfter(reference ArticleCompletionReference, after ArticleDispositionState) bool {
	if !after.Exists || after.ID != reference.ExpectedArticleID || after.Version != reference.BeforeVersion+1 ||
		after.SourceID != nil || after.Publication != dispositionPublished || after.Deletion != dispositionActive ||
		after.OperationID != nil || !validArticleDispositionViolation(after) {
		return false
	}
	if reference.Action == ArticleDispositionPublish {
		return after.AuthorID == nil && after.Violation == dispositionClear &&
			reference.BeforeArticle.Violation == dispositionClear
	}
	return articleDispositionSameValue(after.AuthorID, reference.NewAuthorID) &&
		after.Violation == reference.BeforeArticle.Violation &&
		articleDispositionSameValue(after.Category, reference.BeforeArticle.Category)
}

func validateArticleTransferCompletion(
	reference ArticleCompletionReference, facts VerifiedArticleCompletion, parties LockedArticleParties,
) error {
	notice := facts.Notice
	if notice == nil || notice.ID != *reference.NoticeID || notice.ArticleID != reference.ExpectedArticleID ||
		notice.NewAuthorID != *reference.NewAuthorID || notice.FormerDeletedUserID != reference.SourceID ||
		facts.Audit.Action != dispositionTransferAudit || len(facts.Recipients) != 2 ||
		!articleDispositionSameValue(parties.TargetID, reference.NewAuthorID) ||
		parties.TargetEmail == nil {
		return errors.New("article transfer notice binding mismatch")
	}
	contacts := map[string]*string{dispositionNewAuthor: parties.TargetEmail,
		dispositionFormerAuthor: parties.SourceEmail}
	for _, recipient := range facts.Recipients {
		expected, found := contacts[recipient.Kind]
		if !found || recipient.Status != dispositionPending || !validEmail(recipient.EmailSnapshot) ||
			strings.TrimSpace(recipient.EmailSnapshot) != recipient.EmailSnapshot ||
			expected != nil && recipient.EmailSnapshot != *expected {
			return errors.New("article transfer recipient binding mismatch")
		}
		delete(contacts, recipient.Kind)
	}
	return nil
}

func validateArticleDeletionCompletion(reference ArticleCompletionReference, facts VerifiedArticleCompletion) error {
	operation := facts.CompletedOperation
	if facts.AfterArticle.Exists || facts.Notice != nil || len(facts.Recipients) != 0 || operation == nil ||
		operation.ID != *reference.OperationID || operation.ArticleID != reference.ExpectedArticleID ||
		operation.ArticleVersion != reference.BeforeVersion || operation.Status != dispositionCompleted ||
		operation.AuditEventID != reference.AuditEventID {
		return errors.New("article deletion completion binding mismatch")
	}
	expectedAction := map[string]string{dispositionOrdinary: dispositionDeleteAudit,
		dispositionViolating: dispositionViolationDeleteAudit, dispositionOrphan: dispositionOrphanDeleteAudit}
	action, found := expectedAction[operation.DeleteKind]
	if !found || (facts.Audit.Action != action && facts.Audit.Action != dispositionReplayAudit) {
		return errors.New("article deletion completion audit mismatch")
	}
	if operation.DeleteKind == dispositionOrphan {
		if !articleDispositionValue(operation.SourceID, reference.SourceID) ||
			reference.BeforeArticle.Exists && (reference.BeforeArticle.AuthorID != nil ||
				!articleDispositionValue(reference.BeforeArticle.SourceID, reference.SourceID)) {
			return errors.New("article orphan deletion source mismatch")
		}
	} else if !reference.BeforeArticle.Exists || operation.SourceID != nil || reference.BeforeArticle.SourceID != nil ||
		!articleDispositionValue(reference.BeforeArticle.AuthorID, reference.SourceID) {
		return errors.New("article authored deletion source mismatch")
	}
	return nil
}

func validArticleDispositionViolation(state ArticleDispositionState) bool {
	return state.Violation == dispositionClear && state.Category == nil ||
		state.Violation == dispositionViolating && state.Category != nil &&
			(*state.Category == dispositionViolence || *state.Category == dispositionSexual)
}

func articleDispositionValue(value *string, expected string) bool {
	return value != nil && *value == expected
}

func articleDispositionSameValue(left, right *string) bool {
	return left == nil && right == nil || left != nil && right != nil && *left == *right
}

func (lifecycle *ArticleIdentityLifecycle) releaseArticleIdentityEmail(
	ctx context.Context, transaction pgx.Tx, parties LockedArticleParties, traceID string,
) (bool, error) {
	eventID, err := lifecycle.ids.NewID()
	if err != nil {
		return false, fmt.Errorf("generate article identity release audit ID: %w", err)
	}
	if _, err := articleIdentityIDs(eventID, nil); err != nil {
		return false, err
	}
	now := lifecycle.clock.Now().UTC()
	if now.IsZero() {
		return false, errors.New("article identity release time is required")
	}
	command, err := transaction.Exec(ctx, clearArticleIdentityEmailSQL, parties.SourceID, *parties.SourceEmail, now)
	if err != nil {
		return false, fmt.Errorf("clear article retained email: %w", err)
	}
	if command.RowsAffected() != 1 {
		return false, errors.New("article retained email clear did not affect one row")
	}
	command, err = transaction.Exec(ctx, deleteArticleIdentityEmailSQL,
		parties.SourceID, strings.ToLower(*parties.SourceEmail))
	if err != nil {
		return false, fmt.Errorf("release article email identifier: %w", err)
	}
	if command.RowsAffected() != 1 {
		return false, errors.New("article email identifier release did not affect one row")
	}
	actorSystem := dispositionReleaseSystem
	event := audit.Event{ID: eventID, OccurredAt: now, ActorKind: audit.ActorSystem, ActorSystem: &actorSystem,
		Action: canonicalDeleteIdentifierAuditAction, TargetType: canonicalDeleteAuditTargetIdentifier,
		TargetID: parties.SourceID, Outcome: canonicalDeleteAuditOutcomeSucceeded, TraceID: traceID,
		Metadata: map[string]any{"kind": canonicalDeleteIdentifierKindMetadata}}
	if err := lifecycle.audit.Append(ctx, transaction, event); err != nil {
		return false, fmt.Errorf("append article email release audit: %w", err)
	}
	return true, nil
}

const (
	dispositionActive               = "active"
	dispositionPendingDeletion      = "deletion_pending"
	dispositionHeld                 = "held"
	dispositionPublished            = "published"
	dispositionClear                = "clear"
	dispositionViolating            = "violating"
	dispositionViolence             = "violence"
	dispositionSexual               = "sexual"
	dispositionPending              = "pending"
	dispositionCompleted            = "completed"
	dispositionOrdinary             = "ordinary"
	dispositionOrphan               = "orphan"
	dispositionNewAuthor            = "new_author"
	dispositionFormerAuthor         = "former_deleted_author"
	dispositionPublishAudit         = "article.orphan.publish"
	dispositionTransferAudit        = "article.orphan.transfer"
	dispositionDeleteAudit          = "article.delete"
	dispositionViolationDeleteAudit = "article.violation.delete"
	dispositionOrphanDeleteAudit    = "article.orphan.delete"
	dispositionReplayAudit          = "article.deletion.replay"
	dispositionReleaseSystem        = "article_disposition"
)

const clearArticleIdentityEmailSQL = `
	UPDATE users SET retained_email = NULL, updated_at = $3
	WHERE id = $1 AND status = 'deleted' AND deleted_at IS NOT NULL
		AND retained_email = $2 AND email IS NULL AND phone IS NULL
`

const deleteArticleIdentityEmailSQL = `
	DELETE FROM identity_identifiers
	WHERE user_id = $1 AND kind = 'email' AND normalized_value = $2
		AND lifecycle = 'reserved' AND released_at IS NULL
`

func lockArticleIdentityIdentifiers(
	ctx context.Context, transaction pgx.Tx, userID string,
) ([]IdentityIdentifier, error) {
	rows, err := transaction.Query(ctx, lockArticleIdentityIdentifiersSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("lock article identity identifiers: %w", err)
	}
	defer rows.Close()
	identifiers := make([]IdentityIdentifier, 0)
	for rows.Next() {
		var identifier IdentityIdentifier
		if err := rows.Scan(&identifier.UserID, &identifier.Kind, &identifier.NormalizedValue,
			&identifier.Lifecycle, &identifier.ReleasedAt); err != nil {
			return nil, fmt.Errorf("scan article identity identifier: %w", err)
		}
		identifiers = append(identifiers, identifier)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read article identity identifiers: %w", err)
	}
	return identifiers, nil
}

const lockArticleIdentityUserSQL = `
	SELECT id, username, email, phone, status, retained_email, deleted_at
	FROM users WHERE id = $1
	FOR UPDATE NOWAIT
`

const lockArticleIdentityIdentifiersSQL = `
	SELECT user_id, kind, normalized_value, lifecycle, released_at
	FROM identity_identifiers WHERE user_id = $1
	ORDER BY user_id, kind
	FOR UPDATE NOWAIT
`
