package identity

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"han/internal/audit"
	"han/internal/platform"
	"han/internal/security"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

// RegistrationRequest is the registration HTTP payload plus trusted source evidence.
type RegistrationRequest struct {
	Username             string `json:"username"`
	Email                string `json:"email"`
	Phone                string `json:"phone"`
	Password             string `json:"password"`
	PrivacyNoticeVersion string `json:"privacy_notice_version"`
	PrivacyAcknowledged  bool   `json:"privacy_acknowledged"`
	SourceKind           string `json:"-"`
	SourceFingerprint    string `json:"-"`
}

// RegisteredUser is the safe registration result returned to transports.
type RegisteredUser struct {
	ID        string
	Username  string
	Role      Role
	Status    UserStatus
	CreatedAt time.Time
}

// RegistrationTransactor begins caller-owned PostgreSQL transactions.
type RegistrationTransactor interface {
	Begin(context.Context) (pgx.Tx, error)
}

// RegistrationRepository persists the identity-owned registration aggregate.
type RegistrationRepository interface {
	InsertUser(context.Context, pgx.Tx, User) error
	InsertIdentifier(context.Context, pgx.Tx, IdentityIdentifier) error
	InsertPrivacyAcknowledgement(context.Context, pgx.Tx, PrivacyAcknowledgement) error
}

// RegistrationNoticeReader reads the current notice in the registration transaction.
type RegistrationNoticeReader interface {
	CurrentNotice(context.Context, pgx.Tx) (PrivacyNotice, error)
}

// RegistrationProcessingGate controls whether registration may run.
type RegistrationProcessingGate interface {
	RequireOpen(context.Context) error
}

// RegistrationPasswordPolicy validates the confirmed password rules.
type RegistrationPasswordPolicy interface {
	Validate(string) error
}

// RegistrationPasswordHasher creates the persisted password digest.
type RegistrationPasswordHasher interface {
	Hash(string) (string, error)
}

// RegistrationAuditAppender appends the registration event in the same transaction.
type RegistrationAuditAppender interface {
	Append(context.Context, pgx.Tx, audit.Event) error
}

// RegistrationIDGenerator creates opaque database identifiers.
type RegistrationIDGenerator interface {
	NewID() (string, error)
}

// RegistrationClock supplies registration timestamps.
type RegistrationClock interface {
	Now() time.Time
}

// LoginTransactor begins caller-owned login transactions.
type LoginTransactor interface {
	Begin(context.Context) (pgx.Tx, error)
}

// LoginRepository reads credential state and applies successful rehashes.
type LoginRepository interface {
	FindUserByUsername(context.Context, pgx.Tx, string) (User, error)
	FindLatestTemporaryCredential(context.Context, pgx.Tx, string) (TemporaryCredential, error)
	UpdatePermanentPasswordDigest(context.Context, pgx.Tx, string, string) error
	UpdateTemporaryCredentialDigest(context.Context, pgx.Tx, string, string) error
}

// LoginPasswordHasher verifies and upgrades self-describing credential digests.
type LoginPasswordHasher interface {
	Hash(string) (string, error)
	Verify(string, string) (matched bool, needsRehash bool, err error)
}

// LoginSessionStore creates or rotates a session in the caller-owned transaction.
type LoginSessionStore interface {
	Create(context.Context, session.Executor, session.CreateInput) (session.Credentials, error)
	Rotate(
		context.Context,
		pgx.Tx,
		string,
		time.Time,
		string,
		session.CreateInput,
	) (session.Credentials, error)
}

// LoginSessionIssuer supplies caller-approved randomness and explicit session deadlines.
type LoginSessionIssuer interface {
	Issue(User, session.CredentialKind, *string) (session.CreateInput, error)
}

// LoginAbuseProtector persists and evaluates login attempt buckets.
type LoginAbuseProtector interface {
	Check(context.Context, pgx.Tx, LoginAttempt) (LoginAbuseDecision, error)
	RecordFailure(context.Context, pgx.Tx, LoginAttempt) error
	RecordSuccess(context.Context, pgx.Tx, LoginAttempt) error
}

// LoginRecoveryMode controls the recovery-bootstrap-only credential path.
type LoginRecoveryMode interface {
	RecoveryBootstrapAllowed(context.Context) (bool, error)
}

// LoginAuditAppender appends login evidence in the caller-owned transaction.
type LoginAuditAppender interface {
	Append(context.Context, pgx.Tx, audit.Event) error
}

// LoginIDGenerator creates opaque login audit identifiers.
type LoginIDGenerator interface {
	NewID() (string, error)
}

// LoginClock supplies login timestamps.
type LoginClock interface {
	Now() time.Time
}

