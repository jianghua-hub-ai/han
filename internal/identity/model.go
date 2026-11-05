// Package identity owns account, identifier, session, credential, and identity evidence models.
package identity

import "time"

// Role is an identity authorization role.
type Role string

const (
	// RoleMember is a regular community member.
	RoleMember Role = "member"
	// RoleAdmin is an administrator.
	RoleAdmin Role = "admin"
)

// UserStatus is the authoritative account lifecycle state.
type UserStatus string

const (
	// UserStatusActive permits normal authentication and authorization.
	UserStatusActive UserStatus = "active"
	// UserStatusSuspended denies normal authentication and authorization.
	UserStatusSuspended UserStatus = "suspended"
	// UserStatusDeleted is the irreversible account tombstone.
	UserStatusDeleted UserStatus = "deleted"
)

// CredentialMode selects the only credential path accepted for an account.
type CredentialMode string

const (
	// CredentialModePermanent selects the permanent password digest.
	CredentialModePermanent CredentialMode = "permanent"
	// CredentialModeBootstrap selects the initial administrator bootstrap digest.
	CredentialModeBootstrap CredentialMode = "bootstrap"
	// CredentialModeTemporary selects the current temporary credential.
	CredentialModeTemporary CredentialMode = "temporary"
	// CredentialModeRecoveryLocked denies all credentials during recovery review.
	CredentialModeRecoveryLocked CredentialMode = "recovery_locked"
	// CredentialModeRecoveryBootstrap selects the initial administrator recovery secret.
	CredentialModeRecoveryBootstrap CredentialMode = "recovery_bootstrap"
)

// IdentifierKind identifies a normalized login identifier.
type IdentifierKind string

const (
	// IdentifierUsername is the permanently reserved username.
	IdentifierUsername IdentifierKind = "username"
	// IdentifierEmail is the normalized email address.
	IdentifierEmail IdentifierKind = "email"
	// IdentifierPhone is the normalized phone number.
	IdentifierPhone IdentifierKind = "phone"
)

// IdentifierLifecycle is the identifier reservation state.
type IdentifierLifecycle string

const (
	// IdentifierActive belongs to an active account identity.
	IdentifierActive IdentifierLifecycle = "active"
	// IdentifierReserved remains unavailable after account lifecycle changes.
	IdentifierReserved IdentifierLifecycle = "reserved"
)

// SessionCredentialKind records the credential that authenticated a session.
type SessionCredentialKind string

const (
	// SessionCredentialPermanent records permanent-password authentication.
	SessionCredentialPermanent SessionCredentialKind = "permanent"
	// SessionCredentialBootstrap records initial administrator bootstrap authentication.
	SessionCredentialBootstrap SessionCredentialKind = "bootstrap"
	// SessionCredentialTemporary records temporary-credential authentication.
	SessionCredentialTemporary SessionCredentialKind = "temporary"
	// SessionCredentialRecoveryBootstrap records recovery bootstrap authentication.
	SessionCredentialRecoveryBootstrap SessionCredentialKind = "recovery_bootstrap"
)

// TemporaryCredentialPurpose is the authorized temporary credential use.
type TemporaryCredentialPurpose string

const (
	// TemporaryCredentialNewAdministrator bootstraps a newly created administrator.
	TemporaryCredentialNewAdministrator TemporaryCredentialPurpose = "new_administrator"
	// TemporaryCredentialPasswordReset replaces a forgotten or reset password.
	TemporaryCredentialPasswordReset TemporaryCredentialPurpose = "password_reset"
)

// TemporaryCredentialState is the temporary credential lifecycle state.
type TemporaryCredentialState string

const (
	// TemporaryCredentialPendingDelivery is not usable before successful delivery.
	TemporaryCredentialPendingDelivery TemporaryCredentialState = "pending_delivery"
	// TemporaryCredentialActive is the current usable temporary credential.
	TemporaryCredentialActive TemporaryCredentialState = "active"
	// TemporaryCredentialExpired remains current and blocks permanent-password fallback.
	TemporaryCredentialExpired TemporaryCredentialState = "expired"
	// TemporaryCredentialSuperseded was replaced by a later credential.
	TemporaryCredentialSuperseded TemporaryCredentialState = "superseded"
	// TemporaryCredentialConsumed completed the password-change flow.
	TemporaryCredentialConsumed TemporaryCredentialState = "consumed"
	// TemporaryCredentialRecoveryInvalidated was invalidated during recovery.
	TemporaryCredentialRecoveryInvalidated TemporaryCredentialState = "recovery_invalidated"
)

// PasswordResetRequestKind identifies the durable reset candidate shape.
type PasswordResetRequestKind string

const (
	// PasswordResetInitialReset creates the first durable reset candidate.
	PasswordResetInitialReset PasswordResetRequestKind = "initial_reset"
	// PasswordResetResend replaces a settled password-reset credential.
	PasswordResetResend PasswordResetRequestKind = "resend"
)

