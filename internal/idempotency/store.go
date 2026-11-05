// Package idempotency provides persistent scoped idempotency state.
package idempotency

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	scopeDigestDomain   = "han/idempotency/scope/v1"
	keyLocatorDomain    = "han/idempotency/key-locator/v1"
	keyDigestDomain     = "han/idempotency/key-digest/v1"
	requestDigestDomain = "han/idempotency/request-digest/v1"

	// AdministratorCreateResponseSchemaV1 is the closed administrator response schema.
	AdministratorCreateResponseSchemaV1 = "identity-administrator-create-response/v1"
)

var (
	// ErrInvalidRequest indicates incomplete or inconsistent state-machine input.
	ErrInvalidRequest = errors.New("idempotency request is invalid")
	// ErrDigestConflict indicates reuse of a scoped key for another request.
	ErrDigestConflict = errors.New("idempotency key conflicts with another request")
	// ErrDigestKeyUnavailable indicates that a retained key referenced by a record is missing.
	ErrDigestKeyUnavailable = errors.New("idempotency digest key is unavailable")
	// ErrStateConflict indicates that a conditional state transition did not apply.
	ErrStateConflict = errors.New("idempotency state transition conflict")
	// ErrUnsafeSnapshot indicates that a replay snapshot is outside the declared schema.
	ErrUnsafeSnapshot = errors.New("idempotency response snapshot is unsafe")
)

// Status is the durable execution state.
type Status string

const (
	// StatusProcessing reserves a scoped key for one execution generation.
	StatusProcessing Status = "processing"
	// StatusSucceeded stores a successful replay result.
	StatusSucceeded Status = "succeeded"
	// StatusFailed stores a stable failed replay result.
	StatusFailed Status = "failed"
	// StatusUncertain prevents replay after an indeterminate side effect.
	StatusUncertain Status = "uncertain"
	// StatusRecoveryReviewRequired requires an explicit recovery decision.
	StatusRecoveryReviewRequired Status = "recovery_review_required"
)

// BeginAction tells a consumer whether it may execute or must reuse/stop.
type BeginAction string

const (
	// BeginExecute authorizes one execution of the current generation.
	BeginExecute BeginAction = "execute"
	// BeginReplay returns an existing stable response snapshot.
	BeginReplay BeginAction = "replay"
	// BeginInProgress reports an execution that must not be duplicated.
	BeginInProgress BeginAction = "in_progress"
	// BeginOperationUncertain reports a state that requires recovery handling.
	BeginOperationUncertain BeginAction = "operation_uncertain"
)

// Transaction is the caller-owned PostgreSQL transaction surface used by Store.
type Transaction interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// DigestKeyring contains one active HMAC key and every retained verification key.
type DigestKeyring struct {
	ActiveVersion string
	Keys          map[string][]byte
}

// ResponseSnapshot is the complete allowlist for replayable response data.
type ResponseSnapshot struct {
	Outcome    string `json:"outcome"`
	ResourceID string `json:"resource_id,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`

	SchemaVersion string
	HTTPStatus    int
	ResultCode    string
	Body          *AdministratorCreateResponseBody
}

// ResponseFieldError is one stable public field error in a versioned snapshot.
type ResponseFieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// AdministratorCreateResponseBody is the closed success or stable-failure body.
type AdministratorCreateResponseBody struct {
	ID                           string
	Username                     string
	Role                         string
	Status                       string
	MustChangePassword           bool
	MustAcknowledgePrivacy       bool
	TemporaryCredentialExpiresAt string
	DeliveryStatus               string

	Code        string
	Message     string
	FieldErrors []ResponseFieldError
}

