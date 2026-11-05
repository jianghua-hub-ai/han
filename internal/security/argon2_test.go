package security

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestArgon2idProcessGate_RejectsImmediatelyAndReleasesPermit(t *testing.T) {
	base := mustNewTestHasher(t, &countingReader{})
	entered := make(chan struct{})
	release := make(chan struct{})
	base.derive = func([]byte, []byte, Argon2idConfig) ([]byte, error) {
		close(entered)
		<-release
		return make([]byte, testOnlyArgon2idConfig().KeyLength), nil
	}
	gate, err := NewArgon2idProcessGate(1)
	if err != nil {
		t.Fatalf("NewArgon2idProcessGate() error = %v", err)
	}
	hasher, err := NewGatedArgon2idHasher(base, gate)
	if err != nil {
		t.Fatalf("NewGatedArgon2idHasher() error = %v", err)
	}
	firstDone := make(chan error, 1)
	go func() {
		_, hashErr := hasher.Hash("synthetic-password")
		firstDone <- hashErr
	}()
	<-entered

	_, err = hasher.Hash("second-password")
	var capacity *Argon2idCapacityError
	if !errors.As(err, &capacity) || capacity.RetryAfter() != time.Second {
		t.Fatalf("second Hash() error = %v, want 1s capacity error", err)
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first Hash() error = %v", err)
	}
}

func TestArgon2idProcessGate_ReleasesPermitAfterErrorAndPanic(t *testing.T) {
	for _, test := range []struct {
		name   string
		derive deriveKeyFunc
		panic  bool
	}{
		{
			name: "error",
			derive: func([]byte, []byte, Argon2idConfig) ([]byte, error) {
				return nil, errors.New("synthetic derive failure")
			},
		},
		{
			name: "panic",
			derive: func([]byte, []byte, Argon2idConfig) ([]byte, error) {
				panic("synthetic panic")
			},
			panic: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := mustNewTestHasher(t, &countingReader{})
			base.derive = test.derive
			gate, _ := NewArgon2idProcessGate(1)
			hasher, _ := NewGatedArgon2idHasher(base, gate)
			if test.panic {
				func() {
					defer func() {
						if recover() == nil {
							t.Fatal("Hash() did not propagate panic")
						}
					}()
					_, _ = hasher.Hash("synthetic-password")
				}()
			} else {
				_, _ = hasher.Hash("synthetic-password")
			}
			base.derive = func([]byte, []byte, Argon2idConfig) ([]byte, error) {
				return make([]byte, testOnlyArgon2idConfig().KeyLength), nil
			}
			if _, err := hasher.Hash("next-password"); err != nil {
				t.Fatalf("next Hash() error = %v, permit was not released", err)
			}
		})
	}
}

func TestArgon2idProcessGate_OnePermitSetCoversHashAndVerify(t *testing.T) {
	base := mustNewTestHasher(t, &countingReader{})
	entered := make(chan struct{})
	release := make(chan struct{})
	first := true
	base.derive = func([]byte, []byte, Argon2idConfig) ([]byte, error) {
		if first {
			first = false
			close(entered)
			<-release
		}
		return make([]byte, testOnlyArgon2idConfig().KeyLength), nil
	}
	gate, err := NewArgon2idProcessGate(1)
	if err != nil {
		t.Fatalf("NewArgon2idProcessGate() error = %v", err)
	}
	hasher, err := NewGatedArgon2idHasher(base, gate)
	if err != nil {
		t.Fatalf("NewGatedArgon2idHasher() error = %v", err)
	}
	encoded := encodeArgon2idEncoding(argon2idEncoding{
		config: testOnlyArgon2idConfig(),
		salt:   make([]byte, testOnlyArgon2idConfig().SaltLength),
		digest: make([]byte, testOnlyArgon2idConfig().KeyLength),
	})
	operations := []struct {
		name string
		run  func() error
	}{
		{
			name: "Hash",
			run: func() error {
				_, err := hasher.Hash("synthetic-password")
				return err
			},
		},
		{
			name: "Verify",
			run: func() error {
				_, _, err := hasher.Verify("synthetic-password", encoded)
				return err
			},
		},
	}

	holderDone := make(chan error, 1)
	go func() { holderDone <- operations[0].run() }()
	<-entered
	for _, operation := range operations[1:] {
		err := operation.run()
		var capacity *Argon2idCapacityError
		if !errors.As(err, &capacity) || capacity.RetryAfter() != time.Second {
			t.Fatalf("%s error = %v, want shared 1s capacity error", operation.name, err)
		}
	}
	close(release)
	if err := <-holderDone; err != nil {
		t.Fatalf("permit holder error = %v", err)
	}
	for _, operation := range operations {
		if err := operation.run(); err != nil {
			t.Fatalf("%s after release error = %v", operation.name, err)
		}
	}
}