// PasswordResetOperationState is the durable reset authority lifecycle.
type PasswordResetOperationState string

const (
	PasswordResetOperationPrepared         PasswordResetOperationState = "prepared"
	PasswordResetOperationDispatching      PasswordResetOperationState = "dispatching"
	PasswordResetOperationCommitUnknown    PasswordResetOperationState = "commit_unknown"
	PasswordResetOperationRecoveryRequired PasswordResetOperationState = "recovery_required"
	PasswordResetOperationConverging       PasswordResetOperationState = "converging"
	PasswordResetOperationSucceeded        PasswordResetOperationState = "succeeded"
	PasswordResetOperationInvalidated      PasswordResetOperationState = "invalidated"
	PasswordResetOperationFailed           PasswordResetOperationState = "failed"
)

// PasswordResetDeliveryState records durable mail-dispatch evidence.
type PasswordResetDeliveryState string

const (
	PasswordResetDeliveryNotStarted  PasswordResetDeliveryState = "not_started"
	PasswordResetDeliveryDispatching PasswordResetDeliveryState = "dispatching"
	PasswordResetDeliveryAccepted    PasswordResetDeliveryState = "accepted"
	PasswordResetDeliveryRejected    PasswordResetDeliveryState = "rejected"
	PasswordResetDeliveryUnknown     PasswordResetDeliveryState = "unknown"
)

// PasswordResetSettlementState records the authentication-capability outcome.
type PasswordResetSettlementState string

const (
	PasswordResetSettlementPending            PasswordResetSettlementState = "pending"
	PasswordResetSettlementTemporaryActive    PasswordResetSettlementState = "temporary_active"
	PasswordResetSettlementPermanentSuccessor PasswordResetSettlementState = "permanent_successor"
	PasswordResetSettlementInvalidated        PasswordResetSettlementState = "invalidated"
	PasswordResetSettlementIntegrityHold      PasswordResetSettlementState = "integrity_hold"
)

// PasswordResetRecoveryJobStatus is the durable recovery outbox lifecycle.
type PasswordResetRecoveryJobStatus string

const (
	PasswordResetRecoveryJobReady      PasswordResetRecoveryJobStatus = "ready"
	PasswordResetRecoveryJobLeased     PasswordResetRecoveryJobStatus = "leased"
	PasswordResetRecoveryJobRetryWait  PasswordResetRecoveryJobStatus = "retry_wait"
	PasswordResetRecoveryJobSucceeded  PasswordResetRecoveryJobStatus = "succeeded"
	PasswordResetRecoveryJobDeadLetter PasswordResetRecoveryJobStatus = "dead_letter"
)

// LoginBucketKind identifies the brute-force protection key dimensions.
type LoginBucketKind string

const (
	// LoginBucketSubject groups attempts by normalized login subject.
	LoginBucketSubject LoginBucketKind = "subject"
	// LoginBucketSource groups attempts by canonical request source.
	LoginBucketSource LoginBucketKind = "source"
	// LoginBucketSubjectSource groups attempts by both subject and source.
	LoginBucketSubjectSource LoginBucketKind = "subject_source"
)

// PrivacyNoticeStatus identifies a notice publication state.
type PrivacyNoticeStatus string

const (
	// PrivacyNoticeCurrent is the single notice accepted by onboarding flows.
	PrivacyNoticeCurrent PrivacyNoticeStatus = "current"
)

// RecoveryDecision is the operator review outcome for a recovered account.
type RecoveryDecision string

const (
	// RecoveryDecisionPending keeps the recovered account fail closed.
	RecoveryDecisionPending RecoveryDecision = "pending"
	// RecoveryDecisionRestored records an explicitly restored account.
	RecoveryDecisionRestored RecoveryDecision = "restored"
	// RecoveryDecisionKeptDeleted preserves canonical deletion.
	RecoveryDecisionKeptDeleted RecoveryDecision = "kept_deleted"
	// RecoveryDecisionDenied leaves the recovered account unauthorized.
	RecoveryDecisionDenied RecoveryDecision = "denied"
)

// User is the identity-owned account and tombstone model.
type User struct {
	ID                                   string
	Username                             string
	Email                                *string
	Phone                                *string
	PermanentPasswordDigest              *string
	AuthGeneration                       int64
	PermanentPasswordGeneration          *int64
	PasswordResetExecutionGeneration     int64
	PasswordResetCurrentOperationID      *string
	PermanentPasswordSourceOperationID   *string
	PermanentPasswordSourceCredentialID  *string
	AuthIntegrityHold                    bool
	AuthIntegrityHoldOperationID         *string
	AuthIntegrityHoldReason              *string
	AuthIntegrityHoldAt                  *time.Time
	Role                                 Role
	IsInitialAdmin                       bool
	Status                               UserStatus
	CredentialMode                       CredentialMode
	RecoveryRequired                     bool
	OnboardingPrivacyAcknowledgedVersion *string
	OnboardingCompletedAt                *time.Time
	RetainedEmail                        *string
	CreatedAt                            time.Time
	UpdatedAt                            time.Time
	DeletedAt                            *time.Time
}

