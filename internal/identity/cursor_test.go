package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"han/internal/platform"
)

func TestIdentityCursorRoundTripAndBinding(t *testing.T) {
	now := time.Date(2026, 10, 23, 9, 30, 0, 0, time.UTC)
	codec, err := NewIdentityCursorCodec(
		[]byte("0123456789abcdef0123456789abcdef"),
		15*time.Minute,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatalf("NewIdentityCursorCodec() error = %v", err)
	}
	position := IdentityCursorPosition{
		CreatedAt: now.Add(-time.Hour),
		ID:        "00000000-0000-0000-0000-000000000042",
	}
	binding := IdentityCursorBinding{
		Endpoint:     "/api/v1/admin/users",
		Category:     UserQueryCategoryUsers,
		Visibility:   UserQueryVisibilityAdministrator,
		Limit:        25,
		FilterDigest: "sha256:filters",
	}

	token, err := codec.Encode(position, binding)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	got, err := codec.Decode(token, binding)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !got.CreatedAt.Equal(position.CreatedAt) || got.ID != position.ID {
		t.Fatalf("Decode() = %#v, want %#v", got, position)
	}
}

func TestIdentityCursorRejectsInvalidTokenAndBinding(t *testing.T) {
	now := time.Date(2026, 10, 23, 9, 30, 0, 0, time.UTC)
	current := now
	key := []byte("0123456789abcdef0123456789abcdef")
	codec, err := NewIdentityCursorCodec(key, time.Minute, func() time.Time { return current })
	if err != nil {
		t.Fatalf("NewIdentityCursorCodec() error = %v", err)
	}
	position := IdentityCursorPosition{CreatedAt: now, ID: "user-02"}
	binding := IdentityCursorBinding{
		Endpoint:     "/api/v1/admin/users",
		Category:     UserQueryCategoryUsers,
		Visibility:   UserQueryVisibilityAdministrator,
		Limit:        20,
		FilterDigest: "sha256:none",
	}
	valid, err := codec.Encode(position, binding)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	unknownVersion := signedIdentityCursorForTest(t, key, map[string]any{
		"version":       99,
		"created_at":    now.Format(time.RFC3339Nano),
		"id":            position.ID,
		"endpoint":      string(binding.Endpoint),
		"category":      string(binding.Category),
		"visibility":    string(binding.Visibility),
		"limit":         binding.Limit,
		"filter_digest": binding.FilterDigest,
		"issued_at":     now.Unix(),
		"expires_at":    now.Add(time.Minute).Unix(),
	})
	tamperedSigned, err := base64.RawURLEncoding.DecodeString(valid)
	if err != nil {
		t.Fatalf("DecodeString(valid) error = %v", err)
	}
	tamperedSigned[len(tamperedSigned)-1] ^= 0x01
	tampered := base64.RawURLEncoding.EncodeToString(tamperedSigned)

	tests := []struct {
		name    string
		token   string
		binding IdentityCursorBinding
		advance time.Duration
	}{
		{name: "malformed", token: "not-base64", binding: binding},
		{name: "tampered", token: tampered, binding: binding},
		{name: "unknown version", token: unknownVersion, binding: binding},
		{name: "expired", token: valid, binding: binding, advance: time.Minute},
		{name: "endpoint mismatch", token: valid, binding: withCursorEndpoint(binding, "/api/v1/admin/administrators")},
		{name: "category mismatch", token: valid, binding: withCursorCategory(binding, UserQueryCategoryAdministrators)},
		{name: "visibility mismatch", token: valid, binding: withCursorVisibility(binding, UserQueryVisibilityInitialAdministrator)},
		{name: "limit mismatch", token: valid, binding: withCursorLimit(binding, binding.Limit+1)},
		{name: "filter mismatch", token: valid, binding: withCursorFilter(binding, "sha256:other")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current = now.Add(test.advance)
			_, err := codec.Decode(test.token, test.binding)
			if code := platform.PublicCode(err); code != platform.CodeRequestInvalidCursor {
				t.Fatalf("Decode() code = %q, want %q (err=%v)", code, platform.CodeRequestInvalidCursor, err)
			}
		})
	}
}

func TestIdentityCursorConfigurationValidation(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 10, 23, 9, 30, 0, 0, time.UTC) }
	tests := []struct {
		name  string
		key   []byte
		ttl   time.Duration
		clock func() time.Time
	}{
		{name: "short key", key: []byte("short"), ttl: time.Minute, clock: now},
		{name: "nonpositive ttl", key: []byte("0123456789abcdef0123456789abcdef"), ttl: 0, clock: now},
		{name: "missing clock", key: []byte("0123456789abcdef0123456789abcdef"), ttl: time.Minute},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewIdentityCursorCodec(test.key, test.ttl, test.clock); err == nil {
				t.Fatal("NewIdentityCursorCodec() error = nil, want configuration error")
			}
		})
	}
}

func signedIdentityCursorForTest(t *testing.T, key []byte, payload any) string {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	signed := append(append([]byte(nil), data...), mac.Sum(nil)...)
	return base64.RawURLEncoding.EncodeToString(signed)
}

func withCursorEndpoint(binding IdentityCursorBinding, endpoint string) IdentityCursorBinding {
	binding.Endpoint = endpoint
	return binding
}

func withCursorCategory(binding IdentityCursorBinding, category UserQueryCategory) IdentityCursorBinding {
	binding.Category = category
	return binding
}

func withCursorVisibility(binding IdentityCursorBinding, visibility UserQueryVisibility) IdentityCursorBinding {
	binding.Visibility = visibility
	return binding
}

func withCursorLimit(binding IdentityCursorBinding, limit int) IdentityCursorBinding {
	binding.Limit = limit
	return binding
}

func withCursorFilter(binding IdentityCursorBinding, filter string) IdentityCursorBinding {
	binding.FilterDigest = filter
	return binding
}
