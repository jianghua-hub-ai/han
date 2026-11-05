package identity

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const temporaryCredentialLifetime = 7 * 24 * time.Hour

var (
	// ErrTemporaryCredentialInvalid indicates malformed credential state.
	ErrTemporaryCredentialInvalid = errors.New("temporary credential is invalid")
	// ErrTemporaryCredentialGeneration indicates a non-increasing replacement generation.
	ErrTemporaryCredentialGeneration = errors.New("temporary credential generation is invalid")
	// ErrTemporaryCredentialTransition indicates an unsupported lifecycle transition.
	ErrTemporaryCredentialTransition = errors.New("temporary credential transition is invalid")
)

// TemporaryCredentialHasher is the Argon2id contract used to protect a generated secret.
type TemporaryCredentialHasher interface {
	Hash(secret string) (string, error)
	Verify(secret, encoded string) (matched bool, needsRehash bool, err error)
}

// TemporaryCredentialInput contains the non-persisted input for one pending credential.
type TemporaryCredentialInput struct {
	ID                string
	UserID            string
	Purpose           TemporaryCredentialPurpose
	Generation        int64
	AuthGeneration    int64
	SourceOperationID *string
	Secret            string
	DeliveredToEmail  string
	IssuedAt          time.Time
}

// NewTemporaryCredential hashes and verifies a pending credential with a fixed lifetime.
func NewTemporaryCredential(
	hasher TemporaryCredentialHasher,
	input TemporaryCredentialInput,
) (TemporaryCredential, error) {
	if hasher == nil ||
		strings.TrimSpace(input.ID) == "" ||
		strings.TrimSpace(input.UserID) == "" ||
		input.Generation < 1 ||
		input.Secret == "" ||
		strings.TrimSpace(input.DeliveredToEmail) == "" ||
		input.IssuedAt.IsZero() ||
		!validTemporaryCredentialPurpose(input.Purpose) {
		return TemporaryCredential{}, ErrTemporaryCredentialInvalid
	}

	digest, err := hasher.Hash(input.Secret)
	if err != nil || !validArgon2idEncoding(digest) {
		return TemporaryCredential{}, fmt.Errorf(
			"%w: Argon2id hashing failed",
			ErrTemporaryCredentialInvalid,
		)
	}
	matched, needsRehash, err := hasher.Verify(input.Secret, digest)
	if err != nil || !matched || needsRehash {
		return TemporaryCredential{}, fmt.Errorf(
			"%w: Argon2id verification failed",
			ErrTemporaryCredentialInvalid,
		)
	}

	credential := TemporaryCredential{
		ID:                input.ID,
		UserID:            input.UserID,
		Purpose:           input.Purpose,
		Generation:        input.Generation,
		AuthGeneration:    input.AuthGeneration,
		SourceOperationID: input.SourceOperationID,
		State:             TemporaryCredentialPendingDelivery,
		SecretDigest:      digest,
		DeliveredToEmail:  input.DeliveredToEmail,
		IssuedAt:          input.IssuedAt,
		ExpiresAt:         input.IssuedAt.Add(temporaryCredentialLifetime),
	}
	if err := ValidateTemporaryCredential(credential); err != nil {
		return TemporaryCredential{}, err
	}
	return credential, nil
}