// IdentityIdentifier is the unique normalized identifier truth.
type IdentityIdentifier struct {
	UserID          string
	Kind            IdentifierKind
	NormalizedValue string
	Lifecycle       IdentifierLifecycle
	CreatedAt       time.Time
	ReleasedAt      *time.Time
}

// Session stores only server-side token and CSRF digests.
type Session struct {
	ID                          string
	UserID                      string
	TokenDigest                 []byte
	CSRFTokenDigest             []byte
	AuthenticatedCredentialKind SessionCredentialKind
	AuthenticatedCredentialID   *string
	AuthGeneration              int64
	LineageOperationID          *string
	LineageCredentialID         *string
	IssuedAt                    time.Time
	LastSeenAt                  time.Time
	AbsoluteExpiresAt           time.Time
	IdleExpiresAt               time.Time
	RotatedFromID               *string
	RevokedAt                   *time.Time
	RevokeReason                *string
}

// TemporaryCredential is one generated credential lifecycle record.
type TemporaryCredential struct {
	ID                string
	UserID            string
	Purpose           TemporaryCredentialPurpose
	Generation        int64
	AuthGeneration    int64
	SourceOperationID *string
	State             TemporaryCredentialState
	SecretDigest      string
	DeliveredToEmail  string
	IssuedAt          time.Time
	ExpiresAt         time.Time
	SupersededAt      *time.Time
	ConsumedAt        *time.Time
	InvalidatedAt     *time.Time
}

// PasswordResetOperation is the durable authority for one reset candidate.
type PasswordResetOperation struct {
	Recovery                 *RecoveryResetMetadata
	ID                       string
	UserID                   string
	TemporaryCredentialID    string
	ScopeHash                []byte
	KeyLocator               []byte
	KeyDigest                []byte
	DigestKeyVersion         string
	RequestDigest            []byte
	ExecutionGeneration      int64
	RequestKind              PasswordResetRequestKind
	PredecessorOperationID   *string
	ReplacedCredentialID     *string
	BaseAuthGeneration       int64
	TargetAuthGeneration     int64
	SettlementState          PasswordResetSettlementState
	SettledAuthGeneration    *int64
	SuccessorAuthGeneration  *int64
	SettledAt                *time.Time
	DeliveryState            PasswordResetDeliveryState
	DeliveryAttemptID        *string
	DeliveryAttemptedAt      *time.Time
	DeliveryResultRecordedAt *time.Time
	State                    PasswordResetOperationState
	RecoveryReason           *string
	CreatedAt                time.Time
	UpdatedAt                time.Time
	TerminalAt               *time.Time
}

// PasswordResetRecoveryJob is the one-to-one durable recovery outbox row.
type PasswordResetRecoveryJob struct {
	OperationID     string
	Status          PasswordResetRecoveryJobStatus
	AttemptCount    int64
	NextAttemptAt   time.Time
	LeaseOwner      *string
	LeaseExpiresAt  *time.Time
	LastFailureCode *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     *time.Time
}

// PrivacyNotice is one versioned privacy notice.
type PrivacyNotice struct {
	Version     string
	Content     string
	ContentHash []byte
	EffectiveAt time.Time
	Status      PrivacyNoticeStatus
}

// PrivacyAcknowledgement is append-only user confirmation evidence.
type PrivacyAcknowledgement struct {
	ID                string
	UserID            string
	NoticeVersion     string
	AcknowledgedAt    time.Time
	SourceKind        string
	SourceFingerprint string
}

// AdministratorInformationDeclaration is append-only information provision evidence.
type AdministratorInformationDeclaration struct {
	ID                  string
	ActorUserID         string
	TargetLocatorDigest []byte
	StatementVersion    string
	DeclaredAt          time.Time
	Purpose             string
}

// LoginAttemptBucket is one persistent brute-force protection bucket.
type LoginAttemptBucket struct {
	Kind             LoginBucketKind
	KeyDigest        []byte
	DigestKeyVersion string
	WindowStartedAt  time.Time
	FailureCount     int64
	BlockedUntil     time.Time
	UpdatedAt        time.Time
}

// RecoveryAccountReview binds backup facts to an explicit operator decision.
type RecoveryAccountReview struct {
	RestorationOperationID *string
	RecoveryRunID          string
	UserID                 string
	BackupRole             Role
	BackupStatus           UserStatus
	BackupDeletedAt        *time.Time
	Decision               RecoveryDecision
	EvidenceRef            *string
	ReviewedBy             *string
	ReviewedAt             *time.Time
	CredentialResetAt      *time.Time
}

// RecoveryResetMetadata is the internal immutable 0010 approval projection.
type RecoveryResetMetadata struct {
	RunID, InstallationID, EvidenceRef, EvidenceDigest, ActorID string
	Role                                                        Role
	Status                                                      UserStatus
}