// PasswordChangeRepository owns the authoritative credential transition.
type PasswordChangeRepository interface {
	FindUserForPasswordChange(context.Context, pgx.Tx, string) (User, error)
	FindPasswordResetOperationForPasswordChange(
		context.Context,
		pgx.Tx,
		string,
		string,
		string,
	) (PasswordResetOperation, error)
	FindLatestTemporaryCredential(context.Context, pgx.Tx, string) (TemporaryCredential, error)
	SetPermanentPassword(
		context.Context,
		pgx.Tx,
		string,
		string,
		CredentialMode,
		int64,
		int64,
		*string,
		*string,
		time.Time,
	) error
	SettlePasswordResetSuccessor(
		context.Context,
		pgx.Tx,
		PasswordResetOperation,
		TemporaryCredential,
		int64,
		time.Time,
	) error
	ConsumeTemporaryCredential(context.Context, pgx.Tx, string, time.Time) error
}

// PasswordChangeOnboardingRepository finalizes both administrator onboarding Gates.
type PasswordChangeOnboardingRepository interface {
	CurrentNotice(context.Context, pgx.Tx) (PrivacyNotice, error)
	CompleteAdministratorOnboarding(
		context.Context,
		pgx.Tx,
		string,
		string,
		time.Time,
	) error
}

// PasswordChangeSessionStore rotates the current session and revokes the rest.
type PasswordChangeSessionStore interface {
	Revoke(context.Context, session.Executor, string, time.Time, string) error
	Rotate(
		context.Context,
		pgx.Tx,
		string,
		time.Time,
		string,
		session.CreateInput,
	) (session.Credentials, error)
	RevokeOtherSessions(context.Context, pgx.Tx, string, string, time.Time, string) error
}

// PasswordChangeAuditAppender appends success evidence in the same transaction.
type PasswordChangeAuditAppender interface {
	Append(context.Context, pgx.Tx, audit.Event) error
}

const passwordChangeAuditAction = "identity.password.change"

// PasswordChangeDependencies are the explicit protected onboarding collaborators.
type PasswordChangeDependencies struct {
	// RecoveryAuthority is the same identity executor used by the maintenance surface.
	RecoveryAuthority session.MaintenanceSessionAuthority
	Transactions      LoginTransactor
	Repository        PasswordChangeRepository
	PasswordResetGate session.PasswordResetGate
	Passwords         LoginPasswordHasher
	Policy            RegistrationPasswordPolicy
	Sessions          PasswordChangeSessionStore
	SessionIssuer     LoginSessionIssuer
	Audit             PasswordChangeAuditAppender
	IDs               LoginIDGenerator
	Clock             LoginClock
}

// PasswordChangeRequest contains trusted principal/session evidence and secret inputs.
type PasswordChangeRequest struct {
	Principal           session.Principal
	CurrentSessionToken string
	CurrentPassword     string
	NewPassword         string
	TraceID             string
}

// PasswordChangeResult is the safe user and rotated-session result.
type PasswordChangeResult struct {
	User                   User
	MustAcknowledgePrivacy bool
	SessionToken           string
	CSRFToken              string
}

// PasswordChangeFailureAuditRequest is the safe Handler-to-Service failure fact.
type PasswordChangeFailureAuditRequest struct {
	Principal session.Principal
	ErrorCode platform.Code
	TraceID   string
}

// LoginDependencies are the explicit login collaborators and approved policy inputs.
type LoginDependencies struct {
	Transactions         LoginTransactor
	Repository           LoginRepository
	PasswordResetGate    session.PasswordResetGate
	Passwords            LoginPasswordHasher
	Sessions             LoginSessionStore
	SessionIssuer        LoginSessionIssuer
	Abuse                LoginAbuseProtector
	RecoveryMode         LoginRecoveryMode
	Audit                LoginAuditAppender
	IDs                  LoginIDGenerator
	Clock                LoginClock
	FakeCredentialDigest string
	FakeCredentialProbe  string
}

// RegistrationDependencies are the explicit registration collaborators.
type RegistrationDependencies struct {
	Transactions   RegistrationTransactor
	Repository     RegistrationRepository
	Notices        RegistrationNoticeReader
	Gate           RegistrationProcessingGate
	PasswordPolicy RegistrationPasswordPolicy
	PasswordHasher RegistrationPasswordHasher
	Audit          RegistrationAuditAppender
	IDs            RegistrationIDGenerator
	Clock          RegistrationClock
}

// Service coordinates identity operations.
type Service struct {
	dependencies               RegistrationDependencies
	loginDependencies          LoginDependencies
	passwordChangeDependencies PasswordChangeDependencies
}

// NewService creates an identity service from explicit dependencies.
func NewService(dependencies RegistrationDependencies) (*Service, error) {
	if dependencies.Transactions == nil || dependencies.Repository == nil ||
		dependencies.Notices == nil || dependencies.Gate == nil ||
		dependencies.PasswordPolicy == nil || dependencies.PasswordHasher == nil ||
		dependencies.Audit == nil || dependencies.IDs == nil || dependencies.Clock == nil {
		return nil, errors.New("registration dependencies are incomplete")
	}
	return &Service{dependencies: dependencies}, nil
}

