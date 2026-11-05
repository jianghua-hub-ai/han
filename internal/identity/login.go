package identity

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const (
	loginAuditAction           = "identity.login"
	loginAuditOutcomeSucceeded = "succeeded"
	loginAuditOutcomeFailed    = "failed"
	loginAuditTargetUser       = "user"
	loginAuditTargetAttempt    = "login_attempt"
	loginAuditSubjectDigest    = "login_subject_digest"
	loginAuditSubjectKey       = "login_subject_digest_key_version"
	loginSessionRotateReason   = "login_rotation"
	invalidLoginSubjectBucket  = "\x00invalid-login-subject"
	loginInternalFailureEvent  = "identity.login.internal_failure"
	loginAttemptsMetricName    = "han_identity_login_attempts_total"
	loginFailuresMetricName    = "han_identity_login_internal_failures_total"
	loginDurationMetricName    = "han_identity_login_duration_seconds"
)

// ConfiguredLoginRecoveryMode is the explicit recovery-bootstrap mode.
type ConfiguredLoginRecoveryMode struct {
	maintenance bool
}

// NewConfiguredLoginRecoveryMode validates the startup mode.
func NewConfiguredLoginRecoveryMode(mode string) (ConfiguredLoginRecoveryMode, error) {
	switch strings.TrimSpace(mode) {
	case "normal":
		return ConfiguredLoginRecoveryMode{}, nil
	case "maintenance":
		return ConfiguredLoginRecoveryMode{maintenance: true}, nil
	default:
		return ConfiguredLoginRecoveryMode{}, errors.New("login recovery mode is invalid")
	}
}

// RecoveryBootstrapAllowed reports whether recovery credentials may be verified.
func (m ConfiguredLoginRecoveryMode) RecoveryBootstrapAllowed(context.Context) (bool, error) {
	return m.maintenance, nil
}

// RandomLoginSessionIssuer creates session and CSRF credentials from explicit policy.
type RandomLoginSessionIssuer struct {
	random       io.Reader
	ids          LoginIDGenerator
	clock        LoginClock
	tokenEntropy uint32
	csrfEntropy  uint32
	absoluteTTL  time.Duration
	idleTTL      time.Duration
}

// NewRandomLoginSessionIssuer creates the production session issuer.
func NewRandomLoginSessionIssuer(
	random io.Reader,
	ids LoginIDGenerator,
	clock LoginClock,
	tokenEntropy uint32,
	csrfEntropy uint32,
	absoluteTTL time.Duration,
	idleTTL time.Duration,
) (*RandomLoginSessionIssuer, error) {
	if random == nil || ids == nil || clock == nil ||
		tokenEntropy < 32 || tokenEntropy > 64 ||
		csrfEntropy < 32 || csrfEntropy > 64 ||
		absoluteTTL <= 0 || idleTTL <= 0 || idleTTL > absoluteTTL {
		return nil, errors.New("login session issuer configuration is invalid")
	}
	return &RandomLoginSessionIssuer{
		random: random, ids: ids, clock: clock,
		tokenEntropy: tokenEntropy, csrfEntropy: csrfEntropy,
		absoluteTTL: absoluteTTL, idleTTL: idleTTL,
	}, nil
}

// Issue creates one all-or-nothing session input.
func (i *RandomLoginSessionIssuer) Issue(
	user User,
	kind session.CredentialKind,
	credentialID *string,
) (session.CreateInput, error) {
	if i == nil || user.ID == "" {
		return session.CreateInput{}, errors.New("login session input is invalid")
	}
	id, err := i.ids.NewID()
	if err != nil {
		return session.CreateInput{}, errors.New("issue login session identifier")
	}
	tokenBytes := make([]byte, i.tokenEntropy)
	defer clear(tokenBytes)
	if _, err := io.ReadFull(i.random, tokenBytes); err != nil {
		return session.CreateInput{}, errors.New("issue login session token")
	}
	csrfBytes := make([]byte, i.csrfEntropy)
	defer clear(csrfBytes)
	if _, err := io.ReadFull(i.random, csrfBytes); err != nil {
		return session.CreateInput{}, errors.New("issue login CSRF token")
	}
	issuedAt := i.clock.Now().UTC()
	input := session.CreateInput{
		ID: id, UserID: user.ID,
		Token:                       base64.RawURLEncoding.EncodeToString(tokenBytes),
		CSRFToken:                   base64.RawURLEncoding.EncodeToString(csrfBytes),
		AuthenticatedCredentialKind: kind,
		AuthenticatedCredentialID:   credentialID,
		AuthGeneration:              user.AuthGeneration,
		IssuedAt:                    issuedAt,
		AbsoluteExpiresAt:           issuedAt.Add(i.absoluteTTL),
		IdleExpiresAt:               issuedAt.Add(i.idleTTL),
	}
	switch kind {
	case session.CredentialTemporary:
		if user.PasswordResetCurrentOperationID != nil && credentialID != nil {
			input.LineageOperationID = user.PasswordResetCurrentOperationID
			input.LineageCredentialID = credentialID
		}
	case session.CredentialPermanent:
		input.LineageOperationID = user.PermanentPasswordSourceOperationID
		input.LineageCredentialID = user.PermanentPasswordSourceCredentialID
	}
	return input, nil
}

var errLoginRejected = errors.New("login rejected")

// LoginFailureAuditMode is the closed independent audit operation set.
type LoginFailureAuditMode string

const (
	LoginFailureAuditInsertFailure  LoginFailureAuditMode = "insert_failure"
	LoginFailureAuditVerifyPrepared LoginFailureAuditMode = "verify_prepared"
)

// LoginAuditPersistence is the proven persistence state of this login audit fact.
type LoginAuditPersistence string

const (
	LoginAuditPersistedNo      LoginAuditPersistence = "no"
	LoginAuditPersistedYes     LoginAuditPersistence = "yes"
	LoginAuditPersistedUnknown LoginAuditPersistence = "unknown"
)

// LoginFailureAuditRequest contains one complete immutable audit fact.
type LoginFailureAuditRequest struct {
	Mode  LoginFailureAuditMode
	Event audit.Event
}

// LoginFailureAuditor is the bounded independent login audit entry point.
type LoginFailureAuditor interface {
	RecordOnce(context.Context, LoginFailureAuditRequest) (LoginAuditPersistence, error)
}

// LoginOutcome is the fixed Handler-scope result set.
type LoginOutcome string

const (
	LoginOutcomeSuccess              LoginOutcome = "success"
	LoginOutcomeAuthenticationFailed LoginOutcome = "authentication_failed"
	LoginOutcomeRateLimited          LoginOutcome = "rate_limited"
	LoginOutcomeInvalidRequest       LoginOutcome = "invalid_request"
	LoginOutcomeInternalFailure      LoginOutcome = "internal_failure"
)

// LoginFailureStage is the fixed internal failure location set.
type LoginFailureStage string

const (
	LoginFailureStageNone                     LoginFailureStage = ""
	LoginFailureStageSourceEvidence           LoginFailureStage = "source_evidence"
	LoginFailureStageTransactionBegin         LoginFailureStage = "transaction_begin"
	LoginFailureStageAbuseCheck               LoginFailureStage = "abuse_check"
	LoginFailureStageUserQuery                LoginFailureStage = "user_query"
	LoginFailureStageTemporaryCredentialQuery LoginFailureStage = "temporary_credential_query"
	LoginFailureStageCredentialVerify         LoginFailureStage = "credential_verify"
	LoginFailureStageCredentialRehash         LoginFailureStage = "credential_rehash"
	LoginFailureStageRecoveryMode             LoginFailureStage = "recovery_mode"
	LoginFailureStageAbuseRecord              LoginFailureStage = "abuse_record"
	LoginFailureStageSessionIssue             LoginFailureStage = "session_issue"
	LoginFailureStageSessionPersist           LoginFailureStage = "session_persist"
	LoginFailureStageAuditAppend              LoginFailureStage = "audit_append"
	LoginFailureStageFailureAudit             LoginFailureStage = "failure_audit"
	LoginFailureStageTransactionCommit        LoginFailureStage = "transaction_commit"
	LoginFailureStageCookieOutput             LoginFailureStage = "cookie_output"
	LoginFailureStageResponseWrite            LoginFailureStage = "response_write"
	LoginFailureStageUnknown                  LoginFailureStage = "unknown"
)

type LoginTransactionOutcome string

const (
	LoginTransactionNotStarted LoginTransactionOutcome = "not_started"
	LoginTransactionRolledBack LoginTransactionOutcome = "rolled_back"
	LoginTransactionCommitted  LoginTransactionOutcome = "committed"
	LoginTransactionUnknown    LoginTransactionOutcome = "unknown"
)

type LoginStateChanged string

const (
	LoginStateNone             LoginStateChanged = "none"
	LoginStateRolledBack       LoginStateChanged = "rolled_back"
	LoginStateSessionCommitted LoginStateChanged = "session_committed"
	LoginStateUnknown          LoginStateChanged = "unknown"
)

type LoginRollbackOutcome string

const (
	LoginRollbackNotAttempted LoginRollbackOutcome = "not_attempted"
	LoginRollbackSucceeded    LoginRollbackOutcome = "succeeded"
	LoginRollbackFailed       LoginRollbackOutcome = "failed"
	LoginRollbackUnknown      LoginRollbackOutcome = "unknown"
)

