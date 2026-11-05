package recovery

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// ErrInvalidKeyring indicates that the recovery keyring is incomplete or ambiguous.
var ErrInvalidKeyring = errors.New("invalid recovery keyring")

// ErrKeyRetirementBlocked indicates that a retained key still has either live runs or verifiable records.
var ErrKeyRetirementBlocked = errors.New("recovery key retirement blocked")

var (
	// ErrKeyRetirementVerifierUnavailable indicates that no verifier is configured or it could not decide.
	ErrKeyRetirementVerifierUnavailable = errors.New("recovery key retirement verifier unavailable")
	// ErrKeyRetirementCapabilityRejected indicates that the verifier rejected the capability or request binding.
	ErrKeyRetirementCapabilityRejected = errors.New("recovery key retirement capability rejected")
	// ErrKeyRetirementCapabilityIndeterminate indicates that the verifier returned no authoritative decision.
	ErrKeyRetirementCapabilityIndeterminate = errors.New("recovery key retirement capability indeterminate")
)

// KeyState defines whether a recovery key may write new records or only verify retained records.
type KeyState string

const (
	// KeyStateActive identifies the single key allowed to write new begun records.
	KeyStateActive KeyState = "active"
	// KeyStateVerifyOnly identifies a retained key that may only verify and detect replay.
	KeyStateVerifyOnly KeyState = "verify_only"
)

// Key is one versioned HMAC key in the recovery keyring.
type Key struct {
	ID       string
	Version  string
	State    KeyState
	Material []byte
}

// Keyring contains exactly one active key and every retained verify-only key.
type Keyring struct {
	keys   []Key
	byRef  map[keyReference]Key
	active Key
}

type keyReference struct {
	id      string
	version string
}

// KeyRetirementCapability is an opaque, transient input to a key retirement request.
// S4.2 intentionally provides no production mint or parser.
type KeyRetirementCapability struct {
	opaque string
}

// KeyRetirementRequest binds capability verification to one exact key version.
type KeyRetirementRequest struct {
	KeyID      string
	KeyVersion string
}

// KeyRetirementVerification is the authoritative result of an atomic verify-and-consume attempt.
type KeyRetirementVerification string

const (
	KeyRetirementVerificationApproved      KeyRetirementVerification = "approved"
	KeyRetirementVerificationRejected      KeyRetirementVerification = "rejected"
	KeyRetirementVerificationIndeterminate KeyRetirementVerification = "indeterminate"
)

// KeyRetirementCapabilityVerifier atomically verifies and consumes a capability.
// Implementations must reject copied or replayed capabilities.
type KeyRetirementCapabilityVerifier interface {
	VerifyAndConsume(
		ctx context.Context,
		request KeyRetirementRequest,
		capability KeyRetirementCapability,
	) (KeyRetirementVerification, error)
}

// NewKeyring validates and copies a complete recovery keyring.
func NewKeyring(keys []Key) (Keyring, error) {
	if len(keys) == 0 {
		return Keyring{}, fmt.Errorf("%w: no keys", ErrInvalidKeyring)
	}

	keyring := Keyring{
		keys:  make([]Key, 0, len(keys)),
		byRef: make(map[keyReference]Key, len(keys)),
	}
	activeCount := 0
	for _, candidate := range keys {
		key, err := validateAndCopyKey(candidate)
		if err != nil {
			return Keyring{}, err
		}
		ref := keyReference{id: key.ID, version: key.Version}
		if _, exists := keyring.byRef[ref]; exists {
			return Keyring{}, fmt.Errorf("%w: duplicate key reference", ErrInvalidKeyring)
		}
		if key.State == KeyStateActive {
			activeCount++
			keyring.active = key
		}
		keyring.keys = append(keyring.keys, key)
		keyring.byRef[ref] = key
	}
	if activeCount != 1 {
		return Keyring{}, fmt.Errorf("%w: expected exactly one active key", ErrInvalidKeyring)
	}
	return keyring, nil
}

func validateAndCopyKey(candidate Key) (Key, error) {
	if invalidKeyReferencePart(candidate.ID) {
		return Key{}, fmt.Errorf("%w: invalid key id", ErrInvalidKeyring)
	}
	if invalidKeyReferencePart(candidate.Version) {
		return Key{}, fmt.Errorf("%w: invalid key version", ErrInvalidKeyring)
	}
	if candidate.State != KeyStateActive && candidate.State != KeyStateVerifyOnly {
		return Key{}, fmt.Errorf("%w: invalid key state", ErrInvalidKeyring)
	}
	if len(candidate.Material) == 0 {
		return Key{}, fmt.Errorf("%w: empty key material", ErrInvalidKeyring)
	}
	candidate.Material = append([]byte(nil), candidate.Material...)
	return candidate, nil
}

func (keyring Keyring) retained() []Key {
	keys := make([]Key, len(keyring.keys))
	copy(keys, keyring.keys)
	return keys
}

func (keyring Keyring) activeKey() Key {
	key := keyring.active
	key.Material = append([]byte(nil), key.Material...)
	return key
}

func (keyring Keyring) lookup(keyID, keyVersion string) (Key, bool) {
	key, ok := keyring.byRef[keyReference{id: keyID, version: keyVersion}]
	if !ok {
		return Key{}, false
	}
	key.Material = append([]byte(nil), key.Material...)
	return key, true
}

func invalidKeyReferencePart(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return true
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return true
		}
	}
	return false
}

func fingerprintSecret(key Key, secret []byte) string {
	return hmacDigest(key.Material, secret)
}

func hmacDigest(key, message []byte) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(message)
	return "hmac-sha256:" + hex.EncodeToString(mac.Sum(nil))
}