// Record is one durable scoped idempotency record.
type Record struct {
	ScopeHash           []byte
	KeyLocator          []byte
	KeyDigest           []byte
	RequestDigest       []byte
	DigestKeyVersion    string
	Status              Status
	ExecutionGeneration int64
	ResultCode          string
	ResponseSnapshot    *ResponseSnapshot
	LockedUntil         *time.Time
	ReviewedBy          *string
	ReviewedAt          *time.Time
	ExpiresAt           time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// BeginRequest describes a logical attempt to claim a scoped idempotency key.
type BeginRequest struct {
	Scope       []byte
	Key         string
	Request     []byte
	DigestKeys  DigestKeyring
	LockedUntil time.Time
	ExpiresAt   time.Time
	Now         time.Time
}

// BeginResult is the authoritative action and current durable record.
type BeginResult struct {
	Action BeginAction
	Record Record
}

// Completion records a stable replayable outcome for a processing generation.
type Completion struct {
	ScopeHash           []byte
	KeyLocator          []byte
	KeyDigest           []byte
	RequestDigest       []byte
	DigestKeyVersion    string
	ExecutionGeneration int64
	Status              Status
	ResultCode          string
	ResponseSnapshot    ResponseSnapshot
	Now                 time.Time
}

// Transition identifies a guarded non-result state transition.
type Transition struct {
	ScopeHash           []byte
	KeyLocator          []byte
	KeyDigest           []byte
	RequestDigest       []byte
	DigestKeyVersion    string
	ExecutionGeneration int64
	Now                 time.Time
}

// RecoveryAction is an explicit review decision.
type RecoveryAction string

const (
	// RecoveryFail closes the reviewed generation with a stable failure.
	RecoveryFail RecoveryAction = "fail"
	// RecoveryRetry creates the next processing generation.
	RecoveryRetry RecoveryAction = "retry"
)

// Recovery records an operator-controlled decision for a review-required record.
type Recovery struct {
	ScopeHash           []byte
	KeyLocator          []byte
	KeyDigest           []byte
	RequestDigest       []byte
	DigestKeyVersion    string
	ExecutionGeneration int64
	Action              RecoveryAction
	ReviewedBy          string
	ResultCode          string
	ResponseSnapshot    ResponseSnapshot
	LockedUntil         time.Time
	ExpiresAt           time.Time
	Now                 time.Time
}

type recordIdentity struct {
	ScopeHash        []byte
	KeyLocator       []byte
	KeyDigest        []byte
	RequestDigest    []byte
	DigestKeyVersion string
}

// Store applies the persistent state machine inside caller-owned transactions.
type Store struct{}

// NewPasswordResetClaim derives the exact six-field password-reset identity.
func NewPasswordResetClaim(request BeginRequest, executionGeneration int64) (Record, error) {
	if err := validateBeginRequestValues(request); err != nil || executionGeneration < 1 {
		return Record{}, ErrInvalidRequest
	}
	identity, err := newRecordIdentity(request)
	if err != nil {
		return Record{}, err
	}
	lockedUntil := request.LockedUntil
	return Record{
		ScopeHash:           bytes.Clone(identity.ScopeHash),
		KeyLocator:          bytes.Clone(identity.KeyLocator),
		KeyDigest:           bytes.Clone(identity.KeyDigest),
		RequestDigest:       bytes.Clone(identity.RequestDigest),
		DigestKeyVersion:    identity.DigestKeyVersion,
		Status:              StatusProcessing,
		ExecutionGeneration: executionGeneration,
		LockedUntil:         &lockedUntil,
		ExpiresAt:           request.ExpiresAt,
		CreatedAt:           request.Now,
		UpdatedAt:           request.Now,
	}, nil
}

// InsertPasswordResetClaim inserts a caller-generation claim in a caller-owned transaction.
func (Store) InsertPasswordResetClaim(
	ctx context.Context,
	tx Transaction,
	request BeginRequest,
	executionGeneration int64,
	operationID string,
	userID string,
) (Record, error) {
	if tx == nil || strings.TrimSpace(operationID) == "" || strings.TrimSpace(userID) == "" {
		return Record{}, ErrInvalidRequest
	}
	record, err := NewPasswordResetClaim(request, executionGeneration)
	if err != nil {
		return Record{}, err
	}
	tag, err := tx.Exec(
		ctx,
		insertPasswordResetProcessingSQL,
		record.ScopeHash,
		record.KeyLocator,
		record.KeyDigest,
		record.RequestDigest,
		record.DigestKeyVersion,
		record.ExecutionGeneration,
		operationID,
		userID,
		request.LockedUntil,
		request.ExpiresAt,
		request.Now,
	)
	if err != nil {
		return Record{}, fmt.Errorf("insert password-reset idempotency record: %w", err)
	}
	if err := requireOneRow(tag); err != nil {
		return Record{}, err
	}
	return record, nil
}

// ResolveExisting classifies an existing scoped key without creating a record.
func (Store) ResolveExisting(
	ctx context.Context,
	tx Transaction,
	request BeginRequest,
) (BeginResult, bool, error) {
	if err := validateBeginRequest(tx, request); err != nil {
		return BeginResult{}, false, err
	}
	identity, err := newRecordIdentity(request)
	if err != nil {
		return BeginResult{}, false, err
	}
	record, err := selectRecord(ctx, tx, identity.ScopeHash, identity.KeyLocator)
	if errors.Is(err, pgx.ErrNoRows) {
		return BeginResult{}, false, nil
	}
	if err != nil {
		return BeginResult{}, false, err
	}
	result, err := classifyExisting(request, record)
	if err != nil {
		return BeginResult{}, true, err
	}
	return result, true, nil
}

// ResolvePasswordResetClaim reads and verifies an existing claim without
// treating its generic status as password-reset authority.
func (Store) ResolvePasswordResetClaim(
	ctx context.Context,
	tx Transaction,
	request BeginRequest,
) (Record, bool, error) {
	if err := validateBeginRequest(tx, request); err != nil {
		return Record{}, false, err
	}
	identity, err := newRecordIdentity(request)
	if err != nil {
		return Record{}, false, err
	}
	record, err := selectRecord(ctx, tx, identity.ScopeHash, identity.KeyLocator)
	if errors.Is(err, pgx.ErrNoRows) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	if err := validateExistingIdentity(request, record); err != nil {
		return Record{}, true, err
	}
	return record, true, nil
}

// Begin inserts a new processing generation or classifies the locked existing row.
func (Store) Begin(
	ctx context.Context,
	tx Transaction,
	request BeginRequest,
) (BeginResult, error) {
	if err := validateBeginRequest(tx, request); err != nil {
		return BeginResult{}, err
	}
	identity, err := newRecordIdentity(request)
	if err != nil {
		return BeginResult{}, err
	}

	tag, err := tx.Exec(ctx, insertProcessingSQL,
		identity.ScopeHash,
		identity.KeyLocator,
		identity.KeyDigest,
		identity.RequestDigest,
		identity.DigestKeyVersion,
		request.LockedUntil,
		request.ExpiresAt,
		request.Now,
	)
	if err != nil {
		return BeginResult{}, fmt.Errorf("insert idempotency record: %w", err)
	}
	if tag.RowsAffected() == 1 {
		lockedUntil := request.LockedUntil
		return BeginResult{
			Action: BeginExecute,
			Record: Record{
				ScopeHash:           bytes.Clone(identity.ScopeHash),
				KeyLocator:          bytes.Clone(identity.KeyLocator),
				KeyDigest:           bytes.Clone(identity.KeyDigest),
				RequestDigest:       bytes.Clone(identity.RequestDigest),
				DigestKeyVersion:    identity.DigestKeyVersion,
				Status:              StatusProcessing,
				ExecutionGeneration: 1,
				LockedUntil:         &lockedUntil,
				ExpiresAt:           request.ExpiresAt,
				CreatedAt:           request.Now,
				UpdatedAt:           request.Now,
			},
		}, nil
	}
	if tag.RowsAffected() != 0 {
		return BeginResult{}, fmt.Errorf(
			"%w: insert affected %d rows",
			ErrStateConflict,
			tag.RowsAffected(),
		)
	}

	record, err := selectRecordForUpdate(ctx, tx, identity.ScopeHash, identity.KeyLocator)
	if err != nil {
		return BeginResult{}, err
	}
	return classifyExisting(request, record)
}

// Complete changes one processing generation to a stable replayable outcome.
func (Store) Complete(ctx context.Context, tx Transaction, completion Completion) error {
	if tx == nil || !validIdentity(
		completion.ScopeHash,
		completion.KeyLocator,
		completion.KeyDigest,
		completion.RequestDigest,
		completion.DigestKeyVersion,
		completion.ExecutionGeneration,
		completion.Now,
	) {
		return ErrInvalidRequest
	}
	if completion.Status != StatusSucceeded && completion.Status != StatusFailed {
		return fmt.Errorf("%w: completion status is not stable", ErrInvalidRequest)
	}
	if strings.TrimSpace(completion.ResultCode) == "" {
		return fmt.Errorf("%w: result code is required", ErrInvalidRequest)
	}
	encodedSnapshot, err := encodeResponseSnapshot(completion.ResponseSnapshot)
	if err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, completeSQL,
		completion.Status,
		completion.ResultCode,
		encodedSnapshot,
		completion.Now,
		completion.ScopeHash,
		completion.KeyLocator,
		completion.KeyDigest,
		completion.RequestDigest,
		completion.DigestKeyVersion,
		completion.ExecutionGeneration,
	)
	if err != nil {
		return fmt.Errorf("complete idempotency record: %w", err)
	}
	return requireOneRow(tag)
}