type LoginResponseWrite string

const (
	LoginResponseWriteOK     LoginResponseWrite = "ok"
	LoginResponseWriteFailed LoginResponseWrite = "failed"
)

// LoginObservation is the final low-cardinality login result snapshot.
type LoginObservation struct {
	Endpoint           string
	Duration           time.Duration
	Outcome            LoginOutcome
	ReasonBucket       string
	ErrorCode          string
	FailureStage       LoginFailureStage
	TransactionOutcome LoginTransactionOutcome
	StateChanged       LoginStateChanged
	AuditPersisted     LoginAuditPersistence
	RollbackOutcome    LoginRollbackOutcome
	ResponseWrite      LoginResponseWrite
	TraceID            string
}

// LoginOutcomeObserver observes one final result after its response write attempt.
type LoginOutcomeObserver interface {
	ObserveFinal(context.Context, LoginObservation) error
}

// LoginEventLogger emits the single bounded internal-failure event.
type LoginEventLogger interface {
	LogLoginOutcome(context.Context, LoginObservation) error
}

// SlogLoginEventLogger emits the fixed internal-failure event through slog.
type SlogLoginEventLogger struct {
	logger  *slog.Logger
	service string
}

// NewSlogLoginEventLogger creates the bounded structured login logger.
func NewSlogLoginEventLogger(
	logger *slog.Logger,
	service string,
) (*SlogLoginEventLogger, error) {
	service = strings.TrimSpace(service)
	if logger == nil || service == "" {
		return nil, errors.New("login event logger dependencies are incomplete")
	}
	return &SlogLoginEventLogger{logger: logger, service: service}, nil
}

// LogLoginOutcome emits only the stable low-cardinality internal-failure fields.
func (l *SlogLoginEventLogger) LogLoginOutcome(
	ctx context.Context,
	observation LoginObservation,
) error {
	if l == nil || l.logger == nil || strings.TrimSpace(l.service) == "" {
		return errors.New("login event logger is not initialized")
	}
	if err := loginContextError(ctx); err != nil {
		return err
	}
	if observation.Outcome != LoginOutcomeInternalFailure {
		return errors.New("login event logger requires an internal failure")
	}
	logger, err := platform.LoggerWithTrace(
		l.logger,
		l.service,
		"identity",
		observation.TraceID,
	)
	if err != nil {
		return err
	}
	record := slog.NewRecord(
		time.Now(),
		slog.LevelError,
		loginInternalFailureEvent,
		0,
	)
	record.AddAttrs(
		slog.String("event", loginInternalFailureEvent),
		slog.String("endpoint", observation.Endpoint),
		slog.String("outcome", string(observation.Outcome)),
		slog.String("reason_bucket", observation.ReasonBucket),
		slog.String("error_code", observation.ErrorCode),
		slog.String("failure_stage", string(observation.FailureStage)),
		slog.String("transaction_outcome", string(observation.TransactionOutcome)),
		slog.String("state_changed", string(observation.StateChanged)),
		slog.String("audit_persisted", string(observation.AuditPersisted)),
		slog.String("rollback_outcome", string(observation.RollbackOutcome)),
	)
	if !logger.Enabled(ctx, slog.LevelError) {
		return loginContextError(ctx)
	}
	if err := logger.Handler().Handle(ctx, record); err != nil {
		return err
	}
	return loginContextError(ctx)
}

// LoginMetrics records the fixed Handler-scope login metrics.
type LoginMetrics interface {
	RecordLogin(context.Context, LoginMetricObservation) error
}

// LoginMetricObservation exposes only the fixed metrics allowlist and duration.
type LoginMetricObservation struct {
	Endpoint     string
	Duration     time.Duration
	Outcome      LoginOutcome
	ReasonBucket string
	ErrorCode    string
	FailureStage LoginFailureStage
}

type loginOutcomeObserver struct {
	logger  LoginEventLogger
	metrics LoginMetrics
}

// NewLoginOutcomeObserver composes the fixed logger/metrics fan-out.
func NewLoginOutcomeObserver(
	logger LoginEventLogger,
	metrics LoginMetrics,
) (LoginOutcomeObserver, error) {
	if logger == nil || metrics == nil {
		return nil, errors.New("login outcome observer dependencies are incomplete")
	}
	return &loginOutcomeObserver{logger: logger, metrics: metrics}, nil
}

func (o *loginOutcomeObserver) ObserveFinal(
	ctx context.Context,
	observation LoginObservation,
) error {
	if err := loginContextError(ctx); err != nil {
		return err
	}
	if observation.Outcome == LoginOutcomeInternalFailure {
		if err := o.logger.LogLoginOutcome(ctx, observation); err != nil {
			return err
		}
		if err := loginContextError(ctx); err != nil {
			return err
		}
	}
	return o.metrics.RecordLogin(ctx, LoginMetricObservation{
		Endpoint:     observation.Endpoint,
		Duration:     observation.Duration,
		Outcome:      observation.Outcome,
		ReasonBucket: observation.ReasonBucket,
		ErrorCode:    observation.ErrorCode,
		FailureStage: observation.FailureStage,
	})
}

type loginAttemptPhase string

const (
	loginAttemptPreService     loginAttemptPhase = "pre_service"
	loginAttemptServiceEntered loginAttemptPhase = "service_entered"
)

// LoginAttemptContext carries one immutable ID/time and a shared audit guard.
type LoginAttemptContext struct {
	ID         string
	OccurredAt time.Time

	mu              sync.Mutex
	phase           loginAttemptPhase
	auditClaimed    bool
	auditor         LoginFailureAuditor
	preparedEvent   *audit.Event
	appendAttempted bool
	observation     LoginObservation
}

// LoginAttemptCoordinator creates request-scoped login attempt state.
type LoginAttemptCoordinator struct {
	ids     LoginIDGenerator
	clock   LoginClock
	auditor LoginFailureAuditor
}

// NewLoginAttemptCoordinator creates the source-before-ID/time/guard coordinator.
func NewLoginAttemptCoordinator(
	ids LoginIDGenerator,
	clock LoginClock,
	auditor LoginFailureAuditor,
) (*LoginAttemptCoordinator, error) {
	if ids == nil || clock == nil || auditor == nil {
		return nil, errors.New("login attempt coordinator dependencies are incomplete")
	}
	return &LoginAttemptCoordinator{ids: ids, clock: clock, auditor: auditor}, nil
}

// Begin creates one immutable attempt before any source provider call.
func (c *LoginAttemptCoordinator) Begin() (*LoginAttemptContext, error) {
	if c == nil || c.ids == nil || c.clock == nil || c.auditor == nil {
		return nil, errors.New("login attempt coordinator is not initialized")
	}
	id, err := c.ids.NewID()
	if err != nil || strings.TrimSpace(id) == "" {
		return nil, errors.New("generate login audit identifier")
	}
	occurredAt := c.clock.Now().UTC().Truncate(time.Microsecond)
	if occurredAt.IsZero() {
		return nil, errors.New("login attempt time is required")
	}
	return &LoginAttemptContext{
		ID:         id,
		OccurredAt: occurredAt,
		phase:      loginAttemptPreService,
		auditor:    c.auditor,
		observation: LoginObservation{
			Endpoint:           "POST /api/v1/sessions",
			ReasonBucket:       "none",
			ErrorCode:          "none",
			TransactionOutcome: LoginTransactionNotStarted,
			StateChanged:       LoginStateNone,
			AuditPersisted:     LoginAuditPersistedNo,
			RollbackOutcome:    LoginRollbackNotAttempted,
			ResponseWrite:      LoginResponseWriteOK,
		},
	}, nil
}