func TestNewArgon2idHasher_InvalidConfiguration_FailsClosed(t *testing.T) {
	tests := []struct {
		name   string
		config Argon2idConfig
		random io.Reader
	}{
		{name: "zero config", random: &countingReader{}},
		{name: "missing memory", config: testOnlyArgon2idConfig(), random: &countingReader{}},
		{name: "missing iterations", config: testOnlyArgon2idConfig(), random: &countingReader{}},
		{name: "missing parallelism", config: testOnlyArgon2idConfig(), random: &countingReader{}},
		{name: "missing salt length", config: testOnlyArgon2idConfig(), random: &countingReader{}},
		{name: "missing key length", config: testOnlyArgon2idConfig(), random: &countingReader{}},
		{name: "memory below algorithm minimum", config: testOnlyArgon2idConfig(), random: &countingReader{}},
		{name: "missing random source", config: testOnlyArgon2idConfig()},
	}
	tests[1].config.MemoryKiB = 0
	tests[2].config.Iterations = 0
	tests[3].config.Parallelism = 0
	tests[4].config.SaltLength = 0
	tests[5].config.KeyLength = 0
	tests[6].config.MemoryKiB = 7

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewArgon2idHasher(test.config, test.random)
			if !errors.Is(err, ErrArgon2idInvalidConfig) {
				t.Fatalf("NewArgon2idHasher() error = %v, want %v", err, ErrArgon2idInvalidConfig)
			}
		})
	}
}

func TestArgon2idHasher_Hash_RandomSourceFailureDoesNotExposeSecret(t *testing.T) {
	password := "Sensitive123"
	randomErr := errors.New("entropy unavailable")
	hasher := mustNewTestHasher(t, errorReader{err: randomErr})

	encoded, err := hasher.Hash(password)
	if encoded != "" {
		t.Fatal("Hash() returned a non-empty encoding after random source failure")
	}
	if !errors.Is(err, randomErr) {
		t.Fatalf("Hash() error = %v, want wrapped random error", err)
	}
	assertErrorDoesNotContain(t, err, password)
}

func TestArgon2idHasher_Hash_DerivationFailureDoesNotExposeSecret(t *testing.T) {
	password := "Sensitive123"
	resourceErr := errors.New("resource unavailable")
	hasher := mustNewTestHasher(t, &countingReader{})
	hasher.derive = func([]byte, []byte, Argon2idConfig) ([]byte, error) {
		return nil, resourceErr
	}

	encoded, err := hasher.Hash(password)
	if encoded != "" {
		t.Fatal("Hash() returned a non-empty encoding after derivation failure")
	}
	if !errors.Is(err, ErrArgon2idResource) || !errors.Is(err, resourceErr) {
		t.Fatalf("Hash() error = %v, want resource failure", err)
	}
	assertErrorDoesNotContain(t, err, password)
}

func TestArgon2idHasher_Hash_SamePasswordUsesIndependentSalts(t *testing.T) {
	hasher := mustNewTestHasher(t, &countingReader{})

	first, err := hasher.Hash("Password123")
	if err != nil {
		t.Fatalf("first Hash() error = %v", err)
	}
	second, err := hasher.Hash("Password123")
	if err != nil {
		t.Fatalf("second Hash() error = %v", err)
	}
	if first == second {
		t.Fatal("Hash() returned identical encodings for independent salts")
	}

	firstParsed, err := parseArgon2idEncoding(first)
	if err != nil {
		t.Fatalf("parse first encoding: %v", err)
	}
	secondParsed, err := parseArgon2idEncoding(second)
	if err != nil {
		t.Fatalf("parse second encoding: %v", err)
	}
	if bytes.Equal(firstParsed.salt, secondParsed.salt) {
		t.Fatal("Hash() reused the salt")
	}
}