// MarkUncertain prevents automatic replay of an indeterminate execution.
func (Store) MarkUncertain(ctx context.Context, tx Transaction, transition Transition) error {
	return transitionWithoutResult(ctx, tx, transition, StatusUncertain, markUncertainSQL)
}

// RequireRecoveryReview moves processing or uncertain work behind explicit review.
func (Store) RequireRecoveryReview(
	ctx context.Context,
	tx Transaction,
	transition Transition,
) error {
	return transitionWithoutResult(
		ctx,
		tx,
		transition,
		StatusRecoveryReviewRequired,
		requireRecoveryReviewSQL,
	)
}

// Recover applies an explicit decision to a review-required generation.
func (Store) Recover(ctx context.Context, tx Transaction, recovery Recovery) error {
	if tx == nil || !validIdentity(
		recovery.ScopeHash,
		recovery.KeyLocator,
		recovery.KeyDigest,
		recovery.RequestDigest,
		recovery.DigestKeyVersion,
		recovery.ExecutionGeneration,
		recovery.Now,
	) || strings.TrimSpace(recovery.ReviewedBy) == "" {
		return ErrInvalidRequest
	}

	var (
		tag pgconn.CommandTag
		err error
	)
	switch recovery.Action {
	case RecoveryFail:
		if strings.TrimSpace(recovery.ResultCode) == "" {
			return fmt.Errorf("%w: recovery result code is required", ErrInvalidRequest)
		}
		encodedSnapshot, encodeErr := encodeResponseSnapshot(recovery.ResponseSnapshot)
		if encodeErr != nil {
			return encodeErr
		}
		tag, err = tx.Exec(ctx, recoverAsFailedSQL,
			recovery.ResultCode,
			encodedSnapshot,
			recovery.ReviewedBy,
			recovery.Now,
			recovery.ScopeHash,
			recovery.KeyLocator,
			recovery.KeyDigest,
			recovery.RequestDigest,
			recovery.DigestKeyVersion,
			recovery.ExecutionGeneration,
		)
	case RecoveryRetry:
		if !recovery.LockedUntil.After(recovery.Now) ||
			!recovery.ExpiresAt.After(recovery.Now) {
			return fmt.Errorf("%w: recovery retry deadlines are invalid", ErrInvalidRequest)
		}
		tag, err = tx.Exec(ctx, recoverAsRetrySQL,
			recovery.LockedUntil,
			recovery.ExpiresAt,
			recovery.ReviewedBy,
			recovery.Now,
			recovery.ScopeHash,
			recovery.KeyLocator,
			recovery.KeyDigest,
			recovery.RequestDigest,
			recovery.DigestKeyVersion,
			recovery.ExecutionGeneration,
		)
	default:
		return fmt.Errorf("%w: unsupported recovery action", ErrInvalidRequest)
	}
	if err != nil {
		return fmt.Errorf("recover idempotency record: %w", err)
	}
	return requireOneRow(tag)
}