func (a *LoginAttemptContext) enterService() error {
	if a == nil {
		return errors.New("login attempt context is required")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.phase != loginAttemptPreService {
		return errors.New("login attempt phase transition is invalid")
	}
	a.phase = loginAttemptServiceEntered
	return nil
}

func (a *LoginAttemptContext) serviceEntered() bool {
	if a == nil {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.phase == loginAttemptServiceEntered
}

func (a *LoginAttemptContext) recordOnce(
	ctx context.Context,
	owner loginAttemptPhase,
	request LoginFailureAuditRequest,
) (LoginAuditPersistence, error) {
	if a == nil || a.auditor == nil {
		return LoginAuditPersistedUnknown, errors.New("login failure auditor is unavailable")
	}
	a.mu.Lock()
	if a.phase != owner || a.auditClaimed {
		a.mu.Unlock()
		return LoginAuditPersistedUnknown, errors.New("login failure audit guard rejected duplicate claim")
	}
	a.auditClaimed = true
	a.mu.Unlock()
	return a.auditor.RecordOnce(ctx, request)
}

func (a *LoginAttemptContext) setPrepared(event audit.Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	copied := cloneAuditEvent(event)
	a.preparedEvent = &copied
	a.appendAttempted = true
}

func (a *LoginAttemptContext) prepared() (audit.Event, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.appendAttempted || a.preparedEvent == nil {
		return audit.Event{}, false
	}
	return cloneAuditEvent(*a.preparedEvent), true
}

func (a *LoginAttemptContext) updateObservation(update func(*LoginObservation)) {
	if a == nil || update == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	update(&a.observation)
}

func (a *LoginAttemptContext) observationSnapshot() LoginObservation {
	if a == nil {
		return LoginObservation{
			Endpoint:           "POST /api/v1/sessions",
			ReasonBucket:       "unknown",
			ErrorCode:          string(platform.CodePlatformInternalError),
			FailureStage:       LoginFailureStageUnknown,
			TransactionOutcome: LoginTransactionUnknown,
			StateChanged:       LoginStateUnknown,
			AuditPersisted:     LoginAuditPersistedUnknown,
			RollbackOutcome:    LoginRollbackUnknown,
			ResponseWrite:      LoginResponseWriteOK,
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.observation
}

// LoginRequest is the public credential payload plus trusted HTTP-boundary evidence.
type LoginRequest struct {
	Username             string               `json:"username"`
	Password             string               `json:"password"`
	SourceFingerprint    string               `json:"-"`
	ExistingSessionToken string               `json:"-"`
	Attempt              *LoginAttemptContext `json:"-"`
}

// AuthenticatedUser is the safe identity result returned to transports.
type AuthenticatedUser struct {
	ID                        string
	Username                  string
	Role                      Role
	Status                    UserStatus
	IsInitialAdmin            bool
	MustChangePassword        bool
	MustAcknowledgePrivacy    bool
	AuthenticatedCredential   SessionCredentialKind
	AuthenticatedCredentialID *string
}

// LoginResult contains the safe identity view and raw values required by the transport.
type LoginResult struct {
	User         AuthenticatedUser
	SessionToken string
	CSRFToken    string
}

// LoginAttempt contains only normalized or trusted digested login dimensions.
type LoginAttempt struct {
	Subject           string
	SourceFingerprint string
	Now               time.Time
	subjectDigest     []byte
	subjectKeyVersion string
}

// LoginAbuseDecisionKind identifies the public response class of a persisted block.
type LoginAbuseDecisionKind string

const (
	// LoginAbuseAllowed permits credential verification.
	LoginAbuseAllowed LoginAbuseDecisionKind = ""
	// LoginAbuseSubjectBlocked preserves the unified authentication failure.
	LoginAbuseSubjectBlocked LoginAbuseDecisionKind = "subject_blocked"
	// LoginAbuseSourceBlocked permits the existing source-capacity 429 contract.
	LoginAbuseSourceBlocked LoginAbuseDecisionKind = "source_blocked"
)

// LoginAbuseDecision is the result of checking all persistent bucket dimensions.
type LoginAbuseDecision struct {
	Kind             LoginAbuseDecisionKind
	RetryAfter       time.Duration
	SubjectDigest    []byte
	DigestKeyVersion string
}

// LoginAbuseSuccessAction defines the caller-approved success behavior.
type LoginAbuseSuccessAction string

const (
	// LoginAbuseSuccessClear deletes successful attempt buckets.
	LoginAbuseSuccessClear LoginAbuseSuccessAction = "clear"
	// LoginAbuseSuccessDecay reduces successful attempt bucket counts.
	LoginAbuseSuccessDecay LoginAbuseSuccessAction = "decay"
)

// LoginAbuseConfig contains explicit policy values; no production defaults are supplied.
type LoginAbuseConfig struct {
	Window           time.Duration
	FailureThreshold int64
	BlockDuration    time.Duration
	SuccessAction    LoginAbuseSuccessAction
	SuccessDecay     int64
}

// LoginAbuseDigester creates stable keyed bucket digests without storing raw dimensions.
type LoginAbuseDigester interface {
	Digest(LoginBucketKind, string, string) ([]byte, string, error)
}

// HMACLoginAbuseDigester uses one explicit, versioned key for login bucket identities.
type HMACLoginAbuseDigester struct {
	keyVersion string
	key        []byte
}

// NewHMACLoginAbuseDigester creates a versioned HMAC-SHA-256 bucket digester.
func NewHMACLoginAbuseDigester(
	keyVersion string,
	key []byte,
) (*HMACLoginAbuseDigester, error) {
	if strings.TrimSpace(keyVersion) == "" || len(key) == 0 {
		return nil, errors.New("login abuse digest key and version are required")
	}
	return &HMACLoginAbuseDigester{
		keyVersion: keyVersion,
		key:        append([]byte(nil), key...),
	}, nil
}

// Digest returns a canonical keyed digest and its persisted key version.
func (d *HMACLoginAbuseDigester) Digest(
	kind LoginBucketKind,
	subject string,
	source string,
) ([]byte, string, error) {
	if d == nil || len(d.key) == 0 || strings.TrimSpace(d.keyVersion) == "" {
		return nil, "", errors.New("login abuse digester is not initialized")
	}
	switch kind {
	case LoginBucketSubject:
		source = ""
	case LoginBucketSource:
		subject = ""
	case LoginBucketSubjectSource:
	default:
		return nil, "", errors.New("login abuse bucket kind is invalid")
	}
	if (kind != LoginBucketSource && subject == "") ||
		(kind != LoginBucketSubject && source == "") {
		return nil, "", errors.New("login abuse bucket dimensions are incomplete")
	}

	mac := hmac.New(sha256.New, d.key)
	writeLoginDigestField(mac, string(kind))
	writeLoginDigestField(mac, subject)
	writeLoginDigestField(mac, source)
	return mac.Sum(nil), d.keyVersion, nil
}

// LoginAbuseBucketStore persists attempt buckets in a caller-owned transaction.
type LoginAbuseBucketStore interface {
	LoadForUpdate(context.Context, pgx.Tx, LoginBucketKind, []byte) (LoginAttemptBucket, error)
	Save(context.Context, pgx.Tx, LoginAttemptBucket) error
	Delete(context.Context, pgx.Tx, LoginBucketKind, []byte) error
}

// PersistentLoginAbuseProtector applies explicit policy over a persistent bucket store.
type PersistentLoginAbuseProtector struct {
	store    LoginAbuseBucketStore
	digester LoginAbuseDigester
	config   LoginAbuseConfig
}

// NewPersistentLoginAbuseProtector creates a persistent three-dimension protector.
func NewPersistentLoginAbuseProtector(
	store LoginAbuseBucketStore,
	digester LoginAbuseDigester,
	config LoginAbuseConfig,
) (*PersistentLoginAbuseProtector, error) {
	if store == nil || digester == nil ||
		config.Window <= 0 || config.FailureThreshold <= 0 ||
		config.BlockDuration <= 0 {
		return nil, errors.New("login abuse dependencies and policy are incomplete")
	}
	switch config.SuccessAction {
	case LoginAbuseSuccessClear:
		if config.SuccessDecay != 0 {
			return nil, errors.New("login abuse clear policy cannot specify decay")
		}
	case LoginAbuseSuccessDecay:
		if config.SuccessDecay <= 0 {
			return nil, errors.New("login abuse decay policy requires a positive decrement")
		}
	default:
		return nil, errors.New("login abuse success policy is invalid")
	}
	return &PersistentLoginAbuseProtector{
		store: store, digester: digester, config: config,
	}, nil
}

// Check evaluates persisted source and account dimensions without mutating them.
func (p *PersistentLoginAbuseProtector) Check(
	ctx context.Context,
	tx pgx.Tx,
	attempt LoginAttempt,
) (LoginAbuseDecision, error) {
	keys, err := p.bucketKeys(attempt)
	if err != nil {
		return LoginAbuseDecision{}, err
	}
	decision := LoginAbuseDecision{
		SubjectDigest:    append([]byte(nil), keys[0].digest...),
		DigestKeyVersion: keys[0].keyVersion,
	}

	var subjectRetry, sourceRetry time.Duration
	for _, key := range keys {
		bucket, loadErr := p.store.LoadForUpdate(ctx, tx, key.kind, key.digest)
		if errors.Is(loadErr, pgx.ErrNoRows) {
			continue
		}
		if loadErr != nil {
			return LoginAbuseDecision{}, fmt.Errorf("load login attempt bucket: %w", loadErr)
		}
		if !bucket.BlockedUntil.After(attempt.Now) {
			continue
		}
		retryAfter := bucket.BlockedUntil.Sub(attempt.Now)
		if key.kind == LoginBucketSource {
			if retryAfter > sourceRetry {
				sourceRetry = retryAfter
			}
			continue
		}
		if retryAfter > subjectRetry {
			subjectRetry = retryAfter
		}
	}
	if sourceRetry > 0 {
		decision.Kind = LoginAbuseSourceBlocked
		decision.RetryAfter = sourceRetry
		return decision, nil
	}
	if subjectRetry > 0 {
		decision.Kind = LoginAbuseSubjectBlocked
		decision.RetryAfter = subjectRetry
		return decision, nil
	}
	decision.Kind = LoginAbuseAllowed
	return decision, nil
}

// RecordFailure atomically advances all three persistent bucket dimensions.
func (p *PersistentLoginAbuseProtector) RecordFailure(
	ctx context.Context,
	tx pgx.Tx,
	attempt LoginAttempt,
) error {
	keys, err := p.bucketKeys(attempt)
	if err != nil {
		return err
	}
	for _, key := range keys {
		bucket, loadErr := p.store.LoadForUpdate(ctx, tx, key.kind, key.digest)
		switch {
		case errors.Is(loadErr, pgx.ErrNoRows):
			bucket = LoginAttemptBucket{
				Kind: key.kind, KeyDigest: key.digest,
				DigestKeyVersion: key.keyVersion,
				WindowStartedAt:  attempt.Now,
				BlockedUntil:     attempt.Now,
			}
		case loadErr != nil:
			return fmt.Errorf("load login attempt bucket: %w", loadErr)
		}
		if !attempt.Now.Before(bucket.WindowStartedAt.Add(p.config.Window)) {
			bucket.WindowStartedAt = attempt.Now
			bucket.FailureCount = 0
			bucket.BlockedUntil = attempt.Now
		}
		bucket.FailureCount++
		if bucket.FailureCount >= p.config.FailureThreshold {
			blockedUntil := attempt.Now.Add(p.config.BlockDuration)
			if blockedUntil.After(bucket.BlockedUntil) {
				bucket.BlockedUntil = blockedUntil
			}
		}
		bucket.UpdatedAt = attempt.Now
		if err := p.store.Save(ctx, tx, bucket); err != nil {
			return fmt.Errorf("save login attempt bucket: %w", err)
		}
	}
	return nil
}

// RecordSuccess applies the caller-approved clear or decay policy.
func (p *PersistentLoginAbuseProtector) RecordSuccess(
	ctx context.Context,
	tx pgx.Tx,
	attempt LoginAttempt,
) error {
	keys, err := p.bucketKeys(attempt)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if p.config.SuccessAction == LoginAbuseSuccessClear {
			if err := p.store.Delete(ctx, tx, key.kind, key.digest); err != nil {
				return fmt.Errorf("delete successful login bucket: %w", err)
			}
			continue
		}

		bucket, loadErr := p.store.LoadForUpdate(ctx, tx, key.kind, key.digest)
		if errors.Is(loadErr, pgx.ErrNoRows) {
			continue
		}
		if loadErr != nil {
			return fmt.Errorf("load successful login bucket: %w", loadErr)
		}
		bucket.FailureCount -= p.config.SuccessDecay
		if bucket.FailureCount < 0 {
			bucket.FailureCount = 0
		}
		bucket.BlockedUntil = attempt.Now
		bucket.UpdatedAt = attempt.Now
		if err := p.store.Save(ctx, tx, bucket); err != nil {
			return fmt.Errorf("save successful login bucket: %w", err)
		}
	}
	return nil
}

type loginBucketKey struct {
	kind       LoginBucketKind
	digest     []byte
	keyVersion string
}

func (p *PersistentLoginAbuseProtector) bucketKeys(
	attempt LoginAttempt,
) ([]loginBucketKey, error) {
	if p == nil || p.store == nil || p.digester == nil ||
		attempt.Subject == "" || attempt.SourceFingerprint == "" ||
		attempt.Now.IsZero() {
		return nil, errors.New("login attempt evidence is incomplete")
	}
	kinds := [...]LoginBucketKind{
		LoginBucketSubject,
		LoginBucketSource,
		LoginBucketSubjectSource,
	}
	keys := make([]loginBucketKey, 0, len(kinds))
	for _, kind := range kinds {
		digest, version, err := p.digester.Digest(
			kind,
			attempt.Subject,
			attempt.SourceFingerprint,
		)
		if err != nil {
			return nil, fmt.Errorf("digest login attempt bucket: %w", err)
		}
		keys = append(keys, loginBucketKey{
			kind: kind, digest: digest, keyVersion: version,
		})
	}
	return keys, nil
}

// PostgresLoginAbuseBucketStore persists buckets in login_attempt_buckets.
type PostgresLoginAbuseBucketStore struct{}

// LoadForUpdate locks one bucket for atomic counter updates.
func (PostgresLoginAbuseBucketStore) LoadForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	kind LoginBucketKind,
	digest []byte,
) (LoginAttemptBucket, error) {
	if err := lockLoginAttemptBucket(ctx, tx, kind, digest); err != nil {
		return LoginAttemptBucket{}, err
	}
	var bucket LoginAttemptBucket
	err := tx.QueryRow(ctx, selectLoginAttemptBucketForUpdateSQL, kind, digest).Scan(
		&bucket.Kind,
		&bucket.KeyDigest,
		&bucket.DigestKeyVersion,
		&bucket.WindowStartedAt,
		&bucket.FailureCount,
		&bucket.BlockedUntil,
		&bucket.UpdatedAt,
	)
	return bucket, err
}

// Save inserts or updates one locked bucket.
func (PostgresLoginAbuseBucketStore) Save(
	ctx context.Context,
	tx pgx.Tx,
	bucket LoginAttemptBucket,
) error {
	if err := lockLoginAttemptBucket(ctx, tx, bucket.Kind, bucket.KeyDigest); err != nil {
		return err
	}
	_, err := tx.Exec(
		ctx,
		upsertLoginAttemptBucketSQL,
		bucket.Kind,
		bucket.KeyDigest,
		bucket.DigestKeyVersion,
		bucket.WindowStartedAt,
		bucket.FailureCount,
		bucket.BlockedUntil,
		bucket.UpdatedAt,
	)
	return err
}

// Delete clears one successful login bucket.
func (PostgresLoginAbuseBucketStore) Delete(
	ctx context.Context,
	tx pgx.Tx,
	kind LoginBucketKind,
	digest []byte,
) error {
	if err := lockLoginAttemptBucket(ctx, tx, kind, digest); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, deleteLoginAttemptBucketSQL, kind, digest)
	return err
}

func lockLoginAttemptBucket(
	ctx context.Context,
	tx pgx.Tx,
	kind LoginBucketKind,
	digest []byte,
) error {
	if tx == nil {
		return errors.New("login abuse transaction is required")
	}
	if len(digest) == 0 {
		return errors.New("login abuse bucket digest is required")
	}
	_, err := tx.Exec(ctx, lockLoginAttemptBucketSQL, kind, digest)
	return err
}

// PostgresLoginRepository reads the existing identity schema without adding a second model.
type PostgresLoginRepository struct{}

// FindUserByUsername returns the account bound to the normalized username.
func (PostgresLoginRepository) FindUserByUsername(
	ctx context.Context,
	tx pgx.Tx,
	normalizedUsername string,
) (User, error) {
	if tx == nil {
		return User{}, errors.New("login transaction is required")
	}
	var user User
	err := tx.QueryRow(ctx, findLoginUserSQL, normalizedUsername).Scan(
		&user.ID,
		&user.Username,
		&user.PermanentPasswordDigest,
		&user.AuthGeneration,
		&user.PermanentPasswordGeneration,
		&user.PasswordResetExecutionGeneration,
		&user.PasswordResetCurrentOperationID,
		&user.PermanentPasswordSourceOperationID,
		&user.PermanentPasswordSourceCredentialID,
		&user.AuthIntegrityHold,
		&user.Role,
		&user.IsInitialAdmin,
		&user.Status,
		&user.CredentialMode,
		&user.RecoveryRequired,
		&user.OnboardingPrivacyAcknowledgedVersion,
		&user.OnboardingCompletedAt,
		&user.DeletedAt,
	)
	return user, err
}

// FindLatestTemporaryCredential returns the highest generation for fail-closed selection.
func (PostgresLoginRepository) FindLatestTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (TemporaryCredential, error) {
	if tx == nil {
		return TemporaryCredential{}, errors.New("login transaction is required")
	}
	var credential TemporaryCredential
	err := tx.QueryRow(ctx, findLatestTemporaryCredentialSQL, userID).Scan(
		&credential.ID,
		&credential.UserID,
		&credential.Purpose,
		&credential.Generation,
		&credential.AuthGeneration,
		&credential.SourceOperationID,
		&credential.State,
		&credential.SecretDigest,
		&credential.DeliveredToEmail,
		&credential.IssuedAt,
		&credential.ExpiresAt,
		&credential.SupersededAt,
		&credential.ConsumedAt,
		&credential.InvalidatedAt,
	)
	return credential, err
}

// UpdatePermanentPasswordDigest persists a successful explicit rehash.
func (PostgresLoginRepository) UpdatePermanentPasswordDigest(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	digest string,
) error {
	if tx == nil {
		return errors.New("login transaction is required")
	}
	tag, err := tx.Exec(ctx, updatePermanentLoginDigestSQL, digest, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("permanent credential rehash target changed")
	}
	return nil
}

// UpdateTemporaryCredentialDigest persists a successful explicit rehash.
func (PostgresLoginRepository) UpdateTemporaryCredentialDigest(
	ctx context.Context,
	tx pgx.Tx,
	credentialID string,
	digest string,
) error {
	if tx == nil {
		return errors.New("login transaction is required")
	}
	tag, err := tx.Exec(ctx, updateTemporaryLoginDigestSQL, digest, credentialID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("temporary credential rehash target changed")
	}
	return nil
}

// Login authenticates one credential path and atomically creates or rotates its session.
func (s *Service) Login(ctx context.Context, request LoginRequest) (LoginResult, error) {
	execution, err := newLoginExecution(s, ctx, request)
	if err != nil {
		return LoginResult{}, err
	}
	return execution.run()
}

type loginExecution struct {
	service            *Service
	ctx                context.Context
	request            LoginRequest
	dependencies       LoginDependencies
	attemptContext     *LoginAttemptContext
	source             string
	now                time.Time
	attempt            LoginAttempt
	subjectValid       bool
	tx                 pgx.Tx
	transactionClosed  bool
	user               User
	credential         verifiedLoginCredential
	sessionInput       session.CreateInput
	sessionCredentials session.Credentials
}

func newLoginExecution(
	service *Service,
	ctx context.Context,
	request LoginRequest,
) (*loginExecution, error) {
	if service == nil {
		return nil, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("login service is not initialized"),
		)
	}
	attemptContext := request.Attempt
	if attemptContext == nil || !attemptContext.serviceEntered() {
		return nil, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("login attempt context is incomplete"),
		)
	}
	subject, subjectValid := normalizeLoginSubject(request.Username)
	if !subjectValid {
		subject = invalidLoginSubjectBucket
	}
	source := strings.TrimSpace(request.SourceFingerprint)
	if source == "" {
		return nil, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("login source evidence is incomplete"),
		)
	}
	now := attemptContext.OccurredAt
	return &loginExecution{
		service: service, ctx: ctx, request: request,
		dependencies: service.loginDependencies, attemptContext: attemptContext,
		source: source, now: now, subjectValid: subjectValid,
		attempt: LoginAttempt{Subject: subject, SourceFingerprint: source, Now: now},
	}, nil
}