// NewLoginService creates a login service from explicit persistence and policy inputs.
func NewLoginService(dependencies LoginDependencies) (*Service, error) {
	fakeCredentialProbe := dependencies.FakeCredentialProbe
	dependencies.FakeCredentialProbe = ""
	if dependencies.Transactions == nil || dependencies.Repository == nil ||
		nilIdentityDependency(dependencies.PasswordResetGate) ||
		dependencies.Passwords == nil || dependencies.Sessions == nil ||
		dependencies.SessionIssuer == nil || dependencies.Abuse == nil ||
		dependencies.RecoveryMode == nil || dependencies.Audit == nil ||
		dependencies.FakeCredentialDigest == "" || fakeCredentialProbe == "" {
		return nil, errors.New("login dependencies are incomplete")
	}
	if executor, ok := dependencies.RecoveryMode.(*recoveryIdentityExecution); ok {
		if executor == nil || executor.d.store == nil || nilIdentityDependency(executor.d.runtime) || dependencies.Transactions != executor.d.store.pool {
			return nil, errors.New("maintenance login must share the recovery Store pool")
		}
	}
	matched, needsRehash, err := dependencies.Passwords.Verify(
		fakeCredentialProbe,
		dependencies.FakeCredentialDigest,
	)
	fakeCredentialProbe = ""
	if err != nil || !matched || needsRehash {
		return nil, errors.New("fake login credential is incompatible")
	}
	return &Service{loginDependencies: dependencies}, nil
}

// NewPasswordChangeService creates the protected password transition service.
func NewPasswordChangeService(dependencies PasswordChangeDependencies) (*Service, error) {
	if dependencies.Transactions == nil || dependencies.Repository == nil ||
		nilIdentityDependency(dependencies.PasswordResetGate) ||
		dependencies.Passwords == nil || dependencies.Policy == nil ||
		dependencies.Sessions == nil || dependencies.SessionIssuer == nil ||
		dependencies.Audit == nil || dependencies.IDs == nil ||
		dependencies.Clock == nil {
		return nil, errors.New("password change dependencies are incomplete")
	}
	if dependencies.RecoveryAuthority != nil {
		executor, ok := dependencies.RecoveryAuthority.(*recoveryIdentityExecution)
		if !ok || executor == nil || executor.d.store == nil || nilIdentityDependency(executor.d.runtime) || dependencies.Transactions != executor.d.store.pool {
			return nil, errors.New("recovery change must share the identity Store pool")
		}
	}
	return &Service{passwordChangeDependencies: dependencies}, nil
}

// NewCanonicalDeleteService creates the single daily/recovery deletion coordinator.
func NewCanonicalDeleteService(
	dependencies CanonicalDeleteDependencies,
) (*CanonicalDeleteCoordinator, error) {
	if nilIdentityDependency(dependencies.Transactions) ||
		nilIdentityDependency(dependencies.Repository) ||
		nilIdentityDependency(dependencies.Articles) ||
		nilIdentityDependency(dependencies.Audit) ||
		nilIdentityDependency(dependencies.IDs) ||
		nilIdentityDependency(dependencies.Clock) {
		return nil, errors.New("canonical delete dependencies are incomplete")
	}
	return &CanonicalDeleteCoordinator{dependencies: dependencies}, nil
}

func nilIdentityDependency(dependency any) bool {
	if dependency == nil {
		return true
	}
	value := reflect.ValueOf(dependency)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// ChangePassword replaces the credential bound to the current session.
func (s *Service) ChangePassword(
	ctx context.Context,
	request PasswordChangeRequest,
) (PasswordChangeResult, error) {
	principal, err := s.validatePasswordChangeRequest(ctx, request)
	if err != nil {
		return PasswordChangeResult{}, err
	}
	execution, err := newPasswordChangeExecution(s, ctx, request, principal)
	if err != nil {
		return PasswordChangeResult{}, err
	}
	defer execution.rollback()
	return execution.run()
}

type passwordChangeExecution struct {
	service                *Service
	ctx                    context.Context
	request                PasswordChangeRequest
	principal              session.Principal
	dependencies           PasswordChangeDependencies
	tx                     pgx.Tx
	closed                 bool
	user                   User
	now                    time.Time
	credential             passwordChangeCredentialAuthority
	newDigest              string
	successorGeneration    int64
	sourceOperationID      *string
	sourceCredentialID     *string
	mustAcknowledgePrivacy bool
	credentials            session.Credentials
}

func (s *Service) validatePasswordChangeRequest(
	ctx context.Context,
	request PasswordChangeRequest,
) (session.Principal, error) {
	if s == nil {
		return session.Principal{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("password change service is not initialized"),
		)
	}
	principal := request.Principal
	if principal.UserID == "" || principal.Status != string(UserStatusActive) {
		return session.Principal{}, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("active password change principal is required"),
		)
	}
	if !principal.MustChangePassword && !principal.IsInitialAdmin {
		publicErr := platform.NewPublicError(
			platform.CodeAuthForbidden,
			errors.New("principal is not allowed to change password"),
		)
		err := s.AuditPasswordChangeFailure(ctx, PasswordChangeFailureAuditRequest{
			Principal: principal,
			ErrorCode: platform.CodeAuthForbidden,
			TraceID:   request.TraceID,
		})
		if err != nil {
			return session.Principal{}, passwordChangePersistenceError(
				"append forbidden password change audit",
				err,
			)
		}
		return session.Principal{}, publicErr
	}
	if request.CurrentSessionToken == "" || request.TraceID == "" {
		return session.Principal{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("password change request evidence is incomplete"),
		)
	}
	return principal, nil
}