func validateBeginRequest(tx Transaction, request BeginRequest) error {
	if tx == nil {
		return ErrInvalidRequest
	}
	return validateBeginRequestValues(request)
}

func validateBeginRequestValues(request BeginRequest) error {
	if len(request.Scope) == 0 ||
		strings.TrimSpace(request.Key) == "" ||
		len(request.Request) == 0 ||
		request.Now.IsZero() ||
		!request.LockedUntil.After(request.Now) ||
		!request.ExpiresAt.After(request.Now) {
		return ErrInvalidRequest
	}
	_, err := request.DigestKeys.activeKey()
	return err
}

func newRecordIdentity(request BeginRequest) (recordIdentity, error) {
	activeKey, err := request.DigestKeys.activeKey()
	if err != nil {
		return recordIdentity{}, err
	}
	return recordIdentity{
		ScopeHash: stableDigest(scopeDigestDomain, request.Scope),
		KeyLocator: stableDigest(
			keyLocatorDomain,
			request.Scope,
			[]byte(request.Key),
		),
		KeyDigest: keyedDigest(
			activeKey,
			keyDigestDomain,
			request.Scope,
			[]byte(request.Key),
		),
		RequestDigest: keyedDigest(
			activeKey,
			requestDigestDomain,
			request.Scope,
			request.Request,
		),
		DigestKeyVersion: request.DigestKeys.ActiveVersion,
	}, nil
}

func classifyExisting(request BeginRequest, record Record) (BeginResult, error) {
	if err := validateExistingIdentity(request, record); err != nil {
		return BeginResult{}, err
	}

	result := BeginResult{Record: record}
	switch record.Status {
	case StatusProcessing:
		result.Action = BeginInProgress
	case StatusSucceeded, StatusFailed:
		if record.ResponseSnapshot == nil {
			return BeginResult{}, fmt.Errorf("%w: completed snapshot is missing", ErrStateConflict)
		}
		result.Action = BeginReplay
	case StatusUncertain, StatusRecoveryReviewRequired:
		result.Action = BeginOperationUncertain
	default:
		return BeginResult{}, fmt.Errorf("%w: unsupported stored status", ErrStateConflict)
	}
	return result, nil
}

func validateExistingIdentity(request BeginRequest, record Record) error {
	activeIdentity, err := newRecordIdentity(request)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(activeIdentity.ScopeHash, record.ScopeHash) != 1 ||
		subtle.ConstantTimeCompare(activeIdentity.KeyLocator, record.KeyLocator) != 1 {
		return ErrInvalidRequest
	}

	storedKey, ok := request.DigestKeys.key(record.DigestKeyVersion)
	if !ok {
		return fmt.Errorf(
			"%w: referenced version %q",
			ErrDigestKeyUnavailable,
			record.DigestKeyVersion,
		)
	}
	expectedKeyDigest := keyedDigest(
		storedKey,
		keyDigestDomain,
		request.Scope,
		[]byte(request.Key),
	)
	expectedRequestDigest := keyedDigest(
		storedKey,
		requestDigestDomain,
		request.Scope,
		request.Request,
	)
	if subtle.ConstantTimeCompare(expectedKeyDigest, record.KeyDigest) != 1 ||
		subtle.ConstantTimeCompare(expectedRequestDigest, record.RequestDigest) != 1 {
		return ErrDigestConflict
	}
	return nil
}

func selectRecordForUpdate(
	ctx context.Context,
	tx Transaction,
	scopeHash []byte,
	keyLocator []byte,
) (Record, error) {
	return scanRecord(tx.QueryRow(ctx, selectForUpdateSQL, scopeHash, keyLocator))
}

func selectRecord(
	ctx context.Context,
	tx Transaction,
	scopeHash []byte,
	keyLocator []byte,
) (Record, error) {
	return scanRecord(tx.QueryRow(ctx, selectExistingSQL, scopeHash, keyLocator))
}

func scanRecord(row pgx.Row) (Record, error) {
	var (
		record           Record
		resultCode       *string
		responseSnapshot []byte
	)
	err := row.Scan(
		&record.ScopeHash,
		&record.KeyLocator,
		&record.KeyDigest,
		&record.RequestDigest,
		&record.DigestKeyVersion,
		&record.Status,
		&record.ExecutionGeneration,
		&resultCode,
		&responseSnapshot,
		&record.LockedUntil,
		&record.ReviewedBy,
		&record.ReviewedAt,
		&record.ExpiresAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Record{}, pgx.ErrNoRows
		}
		return Record{}, fmt.Errorf("select idempotency record for update: %w", err)
	}
	if resultCode != nil {
		record.ResultCode = *resultCode
	}
	if responseSnapshot != nil {
		snapshot, err := decodeResponseSnapshot(responseSnapshot)
		if err != nil {
			return Record{}, err
		}
		record.ResponseSnapshot = &snapshot
	}
	return record, nil
}