func (e *loginExecution) run() (LoginResult, error) {
	if err := e.begin(); err != nil {
		return LoginResult{}, err
	}
	defer e.rollback()
	if err := e.checkAbuse(); err != nil {
		return LoginResult{}, err
	}
	if err := e.authenticate(); err != nil {
		return LoginResult{}, err
	}
	if err := e.prepareSession(); err != nil {
		return LoginResult{}, err
	}
	if err := e.storeSession(); err != nil {
		return LoginResult{}, err
	}
	return e.appendAuditAndCommit()
}

func (e *loginExecution) begin() error {
	tx, err := e.dependencies.Transactions.Begin(e.ctx)
	if err == nil {
		e.tx = tx
		return nil
	}
	publicErr := loginPersistenceError("begin login transaction", err)
	return e.service.finishLoginWithoutTransaction(
		e.ctx,
		e.attemptContext,
		"",
		e.source,
		LoginFailureStageTransactionBegin,
		publicErr,
	)
}

func (e *loginExecution) rollback() {
	if !e.transactionClosed {
		_ = e.tx.Rollback(e.ctx)
	}
}

func (e *loginExecution) checkAbuse() error {
	decision, err := e.dependencies.Abuse.Check(e.ctx, e.tx, e.attempt)
	if err != nil {
		return e.fail(
			"", LoginFailureStageAbuseCheck,
			loginPersistenceError("check login attempt buckets", err),
		)
	}
	if len(decision.SubjectDigest) == 0 ||
		strings.TrimSpace(decision.DigestKeyVersion) == "" {
		return e.fail("", LoginFailureStageAbuseCheck, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("login subject audit evidence is incomplete"),
		))
	}
	e.attempt.subjectDigest = append([]byte(nil), decision.SubjectDigest...)
	e.attempt.subjectKeyVersion = decision.DigestKeyVersion
	if decision.Kind == LoginAbuseAllowed {
		return nil
	}
	e.transactionClosed = true
	return e.service.commitLoginFailure(
		e.ctx,
		e.tx,
		"",
		e.attempt,
		"abuse",
		blockedLoginError(decision),
		e.attemptContext,
	)
}