func newPasswordChangeExecution(
	service *Service,
	ctx context.Context,
	request PasswordChangeRequest,
	principal session.Principal,
) (*passwordChangeExecution, error) {
	dependencies := service.passwordChangeDependencies
	tx, err := dependencies.Transactions.Begin(ctx)
	if err != nil {
		return nil, passwordChangePersistenceError("begin password change transaction", err)
	}
	return &passwordChangeExecution{
		service: service, ctx: ctx, request: request, principal: principal,
		dependencies: dependencies, tx: tx,
	}, nil
}

func (e *passwordChangeExecution) run() (PasswordChangeResult, error) {
	if err := e.loadAuthority(); err != nil {
		return PasswordChangeResult{}, err
	}
	if err := e.loadAndVerifyCredential(); err != nil {
		return PasswordChangeResult{}, err
	}
	if err := e.persistPermanentCredential(); err != nil {
		return PasswordChangeResult{}, err
	}
	if err := e.finalizeOnboarding(); err != nil {
		return PasswordChangeResult{}, err
	}
	if err := e.rotateSessions(); err != nil {
		return PasswordChangeResult{}, err
	}
	return e.appendAuditAndCommit()
}

func (e *passwordChangeExecution) rollback() {
	if !e.closed {
		_ = e.tx.Rollback(e.ctx)
	}
}

func (e *passwordChangeExecution) loadAuthority() error {
	user, err := e.dependencies.Repository.FindUserForPasswordChange(
		e.ctx,
		e.tx,
		e.principal.UserID,
	)
	if err != nil {
		return e.failAfterRollback(passwordChangePersistenceError(
			"read password change user",
			err,
		))
	}
	e.user = user
	if user.RecoveryRequired {
		return e.failInTransaction(platform.NewPublicError(platform.CodeAuthSessionRequired, session.ErrSessionRequired))
	}
	if err := e.dependencies.PasswordResetGate.CheckLockedUser(e.ctx, e.tx, user.ID); err != nil {
		if errors.Is(err, session.ErrPasswordResetGateClosed) {
			return e.failInTransaction(platform.NewPublicError(
				platform.CodeAuthSessionRequired,
				err,
			))
		}
		return e.failAfterRollback(passwordChangePersistenceError(
			"check password reset Gate",
			err,
		))
	}
	if passwordChangeRequired(user.CredentialMode) || user.IsInitialAdmin {
		e.now = e.dependencies.Clock.Now().UTC()
		return nil
	}
	if !e.principal.MustChangePassword && !e.principal.IsInitialAdmin {
		return e.failInTransaction(platform.NewPublicError(
			platform.CodeAuthForbidden,
			errors.New("authoritative user is not allowed to change password"),
		))
	}
	e.now = e.dependencies.Clock.Now().UTC()
	_, err = e.service.rejectStalePasswordChange(
		e.ctx, e.tx, e.dependencies, e.request, e.user, e.now, &e.closed,
	)
	return err
}

func (e *passwordChangeExecution) loadAndVerifyCredential() error {
	credential, bindingValid, err := passwordChangeCredential(
		e.ctx,
		e.tx,
		e.dependencies.Repository,
		e.user,
		e.principal,
		e.now,
	)
	if err != nil {
		return e.failAfterRollback(passwordChangePersistenceError(
			"read password change credential",
			err,
		))
	}
	if !bindingValid {
		_, err = e.service.rejectStalePasswordChange(
			e.ctx, e.tx, e.dependencies, e.request, e.user, e.now, &e.closed,
		)
		return err
	}
	e.credential = credential
	if err := e.verifyCurrentPassword(); err != nil {
		return err
	}
	return e.hashReplacementPassword()
}

func (e *passwordChangeExecution) verifyCurrentPassword() error {
	matched, _, err := e.dependencies.Passwords.Verify(
		e.request.CurrentPassword,
		e.credential.digest,
	)
	if err != nil {
		return e.failInTransaction(platform.NewPublicError(
			platform.CodePlatformInternalError,
			fmt.Errorf("verify current password: %w", err),
		))
	}
	if !matched {
		return e.failInTransaction(passwordChangeValidationError(
			"current_password",
			"current_password_mismatch",
			errors.New("current password does not match"),
		))
	}
	return nil
}

