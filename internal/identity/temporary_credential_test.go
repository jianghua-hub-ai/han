package identity

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	"han/internal/security"
)

func TestTemporaryCredentialConstructorUsesArgon2idAndSevenDayLifetime(t *testing.T) {
	hasher := newTemporaryCredentialTestHasher(t)
	issuedAt := time.Date(2026, time.October, 22, 8, 0, 0, 0, time.UTC)
	credential, err := NewTemporaryCredential(hasher, TemporaryCredentialInput{
		ID:               "11111111-1111-1111-1111-111111111111",
		UserID:           "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Purpose:          TemporaryCredentialPasswordReset,
		Generation:       2,
		Secret:           "temporary-secret-value",
		DeliveredToEmail: "recipient@example.test",
		IssuedAt:         issuedAt,
	})
	if err != nil {
		t.Fatalf("NewTemporaryCredential() error = %v", err)
	}
	if credential.State != TemporaryCredentialPendingDelivery {
		t.Fatalf("credential state = %q, want %q", credential.State, TemporaryCredentialPendingDelivery)
	}
	if !credential.ExpiresAt.Equal(issuedAt.Add(7 * 24 * time.Hour)) {
		t.Fatalf("ExpiresAt = %s, want seven days after IssuedAt", credential.ExpiresAt)
	}
	if err := ValidateTemporaryCredential(credential); err != nil {
		t.Fatalf("ValidateTemporaryCredential() error = %v", err)
	}
	matched, _, err := hasher.Verify("temporary-secret-value", credential.SecretDigest)
	if err != nil || !matched {
		t.Fatalf("generated Argon2id digest matched = %v, error = %v", matched, err)
	}
}

func TestTemporaryCredentialValidationRejectsMalformedDigestAndWrongLifetime(t *testing.T) {
	valid := activeTestTemporaryCredential(t)
	tests := []struct {
		name   string
		mutate func(*TemporaryCredential)
	}{
		{
			name: "unsupported purpose",
			mutate: func(credential *TemporaryCredential) {
				credential.Purpose = TemporaryCredentialPurpose("other")
			},
		},
		{
			name: "zero generation",
			mutate: func(credential *TemporaryCredential) {
				credential.Generation = 0
			},
		},
		{
			name: "unsupported state",
			mutate: func(credential *TemporaryCredential) {
				credential.State = TemporaryCredentialState("other")
			},
		},
		{
			name: "one hour lifetime",
			mutate: func(credential *TemporaryCredential) {
				credential.ExpiresAt = credential.IssuedAt.Add(time.Hour)
			},
		},
		{
			name: "empty digest",
			mutate: func(credential *TemporaryCredential) {
				credential.SecretDigest = ""
			},
		},
		{
			name: "plaintext digest",
			mutate: func(credential *TemporaryCredential) {
				credential.SecretDigest = "temporary-secret-value"
			},
		},
		{
			name: "malformed argon2id digest",
			mutate: func(credential *TemporaryCredential) {
				credential.SecretDigest = "$argon2id$v=19$m=8,t=1,p=1$not-base64$also-not-base64!"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			credential := valid
			test.mutate(&credential)
			if err := ValidateTemporaryCredential(credential); !errors.Is(err, ErrTemporaryCredentialInvalid) {
				t.Fatalf("ValidateTemporaryCredential() error = %v, want %v", err, ErrTemporaryCredentialInvalid)
			}
		})
	}
}

