package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"han/internal/audit"
	"han/internal/platform"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	registrationAuditAction           = "identity.register"
	registrationAuditOutcomeSucceeded = "succeeded"
	registrationAuditOutcomeFailed    = "failed"
	registrationAuditTargetUser       = "user"
	registrationAuditTargetAttempt    = "registration_attempt"
)

type normalizedRegistration struct {
	username             string
	normalizedUsername   string
	email                string
	normalizedEmail      string
	phone                string
	password             string
	privacyNoticeVersion string
	sourceKind           string
	sourceFingerprint    string
}

// CurrentNoticeReader reads and locks the current effective notice.
type CurrentNoticeReader struct{}

// CurrentNotice returns the current privacy notice in the caller-owned transaction.
func (CurrentNoticeReader) CurrentNotice(ctx context.Context, tx pgx.Tx) (PrivacyNotice, error) {
	if tx == nil {
		return PrivacyNotice{}, errors.New("privacy notice transaction is required")
	}
	var notice PrivacyNotice
	err := tx.QueryRow(ctx, `
		SELECT version, content, content_hash, effective_at, status
		FROM privacy_notices
		WHERE status = $1 AND effective_at <= now()
		ORDER BY effective_at DESC
		LIMIT 1
		FOR SHARE
	`, PrivacyNoticeCurrent).Scan(
		&notice.Version,
		&notice.Content,
		&notice.ContentHash,
		&notice.EffectiveAt,
		&notice.Status,
	)
	if err != nil {
		return PrivacyNotice{}, fmt.Errorf("query current privacy notice: %w", err)
	}
	return notice, nil
}

// SystemClock supplies UTC wall-clock timestamps.
type SystemClock struct{}

// Now returns the current UTC time.
func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}

// RandomUUIDGenerator creates RFC 4122 version 4 identifiers.
type RandomUUIDGenerator struct{}

// NewID returns one random UUID string.
func (RandomUUIDGenerator) NewID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate registration identifier: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(value[:])
	return strings.Join([]string{
		encoded[0:8],
		encoded[8:12],
		encoded[12:16],
		encoded[16:20],
		encoded[20:32],
	}, "-"), nil
}

func (s *Service) registerInTransaction(
	ctx context.Context,
	tx pgx.Tx,
	request normalizedRegistration,
) (RegisteredUser, error) {
	notice, err := s.dependencies.Notices.CurrentNotice(ctx, tx)
	if err != nil {
		return RegisteredUser{}, currentNoticeError(err)
	}
	if request.privacyNoticeVersion != notice.Version {
		return RegisteredUser{}, platform.NewPublicError(
			platform.CodePrivacyNoticeOutdated,
			errors.New("registration notice version is not current"),
		)
	}

	passwordDigest, err := s.dependencies.PasswordHasher.Hash(request.password)
	if err != nil {
		return RegisteredUser{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			fmt.Errorf("hash registration password: %w", err),
		)
	}
	traceID, ok := platform.TraceIDFromContext(ctx)
	if !ok {
		return RegisteredUser{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("registration trace ID is required"),
		)
	}

	aggregate, err := s.newRegistrationAggregate(request, passwordDigest, traceID)
	if err != nil {
		return RegisteredUser{}, err
	}
	if err := s.persistRegistration(ctx, tx, aggregate); err != nil {
		return aggregate.result, err
	}
	return aggregate.result, nil
}

type registrationAggregate struct {
	user            User
	identifiers     []IdentityIdentifier
	acknowledgement PrivacyAcknowledgement
	auditEvent      audit.Event
	result          RegisteredUser
}