func (e *passwordChangeExecution) hashReplacementPassword() error {
	if err := e.dependencies.Policy.Validate(e.request.NewPassword); err != nil {
		return e.failInTransaction(newPasswordPolicyError(err))
	}
	newDigest, err := e.dependencies.Passwords.Hash(e.request.NewPassword)
	if err != nil {
		return e.failInTransaction(platform.NewPublicError(
			platform.CodePlatformInternalError,
			fmt.Errorf("hash replacement password: %w", err),
		))
	}
	e.newDigest = newDigest
	return nil
}

func (e *passwordChangeExecution) persistPermanentCredential() error {
	e.successorGeneration = e.user.AuthGeneration + 1
	e.sourceOperationID = e.user.PermanentPasswordSourceOperationID
	e.sourceCredentialID = e.user.PermanentPasswordSourceCredentialID
	if err := e.settleCredentialLineage(); err != nil {
		return err
	}
	if err := e.dependencies.Repository.SetPermanentPassword(
		e.ctx,
		e.tx,
		e.user.ID,
		e.newDigest,
		CredentialModePermanent,
		e.user.AuthGeneration,
		e.successorGeneration,
		e.sourceOperationID,
		e.sourceCredentialID,
		e.now,
	); err != nil {
		return e.failAfterRollback(passwordChangePersistenceError(
			"persist replacement password",
			err,
		))
	}
	e.applyPermanentCredential()
	return nil
}

func (e *passwordChangeExecution) settleCredentialLineage() error {
	if e.credential.operation != nil && e.credential.temporary != nil {
		e.sourceOperationID = &e.credential.operation.ID
		e.sourceCredentialID = &e.credential.temporary.ID
		err := e.dependencies.Repository.SettlePasswordResetSuccessor(
			e.ctx,
			e.tx,
			*e.credential.operation,
			*e.credential.temporary,
			e.successorGeneration,
			e.now,
		)
		if err != nil {
			return e.failAfterRollback(passwordChangePersistenceError(
				"settle password reset successor",
				err,
			))
		}
		return nil
	}
	if e.credential.temporary == nil {
		return nil
	}
	err := e.dependencies.Repository.ConsumeTemporaryCredential(
		e.ctx,
		e.tx,
		e.credential.temporary.ID,
		e.now,
	)
	if err != nil {
		return e.failAfterRollback(passwordChangePersistenceError(
			"consume temporary password credential",
			err,
		))
	}
	return nil
}

func (e *passwordChangeExecution) applyPermanentCredential() {
	e.user.PermanentPasswordDigest = &e.newDigest
	e.user.AuthGeneration = e.successorGeneration
	e.user.PermanentPasswordGeneration = &e.successorGeneration
	e.user.PermanentPasswordSourceOperationID = e.sourceOperationID
	e.user.PermanentPasswordSourceCredentialID = e.sourceCredentialID
	e.user.CredentialMode = CredentialModePermanent
	e.user.UpdatedAt = e.now
}

func (e *passwordChangeExecution) finalizeOnboarding() error {
	if e.user.Role != RoleAdmin || e.user.IsInitialAdmin ||
		e.user.OnboardingCompletedAt != nil {
		return nil
	}
	repository, ok := e.dependencies.Repository.(PasswordChangeOnboardingRepository)
	if !ok {
		return e.failAfterRollback(passwordChangePersistenceError(
			"finalize administrator onboarding",
			errors.New("administrator onboarding repository is unavailable"),
		))
	}
	notice, err := repository.CurrentNotice(e.ctx, e.tx)
	if err != nil {
		return e.failAfterRollback(passwordChangePersistenceError(
			"read current administrator privacy notice",
			err,
		))
	}
	if administratorOnboardingComplete(e.user, notice.Version) {
		err = repository.CompleteAdministratorOnboarding(
			e.ctx,
			e.tx,
			e.user.ID,
			notice.Version,
			e.now,
		)
		if err != nil {
			return e.failAfterRollback(passwordChangePersistenceError(
				"complete administrator onboarding",
				err,
			))
		}
		e.user.OnboardingCompletedAt = &e.now
	}
	e.mustAcknowledgePrivacy = e.user.OnboardingCompletedAt == nil
	return nil
}