func TestTemporaryCredentialPasswordResetOperationBindingIsExact(t *testing.T) {
	operationID := "33333333-3333-3333-3333-333333333333"
	issuedAt := time.Date(2026, time.October, 24, 8, 0, 0, 0, time.UTC)
	credential, err := NewTemporaryCredential(
		newTemporaryCredentialTestHasher(t),
		TemporaryCredentialInput{
			ID:                "22222222-2222-2222-2222-222222222222",
			UserID:            "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			Purpose:           TemporaryCredentialPasswordReset,
			Generation:        3,
			AuthGeneration:    7,
			SourceOperationID: &operationID,
			Secret:            "replacement-temporary-secret",
			DeliveredToEmail:  "recipient@example.test",
			IssuedAt:          issuedAt,
		},
	)
	if err != nil {
		t.Fatalf("NewTemporaryCredential() error = %v", err)
	}
	if credential.AuthGeneration != 7 ||
		credential.SourceOperationID == nil ||
		*credential.SourceOperationID != operationID {
		t.Fatalf("durable credential binding = %#v", credential)
	}

	tests := []struct {
		name   string
		mutate func(*TemporaryCredential)
	}{
		{
			name: "generation without operation",
			mutate: func(candidate *TemporaryCredential) {
				candidate.SourceOperationID = nil
			},
		},
		{
			name: "operation without generation",
			mutate: func(candidate *TemporaryCredential) {
				candidate.AuthGeneration = 0
			},
		},
		{
			name: "blank operation",
			mutate: func(candidate *TemporaryCredential) {
				blank := " "
				candidate.SourceOperationID = &blank
			},
		},
		{
			name: "new administrator operation",
			mutate: func(candidate *TemporaryCredential) {
				candidate.Purpose = TemporaryCredentialNewAdministrator
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := credential
			test.mutate(&candidate)
			if err := ValidateTemporaryCredential(candidate); !errors.Is(
				err,
				ErrTemporaryCredentialInvalid,
			) {
				t.Fatalf("ValidateTemporaryCredential() error = %v", err)
			}
		})
	}
}

func TestTemporaryCredentialConstructorFailureLeavesOldGenerationUnchanged(t *testing.T) {
	now := time.Date(2026, time.October, 22, 8, 2, 0, 0, time.UTC)
	current := activeTestTemporaryCredential(t)
	before := current

	_, err := NewTemporaryCredential(failingTemporaryCredentialHasher{}, TemporaryCredentialInput{
		ID:               "22222222-2222-2222-2222-222222222222",
		UserID:           current.UserID,
		Purpose:          current.Purpose,
		Generation:       current.Generation + 1,
		Secret:           "new-temporary-secret",
		DeliveredToEmail: current.DeliveredToEmail,
		IssuedAt:         now,
	})
	if err == nil {
		t.Fatal("NewTemporaryCredential() error = nil")
	}
	if !reflect.DeepEqual(current, before) {
		t.Fatal("failed constructor changed old generation")
	}
	if !TemporaryCredentialAuthenticatable(current, current.Generation, now) {
		t.Fatal("failed constructor invalidated old generation")
	}
}