func (e *loginExecution) authenticate() error {
	if !e.subjectValid || e.request.Password == "" {
		return e.rejectCredentials("", true)
	}
	user, err := e.dependencies.Repository.FindUserByUsername(
		e.ctx,
		e.tx,
		e.attempt.Subject,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return e.rejectCredentials("", true)
	}
	if err != nil {
		return e.fail(
			"", LoginFailureStageUserQuery,
			loginPersistenceError("find login user", err),
		)
	}
	e.user = user
	if err := e.checkPasswordResetGate(); err != nil {
		return err
	}
	if err := e.verifyCredential(); err != nil {
		return err
	}
	if err := e.dependencies.Abuse.RecordSuccess(e.ctx, e.tx, e.attempt); err != nil {
		return e.fail(
			e.user.ID, LoginFailureStageAbuseRecord,
			loginPersistenceError("record successful login buckets", err),
		)
	}
	return nil
}

func (e *loginExecution) rejectCredentials(userID string, verifyFake bool) error {
	if verifyFake {
		if err := e.service.verifyFakeCredential(e.request.Password); err != nil {
			return e.fail(userID, LoginFailureStageCredentialVerify, err)
		}
	}
	e.transactionClosed = true
	return e.service.recordAndCommitLoginFailure(
		e.ctx,
		e.tx,
		userID,
		e.attempt,
		"credentials",
		e.attemptContext,
	)
}

func (e *loginExecution) checkPasswordResetGate() error {
	if err := e.service.checkMaintenanceLogin(e.ctx, e.tx, e.user); err != nil {
		if errors.Is(err, session.ErrSessionRequired) {
			return e.rejectCredentials(e.user.ID, true)
		}
		return e.fail(e.user.ID, LoginFailureStageRecoveryMode, platform.NewPublicError(platform.CodePlatformRecoveryInProgress, err))
	}
	err := e.dependencies.PasswordResetGate.CheckLockedUser(e.ctx, e.tx, e.user.ID)
	if err == nil {
		return nil
	}
	if errors.Is(err, session.ErrPasswordResetGateClosed) {
		return e.rejectCredentials(e.user.ID, false)
	}
	return e.fail(
		e.user.ID,
		LoginFailureStageUserQuery,
		loginFailedError(fmt.Errorf("check password reset Gate: %w", err)),
	)
}

func (e *loginExecution) verifyCredential() error {
	credential, err := e.service.verifyLoginCredential(
		e.ctx,
		e.tx,
		e.user,
		e.request.Password,
		e.now,
	)
	if errors.Is(err, errLoginRejected) {
		return e.rejectCredentials(e.user.ID, false)
	}
	if err != nil {
		return e.fail(
			e.user.ID,
			loginFailureStageFor(err, LoginFailureStageCredentialVerify),
			err,
		)
	}
	e.credential = credential
	return nil
}

func (e *loginExecution) prepareSession() error {
	input, err := e.dependencies.SessionIssuer.Issue(
		e.user,
		session.CredentialKind(e.credential.kind),
		e.credential.id,
	)
	if err != nil {
		return e.fail(
			e.user.ID,
			LoginFailureStageSessionIssue,
			platform.NewPublicError(
				platform.CodePlatformInternalError,
				fmt.Errorf("issue login session: %w", err),
			),
		)
	}
	if err := validateLoginSessionCookieToken(input.Token); err != nil {
		return e.fail(e.user.ID, LoginFailureStageSessionIssue, err)
	}
	e.sessionInput = input
	return nil
}

func (e *loginExecution) storeSession() error {
	var err error
	if _, maintenance := e.dependencies.RecoveryMode.(*recoveryIdentityExecution); maintenance {
		e.sessionCredentials, err = e.service.storeMaintenanceLogin(e.ctx, e.tx, e.request.ExistingSessionToken, e.sessionInput)
	} else if e.request.ExistingSessionToken == "" {
		e.sessionCredentials, err = e.dependencies.Sessions.Create(
			e.ctx,
			e.tx,
			e.sessionInput,
		)
	} else {
		e.sessionCredentials, err = e.dependencies.Sessions.Rotate(
			e.ctx,
			e.tx,
			e.request.ExistingSessionToken,
			e.now,
			loginSessionRotateReason,
			e.sessionInput,
		)
	}
	if err != nil {
		return e.fail(
			e.user.ID,
			LoginFailureStageSessionPersist,
			loginSessionError(err),
		)
	}
	if err := validateLoginSessionCookieToken(e.sessionCredentials.Token); err != nil {
		return e.fail(e.user.ID, LoginFailureStageSessionPersist, err)
	}
	return nil
}

func (e *loginExecution) appendAuditAndCommit() (LoginResult, error) {
	result := e.result()
	if err := e.service.appendLoginAudit(
		e.ctx,
		e.tx,
		e.user.ID,
		e.attempt,
		loginAuditOutcomeSucceeded,
		"",
		"",
		e.attemptContext,
	); err != nil {
		return LoginResult{}, e.fail(
			e.user.ID,
			LoginFailureStageAuditAppend,
			err,
		)
	}
	if err := e.tx.Commit(e.ctx); err != nil {
		e.transactionClosed = true
		return LoginResult{}, e.service.finishUnknownCommit(
			e.ctx,
			e.attemptContext,
			loginPersistenceError("commit successful login", err),
		)
	}
	e.transactionClosed = true
	e.attemptContext.updateObservation(func(observation *LoginObservation) {
		observation.TransactionOutcome = LoginTransactionCommitted
		observation.StateChanged = LoginStateSessionCommitted
		observation.AuditPersisted = LoginAuditPersistedYes
		observation.RollbackOutcome = LoginRollbackNotAttempted
	})
	return result, nil
}

func (e *loginExecution) result() LoginResult {
	return LoginResult{
		User: AuthenticatedUser{
			ID: e.user.ID, Username: e.user.Username, Role: e.user.Role, Status: e.user.Status,
			IsInitialAdmin:     e.user.IsInitialAdmin,
			MustChangePassword: e.credential.kind != SessionCredentialPermanent,
			MustAcknowledgePrivacy: e.user.Role == RoleAdmin &&
				e.user.OnboardingCompletedAt == nil,
			AuthenticatedCredential:   e.credential.kind,
			AuthenticatedCredentialID: e.credential.id,
		},
		SessionToken: e.sessionCredentials.Token,
		CSRFToken:    e.sessionCredentials.CSRFToken,
	}
}