func (e *passwordChangeExecution) rotateSessions() error {
	replacement, err := e.dependencies.SessionIssuer.Issue(
		e.user,
		session.CredentialPermanent,
		nil,
	)
	if err != nil {
		return e.failAfterRollback(platform.NewPublicError(
			platform.CodePlatformInternalError,
			fmt.Errorf("issue replacement password session: %w", err),
		))
	}
	replacement.RotatedFromID = &e.principal.SessionID
	e.credentials, err = e.dependencies.Sessions.Rotate(
		e.ctx,
		e.tx,
		e.request.CurrentSessionToken,
		e.now,
		"password_changed",
		replacement,
	)
	if err != nil {
		return e.failAfterRollback(passwordChangePersistenceError(
			"rotate password change session",
			err,
		))
	}
	err = e.dependencies.Sessions.RevokeOtherSessions(
		e.ctx,
		e.tx,
		e.user.ID,
		replacement.ID,
		e.now,
		"password_changed",
	)
	if err != nil {
		return e.failAfterRollback(passwordChangePersistenceError(
			"revoke other password sessions",
			err,
		))
	}
	return nil
}

func (e *passwordChangeExecution) appendAuditAndCommit() (PasswordChangeResult, error) {
	if err := appendPasswordChangeAudit(
		e.ctx,
		e.tx,
		e.dependencies,
		e.principal,
		e.user.ID,
		"succeeded",
		"",
		e.request.TraceID,
		e.now,
	); err != nil {
		return PasswordChangeResult{}, passwordChangePersistenceError(
			"append password change audit",
			err,
		)
	}
	if err := e.tx.Commit(e.ctx); err != nil {
		e.closed = true
		return PasswordChangeResult{}, passwordChangePersistenceError(
			"commit password change",
			err,
		)
	}
	e.closed = true
	return PasswordChangeResult{
		User: e.user, MustAcknowledgePrivacy: e.mustAcknowledgePrivacy,
		SessionToken: e.credentials.Token, CSRFToken: e.credentials.CSRFToken,
	}, nil
}

func (e *passwordChangeExecution) failInTransaction(publicErr error) error {
	now := e.dependencies.Clock.Now().UTC()
	if err := appendPasswordChangeAudit(
		e.ctx,
		e.tx,
		e.dependencies,
		e.principal,
		e.user.ID,
		"failed",
		platform.PublicCode(publicErr),
		e.request.TraceID,
		now,
	); err != nil {
		return passwordChangePersistenceError("append password change failure audit", err)
	}
	if err := e.tx.Commit(e.ctx); err != nil {
		e.closed = true
		return passwordChangePersistenceError("commit password change failure audit", err)
	}
	e.closed = true
	return publicErr
}

func (e *passwordChangeExecution) failAfterRollback(publicErr error) error {
	if err := e.tx.Rollback(e.ctx); err != nil {
		e.closed = true
		return passwordChangePersistenceError("rollback failed password change", err)
	}
	e.closed = true
	err := e.service.AuditPasswordChangeFailure(e.ctx, PasswordChangeFailureAuditRequest{
		Principal: e.principal,
		ErrorCode: platform.PublicCode(publicErr),
		TraceID:   e.request.TraceID,
	})
	if err != nil {
		return passwordChangePersistenceError(
			"append rolled back password change audit",
			err,
		)
	}
	return publicErr
}

// AuditPasswordChangeFailure records one pre-Service stable rejection.
func (s *Service) AuditPasswordChangeFailure(
	ctx context.Context,
	request PasswordChangeFailureAuditRequest,
) error {
	if s == nil {
		return errors.New("password change service is not initialized")
	}
	dependencies := s.passwordChangeDependencies
	if request.Principal.UserID == "" || request.TraceID == "" || request.ErrorCode == "" {
		return errors.New("password change failure audit request is incomplete")
	}
	tx, err := dependencies.Transactions.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin password change failure audit: %w", err)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := appendPasswordChangeAudit(
		ctx,
		tx,
		dependencies,
		request.Principal,
		request.Principal.UserID,
		"failed",
		request.ErrorCode,
		request.TraceID,
		dependencies.Clock.Now().UTC(),
	); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return fmt.Errorf("commit password change failure audit: %w", err)
	}
	closed = true
	return nil
}

func (s *Service) rejectStalePasswordChange(
	ctx context.Context,
	tx pgx.Tx,
	dependencies PasswordChangeDependencies,
	request PasswordChangeRequest,
	user User,
	now time.Time,
	closed *bool,
) (PasswordChangeResult, error) {
	publicErr := platform.NewPublicError(
		platform.CodeAuthSessionRequired,
		errors.New("session credential binding is no longer current"),
	)
	if err := dependencies.Sessions.Revoke(
		ctx,
		tx,
		request.CurrentSessionToken,
		now,
		"credential_binding_invalid",
	); err != nil {
		persistenceErr := passwordChangePersistenceError(
			"revoke stale password change session",
			err,
		)
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			*closed = true
			return PasswordChangeResult{}, passwordChangePersistenceError(
				"rollback stale password change session",
				rollbackErr,
			)
		}
		*closed = true
		if auditErr := s.AuditPasswordChangeFailure(ctx, PasswordChangeFailureAuditRequest{
			Principal: request.Principal,
			ErrorCode: platform.PublicCode(persistenceErr),
			TraceID:   request.TraceID,
		}); auditErr != nil {
			return PasswordChangeResult{}, passwordChangePersistenceError(
				"append stale password change failure audit",
				auditErr,
			)
		}
		return PasswordChangeResult{}, persistenceErr
	}
	if err := appendPasswordChangeAudit(
		ctx,
		tx,
		dependencies,
		request.Principal,
		user.ID,
		"failed",
		platform.CodeAuthSessionRequired,
		request.TraceID,
		now,
	); err != nil {
		return PasswordChangeResult{}, passwordChangePersistenceError(
			"append stale password change audit",
			err,
		)
	}
	if err := tx.Commit(ctx); err != nil {
		*closed = true
		return PasswordChangeResult{}, passwordChangePersistenceError(
			"commit stale password change session revocation",
			err,
		)
	}
	*closed = true
	return PasswordChangeResult{}, publicErr
}