func TestArgon2idHasher_Verify_CorrectAndIncorrectPasswords(t *testing.T) {
	hasher := mustNewTestHasher(t, &countingReader{})
	encoded, err := hasher.Hash("Password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	tests := []struct {
		name     string
		password string
		matched  bool
	}{
		{name: "correct password", password: "Password123", matched: true},
		{name: "incorrect password", password: "Password124"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matched, needsRehash, err := hasher.Verify(test.password, encoded)
			if err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if matched != test.matched {
				t.Fatalf("Verify() matched = %v, want %v", matched, test.matched)
			}
			if needsRehash {
				t.Fatal("Verify() needsRehash = true for current parameters")
			}
		})
	}
}

func TestArgon2idHasher_Verify_MalformedOrUnsupportedEncodingFailsClosed(t *testing.T) {
	validHasher := mustNewTestHasher(t, &countingReader{})
	valid, err := validHasher.Hash("Password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	parsed, err := parseArgon2idEncoding(valid)
	if err != nil {
		t.Fatalf("parse valid encoding: %v", err)
	}

	tests := []struct {
		name    string
		encoded string
		wantErr error
	}{
		{name: "empty", encoded: "", wantErr: ErrArgon2idMalformedEncoding},
		{name: "truncated", encoded: "$argon2id$v=19$m=8,t=1,p=1", wantErr: ErrArgon2idMalformedEncoding},
		{name: "invalid parameter", encoded: "$argon2id$v=19$m=x,t=1,p=1$YWJj$ZGVm", wantErr: ErrArgon2idMalformedEncoding},
		{name: "invalid salt base64", encoded: "$argon2id$v=19$m=8,t=1,p=1$***$ZGVm", wantErr: ErrArgon2idMalformedEncoding},
		{name: "unsupported algorithm", encoded: strings.Replace(valid, "$argon2id$", "$argon2i$", 1), wantErr: ErrArgon2idUnsupportedAlgorithm},
		{name: "unsupported version", encoded: strings.Replace(valid, "$v=19$", "$v=16$", 1), wantErr: ErrArgon2idUnsupportedVersion},
		{name: "empty salt", encoded: encodeArgon2idEncoding(argon2idEncoding{config: parsed.config, digest: parsed.digest}), wantErr: ErrArgon2idMalformedEncoding},
		{name: "empty digest", encoded: encodeArgon2idEncoding(argon2idEncoding{config: parsed.config, salt: parsed.salt}), wantErr: ErrArgon2idMalformedEncoding},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matched, needsRehash, err := validHasher.Verify("Password123", test.encoded)
			if matched || needsRehash {
				t.Fatalf("Verify() = (%v, %v), want (false, false)", matched, needsRehash)
			}
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Verify() error = %v, want %v", err, test.wantErr)
			}
			assertErrorDoesNotContain(t, err, test.encoded)
		})
	}
}