func (s *Service) newRegistrationAggregate(
	request normalizedRegistration,
	passwordDigest string,
	traceID string,
) (registrationAggregate, error) {
	userID, acknowledgementID, auditID, err := s.registrationIDs()
	if err != nil {
		return registrationAggregate{}, err
	}
	now := s.dependencies.Clock.Now().UTC()
	email, phone, digest := request.email, request.phone, passwordDigest
	noticeVersion := request.privacyNoticeVersion

	user := User{
		ID: userID, Username: request.username, Email: &email, Phone: &phone,
		PermanentPasswordDigest: &digest, Role: RoleMember, Status: UserStatusActive,
		CredentialMode: CredentialModePermanent, RecoveryRequired: false,
		OnboardingPrivacyAcknowledgedVersion: &noticeVersion,
		OnboardingCompletedAt:                &now,
		CreatedAt:                            now,
		UpdatedAt:                            now,
	}
	sourceFingerprint := request.sourceFingerprint
	return registrationAggregate{
		user: user,
		identifiers: []IdentityIdentifier{
			newRegistrationIdentifier(userID, IdentifierUsername, request.normalizedUsername, now),
			newRegistrationIdentifier(userID, IdentifierEmail, request.normalizedEmail, now),
			newRegistrationIdentifier(userID, IdentifierPhone, request.phone, now),
		},
		acknowledgement: PrivacyAcknowledgement{
			ID: acknowledgementID, UserID: userID, NoticeVersion: noticeVersion,
			AcknowledgedAt: now, SourceKind: request.sourceKind,
			SourceFingerprint: request.sourceFingerprint,
		},
		auditEvent: audit.Event{
			ID: auditID, OccurredAt: now, ActorKind: audit.ActorAnonymous,
			Action: registrationAuditAction, TargetType: registrationAuditTargetUser,
			TargetID: userID, Outcome: registrationAuditOutcomeSucceeded, TraceID: traceID,
			SourceFingerprint: &sourceFingerprint,
			Metadata:          map[string]any{"notice_version": noticeVersion},
		},
		result: RegisteredUser{
			ID: userID, Username: request.username, Role: RoleMember,
			Status: UserStatusActive, CreatedAt: now,
		},
	}, nil
}

func (s *Service) persistRegistration(
	ctx context.Context,
	tx pgx.Tx,
	aggregate registrationAggregate,
) error {
	if err := s.dependencies.Repository.InsertUser(ctx, tx, aggregate.user); err != nil {
		return markRegistrationPersistenceFailure(
			persistenceError("insert registered user", err),
		)
	}
	for _, identifier := range aggregate.identifiers {
		if err := s.dependencies.Repository.InsertIdentifier(ctx, tx, identifier); err != nil {
			if isUniqueViolation(err) {
				return markRegistrationPersistenceFailure(
					platform.NewPublicError(platform.CodeIdentityIdentifierConflict, err),
				)
			}
			return markRegistrationPersistenceFailure(
				persistenceError("insert registration identifier", err),
			)
		}
	}
	if err := s.dependencies.Repository.InsertPrivacyAcknowledgement(
		ctx,
		tx,
		aggregate.acknowledgement,
	); err != nil {
		return markRegistrationPersistenceFailure(
			persistenceError("insert registration privacy acknowledgement", err),
		)
	}
	if err := s.dependencies.Audit.Append(ctx, tx, aggregate.auditEvent); err != nil {
		return markRegistrationPersistenceFailure(
			persistenceError("append registration audit event", err),
		)
	}
	return nil
}

func (s *Service) finishFailedRegistration(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	cause error,
	rollbackConfirmed bool,
) error {
	if !rollbackConfirmed {
		if err := tx.Rollback(ctx); err != nil {
			return platform.NewPublicError(
				platform.CodePlatformDatabaseUnavailable,
				errors.Join(cause, fmt.Errorf("rollback failed registration: %w", err)),
			)
		}
	}
	if !isRegistrationPersistenceFailure(cause) {
		return cause
	}
	return s.appendRegistrationFailureAudit(ctx, userID, cause)
}

func (s *Service) appendRegistrationFailureAudit(
	ctx context.Context,
	userID string,
	cause error,
) error {
	event, err := s.newRegistrationFailureEvent(ctx, userID, cause)
	if err != nil {
		return platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.Join(cause, err),
		)
	}
	tx, err := s.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return failureAuditPersistenceError(cause, "begin failure audit transaction", err)
	}
	if err := s.appendFailureAuditInTransaction(ctx, tx, event); err != nil {
		return failureAuditPersistenceError(cause, "persist registration failure audit", err)
	}
	return cause
}

func (s *Service) appendFailureAuditInTransaction(
	ctx context.Context,
	tx pgx.Tx,
	event audit.Event,
) error {
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := s.dependencies.Audit.Append(ctx, tx, event); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *Service) newRegistrationFailureEvent(
	ctx context.Context,
	userID string,
	cause error,
) (audit.Event, error) {
	traceID, ok := platform.TraceIDFromContext(ctx)
	if !ok {
		return audit.Event{}, errors.New("registration trace ID is required")
	}
	eventID, err := s.dependencies.IDs.NewID()
	if err != nil {
		return audit.Event{}, fmt.Errorf("generate registration failure audit ID: %w", err)
	}
	targetType, targetID := registrationAuditTargetUser, userID
	if targetID == "" {
		targetType, targetID = registrationAuditTargetAttempt, traceID
	}
	errorCode := string(platform.PublicCode(cause))
	return audit.Event{
		ID: eventID, OccurredAt: s.dependencies.Clock.Now().UTC(),
		ActorKind: audit.ActorAnonymous, Action: registrationAuditAction,
		TargetType: targetType, TargetID: targetID,
		Outcome: registrationAuditOutcomeFailed, ErrorCode: &errorCode,
		TraceID: traceID,
	}, nil
}