func appendPasswordChangeAudit(
	ctx context.Context,
	tx pgx.Tx,
	dependencies PasswordChangeDependencies,
	principal session.Principal,
	targetUserID string,
	outcome string,
	errorCode platform.Code,
	traceID string,
	now time.Time,
) error {
	auditID, err := dependencies.IDs.NewID()
	if err != nil {
		return fmt.Errorf("create password change audit identifier: %w", err)
	}
	actorID := principal.UserID
	role := principal.Role
	event := audit.Event{
		ID: auditID, OccurredAt: now, ActorKind: audit.ActorUser,
		ActorUserID: &actorID, ActorRole: &role,
		Action: passwordChangeAuditAction, TargetType: "user",
		TargetID: targetUserID, Outcome: outcome, TraceID: traceID,
	}
	if errorCode != "" {
		code := string(errorCode)
		event.ErrorCode = &code
	}
	return dependencies.Audit.Append(ctx, tx, event)
}

type passwordChangeCredentialAuthority struct {
	digest    string
	temporary *TemporaryCredential
	operation *PasswordResetOperation
}

func passwordChangeCredential(
	ctx context.Context,
	tx pgx.Tx,
	repository PasswordChangeRepository,
	user User,
	principal session.Principal,
	now time.Time,
) (passwordChangeCredentialAuthority, bool, error) {
	if user.ID != principal.UserID || user.Status != UserStatusActive {
		return passwordChangeCredentialAuthority{}, false, nil
	}
	switch user.CredentialMode {
	case CredentialModePermanent:
		return passwordChangeStoredCredential(
			principal,
			session.CredentialPermanent,
			user.PermanentPasswordDigest,
		)
	case CredentialModeBootstrap:
		return passwordChangeStoredCredential(
			principal,
			session.CredentialBootstrap,
			user.PermanentPasswordDigest,
		)
	case CredentialModeRecoveryBootstrap:
		return passwordChangeStoredCredential(
			principal,
			session.CredentialRecoveryBootstrap,
			user.PermanentPasswordDigest,
		)
	case CredentialModeTemporary:
		return passwordChangeTemporaryCredential(
			ctx,
			tx,
			repository,
			user,
			principal,
			now,
		)
	default:
		return passwordChangeCredentialAuthority{}, false, nil
	}
}

func passwordChangeStoredCredential(
	principal session.Principal,
	expected session.CredentialKind,
	digest *string,
) (passwordChangeCredentialAuthority, bool, error) {
	if principal.AuthenticatedCredential != expected ||
		principal.AuthenticatedCredentialID != nil ||
		digest == nil {
		return passwordChangeCredentialAuthority{}, false, nil
	}
	return passwordChangeCredentialAuthority{digest: *digest}, true, nil
}

func passwordChangeTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	repository PasswordChangeRepository,
	user User,
	principal session.Principal,
	now time.Time,
) (passwordChangeCredentialAuthority, bool, error) {
	if principal.AuthenticatedCredential != session.CredentialTemporary ||
		principal.AuthenticatedCredentialID == nil {
		return passwordChangeCredentialAuthority{}, false, nil
	}
	operation, valid, err := passwordChangeResetOperation(
		ctx, tx, repository, user, *principal.AuthenticatedCredentialID,
	)
	if err != nil || !valid {
		return passwordChangeCredentialAuthority{}, false, err
	}
	credential, err := repository.FindLatestTemporaryCredential(ctx, tx, user.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return passwordChangeCredentialAuthority{}, false, nil
	}
	if err != nil {
		return passwordChangeCredentialAuthority{}, false, err
	}
	if !passwordChangeTemporaryBindingValid(
		credential, *principal.AuthenticatedCredentialID, operation, user, now,
	) {
		return passwordChangeCredentialAuthority{}, false, nil
	}
	return passwordChangeCredentialAuthority{
		digest: credential.SecretDigest, temporary: &credential, operation: operation,
	}, true, nil
}