func TestArgon2idHasher_Verify_OldParametersSignalsRehashAfterSuccessfulMatch(t *testing.T) {
	oldConfig := testOnlyArgon2idConfig()
	oldHasher, err := NewArgon2idHasher(oldConfig, &countingReader{})
	if err != nil {
		t.Fatalf("NewArgon2idHasher(old) error = %v", err)
	}
	encoded, err := oldHasher.Hash("Password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	currentConfig := oldConfig
	currentConfig.MemoryKiB *= 2
	currentConfig.Iterations++
	currentHasher, err := NewArgon2idHasher(currentConfig, &countingReader{})
	if err != nil {
		t.Fatalf("NewArgon2idHasher(current) error = %v", err)
	}

	matched, needsRehash, err := currentHasher.Verify("Password123", encoded)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !matched || !needsRehash {
		t.Fatalf("Verify() = (%v, %v), want (true, true)", matched, needsRehash)
	}

	matched, needsRehash, err = currentHasher.Verify("WrongPassword123", encoded)
	if err != nil {
		t.Fatalf("Verify(wrong) error = %v", err)
	}
	if matched || needsRehash {
		t.Fatalf("Verify(wrong) = (%v, %v), want (false, false)", matched, needsRehash)
	}
}

func TestArgon2idHasher_Verify_RefusesParameterDowngrade(t *testing.T) {
	strongConfig := testOnlyArgon2idConfig()
	strongConfig.MemoryKiB *= 2
	strongConfig.Iterations++
	strongHasher, err := NewArgon2idHasher(strongConfig, &countingReader{})
	if err != nil {
		t.Fatalf("NewArgon2idHasher(strong) error = %v", err)
	}
	encoded, err := strongHasher.Hash("Password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	weakHasher := mustNewTestHasher(t, &countingReader{})
	matched, needsRehash, err := weakHasher.Verify("Password123", encoded)
	if matched || needsRehash {
		t.Fatalf("Verify() = (%v, %v), want (false, false)", matched, needsRehash)
	}
	if !errors.Is(err, ErrArgon2idParameterDowngrade) {
		t.Fatalf("Verify() error = %v, want %v", err, ErrArgon2idParameterDowngrade)
	}
}

func TestArgon2idHasher_Verify_DerivationFailureFailsClosed(t *testing.T) {
	hasher := mustNewTestHasher(t, &countingReader{})
	encoded, err := hasher.Hash("Password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	resourceErr := errors.New("resource unavailable")
	hasher.derive = func([]byte, []byte, Argon2idConfig) ([]byte, error) {
		return nil, resourceErr
	}

	matched, needsRehash, err := hasher.Verify("Password123", encoded)
	if matched || needsRehash {
		t.Fatalf("Verify() = (%v, %v), want (false, false)", matched, needsRehash)
	}
	if !errors.Is(err, ErrArgon2idResource) || !errors.Is(err, resourceErr) {
		t.Fatalf("Verify() error = %v, want resource failure", err)
	}
}

func TestArgon2idEncoding_RoundTripPreservesSelfDescribingFields(t *testing.T) {
	hasher := mustNewTestHasher(t, &countingReader{})
	passwords := []string{
		"abcd1234",
		"LongerPassword456",
		"密码abcd12",
		"symbols-123-XYZ!",
	}

	for _, password := range passwords {
		encoded, err := hasher.Hash(password)
		if err != nil {
			t.Fatalf("Hash() error = %v", err)
		}
		parsed, err := parseArgon2idEncoding(encoded)
		if err != nil {
			t.Fatalf("parseArgon2idEncoding() error = %v", err)
		}
		if parsed.config != testOnlyArgon2idConfig() {
			t.Fatalf("parsed config = %#v, want %#v", parsed.config, testOnlyArgon2idConfig())
		}
		if len(parsed.salt) != int(testOnlyArgon2idConfig().SaltLength) {
			t.Fatalf("salt length = %d", len(parsed.salt))
		}
		if len(parsed.digest) != int(testOnlyArgon2idConfig().KeyLength) {
			t.Fatalf("digest length = %d", len(parsed.digest))
		}
		if roundTrip := encodeArgon2idEncoding(parsed); roundTrip != encoded {
			t.Fatalf("encoding round trip changed the value")
		}
	}
}

func testOnlyArgon2idConfig() Argon2idConfig {
	return Argon2idConfig{
		MemoryKiB:   8,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  8,
		KeyLength:   16,
	}
}

func mustNewTestHasher(t *testing.T, random io.Reader) *Argon2idHasher {
	t.Helper()
	hasher, err := NewArgon2idHasher(testOnlyArgon2idConfig(), random)
	if err != nil {
		t.Fatalf("NewArgon2idHasher() error = %v", err)
	}
	return hasher
}

func assertErrorDoesNotContain(t *testing.T, err error, secret string) {
	t.Helper()
	if err == nil {
		t.Fatal("error = nil")
	}
	if secret != "" && strings.Contains(err.Error(), secret) {
		t.Fatal("error contains sensitive input")
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

type countingReader struct {
	next byte
}

func (r *countingReader) Read(target []byte) (int, error) {
	for index := range target {
		r.next++
		target[index] = r.next
	}
	return len(target), nil
}