func (e *loginExecution) fail(
	userID string,
	stage LoginFailureStage,
	publicErr error,
) error {
	e.transactionClosed = true
	return e.service.failLoginTransaction(
		e.ctx,
		e.tx,
		e.attemptContext,
		userID,
		e.source,
		stage,
		publicErr,
	)
}

type verifiedLoginCredential struct {
	kind SessionCredentialKind
	id   *string
}

func (s *Service) verifyLoginCredential(
	ctx context.Context,
	tx pgx.Tx,
	user User,
	password string,
	now time.Time,
) (verifiedLoginCredential, error) {
	if user.Status != UserStatusActive {
		return verifiedLoginCredential{}, s.rejectAfterFakeVerification(password)
	}
	if user.CredentialMode == CredentialModeRecoveryBootstrap {
		if !user.IsInitialAdmin || !user.RecoveryRequired {
			return verifiedLoginCredential{}, errLoginRejected
		}
	} else if user.RecoveryRequired {
		return verifiedLoginCredential{}, s.rejectAfterFakeVerification(password)
	}

	switch user.CredentialMode {
	case CredentialModePermanent:
		return s.verifyStoredCredential(
			ctx,
			tx,
			user,
			password,
			SessionCredentialPermanent,
			nil,
			user.PermanentPasswordDigest,
		)
	case CredentialModeBootstrap:
		if !user.IsInitialAdmin {
			return verifiedLoginCredential{}, s.rejectAfterFakeVerification(password)
		}
		return s.verifyStoredCredential(
			ctx,
			tx,
			user,
			password,
			SessionCredentialBootstrap,
			nil,
			user.PermanentPasswordDigest,
		)
	case CredentialModeTemporary:
		return s.verifyTemporaryCredential(ctx, tx, user, password, now)
	case CredentialModeRecoveryBootstrap:
		allowed, err := s.loginDependencies.RecoveryMode.RecoveryBootstrapAllowed(ctx)
		if err != nil {
			return verifiedLoginCredential{}, withLoginFailureStage(
				LoginFailureStageRecoveryMode,
				platform.NewPublicError(
					platform.CodePlatformRecoveryInProgress,
					fmt.Errorf("check recovery bootstrap mode: %w", err),
				),
			)
		}
		_, currentAuthority := s.loginDependencies.RecoveryMode.(*recoveryIdentityExecution)
		if !allowed || !currentAuthority {
			return verifiedLoginCredential{}, errLoginRejected
		}
		return s.verifyStoredCredential(
			ctx,
			tx,
			user,
			password,
			SessionCredentialRecoveryBootstrap,
			nil,
			user.PermanentPasswordDigest,
		)
	case CredentialModeRecoveryLocked:
		return verifiedLoginCredential{}, s.rejectAfterFakeVerification(password)
	default:
		return verifiedLoginCredential{}, s.rejectAfterFakeVerification(password)
	}
}

func (s *Service) verifyTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	user User,
	password string,
	now time.Time,
) (verifiedLoginCredential, error) {
	credential, err := s.loginDependencies.Repository.FindLatestTemporaryCredential(
		ctx,
		tx,
		user.ID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return verifiedLoginCredential{}, s.rejectAfterFakeVerification(password)
	}
	if err != nil {
		return verifiedLoginCredential{}, withLoginFailureStage(
			LoginFailureStageTemporaryCredentialQuery,
			loginPersistenceError(
				"find latest temporary credential",
				err,
			),
		)
	}

	matched, needsRehash, err := s.loginDependencies.Passwords.Verify(
		password,
		credential.SecretDigest,
	)
	if err != nil {
		return verifiedLoginCredential{}, loginCredentialVerifierError(err)
	}
	if !matched ||
		!TemporaryCredentialAuthenticatable(credential, credential.Generation, now) {
		return verifiedLoginCredential{}, errLoginRejected
	}
	if needsRehash {
		digest, err := s.loginDependencies.Passwords.Hash(password)
		if err != nil {
			return verifiedLoginCredential{}, withLoginFailureStage(
				LoginFailureStageCredentialRehash,
				loginCredentialVerifierError(err),
			)
		}
		if err := s.loginDependencies.Repository.UpdateTemporaryCredentialDigest(
			ctx,
			tx,
			credential.ID,
			digest,
		); err != nil {
			return verifiedLoginCredential{}, withLoginFailureStage(
				LoginFailureStageCredentialRehash,
				loginPersistenceError(
					"rehash temporary credential",
					err,
				),
			)
		}
	}
	credentialID := credential.ID
	return verifiedLoginCredential{
		kind: SessionCredentialTemporary,
		id:   &credentialID,
	}, nil
}

func (s *Service) verifyStoredCredential(
	ctx context.Context,
	tx pgx.Tx,
	user User,
	password string,
	kind SessionCredentialKind,
	credentialID *string,
	digest *string,
) (verifiedLoginCredential, error) {
	if digest == nil || *digest == "" {
		return verifiedLoginCredential{}, s.rejectAfterFakeVerification(password)
	}
	matched, needsRehash, err := s.loginDependencies.Passwords.Verify(password, *digest)
	if err != nil {
		return verifiedLoginCredential{}, loginCredentialVerifierError(err)
	}
	if !matched {
		return verifiedLoginCredential{}, errLoginRejected
	}
	if needsRehash {
		rehashed, err := s.loginDependencies.Passwords.Hash(password)
		if err != nil {
			return verifiedLoginCredential{}, withLoginFailureStage(
				LoginFailureStageCredentialRehash,
				loginCredentialVerifierError(err),
			)
		}
		if err := s.loginDependencies.Repository.UpdatePermanentPasswordDigest(
			ctx,
			tx,
			user.ID,
			rehashed,
		); err != nil {
			return verifiedLoginCredential{}, withLoginFailureStage(
				LoginFailureStageCredentialRehash,
				loginPersistenceError(
					"rehash permanent credential",
					err,
				),
			)
		}
	}
	return verifiedLoginCredential{kind: kind, id: credentialID}, nil
}

func (s *Service) rejectAfterFakeVerification(password string) error {
	if err := s.verifyFakeCredential(password); err != nil {
		return err
	}
	return errLoginRejected
}

func (s *Service) verifyFakeCredential(password string) error {
	_, _, err := s.loginDependencies.Passwords.Verify(
		password,
		s.loginDependencies.FakeCredentialDigest,
	)
	if err != nil {
		return loginCredentialVerifierError(err)
	}
	return nil
}

func (s *Service) recordAndCommitLoginFailure(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	attempt LoginAttempt,
	reason string,
	attemptContext *LoginAttemptContext,
) error {
	if err := s.loginDependencies.Abuse.RecordFailure(ctx, tx, attempt); err != nil {
		return s.failLoginTransaction(
			ctx, tx, attemptContext, userID, attempt.SourceFingerprint,
			LoginFailureStageAbuseRecord,
			loginPersistenceError("record failed login buckets", err),
		)
	}
	return s.commitLoginFailure(
		ctx,
		tx,
		userID,
		attempt,
		reason,
		loginFailedError(errLoginRejected),
		attemptContext,
	)
}

func (s *Service) commitLoginFailure(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	attempt LoginAttempt,
	reason string,
	publicErr error,
	attemptContext *LoginAttemptContext,
) error {
	if err := s.appendLoginAudit(
		ctx,
		tx,
		userID,
		attempt,
		loginAuditOutcomeFailed,
		string(platform.PublicCode(publicErr)),
		reason,
		attemptContext,
	); err != nil {
		return s.failLoginTransaction(
			ctx, tx, attemptContext, userID, attempt.SourceFingerprint,
			LoginFailureStageAuditAppend, err,
		)
	}
	if err := tx.Commit(ctx); err != nil {
		return s.finishUnknownCommit(
			ctx,
			attemptContext,
			loginPersistenceError("commit failed login evidence", err),
		)
	}
	attemptContext.updateObservation(func(observation *LoginObservation) {
		observation.TransactionOutcome = LoginTransactionCommitted
		observation.StateChanged = LoginStateNone
		observation.AuditPersisted = LoginAuditPersistedYes
		observation.RollbackOutcome = LoginRollbackNotAttempted
	})
	return publicErr
}

func (s *Service) appendLoginAudit(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	attempt LoginAttempt,
	outcome string,
	errorCode string,
	reason string,
	attemptContext *LoginAttemptContext,
) error {
	traceID, ok := platform.TraceIDFromContext(ctx)
	if !ok {
		return platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("login trace ID is required"),
		)
	}
	if attemptContext == nil || strings.TrimSpace(attemptContext.ID) == "" ||
		attemptContext.OccurredAt.IsZero() {
		return platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("login attempt audit identity is required"),
		)
	}
	targetType, targetID := loginAuditTargetUser, userID
	if targetID == "" {
		targetType, targetID = loginAuditTargetAttempt, traceID
	}
	source := attempt.SourceFingerprint
	if len(attempt.subjectDigest) == 0 ||
		strings.TrimSpace(attempt.subjectKeyVersion) == "" {
		return platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("login subject audit evidence is incomplete"),
		)
	}
	metadata := map[string]any{
		loginAuditSubjectDigest: hex.EncodeToString(attempt.subjectDigest),
		loginAuditSubjectKey:    attempt.subjectKeyVersion,
	}
	event := audit.Event{
		ID: attemptContext.ID, OccurredAt: attemptContext.OccurredAt,
		ActorKind: audit.ActorAnonymous, Action: loginAuditAction,
		TargetType: targetType, TargetID: targetID, Outcome: outcome,
		TraceID: traceID, SourceFingerprint: &source, Metadata: metadata,
	}
	if errorCode != "" {
		event.ErrorCode = &errorCode
	}
	if reason != "" {
		event.Metadata["reason_bucket"] = reason
	}
	attemptContext.setPrepared(event)
	if err := s.loginDependencies.Audit.Append(ctx, tx, event); err != nil {
		return loginPersistenceError("append login audit event", err)
	}
	return nil
}