func transitionWithoutResult(
	ctx context.Context,
	tx Transaction,
	transition Transition,
	status Status,
	statement string,
) error {
	if tx == nil || !validIdentity(
		transition.ScopeHash,
		transition.KeyLocator,
		transition.KeyDigest,
		transition.RequestDigest,
		transition.DigestKeyVersion,
		transition.ExecutionGeneration,
		transition.Now,
	) {
		return ErrInvalidRequest
	}
	tag, err := tx.Exec(ctx, statement,
		status,
		transition.Now,
		transition.ScopeHash,
		transition.KeyLocator,
		transition.KeyDigest,
		transition.RequestDigest,
		transition.DigestKeyVersion,
		transition.ExecutionGeneration,
	)
	if err != nil {
		return fmt.Errorf("transition idempotency record: %w", err)
	}
	return requireOneRow(tag)
}

func validIdentity(
	scopeHash []byte,
	keyLocator []byte,
	keyDigest []byte,
	requestDigest []byte,
	digestKeyVersion string,
	generation int64,
	now time.Time,
) bool {
	return len(scopeHash) > 0 &&
		len(keyLocator) > 0 &&
		len(keyDigest) > 0 &&
		len(requestDigest) > 0 &&
		strings.TrimSpace(digestKeyVersion) != "" &&
		generation >= 1 &&
		!now.IsZero()
}

func requireOneRow(tag pgconn.CommandTag) error {
	if tag.RowsAffected() != 1 {
		return ErrStateConflict
	}
	return nil
}

func (keyring DigestKeyring) activeKey() ([]byte, error) {
	activeVersion := strings.TrimSpace(keyring.ActiveVersion)
	if activeVersion == "" || activeVersion != keyring.ActiveVersion || len(keyring.Keys) == 0 {
		return nil, ErrInvalidRequest
	}
	for version, key := range keyring.Keys {
		if strings.TrimSpace(version) == "" || version != strings.TrimSpace(version) || len(key) == 0 {
			return nil, ErrInvalidRequest
		}
	}
	key, ok := keyring.key(activeVersion)
	if !ok {
		return nil, fmt.Errorf("%w: active version %q", ErrDigestKeyUnavailable, activeVersion)
	}
	return key, nil
}

func (keyring DigestKeyring) key(version string) ([]byte, bool) {
	key, ok := keyring.Keys[version]
	return key, ok && len(key) > 0
}

func stableDigest(domain string, parts ...[]byte) []byte {
	digest := sha256.New()
	writeDigestPart(digest, []byte(domain))
	for _, part := range parts {
		writeDigestPart(digest, part)
	}
	return digest.Sum(nil)
}

func keyedDigest(key []byte, domain string, parts ...[]byte) []byte {
	digest := hmac.New(sha256.New, key)
	writeDigestPart(digest, []byte(domain))
	for _, part := range parts {
		writeDigestPart(digest, part)
	}
	return digest.Sum(nil)
}

func writeDigestPart(digest hash.Hash, part []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(part)))
	_, _ = digest.Write(length[:])
	_, _ = digest.Write(part)
}

