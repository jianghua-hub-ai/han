package identity

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"han/internal/platform"
)

const (
	identityCursorVersion        = 1
	identityCursorSignatureBytes = sha256.Size
	maxIdentityCursorTokenBytes  = 4096
)

// IdentityCursorPosition is the exclusive keyset boundary for the next page.
type IdentityCursorPosition struct {
	CreatedAt time.Time
	ID        string
}

// IdentityCursorBinding prevents a cursor from crossing query contracts.
type IdentityCursorBinding struct {
	Endpoint     string
	Category     UserQueryCategory
	Visibility   UserQueryVisibility
	Limit        int
	FilterDigest string
}

// IdentityCursorCodec signs and validates versioned identity query cursors.
type IdentityCursorCodec struct {
	key []byte
	ttl time.Duration
	now func() time.Time
}

type identityCursorPayload struct {
	Version      int                 `json:"version"`
	CreatedAt    string              `json:"created_at"`
	ID           string              `json:"id"`
	Endpoint     string              `json:"endpoint"`
	Category     UserQueryCategory   `json:"category"`
	Visibility   UserQueryVisibility `json:"visibility"`
	Limit        int                 `json:"limit"`
	FilterDigest string              `json:"filter_digest"`
	IssuedAt     int64               `json:"issued_at"`
	ExpiresAt    int64               `json:"expires_at"`
}

// NewIdentityCursorCodec creates a cursor codec with injected key, lifetime, and clock.
func NewIdentityCursorCodec(
	key []byte,
	ttl time.Duration,
	now func() time.Time,
) (*IdentityCursorCodec, error) {
	if len(key) < 32 {
		return nil, errors.New("identity cursor HMAC key must contain at least 32 bytes")
	}
	if ttl <= 0 {
		return nil, errors.New("identity cursor lifetime must be positive")
	}
	if now == nil {
		return nil, errors.New("identity cursor clock is required")
	}
	return &IdentityCursorCodec{
		key: append([]byte(nil), key...),
		ttl: ttl,
		now: now,
	}, nil
}

// Encode creates one opaque cursor bound to the exact query contract.
func (c *IdentityCursorCodec) Encode(
	position IdentityCursorPosition,
	binding IdentityCursorBinding,
) (string, error) {
	if c == nil || len(c.key) < 32 || c.now == nil || c.ttl <= 0 {
		return "", errors.New("identity cursor codec is unavailable")
	}
	if position.CreatedAt.IsZero() || strings.TrimSpace(position.ID) == "" {
		return "", errors.New("identity cursor position is invalid")
	}
	if err := validateIdentityCursorBinding(binding); err != nil {
		return "", err
	}

	issuedAt := c.now().UTC()
	payload := identityCursorPayload{
		Version:      identityCursorVersion,
		CreatedAt:    position.CreatedAt.UTC().Format(time.RFC3339Nano),
		ID:           position.ID,
		Endpoint:     binding.Endpoint,
		Category:     binding.Category,
		Visibility:   binding.Visibility,
		Limit:        binding.Limit,
		FilterDigest: binding.FilterDigest,
		IssuedAt:     issuedAt.Unix(),
		ExpiresAt:    issuedAt.Add(c.ttl).Unix(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", errors.New("encode identity cursor payload")
	}
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write(data)
	signed := append(append([]byte(nil), data...), mac.Sum(nil)...)
	return base64.RawURLEncoding.EncodeToString(signed), nil
}

// Decode authenticates a cursor and enforces its complete request binding.
func (c *IdentityCursorCodec) Decode(
	token string,
	binding IdentityCursorBinding,
) (IdentityCursorPosition, error) {
	if c == nil || len(c.key) < 32 || c.now == nil || c.ttl <= 0 {
		return IdentityCursorPosition{}, invalidIdentityCursor(errors.New("identity cursor codec is unavailable"))
	}
	if err := validateIdentityCursorBinding(binding); err != nil {
		return IdentityCursorPosition{}, invalidIdentityCursor(err)
	}
	if token == "" || len(token) > maxIdentityCursorTokenBytes {
		return IdentityCursorPosition{}, invalidIdentityCursor(errors.New("identity cursor token length is invalid"))
	}
	signed, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(signed) <= identityCursorSignatureBytes {
		return IdentityCursorPosition{}, invalidIdentityCursor(errors.New("identity cursor encoding is invalid"))
	}
	data := signed[:len(signed)-identityCursorSignatureBytes]
	signature := signed[len(signed)-identityCursorSignatureBytes:]
	mac := hmac.New(sha256.New, c.key)
	_, _ = mac.Write(data)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return IdentityCursorPosition{}, invalidIdentityCursor(errors.New("identity cursor signature is invalid"))
	}

	var payload identityCursorPayload
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return IdentityCursorPosition{}, invalidIdentityCursor(errors.New("identity cursor payload is invalid"))
	}
	if err := requireCursorEOF(decoder); err != nil {
		return IdentityCursorPosition{}, invalidIdentityCursor(err)
	}
	if payload.Version != identityCursorVersion {
		return IdentityCursorPosition{}, invalidIdentityCursor(errors.New("identity cursor version is unsupported"))
	}
	if payload.Endpoint != binding.Endpoint ||
		payload.Category != binding.Category ||
		payload.Visibility != binding.Visibility ||
		payload.Limit != binding.Limit ||
		payload.FilterDigest != binding.FilterDigest {
		return IdentityCursorPosition{}, invalidIdentityCursor(errors.New("identity cursor binding does not match"))
	}
	now := c.now().UTC().Unix()
	if payload.IssuedAt > now || payload.ExpiresAt <= payload.IssuedAt || now >= payload.ExpiresAt {
		return IdentityCursorPosition{}, invalidIdentityCursor(errors.New("identity cursor lifetime is invalid"))
	}
	createdAt, err := time.Parse(time.RFC3339Nano, payload.CreatedAt)
	if err != nil || strings.TrimSpace(payload.ID) == "" {
		return IdentityCursorPosition{}, invalidIdentityCursor(errors.New("identity cursor position is invalid"))
	}
	return IdentityCursorPosition{CreatedAt: createdAt.UTC(), ID: payload.ID}, nil
}

func validateIdentityCursorBinding(binding IdentityCursorBinding) error {
	if binding.Endpoint == "" ||
		!binding.Category.valid() ||
		!binding.Visibility.valid() ||
		binding.Limit <= 0 ||
		binding.FilterDigest == "" {
		return errors.New("identity cursor binding is invalid")
	}
	return nil
}

func invalidIdentityCursor(cause error) error {
	return platform.NewPublicError(platform.CodeRequestInvalidCursor, cause)
}

func requireCursorEOF(decoder *json.Decoder) error {
	var trailing any
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("identity cursor payload has trailing data")
}
