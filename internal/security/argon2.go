package security

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	argon2idAlgorithm      = "argon2id"
	argon2idMinimumBlocks  = 8
	argon2idParameterCount = 3
)

var (
	// ErrArgon2idInvalidConfig indicates missing or invalid hashing configuration.
	ErrArgon2idInvalidConfig = errors.New("argon2id configuration is invalid")
	// ErrArgon2idMalformedEncoding indicates an invalid self-describing digest.
	ErrArgon2idMalformedEncoding = errors.New("argon2id encoding is malformed")
	// ErrArgon2idUnsupportedAlgorithm indicates a non-Argon2id encoding.
	ErrArgon2idUnsupportedAlgorithm = errors.New("argon2id algorithm is unsupported")
	// ErrArgon2idUnsupportedVersion indicates an unsupported Argon2 version.
	ErrArgon2idUnsupportedVersion = errors.New("argon2id version is unsupported")
	// ErrArgon2idParameterDowngrade indicates that verification would lower strength.
	ErrArgon2idParameterDowngrade = errors.New("argon2id parameter downgrade is forbidden")
	// ErrArgon2idResource indicates a hashing resource failure.
	ErrArgon2idResource = errors.New("argon2id resource failure")
)

const argon2idCapacityRetryAfter = time.Second

// Argon2idCapacityError reports immediate process-capacity exhaustion.
type Argon2idCapacityError struct {
	retryAfter time.Duration
}

func (e *Argon2idCapacityError) Error() string {
	return "argon2id process capacity exhausted"
}

// RetryAfter is the sole capacity retry hint owner.
func (e *Argon2idCapacityError) RetryAfter() time.Duration {
	if e == nil {
		return 0
	}
	return e.retryAfter
}

// Argon2idProcessGate is one non-blocking process-wide permit set.
type Argon2idProcessGate struct {
	permits chan struct{}
}

// NewArgon2idProcessGate creates a bounded process gate.
func NewArgon2idProcessGate(maxConcurrency uint8) (*Argon2idProcessGate, error) {
	if maxConcurrency < 1 || maxConcurrency > 4 {
		return nil, fmt.Errorf("%w: max concurrency must be between 1 and 4", ErrArgon2idInvalidConfig)
	}
	return &Argon2idProcessGate{permits: make(chan struct{}, maxConcurrency)}, nil
}

func (g *Argon2idProcessGate) acquire() error {
	if g == nil || g.permits == nil {
		return fmt.Errorf("%w: process gate is not initialized", ErrArgon2idInvalidConfig)
	}
	select {
	case g.permits <- struct{}{}:
		return nil
	default:
		return &Argon2idCapacityError{retryAfter: argon2idCapacityRetryAfter}
	}
}

func (g *Argon2idProcessGate) release() {
	<-g.permits
}

// GatedArgon2idHasher applies one shared gate to every Hash and Verify.
type GatedArgon2idHasher struct {
	hasher *Argon2idHasher
	gate   *Argon2idProcessGate
}

// NewGatedArgon2idHasher composes an initialized hasher and process gate.
func NewGatedArgon2idHasher(
	hasher *Argon2idHasher,
	gate *Argon2idProcessGate,
) (*GatedArgon2idHasher, error) {
	if hasher == nil || gate == nil {
		return nil, fmt.Errorf("%w: gated hasher dependencies are incomplete", ErrArgon2idInvalidConfig)
	}
	if err := hasher.validateReady(); err != nil {
		return nil, err
	}
	if gate.permits == nil {
		return nil, fmt.Errorf("%w: process gate is not initialized", ErrArgon2idInvalidConfig)
	}
	return &GatedArgon2idHasher{hasher: hasher, gate: gate}, nil
}

// Hash hashes while holding one process permit.
func (h *GatedArgon2idHasher) Hash(password string) (string, error) {
	if h == nil || h.hasher == nil || h.gate == nil {
		return "", fmt.Errorf("%w: gated hasher is not initialized", ErrArgon2idInvalidConfig)
	}
	if err := h.gate.acquire(); err != nil {
		return "", err
	}
	defer h.gate.release()
	return h.hasher.Hash(password)
}