func encodeResponseSnapshot(snapshot ResponseSnapshot) ([]byte, error) {
	if err := validateResponseSnapshot(snapshot); err != nil {
		return nil, err
	}
	var value any
	if snapshot.SchemaVersion == "" {
		value = struct {
			Outcome    string `json:"outcome"`
			ResourceID string `json:"resource_id,omitempty"`
			ErrorCode  string `json:"error_code,omitempty"`
		}{
			Outcome: snapshot.Outcome, ResourceID: snapshot.ResourceID,
			ErrorCode: snapshot.ErrorCode,
		}
	} else if snapshot.HTTPStatus == 201 {
		value = struct {
			SchemaVersion string                                 `json:"schema_version"`
			HTTPStatus    int                                    `json:"http_status"`
			ResultCode    string                                 `json:"result_code"`
			Body          administratorCreateSuccessResponseBody `json:"body"`
		}{
			SchemaVersion: snapshot.SchemaVersion,
			HTTPStatus:    snapshot.HTTPStatus,
			ResultCode:    snapshot.ResultCode,
			Body: administratorCreateSuccessResponseBody{
				ID: snapshot.Body.ID, Username: snapshot.Body.Username,
				Role: snapshot.Body.Role, Status: snapshot.Body.Status,
				MustChangePassword:           snapshot.Body.MustChangePassword,
				MustAcknowledgePrivacy:       snapshot.Body.MustAcknowledgePrivacy,
				TemporaryCredentialExpiresAt: snapshot.Body.TemporaryCredentialExpiresAt,
				DeliveryStatus:               snapshot.Body.DeliveryStatus,
			},
		}
	} else {
		value = struct {
			SchemaVersion string                                 `json:"schema_version"`
			HTTPStatus    int                                    `json:"http_status"`
			ResultCode    string                                 `json:"result_code"`
			Body          administratorCreateFailureResponseBody `json:"body"`
		}{
			SchemaVersion: snapshot.SchemaVersion,
			HTTPStatus:    snapshot.HTTPStatus,
			ResultCode:    snapshot.ResultCode,
			Body: administratorCreateFailureResponseBody{
				Code: snapshot.Body.Code, Message: snapshot.Body.Message,
				FieldErrors: snapshot.Body.FieldErrors,
			},
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: encode declared snapshot", ErrUnsafeSnapshot)
	}
	return encoded, nil
}

func decodeResponseSnapshot(encoded json.RawMessage) (ResponseSnapshot, error) {
	if len(encoded) == 0 || !json.Valid(encoded) {
		return ResponseSnapshot{}, fmt.Errorf("%w: snapshot must be valid JSON", ErrUnsafeSnapshot)
	}
	fields, err := decodeSnapshotObject(encoded)
	if err != nil {
		return ResponseSnapshot{}, err
	}
	if _, versioned := fields["schema_version"]; !versioned {
		return decodeLegacyResponseSnapshot(fields)
	}
	return decodeAdministratorCreateResponseSnapshot(fields)
}

type administratorCreateSuccessResponseBody struct {
	ID                           string `json:"id"`
	Username                     string `json:"username"`
	Role                         string `json:"role"`
	Status                       string `json:"status"`
	MustChangePassword           bool   `json:"must_change_password"`
	MustAcknowledgePrivacy       bool   `json:"must_acknowledge_privacy"`
	TemporaryCredentialExpiresAt string `json:"temporary_credential_expires_at"`
	DeliveryStatus               string `json:"delivery_status"`
}

type administratorCreateFailureResponseBody struct {
	Code        string               `json:"code"`
	Message     string               `json:"message"`
	FieldErrors []ResponseFieldError `json:"field_errors"`
}

func decodeLegacyResponseSnapshot(
	fields map[string]json.RawMessage,
) (ResponseSnapshot, error) {
	if !exactSnapshotFields(fields, "outcome", "resource_id", "error_code") ||
		fields["outcome"] == nil {
		return ResponseSnapshot{}, fmt.Errorf("%w: legacy fields are invalid", ErrUnsafeSnapshot)
	}
	var snapshot ResponseSnapshot
	if err := decodeSnapshotString(fields["outcome"], &snapshot.Outcome); err != nil {
		return ResponseSnapshot{}, err
	}
	if raw := fields["resource_id"]; raw != nil {
		if err := decodeSnapshotString(raw, &snapshot.ResourceID); err != nil {
			return ResponseSnapshot{}, err
		}
	}
	if raw := fields["error_code"]; raw != nil {
		if err := decodeSnapshotString(raw, &snapshot.ErrorCode); err != nil {
			return ResponseSnapshot{}, err
		}
	}
	if err := validateResponseSnapshot(snapshot); err != nil {
		return ResponseSnapshot{}, err
	}
	return snapshot, nil
}

func decodeAdministratorCreateResponseSnapshot(
	fields map[string]json.RawMessage,
) (ResponseSnapshot, error) {
	if !exactSnapshotFields(fields, "schema_version", "http_status", "result_code", "body") ||
		fields["schema_version"] == nil || fields["http_status"] == nil ||
		fields["result_code"] == nil || fields["body"] == nil {
		return ResponseSnapshot{}, fmt.Errorf("%w: versioned fields are invalid", ErrUnsafeSnapshot)
	}
	var snapshot ResponseSnapshot
	if err := decodeSnapshotString(fields["schema_version"], &snapshot.SchemaVersion); err != nil {
		return ResponseSnapshot{}, err
	}
	if snapshot.SchemaVersion != AdministratorCreateResponseSchemaV1 {
		return ResponseSnapshot{}, fmt.Errorf("%w: snapshot version is unsupported", ErrUnsafeSnapshot)
	}
	if err := decodeSnapshotValue(fields["http_status"], &snapshot.HTTPStatus); err != nil {
		return ResponseSnapshot{}, err
	}
	if err := decodeSnapshotString(fields["result_code"], &snapshot.ResultCode); err != nil {
		return ResponseSnapshot{}, err
	}
	bodyFields, err := decodeSnapshotObject(fields["body"])
	if err != nil {
		return ResponseSnapshot{}, err
	}
	body := &AdministratorCreateResponseBody{}
	switch {
	case snapshot.HTTPStatus == 201 && snapshot.ResultCode == "created":
		if !exactRequiredSnapshotFields(bodyFields,
			"id", "username", "role", "status", "must_change_password",
			"must_acknowledge_privacy", "temporary_credential_expires_at",
			"delivery_status",
		) {
			return ResponseSnapshot{}, fmt.Errorf("%w: success body fields are invalid", ErrUnsafeSnapshot)
		}
		if err := decodeSnapshotString(bodyFields["id"], &body.ID); err != nil {
			return ResponseSnapshot{}, err
		}
		if err := decodeSnapshotString(bodyFields["username"], &body.Username); err != nil {
			return ResponseSnapshot{}, err
		}
		if err := decodeSnapshotString(bodyFields["role"], &body.Role); err != nil {
			return ResponseSnapshot{}, err
		}
		if err := decodeSnapshotString(bodyFields["status"], &body.Status); err != nil {
			return ResponseSnapshot{}, err
		}
		if err := decodeSnapshotValue(bodyFields["must_change_password"], &body.MustChangePassword); err != nil {
			return ResponseSnapshot{}, err
		}
		if err := decodeSnapshotValue(bodyFields["must_acknowledge_privacy"], &body.MustAcknowledgePrivacy); err != nil {
			return ResponseSnapshot{}, err
		}
		if err := decodeSnapshotString(bodyFields["temporary_credential_expires_at"], &body.TemporaryCredentialExpiresAt); err != nil {
			return ResponseSnapshot{}, err
		}
		if err := decodeSnapshotString(bodyFields["delivery_status"], &body.DeliveryStatus); err != nil {
			return ResponseSnapshot{}, err
		}
	case snapshot.HTTPStatus == 409 &&
		snapshot.ResultCode == "identity.identifier_conflict",
		snapshot.HTTPStatus == 503 &&
			snapshot.ResultCode == "identity.email_delivery_failed":
		if !exactRequiredSnapshotFields(bodyFields, "code", "message", "field_errors") {
			return ResponseSnapshot{}, fmt.Errorf("%w: failure body fields are invalid", ErrUnsafeSnapshot)
		}
		if err := decodeSnapshotString(bodyFields["code"], &body.Code); err != nil {
			return ResponseSnapshot{}, err
		}
		if err := decodeSnapshotString(bodyFields["message"], &body.Message); err != nil {
			return ResponseSnapshot{}, err
		}
		if err := decodeSnapshotValue(bodyFields["field_errors"], &body.FieldErrors); err != nil ||
			body.FieldErrors == nil || len(body.FieldErrors) != 0 {
			return ResponseSnapshot{}, fmt.Errorf("%w: failure field errors are invalid", ErrUnsafeSnapshot)
		}
	default:
		return ResponseSnapshot{}, fmt.Errorf("%w: response mapping is unsupported", ErrUnsafeSnapshot)
	}
	snapshot.Body = body
	if err := validateResponseSnapshot(snapshot); err != nil {
		return ResponseSnapshot{}, err
	}
	return snapshot, nil
}

func decodeSnapshotObject(encoded json.RawMessage) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, fmt.Errorf("%w: snapshot must be one object", ErrUnsafeSnapshot)
	}

	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("%w: decode snapshot field", ErrUnsafeSnapshot)
		}
		name, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("%w: snapshot field name is invalid", ErrUnsafeSnapshot)
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, fmt.Errorf("%w: snapshot field is duplicated", ErrUnsafeSnapshot)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, fmt.Errorf("%w: decode snapshot value", ErrUnsafeSnapshot)
		}
		fields[name] = value
	}

	token, err = decoder.Token()
	if err != nil || token != json.Delim('}') {
		return nil, fmt.Errorf("%w: snapshot object is incomplete", ErrUnsafeSnapshot)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%w: snapshot has trailing data", ErrUnsafeSnapshot)
	}
	return fields, nil
}