func (s *Service) finishLoginWithoutTransaction(
	ctx context.Context,
	attempt *LoginAttemptContext,
	userID string,
	source string,
	stage LoginFailureStage,
	cause error,
) error {
	traceID, _ := platform.TraceIDFromContext(ctx)
	event := loginFailureAuditEvent(attempt, traceID, optionalString(source), userID, stage, cause)
	persisted, _ := attempt.recordOnce(ctx, loginAttemptServiceEntered, LoginFailureAuditRequest{
		Mode: LoginFailureAuditInsertFailure, Event: event,
	})
	attempt.updateObservation(func(observation *LoginObservation) {
		observation.FailureStage = stage
		observation.ReasonBucket = loginReasonBucketFor(stage, cause)
		observation.ErrorCode = string(platform.PublicCode(cause))
		observation.TransactionOutcome = LoginTransactionNotStarted
		observation.StateChanged = LoginStateNone
		observation.AuditPersisted = persisted
		observation.RollbackOutcome = LoginRollbackNotAttempted
	})
	return cause
}

func (s *Service) failLoginTransaction(
	ctx context.Context,
	tx pgx.Tx,
	attempt *LoginAttemptContext,
	userID string,
	source string,
	stage LoginFailureStage,
	cause error,
) error {
	rollbackErr := tx.Rollback(ctx)
	transactionOutcome := LoginTransactionRolledBack
	stateChanged := LoginStateRolledBack
	rollbackOutcome := LoginRollbackSucceeded
	persisted := LoginAuditPersistedNo
	if rollbackErr == nil {
		traceID, _ := platform.TraceIDFromContext(ctx)
		event := loginFailureAuditEvent(
			attempt,
			traceID,
			optionalString(source),
			userID,
			stage,
			cause,
		)
		persisted, _ = attempt.recordOnce(
			ctx,
			loginAttemptServiceEntered,
			LoginFailureAuditRequest{
				Mode: LoginFailureAuditInsertFailure, Event: event,
			},
		)
	} else {
		transactionOutcome = LoginTransactionUnknown
		stateChanged = LoginStateUnknown
		rollbackOutcome = LoginRollbackUnknown
		persisted = LoginAuditPersistedUnknown
		if prepared, ok := attempt.prepared(); ok {
			persisted, _ = attempt.recordOnce(
				ctx,
				loginAttemptServiceEntered,
				LoginFailureAuditRequest{
					Mode: LoginFailureAuditVerifyPrepared, Event: prepared,
				},
			)
		}
	}
	attempt.updateObservation(func(observation *LoginObservation) {
		observation.FailureStage = stage
		observation.ReasonBucket = loginReasonBucketFor(stage, cause)
		observation.ErrorCode = string(platform.PublicCode(cause))
		observation.TransactionOutcome = transactionOutcome
		observation.StateChanged = stateChanged
		observation.AuditPersisted = persisted
		observation.RollbackOutcome = rollbackOutcome
	})
	return cause
}

func (s *Service) finishUnknownCommit(
	ctx context.Context,
	attempt *LoginAttemptContext,
	cause error,
) error {
	persisted := LoginAuditPersistedUnknown
	if prepared, ok := attempt.prepared(); ok {
		persisted, _ = attempt.recordOnce(
			ctx,
			loginAttemptServiceEntered,
			LoginFailureAuditRequest{
				Mode: LoginFailureAuditVerifyPrepared, Event: prepared,
			},
		)
	}
	attempt.updateObservation(func(observation *LoginObservation) {
		observation.FailureStage = LoginFailureStageTransactionCommit
		observation.ReasonBucket = "persistence"
		observation.ErrorCode = string(platform.PublicCode(cause))
		observation.TransactionOutcome = LoginTransactionUnknown
		observation.StateChanged = LoginStateUnknown
		observation.AuditPersisted = persisted
		observation.RollbackOutcome = LoginRollbackNotAttempted
	})
	return cause
}

func loginFailureAuditEvent(
	attempt *LoginAttemptContext,
	traceID string,
	source *string,
	userID string,
	stage LoginFailureStage,
	cause error,
) audit.Event {
	targetType, targetID := loginAuditTargetUser, userID
	if targetID == "" {
		targetType, targetID = loginAuditTargetAttempt, traceID
	}
	errorCode := string(platform.PublicCode(cause))
	event := audit.Event{
		ActorKind:         audit.ActorAnonymous,
		Action:            loginAuditAction,
		TargetType:        targetType,
		TargetID:          targetID,
		Outcome:           loginAuditOutcomeFailed,
		ErrorCode:         &errorCode,
		TraceID:           traceID,
		SourceFingerprint: cloneStringPointer(source),
		Metadata: map[string]any{
			"failure_stage": string(stage),
			"reason_bucket": loginReasonBucketFor(stage, cause),
		},
	}
	if attempt != nil {
		event.ID = attempt.ID
		event.OccurredAt = attempt.OccurredAt
	}
	return event
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func loginReasonBucket(stage LoginFailureStage) string {
	switch stage {
	case LoginFailureStageTransactionBegin,
		LoginFailureStageAbuseCheck,
		LoginFailureStageUserQuery,
		LoginFailureStageTemporaryCredentialQuery,
		LoginFailureStageAbuseRecord,
		LoginFailureStageTransactionCommit:
		return "persistence"
	case LoginFailureStageCredentialVerify, LoginFailureStageCredentialRehash,
		LoginFailureStageRecoveryMode:
		return "credential_processing"
	case LoginFailureStageSessionIssue, LoginFailureStageSessionPersist:
		return "session"
	case LoginFailureStageAuditAppend, LoginFailureStageFailureAudit:
		return "audit"
	case LoginFailureStageCookieOutput, LoginFailureStageResponseWrite:
		return "response"
	case LoginFailureStageSourceEvidence:
		return "request_context"
	default:
		return "unknown"
	}
}

func loginReasonBucketFor(stage LoginFailureStage, cause error) string {
	var rateLimit *loginRateLimitError
	if errors.As(cause, &rateLimit) && rateLimit.resourceCapacity {
		return "resource_capacity"
	}
	return loginReasonBucket(stage)
}

func validateLoginSessionCookieToken(token string) error {
	if token == "" {
		return platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("issued login session token is invalid"),
		)
	}
	cookie := http.Cookie{Name: platform.SessionCookieName, Value: token}
	if err := cookie.Valid(); err != nil {
		return platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("issued login session token is invalid"),
		)
	}
	return nil
}

func blockedLoginError(decision LoginAbuseDecision) error {
	switch decision.Kind {
	case LoginAbuseSourceBlocked:
		publicErr := platform.NewPublicError(
			platform.CodeRequestRateLimited,
			errors.New("login source is blocked"),
		)
		return &loginRateLimitError{
			retryAfter: decision.RetryAfter,
			cause:      publicErr,
		}
	case LoginAbuseSubjectBlocked:
		return loginFailedError(errors.New("login subject is blocked"))
	default:
		return platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("login abuse decision is invalid"),
		)
	}
}

type loginRateLimitError struct {
	retryAfter       time.Duration
	cause            error
	resourceCapacity bool
}

func (e *loginRateLimitError) Error() string {
	if e == nil || e.cause == nil {
		return ""
	}
	return e.cause.Error()
}