// Verify verifies while holding one process permit.
func (h *GatedArgon2idHasher) Verify(password, encoded string) (bool, bool, error) {
	if h == nil || h.hasher == nil || h.gate == nil {
		return false, false, fmt.Errorf("%w: gated hasher is not initialized", ErrArgon2idInvalidConfig)
	}
	if err := h.gate.acquire(); err != nil {
		return false, false, err
	}
	defer h.gate.release()
	return h.hasher.Verify(password, encoded)
}

// Argon2idConfig contains all parameters required to hash a credential.
type Argon2idConfig struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

type deriveKeyFunc func(password, salt []byte, config Argon2idConfig) ([]byte, error)

// Argon2idHasher creates and verifies self-describing Argon2id encodings.
type Argon2idHasher struct {
	config Argon2idConfig
	random io.Reader
	derive deriveKeyFunc
}

type argon2idEncoding struct {
	config Argon2idConfig
	salt   []byte
	digest []byte
}

// NewArgon2idHasher creates a hasher from explicit parameters and randomness.
func NewArgon2idHasher(config Argon2idConfig, random io.Reader) (*Argon2idHasher, error) {
	if err := validateArgon2idConfig(config); err != nil {
		return nil, err
	}
	if random == nil {
		return nil, fmt.Errorf("%w: random source is required", ErrArgon2idInvalidConfig)
	}
	return &Argon2idHasher{
		config: config,
		random: random,
		derive: deriveArgon2idKey,
	}, nil
}

// Hash creates a self-describing Argon2id credential encoding.
func (h *Argon2idHasher) Hash(password string) (string, error) {
	if err := h.validateReady(); err != nil {
		return "", err
	}

	salt := make([]byte, h.config.SaltLength)
	if _, err := io.ReadFull(h.random, salt); err != nil {
		return "", fmt.Errorf("read argon2id salt: %w", err)
	}

	digest, err := h.derive([]byte(password), salt, h.config)
	if err != nil {
		return "", resourceError(err)
	}
	defer clear(digest)
	if len(digest) != int(h.config.KeyLength) {
		return "", resourceError(errors.New("unexpected digest length"))
	}
	return encodeArgon2idEncoding(argon2idEncoding{
		config: h.config,
		salt:   salt,
		digest: digest,
	}), nil
}

// Verify compares a credential with a self-describing Argon2id encoding.
func (h *Argon2idHasher) Verify(password, encoded string) (bool, bool, error) {
	if err := h.validateReady(); err != nil {
		return false, false, err
	}
	stored, err := parseArgon2idEncoding(encoded)
	if err != nil {
		return false, false, err
	}
	needsRehash, err := rehashDecision(h.config, stored.config)
	if err != nil {
		return false, false, err
	}

	digest, err := h.derive([]byte(password), stored.salt, stored.config)
	if err != nil {
		return false, false, resourceError(err)
	}
	defer clear(digest)
	if len(digest) != len(stored.digest) {
		return false, false, resourceError(errors.New("unexpected digest length"))
	}
	if subtle.ConstantTimeCompare(digest, stored.digest) != 1 {
		return false, false, nil
	}
	return true, needsRehash, nil
}

func parseArgon2idEncoding(encoded string) (argon2idEncoding, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" {
		return argon2idEncoding{}, ErrArgon2idMalformedEncoding
	}
	if parts[1] != argon2idAlgorithm {
		return argon2idEncoding{}, ErrArgon2idUnsupportedAlgorithm
	}
	if parts[2] != "v="+strconv.Itoa(argon2.Version) {
		return argon2idEncoding{}, ErrArgon2idUnsupportedVersion
	}

	config, err := parseArgon2idParameters(parts[3])
	if err != nil {
		return argon2idEncoding{}, err
	}
	salt, err := decodeArgon2idField(parts[4])
	if err != nil {
		return argon2idEncoding{}, err
	}
	digest, err := decodeArgon2idField(parts[5])
	if err != nil {
		return argon2idEncoding{}, err
	}
	config.SaltLength = uint32(len(salt))
	config.KeyLength = uint32(len(digest))
	if err := validateArgon2idConfig(config); err != nil {
		return argon2idEncoding{}, ErrArgon2idMalformedEncoding
	}
	return argon2idEncoding{config: config, salt: salt, digest: digest}, nil
}