func exactSnapshotFields(fields map[string]json.RawMessage, allowed ...string) bool {
	if len(fields) == 0 || len(fields) > len(allowed) {
		return false
	}
	allowlist := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		allowlist[name] = struct{}{}
	}
	for name := range fields {
		if _, ok := allowlist[name]; !ok {
			return false
		}
	}
	return true
}

func exactRequiredSnapshotFields(fields map[string]json.RawMessage, required ...string) bool {
	if len(fields) != len(required) {
		return false
	}
	for _, name := range required {
		if fields[name] == nil {
			return false
		}
	}
	return true
}

func decodeSnapshotString(encoded json.RawMessage, destination *string) error {
	if err := decodeSnapshotValue(encoded, destination); err != nil ||
		destination == nil || strings.TrimSpace(*destination) == "" {
		return fmt.Errorf("%w: snapshot string is invalid", ErrUnsafeSnapshot)
	}
	return nil
}

func decodeSnapshotValue(encoded json.RawMessage, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("%w: decode snapshot value", ErrUnsafeSnapshot)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: snapshot value has trailing data", ErrUnsafeSnapshot)
	}
	return nil
}

func validateResponseSnapshot(snapshot ResponseSnapshot) error {
	if snapshot.SchemaVersion == "" {
		if snapshot.HTTPStatus != 0 || snapshot.ResultCode != "" || snapshot.Body != nil ||
			strings.TrimSpace(snapshot.Outcome) == "" ||
			(snapshot.ResourceID != "" && strings.TrimSpace(snapshot.ResourceID) == "") ||
			(snapshot.ErrorCode != "" && strings.TrimSpace(snapshot.ErrorCode) == "") {
			return fmt.Errorf("%w: declared snapshot fields are invalid", ErrUnsafeSnapshot)
		}
		return nil
	}
	if snapshot.SchemaVersion != AdministratorCreateResponseSchemaV1 ||
		snapshot.Outcome != "" || snapshot.ResourceID != "" || snapshot.ErrorCode != "" ||
		snapshot.Body == nil || strings.TrimSpace(snapshot.ResultCode) == "" {
		return fmt.Errorf("%w: declared snapshot fields are invalid", ErrUnsafeSnapshot)
	}
	body := snapshot.Body
	switch {
	case snapshot.HTTPStatus == 201 && snapshot.ResultCode == "created":
		if strings.TrimSpace(body.ID) == "" || strings.TrimSpace(body.Username) == "" ||
			body.Role != "admin" || body.Status != "active" ||
			!body.MustChangePassword || !body.MustAcknowledgePrivacy ||
			strings.TrimSpace(body.TemporaryCredentialExpiresAt) == "" ||
			body.DeliveryStatus != "accepted" || body.Code != "" ||
			body.Message != "" || body.FieldErrors != nil {
			return fmt.Errorf("%w: administrator success snapshot is invalid", ErrUnsafeSnapshot)
		}
		if parsed, err := time.Parse(time.RFC3339, body.TemporaryCredentialExpiresAt); err != nil ||
			parsed.UTC().Format(time.RFC3339) != body.TemporaryCredentialExpiresAt {
			return fmt.Errorf("%w: administrator success expiry is invalid", ErrUnsafeSnapshot)
		}
	case snapshot.HTTPStatus == 409 &&
		snapshot.ResultCode == "identity.identifier_conflict":
		if body.Code != snapshot.ResultCode || body.Message != "身份标识已被使用" ||
			body.FieldErrors == nil || len(body.FieldErrors) != 0 ||
			administratorSuccessBodyPopulated(body) {
			return fmt.Errorf("%w: identifier conflict snapshot is invalid", ErrUnsafeSnapshot)
		}
	case snapshot.HTTPStatus == 503 &&
		snapshot.ResultCode == "identity.email_delivery_failed":
		if body.Code != snapshot.ResultCode || body.Message != "邮件暂未受理" ||
			body.FieldErrors == nil || len(body.FieldErrors) != 0 ||
			administratorSuccessBodyPopulated(body) {
			return fmt.Errorf("%w: email delivery snapshot is invalid", ErrUnsafeSnapshot)
		}
	default:
		return fmt.Errorf("%w: declared snapshot fields are invalid", ErrUnsafeSnapshot)
	}
	return nil
}