func (e *loginRateLimitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// LoginRateLimitRetryAfter returns the trusted duration carried by a login 429.
func LoginRateLimitRetryAfter(err error) (time.Duration, bool) {
	var rateLimit *loginRateLimitError
	if errors.As(err, &rateLimit) && rateLimit.retryAfter > 0 {
		return rateLimit.retryAfter, true
	}
	return 0, false
}

func loginFailedError(cause error) error {
	return platform.NewPublicError(platform.CodeAuthLoginFailed, cause)
}

func loginPersistenceError(operation string, cause error) error {
	return platform.NewPublicError(
		platform.CodePlatformDatabaseUnavailable,
		fmt.Errorf("%s: %w", operation, cause),
	)
}

func loginCredentialVerifierError(cause error) error {
	var capacity *security.Argon2idCapacityError
	if errors.As(cause, &capacity) && capacity.RetryAfter() > 0 {
		return &loginRateLimitError{
			cause:            platform.NewPublicError(platform.CodeRequestRateLimited, cause),
			retryAfter:       capacity.RetryAfter(),
			resourceCapacity: true,
		}
	}
	return platform.NewPublicError(
		platform.CodePlatformInternalError,
		fmt.Errorf("verify login credential: %w", cause),
	)
}

type loginFailureStageError struct {
	stage LoginFailureStage
	cause error
}

func (e *loginFailureStageError) Error() string {
	if e == nil || e.cause == nil {
		return ""
	}
	return e.cause.Error()
}

func (e *loginFailureStageError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func withLoginFailureStage(stage LoginFailureStage, cause error) error {
	if cause == nil {
		return nil
	}
	return &loginFailureStageError{stage: stage, cause: cause}
}

func loginFailureStageFor(err error, fallback LoginFailureStage) LoginFailureStage {
	var staged *loginFailureStageError
	if errors.As(err, &staged) && staged.stage != LoginFailureStageNone {
		return staged.stage
	}
	return fallback
}

func loginSessionError(cause error) error {
	code := platform.CodePlatformInternalError
	if errors.Is(cause, session.ErrStoreUnavailable) {
		code = platform.CodePlatformDatabaseUnavailable
	}
	return platform.NewPublicError(code, fmt.Errorf("persist login session: %w", cause))
}

func rollbackLogin(ctx context.Context, tx pgx.Tx, cause error) error {
	if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return platform.NewPublicError(
			platform.CodePlatformDatabaseUnavailable,
			errors.Join(cause, fmt.Errorf("rollback login transaction: %w", err)),
		)
	}
	return cause
}

// LoginFailureAuditDatabase is the minimum independent audit database contract.
type LoginFailureAuditDatabase interface {
	Begin(context.Context) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// LoginFailureAuditStore appends and reads immutable audit events.
type LoginFailureAuditStore interface {
	Append(context.Context, pgx.Tx, audit.Event) error
	Get(context.Context, audit.QueryRower, string) (audit.Event, error)
}

// PostgresLoginFailureAuditor implements insert-or-verify without blind rewrites.
type PostgresLoginFailureAuditor struct {
	database LoginFailureAuditDatabase
	store    LoginFailureAuditStore
}

// NewPostgresLoginFailureAuditor creates the controlled independent audit entry point.
func NewPostgresLoginFailureAuditor(
	database LoginFailureAuditDatabase,
	store LoginFailureAuditStore,
) (*PostgresLoginFailureAuditor, error) {
	if database == nil || store == nil {
		return nil, errors.New("login failure audit dependencies are incomplete")
	}
	return &PostgresLoginFailureAuditor{database: database, store: store}, nil
}

// RecordOnce performs one closed audit operation and never updates an event.
func (a *PostgresLoginFailureAuditor) RecordOnce(
	ctx context.Context,
	request LoginFailureAuditRequest,
) (LoginAuditPersistence, error) {
	if a == nil || a.database == nil || a.store == nil ||
		strings.TrimSpace(request.Event.ID) == "" {
		return LoginAuditPersistedUnknown, errors.New("login failure audit request is incomplete")
	}
	switch request.Mode {
	case LoginFailureAuditVerifyPrepared:
		return a.verifyPrepared(ctx, request.Event)
	case LoginFailureAuditInsertFailure:
	default:
		return LoginAuditPersistedUnknown, errors.New("login failure audit mode is invalid")
	}

	tx, err := a.database.Begin(ctx)
	if err != nil {
		return LoginAuditPersistedUnknown, errors.New("begin login failure audit")
	}
	if err := a.store.Append(ctx, tx, request.Event); err != nil {
		rollbackErr := tx.Rollback(ctx)
		if persisted, verifyErr := a.verifyPrepared(ctx, request.Event); persisted == LoginAuditPersistedYes {
			return persisted, verifyErr
		}
		if rollbackErr == nil {
			return LoginAuditPersistedNo, errors.New("append login failure audit")
		}
		return LoginAuditPersistedUnknown, errors.New("append login failure audit")
	}
	if err := tx.Commit(ctx); err != nil {
		return a.verifyPrepared(ctx, request.Event)
	}
	return LoginAuditPersistedYes, nil
}

func (a *PostgresLoginFailureAuditor) verifyPrepared(
	ctx context.Context,
	prepared audit.Event,
) (LoginAuditPersistence, error) {
	stored, err := a.store.Get(ctx, a.database, prepared.ID)
	if err != nil {
		return LoginAuditPersistedUnknown, errors.New("verify prepared login audit")
	}
	if !sameAuditEvent(stored, prepared) {
		return LoginAuditPersistedUnknown, errors.New("prepared login audit does not match")
	}
	return LoginAuditPersistedYes, nil
}

func sameAuditEvent(left audit.Event, right audit.Event) bool {
	if left.ID != right.ID ||
		!left.OccurredAt.Equal(right.OccurredAt) ||
		left.ActorKind != right.ActorKind ||
		!sameStringPointer(left.ActorUserID, right.ActorUserID) ||
		!sameStringPointer(left.ActorSystem, right.ActorSystem) ||
		!sameStringPointer(left.ActorRole, right.ActorRole) ||
		left.Action != right.Action ||
		left.TargetType != right.TargetType ||
		left.TargetID != right.TargetID ||
		left.Outcome != right.Outcome ||
		!sameStringPointer(left.ErrorCode, right.ErrorCode) ||
		left.TraceID != right.TraceID ||
		!sameStringPointer(left.SourceFingerprint, right.SourceFingerprint) {
		return false
	}
	leftMetadata, leftErr := canonicalAuditMetadata(left.Metadata)
	rightMetadata, rightErr := canonicalAuditMetadata(right.Metadata)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftMetadata, rightMetadata)
}

func sameStringPointer(left *string, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func canonicalAuditMetadata(metadata map[string]any) ([]byte, error) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return json.Marshal(metadata)
}

func cloneAuditEvent(event audit.Event) audit.Event {
	cloned := event
	cloned.ActorUserID = cloneStringPointer(event.ActorUserID)
	cloned.ActorSystem = cloneStringPointer(event.ActorSystem)
	cloned.ActorRole = cloneStringPointer(event.ActorRole)
	cloned.ErrorCode = cloneStringPointer(event.ErrorCode)
	cloned.SourceFingerprint = cloneStringPointer(event.SourceFingerprint)
	if event.Metadata != nil {
		cloned.Metadata = make(map[string]any, len(event.Metadata))
		for key, value := range event.Metadata {
			cloned.Metadata[key] = value
		}
	}
	return cloned
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func loginContextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("login observation context is required")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func normalizeLoginSubject(username string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(username))
	if normalized == "" {
		return "", false
	}
	for _, character := range normalized {
		if unicode.IsControl(character) {
			return "", false
		}
	}
	return normalized, true
}

func writeLoginDigestField(mac interface{ Write([]byte) (int, error) }, value string) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = mac.Write(length[:])
	_, _ = mac.Write([]byte(value))
}

const findLoginUserSQL = `
	SELECT
		u.id,
		u.username,
		u.permanent_password_digest,
		u.auth_generation,
		u.permanent_password_generation,
		u.password_reset_execution_generation,
		u.password_reset_current_operation_id,
		u.permanent_password_source_operation_id,
		u.permanent_password_source_credential_id,
		u.auth_integrity_hold,
		u.role,
		u.is_initial_admin,
		u.status,
		u.credential_mode,
		u.recovery_required,
		u.onboarding_privacy_acknowledged_version,
		u.onboarding_completed_at,
		u.deleted_at
	FROM identity_identifiers AS identifier
	JOIN users AS u ON u.id = identifier.user_id
	WHERE
		identifier.kind = 'username'
		AND identifier.normalized_value = $1
	FOR UPDATE OF u
`

const findLatestTemporaryCredentialSQL = `
	SELECT
		id,
		user_id,
		purpose,
		generation,
		auth_generation,
		source_operation_id,
		state,
		secret_digest,
		delivered_to_email,
		issued_at,
		expires_at,
		superseded_at,
		consumed_at,
		invalidated_at
	FROM temporary_credentials
	WHERE user_id = $1
	ORDER BY generation DESC
	LIMIT 1
	FOR SHARE
`

const updatePermanentLoginDigestSQL = `
	UPDATE users
	SET permanent_password_digest = $1
	WHERE id = $2
`

const updateTemporaryLoginDigestSQL = `
	UPDATE temporary_credentials
	SET secret_digest = $1
	WHERE id = $2
`

const selectLoginAttemptBucketForUpdateSQL = `
	SELECT
		bucket_kind,
		bucket_key_digest,
		digest_key_version,
		window_started_at,
		failure_count,
		blocked_until,
		updated_at
	FROM login_attempt_buckets
	WHERE bucket_kind = $1 AND bucket_key_digest = $2
	FOR UPDATE
`

const lockLoginAttemptBucketSQL = `
	SELECT pg_advisory_xact_lock(
		hashtextextended($1::text || ':' || encode($2::bytea, 'hex'), 0)
	)
`

const upsertLoginAttemptBucketSQL = `
	INSERT INTO login_attempt_buckets (
		bucket_kind,
		bucket_key_digest,
		digest_key_version,
		window_started_at,
		failure_count,
		blocked_until,
		updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (bucket_kind, bucket_key_digest)
	DO UPDATE SET
		digest_key_version = EXCLUDED.digest_key_version,
		window_started_at = EXCLUDED.window_started_at,
		failure_count = EXCLUDED.failure_count,
		blocked_until = EXCLUDED.blocked_until,
		updated_at = EXCLUDED.updated_at
`

const deleteLoginAttemptBucketSQL = `
	DELETE FROM login_attempt_buckets
	WHERE bucket_kind = $1 AND bucket_key_digest = $2
`