func (s *Service) registrationIDs() (string, string, string, error) {
	userID, err := s.dependencies.IDs.NewID()
	if err != nil {
		return "", "", "", identifierGenerationError(err)
	}
	acknowledgementID, err := s.dependencies.IDs.NewID()
	if err != nil {
		return "", "", "", identifierGenerationError(err)
	}
	auditID, err := s.dependencies.IDs.NewID()
	if err != nil {
		return "", "", "", identifierGenerationError(err)
	}
	return userID, acknowledgementID, auditID, nil
}

func validateRegistrationRequest(
	request RegistrationRequest,
	passwordPolicy RegistrationPasswordPolicy,
) (normalizedRegistration, error) {
	if !request.PrivacyAcknowledged {
		return normalizedRegistration{}, platform.NewPublicError(
			platform.CodePrivacyAcknowledgementRequired,
			errors.New("registration privacy acknowledgement is required"),
		)
	}

	username := strings.TrimSpace(request.Username)
	email := strings.TrimSpace(request.Email)
	phone := strings.TrimSpace(request.Phone)
	noticeVersion := strings.TrimSpace(request.PrivacyNoticeVersion)
	if !validUsername(username) || !validEmail(email) || phone == "" || noticeVersion == "" {
		return normalizedRegistration{}, validationError(errors.New("registration fields are invalid"))
	}
	if err := passwordPolicy.Validate(request.Password); err != nil {
		return normalizedRegistration{}, validationError(err)
	}
	sourceKind := strings.TrimSpace(request.SourceKind)
	sourceFingerprint := strings.TrimSpace(request.SourceFingerprint)
	if sourceKind == "" || sourceFingerprint == "" {
		return normalizedRegistration{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("registration source evidence is incomplete"),
		)
	}
	return normalizedRegistration{
		username: username, normalizedUsername: strings.ToLower(username),
		email: email, normalizedEmail: strings.ToLower(email), phone: phone,
		password: request.Password, privacyNoticeVersion: noticeVersion,
		sourceKind: sourceKind, sourceFingerprint: sourceFingerprint,
	}, nil
}

func validUsername(username string) bool {
	if username == "" {
		return false
	}
	for _, character := range username {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Name == "" && strings.EqualFold(address.Address, email)
}

func newRegistrationIdentifier(
	userID string,
	kind IdentifierKind,
	value string,
	createdAt time.Time,
) IdentityIdentifier {
	return IdentityIdentifier{
		UserID: userID, Kind: kind, NormalizedValue: value,
		Lifecycle: IdentifierActive, CreatedAt: createdAt,
	}
}

func processingGateError(cause error) error {
	return platform.NewPublicError(platform.CodePlatformProcessingGateClosed, cause)
}

func validationError(cause error) error {
	return platform.NewPublicError(platform.CodeIdentityValidationFailed, cause)
}

func currentNoticeError(cause error) error {
	if errors.Is(cause, pgx.ErrNoRows) {
		return platform.NewPublicError(platform.CodePrivacyNoticeOutdated, cause)
	}
	return markRegistrationPersistenceFailure(
		persistenceError("read current privacy notice", cause),
	)
}

func persistenceError(operation string, cause error) error {
	return platform.NewPublicError(
		platform.CodePlatformDatabaseUnavailable,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

func identifierGenerationError(cause error) error {
	return platform.NewPublicError(
		platform.CodePlatformInternalError,
		fmt.Errorf("generate registration identifier: %w", cause),
	)
}

type registrationPersistenceFailure struct {
	cause error
}

func (e *registrationPersistenceFailure) Error() string {
	return e.cause.Error()
}

func (e *registrationPersistenceFailure) Unwrap() error {
	return e.cause
}

func markRegistrationPersistenceFailure(cause error) error {
	return &registrationPersistenceFailure{cause: cause}
}

func isRegistrationPersistenceFailure(err error) bool {
	var failure *registrationPersistenceFailure
	return errors.As(err, &failure)
}

func failureAuditPersistenceError(cause error, operation string, auditCause error) error {
	return platform.NewPublicError(
		platform.CodePlatformDatabaseUnavailable,
		errors.Join(cause, fmt.Errorf("%s: %w", operation, auditCause)),
	)
}

func isUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}