func administratorSuccessBodyPopulated(body *AdministratorCreateResponseBody) bool {
	return body.ID != "" || body.Username != "" || body.Role != "" || body.Status != "" ||
		body.MustChangePassword || body.MustAcknowledgePrivacy ||
		body.TemporaryCredentialExpiresAt != "" || body.DeliveryStatus != ""
}

const insertProcessingSQL = `
	INSERT INTO idempotency_records (
		scope_hash, key_locator, key_digest, request_digest, digest_key_version,
		status, execution_generation, locked_until, expires_at, created_at, updated_at
	)
	VALUES ($1, $2, $3, $4, $5, 'processing', 1, $6, $7, $8, $8)
	ON CONFLICT (scope_hash, key_locator) DO NOTHING
`

const insertPasswordResetProcessingSQL = `
	INSERT INTO idempotency_records (
		scope_hash, key_locator, key_digest, request_digest, digest_key_version,
		status, execution_generation, password_reset_operation_id, password_reset_user_id,
		locked_until, expires_at, created_at, updated_at
	)
	VALUES ($1, $2, $3, $4, $5, 'processing', $6, $7, $8, $9, $10, $11, $11)
`

const selectForUpdateSQL = `
	SELECT
		scope_hash, key_locator, key_digest, request_digest, digest_key_version,
		status, execution_generation, result_code, response_snapshot, locked_until,
		reviewed_by, reviewed_at, expires_at, created_at, updated_at
	FROM idempotency_records
	WHERE scope_hash = $1 AND key_locator = $2
	FOR UPDATE
`

const selectExistingSQL = `
	SELECT
		scope_hash,
		key_locator,
		key_digest,
		request_digest,
		digest_key_version,
		status,
		execution_generation,
		result_code,
		response_snapshot,
		locked_until,
		reviewed_by,
		reviewed_at,
		expires_at,
		created_at,
		updated_at
	FROM idempotency_records
	WHERE scope_hash = $1
		AND key_locator = $2
`

const completeSQL = `
	UPDATE idempotency_records
	SET
		status = $1,
		result_code = $2,
		response_snapshot = $3,
		locked_until = NULL,
		updated_at = $4
	WHERE scope_hash = $5
		AND key_locator = $6
		AND key_digest = $7
		AND request_digest = $8
		AND digest_key_version = $9
		AND execution_generation = $10
		AND status = 'processing'
`

const markUncertainSQL = `
	UPDATE idempotency_records
	SET status = $1, locked_until = NULL, updated_at = $2
	WHERE scope_hash = $3
		AND key_locator = $4
		AND key_digest = $5
		AND request_digest = $6
		AND digest_key_version = $7
		AND execution_generation = $8
		AND status = 'processing'
`

const requireRecoveryReviewSQL = `
	UPDATE idempotency_records
	SET status = $1, locked_until = NULL, updated_at = $2
	WHERE scope_hash = $3
		AND key_locator = $4
		AND key_digest = $5
		AND request_digest = $6
		AND digest_key_version = $7
		AND execution_generation = $8
		AND status IN ('processing', 'uncertain')
`

const recoverAsFailedSQL = `
	UPDATE idempotency_records
	SET
		status = 'failed',
		result_code = $1,
		response_snapshot = $2,
		reviewed_by = $3,
		reviewed_at = $4,
		updated_at = $4
	WHERE scope_hash = $5
		AND key_locator = $6
		AND key_digest = $7
		AND request_digest = $8
		AND digest_key_version = $9
		AND execution_generation = $10
		AND status = 'recovery_review_required'
`

const recoverAsRetrySQL = `
	UPDATE idempotency_records
	SET
		status = 'processing',
		execution_generation = execution_generation + 1,
		locked_until = $1,
		expires_at = $2,
		reviewed_by = $3,
		reviewed_at = $4,
		updated_at = $4
	WHERE scope_hash = $5
		AND key_locator = $6
		AND key_digest = $7
		AND request_digest = $8
		AND digest_key_version = $9
		AND execution_generation = $10
		AND status = 'recovery_review_required'
`