// ValidateTemporaryCredential validates the structural credential state.
func ValidateTemporaryCredential(credential TemporaryCredential) error {
	if strings.TrimSpace(credential.ID) == "" ||
		strings.TrimSpace(credential.UserID) == "" ||
		!validArgon2idEncoding(credential.SecretDigest) ||
		strings.TrimSpace(credential.DeliveredToEmail) == "" ||
		credential.Generation < 1 ||
		credential.IssuedAt.IsZero() ||
		!credential.ExpiresAt.Equal(
			credential.IssuedAt.Add(temporaryCredentialLifetime),
		) {
		return ErrTemporaryCredentialInvalid
	}
	if !validTemporaryCredentialPurpose(credential.Purpose) {
		return ErrTemporaryCredentialInvalid
	}
	if credential.AuthGeneration < 0 ||
		(credential.SourceOperationID == nil) != (credential.AuthGeneration == 0) ||
		(credential.SourceOperationID != nil &&
			(credential.Purpose != TemporaryCredentialPasswordReset ||
				strings.TrimSpace(*credential.SourceOperationID) == "")) {
		return ErrTemporaryCredentialInvalid
	}

	switch credential.State {
	case TemporaryCredentialPendingDelivery,
		TemporaryCredentialActive,
		TemporaryCredentialExpired:
		if credential.SupersededAt != nil ||
			credential.ConsumedAt != nil ||
			credential.InvalidatedAt != nil {
			return ErrTemporaryCredentialInvalid
		}
	case TemporaryCredentialSuperseded:
		if credential.SupersededAt == nil ||
			credential.ConsumedAt != nil ||
			credential.InvalidatedAt != nil {
			return ErrTemporaryCredentialInvalid
		}
	case TemporaryCredentialConsumed:
		if credential.SupersededAt != nil ||
			credential.ConsumedAt == nil ||
			credential.InvalidatedAt != nil {
			return ErrTemporaryCredentialInvalid
		}
	case TemporaryCredentialRecoveryInvalidated:
		if credential.SupersededAt != nil ||
			credential.ConsumedAt != nil ||
			credential.InvalidatedAt == nil {
			return ErrTemporaryCredentialInvalid
		}
	default:
		return ErrTemporaryCredentialInvalid
	}
	return nil
}

// TemporaryCredentialAuthenticatable reports whether the credential is the active,
// unexpired latest generation.
func TemporaryCredentialAuthenticatable(
	credential TemporaryCredential,
	latestGeneration int64,
	now time.Time,
) bool {
	if ValidateTemporaryCredential(credential) != nil ||
		now.IsZero() ||
		credential.State != TemporaryCredentialActive ||
		credential.Generation != latestGeneration {
		return false
	}
	return now.Before(credential.ExpiresAt)
}

// TemporaryCredentialCurrent reports whether the record occupies the current slot.
func TemporaryCredentialCurrent(credential TemporaryCredential) bool {
	if ValidateTemporaryCredential(credential) != nil {
		return false
	}
	switch credential.State {
	case TemporaryCredentialPendingDelivery,
		TemporaryCredentialActive,
		TemporaryCredentialExpired:
		return true
	default:
		return false
	}
}

// TransitionTemporaryCredential returns a transitioned copy without mutating input.
func TransitionTemporaryCredential(
	credential TemporaryCredential,
	target TemporaryCredentialState,
	at time.Time,
) (TemporaryCredential, error) {
	if err := ValidateTemporaryCredential(credential); err != nil {
		return TemporaryCredential{}, err
	}
	if at.IsZero() || at.Before(credential.IssuedAt) {
		return TemporaryCredential{}, ErrTemporaryCredentialTransition
	}
	if !allowedTemporaryCredentialTransition(credential.State, target) {
		return TemporaryCredential{}, ErrTemporaryCredentialTransition
	}
	if target == TemporaryCredentialActive && !at.Before(credential.ExpiresAt) {
		return TemporaryCredential{}, ErrTemporaryCredentialTransition
	}
	if target == TemporaryCredentialExpired && at.Before(credential.ExpiresAt) {
		return TemporaryCredential{}, ErrTemporaryCredentialTransition
	}

	transitioned := credential
	transitioned.State = target
	switch target {
	case TemporaryCredentialSuperseded:
		transitioned.SupersededAt = timePointer(at)
	case TemporaryCredentialConsumed:
		transitioned.ConsumedAt = timePointer(at)
	case TemporaryCredentialRecoveryInvalidated:
		transitioned.InvalidatedAt = timePointer(at)
	}
	if err := ValidateTemporaryCredential(transitioned); err != nil {
		return TemporaryCredential{}, fmt.Errorf("%w: %v", ErrTemporaryCredentialTransition, err)
	}
	return transitioned, nil
}