func encodeArgon2idEncoding(encoding argon2idEncoding) string {
	return fmt.Sprintf(
		"$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2idAlgorithm,
		argon2.Version,
		encoding.config.MemoryKiB,
		encoding.config.Iterations,
		encoding.config.Parallelism,
		base64.RawStdEncoding.EncodeToString(encoding.salt),
		base64.RawStdEncoding.EncodeToString(encoding.digest),
	)
}

func (h *Argon2idHasher) validateReady() error {
	if h == nil || h.random == nil || h.derive == nil {
		return fmt.Errorf("%w: hasher is not initialized", ErrArgon2idInvalidConfig)
	}
	return validateArgon2idConfig(h.config)
}

func validateArgon2idConfig(config Argon2idConfig) error {
	if config.MemoryKiB == 0 || config.Iterations == 0 || config.Parallelism == 0 ||
		config.SaltLength == 0 || config.KeyLength == 0 {
		return fmt.Errorf("%w: all parameters are required", ErrArgon2idInvalidConfig)
	}
	if config.MemoryKiB < argon2idMinimumBlocks*uint32(config.Parallelism) {
		return fmt.Errorf("%w: memory is below the algorithm minimum", ErrArgon2idInvalidConfig)
	}
	maximumInt := uint64(^uint(0) >> 1)
	if uint64(config.MemoryKiB) > maximumInt/1024 ||
		uint64(config.SaltLength) > maximumInt ||
		uint64(config.KeyLength) > maximumInt {
		return fmt.Errorf("%w: parameter exceeds platform capacity", ErrArgon2idInvalidConfig)
	}
	return nil
}

func parseArgon2idParameters(encoded string) (Argon2idConfig, error) {
	parameters := strings.Split(encoded, ",")
	if len(parameters) != argon2idParameterCount {
		return Argon2idConfig{}, ErrArgon2idMalformedEncoding
	}
	memory, err := parseArgon2idParameter(parameters[0], "m=", 32)
	if err != nil {
		return Argon2idConfig{}, err
	}
	iterations, err := parseArgon2idParameter(parameters[1], "t=", 32)
	if err != nil {
		return Argon2idConfig{}, err
	}
	parallelism, err := parseArgon2idParameter(parameters[2], "p=", 8)
	if err != nil {
		return Argon2idConfig{}, err
	}
	return Argon2idConfig{
		MemoryKiB:   uint32(memory),
		Iterations:  uint32(iterations),
		Parallelism: uint8(parallelism),
	}, nil
}

func parseArgon2idParameter(encoded, prefix string, bitSize int) (uint64, error) {
	value, found := strings.CutPrefix(encoded, prefix)
	if !found || value == "" {
		return 0, ErrArgon2idMalformedEncoding
	}
	parsed, err := strconv.ParseUint(value, 10, bitSize)
	if err != nil {
		return 0, ErrArgon2idMalformedEncoding
	}
	return parsed, nil
}

func decodeArgon2idField(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, ErrArgon2idMalformedEncoding
	}
	decoded, err := base64.RawStdEncoding.Strict().DecodeString(encoded)
	if err != nil || len(decoded) == 0 {
		return nil, ErrArgon2idMalformedEncoding
	}
	return decoded, nil
}

func rehashDecision(current, stored Argon2idConfig) (bool, error) {
	if stored.MemoryKiB > current.MemoryKiB ||
		stored.Iterations > current.Iterations ||
		stored.Parallelism > current.Parallelism ||
		stored.SaltLength > current.SaltLength ||
		stored.KeyLength > current.KeyLength {
		return false, ErrArgon2idParameterDowngrade
	}
	return stored != current, nil
}

func deriveArgon2idKey(password, salt []byte, config Argon2idConfig) (digest []byte, err error) {
	defer func() {
		if recover() != nil {
			digest = nil
			err = ErrArgon2idResource
		}
	}()
	return argon2.IDKey(
		password,
		salt,
		config.Iterations,
		config.MemoryKiB,
		config.Parallelism,
		config.KeyLength,
	), nil
}

func resourceError(cause error) error {
	return fmt.Errorf("%w: %w", ErrArgon2idResource, cause)
}