func TestTemporaryCredentialAuthenticatableRequiresActiveCurrentUnexpiredGeneration(t *testing.T) {
	base := activeTestTemporaryCredential(t)
	now := base.IssuedAt.Add(time.Minute)
	tests := []struct {
		name             string
		mutate           func(*TemporaryCredential)
		latestGeneration int64
		want             bool
	}{
		{name: "active current", latestGeneration: base.Generation, want: true},
		{
			name: "old generation",
			mutate: func(credential *TemporaryCredential) {
				credential.Generation--
			},
			latestGeneration: base.Generation,
		},
		{
			name: "pending",
			mutate: func(credential *TemporaryCredential) {
				credential.State = TemporaryCredentialPendingDelivery
			},
			latestGeneration: base.Generation,
		},
		{
			name: "expired by time",
			mutate: func(credential *TemporaryCredential) {
				credential.ExpiresAt = now
			},
			latestGeneration: base.Generation,
		},
		{
			name: "superseded",
			mutate: func(credential *TemporaryCredential) {
				credential.State = TemporaryCredentialSuperseded
				credential.SupersededAt = ptrTime(now)
			},
			latestGeneration: base.Generation,
		},
		{
			name: "consumed",
			mutate: func(credential *TemporaryCredential) {
				credential.State = TemporaryCredentialConsumed
				credential.ConsumedAt = ptrTime(now)
			},
			latestGeneration: base.Generation,
		},
		{
			name: "recovery invalidated",
			mutate: func(credential *TemporaryCredential) {
				credential.State = TemporaryCredentialRecoveryInvalidated
				credential.InvalidatedAt = ptrTime(now)
			},
			latestGeneration: base.Generation,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			credential := base
			if test.mutate != nil {
				test.mutate(&credential)
			}
			if got := TemporaryCredentialAuthenticatable(credential, test.latestGeneration, now); got != test.want {
				t.Fatalf("TemporaryCredentialAuthenticatable() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestTemporaryCredentialCurrentIncludesOnlyPendingActiveAndExpired(t *testing.T) {
	base := activeTestTemporaryCredential(t)
	for _, state := range []TemporaryCredentialState{
		TemporaryCredentialPendingDelivery,
		TemporaryCredentialActive,
		TemporaryCredentialExpired,
	} {
		credential := base
		credential.State = state
		if !TemporaryCredentialCurrent(credential) {
			t.Fatalf("TemporaryCredentialCurrent(%q) = false", state)
		}
	}
	for _, state := range []TemporaryCredentialState{
		TemporaryCredentialSuperseded,
		TemporaryCredentialConsumed,
		TemporaryCredentialRecoveryInvalidated,
	} {
		credential := base
		credential.State = state
		switch state {
		case TemporaryCredentialSuperseded:
			credential.SupersededAt = ptrTime(base.ExpiresAt)
		case TemporaryCredentialConsumed:
			credential.ConsumedAt = ptrTime(base.ExpiresAt)
		case TemporaryCredentialRecoveryInvalidated:
			credential.InvalidatedAt = ptrTime(base.ExpiresAt)
		}
		if TemporaryCredentialCurrent(credential) {
			t.Fatalf("TemporaryCredentialCurrent(%q) = true", state)
		}
	}
}

func TestTemporaryCredentialReplacementActivatesNewGenerationAndSupersedesOld(t *testing.T) {
	current := activeTestTemporaryCredential(t)
	now := current.IssuedAt.Add(2 * time.Minute)
	pending := pendingTestTemporaryCredential(t, current.Generation+1, now)
	currentBefore, pendingBefore := current, pending

	superseded, active, err := ActivateTemporaryCredentialReplacement(current, pending, now)
	if err != nil {
		t.Fatalf("ActivateTemporaryCredentialReplacement() error = %v", err)
	}
	if !reflect.DeepEqual(current, currentBefore) || !reflect.DeepEqual(pending, pendingBefore) {
		t.Fatal("ActivateTemporaryCredentialReplacement() mutated caller-owned values")
	}
	if superseded.State != TemporaryCredentialSuperseded ||
		superseded.SupersededAt == nil || !superseded.SupersededAt.Equal(now) {
		t.Fatalf("superseded credential = %#v", superseded)
	}
	if active.State != TemporaryCredentialActive || active.Generation != current.Generation+1 {
		t.Fatalf("active credential = %#v", active)
	}
	if TemporaryCredentialAuthenticatable(superseded, active.Generation, now) {
		t.Fatal("superseded generation remained authenticatable")
	}
	if !TemporaryCredentialAuthenticatable(active, active.Generation, now) {
		t.Fatal("new active generation is not authenticatable")
	}
}

func TestTemporaryCredentialReplacementFailureLeavesOldCredentialUsable(t *testing.T) {
	current := activeTestTemporaryCredential(t)
	now := current.IssuedAt.Add(2 * time.Minute)
	pending := pendingTestTemporaryCredential(t, current.Generation, now)
	currentBefore, pendingBefore := current, pending

	_, _, err := ActivateTemporaryCredentialReplacement(current, pending, now)
	if !errors.Is(err, ErrTemporaryCredentialGeneration) {
		t.Fatalf("ActivateTemporaryCredentialReplacement() error = %v, want %v", err, ErrTemporaryCredentialGeneration)
	}
	if !reflect.DeepEqual(current, currentBefore) || !reflect.DeepEqual(pending, pendingBefore) {
		t.Fatal("failed replacement mutated caller-owned values")
	}
	if !TemporaryCredentialAuthenticatable(current, current.Generation, now) {
		t.Fatal("failed replacement invalidated old credential")
	}
}

func TestTemporaryCredentialTransitionRejectsIllegalReactivation(t *testing.T) {
	credential := activeTestTemporaryCredential(t)
	now := credential.IssuedAt.Add(2 * time.Minute)
	credential.State = TemporaryCredentialConsumed
	credential.ConsumedAt = ptrTime(now)

	_, err := TransitionTemporaryCredential(credential, TemporaryCredentialActive, now)
	if !errors.Is(err, ErrTemporaryCredentialTransition) {
		t.Fatalf("TransitionTemporaryCredential() error = %v, want %v", err, ErrTemporaryCredentialTransition)
	}
}

func TestTemporaryCredentialTransitionRejectsPrematureExpiry(t *testing.T) {
	credential := activeTestTemporaryCredential(t)
	_, err := TransitionTemporaryCredential(
		credential,
		TemporaryCredentialExpired,
		credential.ExpiresAt.Add(-time.Second),
	)
	if !errors.Is(err, ErrTemporaryCredentialTransition) {
		t.Fatalf("TransitionTemporaryCredential(expired early) error = %v", err)
	}
}

func activeTestTemporaryCredential(t *testing.T) TemporaryCredential {
	t.Helper()
	issuedAt := time.Date(2026, time.October, 22, 8, 0, 0, 0, time.UTC)
	credential, err := NewTemporaryCredential(
		newTemporaryCredentialTestHasher(t),
		TemporaryCredentialInput{
			ID:               "11111111-1111-1111-1111-111111111111",
			UserID:           "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			Purpose:          TemporaryCredentialPasswordReset,
			Generation:       2,
			Secret:           "temporary-secret-value",
			DeliveredToEmail: "recipient@example.test",
			IssuedAt:         issuedAt,
		},
	)
	if err != nil {
		t.Fatalf("NewTemporaryCredential() error = %v", err)
	}
	active, err := TransitionTemporaryCredential(
		credential,
		TemporaryCredentialActive,
		issuedAt.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("activate temporary credential: %v", err)
	}
	return active
}

func pendingTestTemporaryCredential(
	t *testing.T,
	generation int64,
	issuedAt time.Time,
) TemporaryCredential {
	t.Helper()
	credential, err := NewTemporaryCredential(
		newTemporaryCredentialTestHasher(t),
		TemporaryCredentialInput{
			ID:               "22222222-2222-2222-2222-222222222222",
			UserID:           "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			Purpose:          TemporaryCredentialPasswordReset,
			Generation:       generation,
			Secret:           "replacement-temporary-secret",
			DeliveredToEmail: "recipient@example.test",
			IssuedAt:         issuedAt,
		},
	)
	if err != nil {
		t.Fatalf("NewTemporaryCredential() error = %v", err)
	}
	return credential
}

func newTemporaryCredentialTestHasher(t *testing.T) *security.Argon2idHasher {
	t.Helper()
	hasher, err := security.NewArgon2idHasher(
		security.Argon2idConfig{
			MemoryKiB:   8,
			Iterations:  1,
			Parallelism: 1,
			SaltLength:  8,
			KeyLength:   16,
		},
		bytes.NewReader(bytes.Repeat([]byte{0x42}, 64)),
	)
	if err != nil {
		t.Fatalf("NewArgon2idHasher() error = %v", err)
	}
	return hasher
}

type failingTemporaryCredentialHasher struct{}

func (failingTemporaryCredentialHasher) Hash(string) (string, error) {
	return "", errors.New("hash failed")
}

func (failingTemporaryCredentialHasher) Verify(string, string) (bool, bool, error) {
	return false, false, errors.New("verify failed")
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