// ActivateTemporaryCredentialReplacement atomically derives the two records that
// a caller must persist after successful delivery. On error neither input changes.
func ActivateTemporaryCredentialReplacement(
	current TemporaryCredential,
	pending TemporaryCredential,
	deliveredAt time.Time,
) (TemporaryCredential, TemporaryCredential, error) {
	if err := ValidateTemporaryCredential(current); err != nil {
		return TemporaryCredential{}, TemporaryCredential{}, err
	}
	if err := ValidateTemporaryCredential(pending); err != nil {
		return TemporaryCredential{}, TemporaryCredential{}, err
	}
	if current.UserID != pending.UserID ||
		current.Purpose != pending.Purpose ||
		pending.Generation <= current.Generation {
		return TemporaryCredential{}, TemporaryCredential{}, ErrTemporaryCredentialGeneration
	}
	if pending.State != TemporaryCredentialPendingDelivery {
		return TemporaryCredential{}, TemporaryCredential{}, ErrTemporaryCredentialTransition
	}
	switch current.State {
	case TemporaryCredentialPendingDelivery,
		TemporaryCredentialActive,
		TemporaryCredentialExpired:
	default:
		return TemporaryCredential{}, TemporaryCredential{}, ErrTemporaryCredentialTransition
	}

	superseded, err := TransitionTemporaryCredential(
		current,
		TemporaryCredentialSuperseded,
		deliveredAt,
	)
	if err != nil {
		return TemporaryCredential{}, TemporaryCredential{}, err
	}
	active, err := TransitionTemporaryCredential(
		pending,
		TemporaryCredentialActive,
		deliveredAt,
	)
	if err != nil {
		return TemporaryCredential{}, TemporaryCredential{}, err
	}
	return superseded, active, nil
}

func allowedTemporaryCredentialTransition(
	current TemporaryCredentialState,
	target TemporaryCredentialState,
) bool {
	switch current {
	case TemporaryCredentialPendingDelivery:
		return target == TemporaryCredentialActive ||
			target == TemporaryCredentialSuperseded ||
			target == TemporaryCredentialRecoveryInvalidated
	case TemporaryCredentialActive:
		return target == TemporaryCredentialExpired ||
			target == TemporaryCredentialSuperseded ||
			target == TemporaryCredentialConsumed ||
			target == TemporaryCredentialRecoveryInvalidated
	case TemporaryCredentialExpired:
		return target == TemporaryCredentialSuperseded ||
			target == TemporaryCredentialRecoveryInvalidated
	default:
		return false
	}
}

func validTemporaryCredentialPurpose(purpose TemporaryCredentialPurpose) bool {
	switch purpose {
	case TemporaryCredentialNewAdministrator, TemporaryCredentialPasswordReset:
		return true
	default:
		return false
	}
}

func validArgon2idEncoding(encoded string) bool {
	if encoded == "" || strings.TrimSpace(encoded) != encoded {
		return false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 ||
		parts[0] != "" ||
		parts[1] != "argon2id" ||
		parts[2] != "v=19" {
		return false
	}

	parameters := strings.Split(parts[3], ",")
	if len(parameters) != 3 {
		return false
	}
	memory, ok := positiveArgon2idParameter(parameters[0], "m=", 32)
	if !ok {
		return false
	}
	iterations, ok := positiveArgon2idParameter(parameters[1], "t=", 32)
	if !ok || iterations == 0 {
		return false
	}
	parallelism, ok := positiveArgon2idParameter(parameters[2], "p=", 8)
	if !ok || parallelism == 0 || memory < 8*parallelism {
		return false
	}

	decoder := base64.RawStdEncoding.Strict()
	salt, err := decoder.DecodeString(parts[4])
	if err != nil || len(salt) == 0 {
		return false
	}
	digest, err := decoder.DecodeString(parts[5])
	return err == nil && len(digest) > 0
}

func positiveArgon2idParameter(encoded, prefix string, bitSize int) (uint64, bool) {
	if !strings.HasPrefix(encoded, prefix) {
		return 0, false
	}
	value := strings.TrimPrefix(encoded, prefix)
	if value == "" || (len(value) > 1 && value[0] == '0') {
		return 0, false
	}
	parsed, err := strconv.ParseUint(value, 10, bitSize)
	return parsed, err == nil && parsed > 0
}

func timePointer(value time.Time) *time.Time {
	return &value
}