func passwordChangeResetOperation(
	ctx context.Context,
	tx pgx.Tx,
	repository PasswordChangeRepository,
	user User,
	credentialID string,
) (*PasswordResetOperation, bool, error) {
	if user.PasswordResetCurrentOperationID == nil {
		return nil, true, nil
	}
	current, err := repository.FindPasswordResetOperationForPasswordChange(
		ctx, tx, user.ID, *user.PasswordResetCurrentOperationID, credentialID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !passwordChangeResetOperationValid(current, user) {
		return nil, false, nil
	}
	return &current, true, nil
}

func passwordChangeTemporaryBindingValid(
	credential TemporaryCredential,
	credentialID string,
	operation *PasswordResetOperation,
	user User,
	now time.Time,
) bool {
	if credential.ID != credentialID ||
		!TemporaryCredentialAuthenticatable(credential, credential.Generation, now) {
		return false
	}
	if operation == nil {
		return credential.Purpose == TemporaryCredentialNewAdministrator &&
			credential.SourceOperationID == nil &&
			credential.AuthGeneration == user.AuthGeneration
	}
	return credential.Purpose == TemporaryCredentialPasswordReset &&
		credential.SourceOperationID != nil &&
		*credential.SourceOperationID == operation.ID &&
		credential.AuthGeneration == operation.TargetAuthGeneration
}

func passwordChangeResetOperationValid(
	operation PasswordResetOperation,
	user User,
) bool {
	return operation.ID != "" &&
		operation.UserID == user.ID &&
		operation.TemporaryCredentialID != "" &&
		operation.State == PasswordResetOperationSucceeded &&
		operation.DeliveryState == PasswordResetDeliveryAccepted &&
		operation.SettlementState == PasswordResetSettlementTemporaryActive &&
		operation.SettledAuthGeneration != nil &&
		*operation.SettledAuthGeneration == operation.TargetAuthGeneration &&
		operation.SuccessorAuthGeneration == nil &&
		operation.TargetAuthGeneration == user.AuthGeneration
}

func passwordChangeRequired(mode CredentialMode) bool {
	switch mode {
	case CredentialModeBootstrap, CredentialModeTemporary, CredentialModeRecoveryBootstrap:
		return true
	default:
		return false
	}
}

func passwordChangeValidationError(field, code string, cause error) error {
	return platform.NewPublicError(
		platform.CodeIdentityValidationFailed,
		cause,
		platform.FieldError{Field: field, Code: code},
	)
}

func newPasswordPolicyError(err error) error {
	switch {
	case errors.Is(err, security.ErrPasswordTooShort):
		return passwordChangeValidationError("new_password", "password_too_short", err)
	case errors.Is(err, security.ErrPasswordRequiresLetter):
		return passwordChangeValidationError("new_password", "password_requires_letter", err)
	case errors.Is(err, security.ErrPasswordRequiresDigit):
		return passwordChangeValidationError("new_password", "password_requires_digit", err)
	default:
		return platform.NewPublicError(platform.CodePlatformInternalError, err)
	}
}

func passwordChangePersistenceError(operation string, err error) error {
	return platform.NewPublicError(
		platform.CodePlatformDatabaseUnavailable,
		fmt.Errorf("%s: %w", operation, err),
	)
}

// Register creates one member identity.
func (s *Service) Register(
	ctx context.Context,
	request RegistrationRequest,
) (RegisteredUser, error) {
	if err := s.dependencies.Gate.RequireOpen(ctx); err != nil {
		return RegisteredUser{}, processingGateError(err)
	}

	normalized, err := validateRegistrationRequest(request, s.dependencies.PasswordPolicy)
	if err != nil {
		return RegisteredUser{}, err
	}

	tx, err := s.dependencies.Transactions.Begin(ctx)
	if err != nil {
		beginErr := markRegistrationPersistenceFailure(
			persistenceError("begin registration transaction", err),
		)
		return RegisteredUser{}, s.appendRegistrationFailureAudit(ctx, "", beginErr)
	}
	transactionClosed := false
	defer func() {
		if !transactionClosed {
			_ = tx.Rollback(ctx)
		}
	}()

	registered, err := s.registerInTransaction(ctx, tx, normalized)
	if err != nil {
		transactionClosed = true
		return RegisteredUser{}, s.finishFailedRegistration(ctx, tx, registered.ID, err, false)
	}
	if err := tx.Commit(ctx); err != nil {
		transactionClosed = true
		commitErr := markRegistrationPersistenceFailure(
			persistenceError("commit registration transaction", err),
		)
		rollbackConfirmed := errors.Is(err, pgx.ErrTxCommitRollback)
		return RegisteredUser{}, s.finishFailedRegistration(
			ctx,
			tx,
			registered.ID,
			commitErr,
			rollbackConfirmed,
		)
	}
	transactionClosed = true
	return registered, nil
}
