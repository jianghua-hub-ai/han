package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const (
	administratorCreateOperation  = "identity.administrator.create"
	administratorPrivacyAction    = "identity.privacy.acknowledge"
	administratorDeliveryAccepted = "accepted"
	administratorCreateOutcome    = "created"
	administratorCreateFailed     = "failed"
	administratorLocatorDomain    = "han.identity.administrator-locator.v1"
)

// AdministratorInformationNotice is the current declaration shown before creation.
type AdministratorInformationNotice struct {
	Version     string
	EffectiveAt time.Time
	Content     string
	ContentHash string
}

// CreateAdministratorRequest contains trusted actor evidence and the HTTP payload.
type CreateAdministratorRequest struct {
	Principal                         session.Principal
	IdempotencyKey                    string
	Username                          string `json:"username"`
	Email                             string `json:"email"`
	Phone                             string `json:"phone"`
	TemporaryPassword                 string `json:"temporary_password"`
	InformationProvisionNoticeVersion string `json:"information_provision_notice_version"`
	InformationProvisionDeclared      bool   `json:"information_provision_declared"`
	TraceID                           string `json:"-"`
}

// CreatedAdministrator is the minimum safe creation result.
type CreatedAdministrator struct {
	ID                           string
	Username                     string
	Role                         Role
	Status                       UserStatus
	MustChangePassword           bool
	MustAcknowledgePrivacy       bool
	TemporaryCredentialExpiresAt time.Time
	DeliveryStatus               string
}

// AdministratorPrivacyAcknowledgementRequest contains trusted session/source evidence.
type AdministratorPrivacyAcknowledgementRequest struct {
	Principal            session.Principal
	CurrentSessionToken  string
	PrivacyNoticeVersion string
	Acknowledged         bool
	SourceKind           string
	SourceFingerprint    string
	TraceID              string
}

// AdministratorOnboardingResult is the safe post-acknowledgement session view.
type AdministratorOnboardingResult struct {
	User                   User
	MustChangePassword     bool
	MustAcknowledgePrivacy bool
	SessionToken           string
	CSRFToken              string
}

// AdministratorInformationNoticeReader supplies the current declaration contract.
type AdministratorInformationNoticeReader interface {
	CurrentAdministratorInformationNotice(
		context.Context,
		pgx.Tx,
	) (AdministratorInformationNotice, error)
}

// AdministratorCreateRepository persists the creation aggregate.
type AdministratorCreateRepository interface {
	FindAdministratorCreationActor(context.Context, pgx.Tx, string) (User, error)
	InsertUser(context.Context, pgx.Tx, User) error
	InsertIdentifier(context.Context, pgx.Tx, IdentityIdentifier) error
	InsertTemporaryCredential(context.Context, pgx.Tx, TemporaryCredential) error
	InsertAdministratorInformationDeclaration(
		context.Context,
		pgx.Tx,
		AdministratorInformationDeclaration,
	) error
	ActivateTemporaryCredential(context.Context, pgx.Tx, string, time.Time) error
	FindCreatedAdministrator(
		context.Context,
		pgx.Tx,
		string,
	) (CreatedAdministrator, error)
}

// AdministratorPrivacyRepository owns the onboarding privacy transition.
type AdministratorPrivacyRepository interface {
	FindAdministratorForOnboarding(context.Context, pgx.Tx, string) (User, error)
	InsertPrivacyAcknowledgement(context.Context, pgx.Tx, PrivacyAcknowledgement) error
	SetAdministratorPrivacyAcknowledgement(
		context.Context,
		pgx.Tx,
		string,
		string,
		bool,
		time.Time,
	) error
}

// AdministratorIdempotencyStore is the scoped persistent state-machine seam.
type AdministratorIdempotencyStore interface {
	ResolveExisting(
		context.Context,
		idempotency.Transaction,
		idempotency.BeginRequest,
	) (idempotency.BeginResult, bool, error)
	Begin(
		context.Context,
		idempotency.Transaction,
		idempotency.BeginRequest,
	) (idempotency.BeginResult, error)
	Complete(context.Context, idempotency.Transaction, idempotency.Completion) error
	MarkUncertain(context.Context, idempotency.Transaction, idempotency.Transition) error
}

// AdministratorOnboardingSessionStore rotates the current session.
type AdministratorOnboardingSessionStore interface {
	Rotate(
		context.Context,
		pgx.Tx,
		string,
		time.Time,
		string,
		session.CreateInput,
	) (session.Credentials, error)
}

// AdministratorCreateDependencies are the explicit creation collaborators.
type AdministratorCreateDependencies struct {
	Transactions        RegistrationTransactor
	Repository          AdministratorCreateRepository
	InformationNotices  AdministratorInformationNoticeReader
	Gate                RegistrationProcessingGate
	PasswordPolicy      RegistrationPasswordPolicy
	PasswordHasher      TemporaryCredentialHasher
	Mailer              mailer.Mailer
	Idempotency         AdministratorIdempotencyStore
	IdempotencyKeys     idempotency.DigestKeyring
	IdempotencyLock     time.Duration
	IdempotencyTTL      time.Duration
	SMTPOperationBudget time.Duration
	Audit               RegistrationAuditAppender
	IDs                 RegistrationIDGenerator
	Clock               RegistrationClock
}

// AdministratorPrivacyDependencies are the explicit privacy-Gate collaborators.
type AdministratorPrivacyDependencies struct {
	Transactions  RegistrationTransactor
	Repository    AdministratorPrivacyRepository
	Notices       RegistrationNoticeReader
	Sessions      AdministratorOnboardingSessionStore
	SessionIssuer LoginSessionIssuer
	Audit         RegistrationAuditAppender
	IDs           RegistrationIDGenerator
	Clock         RegistrationClock
}

// AdministratorCreateCoordinator creates ordinary administrators.
type AdministratorCreateCoordinator struct {
	dependencies AdministratorCreateDependencies
}

// AdministratorPrivacyCoordinator completes the privacy onboarding Gate.
type AdministratorPrivacyCoordinator struct {
	dependencies AdministratorPrivacyDependencies
}

// NewAdministratorCreateService creates the initial-admin-only coordinator.
func NewAdministratorCreateService(
	dependencies AdministratorCreateDependencies,
) (*AdministratorCreateCoordinator, error) {
	activeKey := dependencies.IdempotencyKeys.Keys[dependencies.IdempotencyKeys.ActiveVersion]
	if dependencies.Transactions == nil || dependencies.Repository == nil ||
		dependencies.InformationNotices == nil || dependencies.Gate == nil ||
		dependencies.PasswordPolicy == nil || dependencies.PasswordHasher == nil ||
		dependencies.Mailer == nil || dependencies.Idempotency == nil ||
		strings.TrimSpace(dependencies.IdempotencyKeys.ActiveVersion) == "" ||
		len(activeKey) == 0 || dependencies.IdempotencyLock <= 0 ||
		dependencies.IdempotencyTTL <= 0 || dependencies.SMTPOperationBudget <= 0 ||
		dependencies.Audit == nil ||
		dependencies.IDs == nil || dependencies.Clock == nil {
		return nil, errors.New("administrator creation dependencies are incomplete")
	}
	return &AdministratorCreateCoordinator{dependencies: dependencies}, nil
}

// NewAdministratorPrivacyService creates the privacy acknowledgement coordinator.
func NewAdministratorPrivacyService(
	dependencies AdministratorPrivacyDependencies,
) (*AdministratorPrivacyCoordinator, error) {
	if dependencies.Transactions == nil || dependencies.Repository == nil ||
		dependencies.Notices == nil || dependencies.Sessions == nil ||
		dependencies.SessionIssuer == nil || dependencies.Audit == nil ||
		dependencies.IDs == nil || dependencies.Clock == nil {
		return nil, errors.New("administrator privacy dependencies are incomplete")
	}
	return &AdministratorPrivacyCoordinator{dependencies: dependencies}, nil
}

// CurrentAdministratorInformationNotice reads the declaration shown to the actor.
func (c *AdministratorCreateCoordinator) CurrentAdministratorInformationNotice(
	ctx context.Context,
	principal session.Principal,
) (AdministratorInformationNotice, error) {
	if c == nil {
		return AdministratorInformationNotice{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator creation service is not initialized"),
		)
	}
	if err := requireInitialAdministrator(principal); err != nil {
		return AdministratorInformationNotice{}, err
	}
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return AdministratorInformationNotice{}, administratorPersistenceError(
			"begin administrator information notice transaction",
			err,
		)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()
	notice, err := c.dependencies.InformationNotices.
		CurrentAdministratorInformationNotice(ctx, tx)
	if err != nil {
		return AdministratorInformationNotice{}, administratorPersistenceError(
			"read administrator information notice",
			err,
		)
	}
	if err := validateAdministratorInformationNotice(notice); err != nil {
		return AdministratorInformationNotice{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			err,
		)
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return AdministratorInformationNotice{}, administratorPersistenceError(
			"commit administrator information notice transaction",
			err,
		)
	}
	closed = true
	return notice, nil
}

// CreateAdministrator creates and delivers one temporary administrator credential.
func (c *AdministratorCreateCoordinator) CreateAdministrator(
	ctx context.Context,
	request CreateAdministratorRequest,
) (CreatedAdministrator, error) {
	if c == nil {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator creation service is not initialized"),
		)
	}
	if request.Principal.UserID == "" {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodeAuthForbidden,
			errors.New("current active initial administrator capability is required"),
		)
	}
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return CreatedAdministrator{}, administratorPersistenceError(
			"begin administrator creation transaction",
			err,
		)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()

	actor, err := c.dependencies.Repository.FindAdministratorCreationActor(
		ctx,
		tx,
		request.Principal.UserID,
	)
	if err != nil {
		return CreatedAdministrator{}, administratorPersistenceError(
			"read administrator creation actor",
			err,
		)
	}
	if err := requireCurrentInitialAdministrator(request.Principal, actor); err != nil {
		return CreatedAdministrator{}, err
	}
	if request.TraceID == "" {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator creation trace ID is required"),
		)
	}
	if !validIdempotencyKey(request.IdempotencyKey) {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodeIdentityIdempotencyKeyRequired,
			errors.New("administrator creation idempotency key is invalid"),
		)
	}
	normalized := normalizeCreateAdministratorRequest(request)
	beginRequest, err := c.idempotencyBeginRequest(request, normalized)
	if err != nil {
		return CreatedAdministrator{}, err
	}

	existing, found, err := c.dependencies.Idempotency.ResolveExisting(ctx, tx, beginRequest)
	if err != nil {
		return CreatedAdministrator{}, administratorIdempotencyError(err)
	}
	if found {
		switch existing.Action {
		case idempotency.BeginReplay:
			result, replayErr := c.replayCreateAdministrator(existing.Record)
			if err := tx.Commit(ctx); err != nil {
				closed = true
				return CreatedAdministrator{}, administratorPersistenceError(
					"commit administrator creation replay",
					err,
				)
			}
			closed = true
			return result, replayErr
		case idempotency.BeginInProgress:
			return CreatedAdministrator{}, platform.NewPublicError(
				platform.CodeIdentityIdempotencyInProgress,
				errors.New("administrator creation is already processing"),
			)
		case idempotency.BeginOperationUncertain:
			return CreatedAdministrator{}, administratorOperationUncertain(
				errors.New("administrator creation requires controlled review"),
			)
		default:
			return CreatedAdministrator{}, platform.NewPublicError(
				platform.CodePlatformInternalError,
				errors.New("administrator creation idempotency action is invalid"),
			)
		}
	}
	if err := c.dependencies.Gate.RequireOpen(ctx); err != nil {
		return CreatedAdministrator{}, processingGateError(err)
	}
	normalized, err = validateCreateAdministratorRequest(
		request,
		c.dependencies.PasswordPolicy,
	)
	if err != nil {
		return CreatedAdministrator{}, err
	}
	notice, err := c.dependencies.InformationNotices.
		CurrentAdministratorInformationNotice(ctx, tx)
	if err != nil {
		return CreatedAdministrator{}, administratorPersistenceError(
			"read current administrator information notice",
			err,
		)
	}
	if err := validateAdministratorInformationNotice(notice); err != nil {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			err,
		)
	}
	if normalized.noticeVersion != notice.Version {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodePrivacyNoticeOutdated,
			errors.New("administrator information notice version is not current"),
		)
	}

	begin, err := c.dependencies.Idempotency.Begin(ctx, tx, beginRequest)
	if err != nil {
		return CreatedAdministrator{}, administratorIdempotencyError(err)
	}
	switch begin.Action {
	case idempotency.BeginReplay:
		result, replayErr := c.replayCreateAdministrator(begin.Record)
		if replayErr != nil {
			return CreatedAdministrator{}, replayErr
		}
		if err := tx.Commit(ctx); err != nil {
			closed = true
			return CreatedAdministrator{}, administratorPersistenceError(
				"commit administrator creation replay",
				err,
			)
		}
		closed = true
		return result, nil
	case idempotency.BeginInProgress:
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodeIdentityIdempotencyInProgress,
			errors.New("administrator creation is already processing"),
		)
	case idempotency.BeginOperationUncertain:
		return CreatedAdministrator{}, administratorOperationUncertain(
			errors.New("administrator creation requires controlled review"),
		)
	case idempotency.BeginExecute:
	default:
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator creation idempotency action is invalid"),
		)
	}

	aggregate, err := c.newAdministratorAggregate(request, normalized)
	if err != nil {
		return CreatedAdministrator{}, err
	}
	if err := c.persistPendingAdministrator(ctx, tx, aggregate); err != nil {
		closed = true
		_ = tx.Rollback(ctx)
		return CreatedAdministrator{}, c.persistKnownCreateFailure(
			ctx,
			beginRequest,
			aggregate,
			err,
		)
	}
	mailContext, cancelMail := context.WithTimeout(
		ctx,
		c.dependencies.SMTPOperationBudget,
	)
	mailErr := c.dependencies.Mailer.Send(mailContext, mailer.Message{
		Purpose: mailer.PurposeNewAdministratorCredential,
		To:      normalized.email,
		Subject: "Your administrator temporary credential",
		Body:    "Temporary credential: " + request.TemporaryPassword,
	})
	cancelMail()
	if mailErr != nil {
		cleanupCtx, cancelCleanup := c.administratorReconciliationContext(ctx)
		defer cancelCleanup()
		closed = true
		_ = tx.Rollback(cleanupCtx)
		publicErr := platform.NewPublicError(
			platform.CodeIdentityEmailDeliveryFailed,
			fmt.Errorf("deliver administrator temporary credential: %w", mailErr),
		)
		return CreatedAdministrator{}, c.persistKnownCreateFailure(
			cleanupCtx,
			beginRequest,
			aggregate,
			publicErr,
		)
	}

	now := c.dependencies.Clock.Now().UTC()
	if err := c.dependencies.Repository.ActivateTemporaryCredential(
		ctx,
		tx,
		aggregate.credential.ID,
		now,
	); err != nil {
		cleanupCtx, cancelCleanup := c.administratorReconciliationContext(ctx)
		defer cancelCleanup()
		closed = true
		_ = tx.Rollback(cleanupCtx)
		return CreatedAdministrator{}, c.markCreateUncertain(
			cleanupCtx,
			beginRequest,
			begin.Record,
			fmt.Errorf("activate delivered administrator credential: %w", err),
		)
	}
	aggregate.result.DeliveryStatus = administratorDeliveryAccepted
	if err := c.dependencies.Audit.Append(ctx, tx, aggregate.auditEvent); err != nil {
		cleanupCtx, cancelCleanup := c.administratorReconciliationContext(ctx)
		defer cancelCleanup()
		closed = true
		_ = tx.Rollback(cleanupCtx)
		return CreatedAdministrator{}, c.markCreateUncertain(
			cleanupCtx,
			beginRequest,
			begin.Record,
			fmt.Errorf("append administrator creation audit: %w", err),
		)
	}
	if err := c.dependencies.Idempotency.Complete(
		ctx,
		tx,
		idempotencyCompletion(
			begin.Record,
			idempotency.StatusSucceeded,
			administratorCreateOutcome,
			administratorSuccessSnapshot(aggregate.result),
			now,
		),
	); err != nil {
		cleanupCtx, cancelCleanup := c.administratorReconciliationContext(ctx)
		defer cancelCleanup()
		closed = true
		_ = tx.Rollback(cleanupCtx)
		return CreatedAdministrator{}, c.markCreateUncertain(
			cleanupCtx,
			beginRequest,
			begin.Record,
			fmt.Errorf("complete administrator creation idempotency: %w", err),
		)
	}
	if err := tx.Commit(ctx); err != nil {
		cleanupCtx, cancelCleanup := c.administratorReconciliationContext(ctx)
		defer cancelCleanup()
		closed = true
		return CreatedAdministrator{}, c.markCreateUncertain(
			cleanupCtx,
			beginRequest,
			begin.Record,
			fmt.Errorf("commit delivered administrator creation: %w", err),
		)
	}
	closed = true
	return aggregate.result, nil
}

func (c *AdministratorCreateCoordinator) administratorReconciliationContext(
	ctx context.Context,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		context.WithoutCancel(ctx),
		c.dependencies.SMTPOperationBudget,
	)
}

// AcknowledgePrivacy appends the current notice evidence and recomputes both Gates.
func (c *AdministratorPrivacyCoordinator) AcknowledgePrivacy(
	ctx context.Context,
	request AdministratorPrivacyAcknowledgementRequest,
) (AdministratorOnboardingResult, error) {
	if c == nil {
		return AdministratorOnboardingResult{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator privacy service is not initialized"),
		)
	}
	if request.Principal.UserID == "" ||
		request.Principal.Role != string(RoleAdmin) ||
		request.Principal.IsInitialAdmin ||
		request.Principal.Status != string(UserStatusActive) {
		return AdministratorOnboardingResult{}, platform.NewPublicError(
			platform.CodeAuthForbidden,
			errors.New("administrator privacy acknowledgement is not allowed"),
		)
	}
	if !request.Acknowledged {
		return AdministratorOnboardingResult{}, platform.NewPublicError(
			platform.CodePrivacyAcknowledgementRequired,
			errors.New("current privacy notice acknowledgement is required"),
		)
	}
	if strings.TrimSpace(request.PrivacyNoticeVersion) == "" {
		return AdministratorOnboardingResult{}, validationError(
			errors.New("privacy notice version is required"),
		)
	}
	if request.CurrentSessionToken == "" || request.TraceID == "" ||
		strings.TrimSpace(request.SourceKind) == "" ||
		strings.TrimSpace(request.SourceFingerprint) == "" {
		return AdministratorOnboardingResult{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator privacy acknowledgement evidence is incomplete"),
		)
	}

	dependencies := c.dependencies
	tx, err := dependencies.Transactions.Begin(ctx)
	if err != nil {
		return AdministratorOnboardingResult{}, administratorPersistenceError(
			"begin administrator privacy transaction",
			err,
		)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()
	user, err := dependencies.Repository.FindAdministratorForOnboarding(
		ctx,
		tx,
		request.Principal.UserID,
	)
	if err != nil {
		return AdministratorOnboardingResult{}, administratorPersistenceError(
			"read administrator onboarding state",
			err,
		)
	}
	if user.ID != request.Principal.UserID || user.Role != RoleAdmin ||
		user.IsInitialAdmin || user.Status != UserStatusActive ||
		user.OnboardingCompletedAt != nil {
		return AdministratorOnboardingResult{}, platform.NewPublicError(
			platform.CodeIdentityInvalidState,
			errors.New("administrator onboarding state is not eligible"),
		)
	}
	notice, err := dependencies.Notices.CurrentNotice(ctx, tx)
	if err != nil {
		return AdministratorOnboardingResult{}, currentNoticeError(err)
	}
	requestVersion := strings.TrimSpace(request.PrivacyNoticeVersion)
	if requestVersion != notice.Version {
		return AdministratorOnboardingResult{}, platform.NewPublicError(
			platform.CodePrivacyNoticeOutdated,
			errors.New("administrator privacy notice version is not current"),
		)
	}
	now := dependencies.Clock.Now().UTC()
	acknowledgementID, err := dependencies.IDs.NewID()
	if err != nil {
		return AdministratorOnboardingResult{}, identifierGenerationError(err)
	}
	auditID, err := dependencies.IDs.NewID()
	if err != nil {
		return AdministratorOnboardingResult{}, identifierGenerationError(err)
	}
	if err := dependencies.Repository.InsertPrivacyAcknowledgement(
		ctx,
		tx,
		PrivacyAcknowledgement{
			ID: acknowledgementID, UserID: user.ID, NoticeVersion: notice.Version,
			AcknowledgedAt: now, SourceKind: strings.TrimSpace(request.SourceKind),
			SourceFingerprint: strings.TrimSpace(request.SourceFingerprint),
		},
	); err != nil {
		return AdministratorOnboardingResult{}, administratorPersistenceError(
			"insert administrator privacy acknowledgement",
			err,
		)
	}
	completed := user.CredentialMode == CredentialModePermanent
	if err := dependencies.Repository.SetAdministratorPrivacyAcknowledgement(
		ctx,
		tx,
		user.ID,
		notice.Version,
		completed,
		now,
	); err != nil {
		return AdministratorOnboardingResult{}, administratorPersistenceError(
			"update administrator privacy Gate",
			err,
		)
	}
	user.OnboardingPrivacyAcknowledgedVersion = &notice.Version
	user.UpdatedAt = now
	if completed {
		user.OnboardingCompletedAt = &now
	}
	replacement, err := dependencies.SessionIssuer.Issue(
		user,
		request.Principal.AuthenticatedCredential,
		request.Principal.AuthenticatedCredentialID,
	)
	if err != nil {
		return AdministratorOnboardingResult{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			fmt.Errorf("issue administrator privacy session: %w", err),
		)
	}
	replacement.RotatedFromID = &request.Principal.SessionID
	credentials, err := dependencies.Sessions.Rotate(
		ctx,
		tx,
		request.CurrentSessionToken,
		now,
		"privacy_acknowledged",
		replacement,
	)
	if err != nil {
		return AdministratorOnboardingResult{}, administratorPersistenceError(
			"rotate administrator privacy session",
			err,
		)
	}
	actorID, actorRole := request.Principal.UserID, request.Principal.Role
	if err := dependencies.Audit.Append(ctx, tx, audit.Event{
		ID: auditID, OccurredAt: now, ActorKind: audit.ActorUser,
		ActorUserID: &actorID, ActorRole: &actorRole,
		Action: administratorPrivacyAction, TargetType: "user",
		TargetID: user.ID, Outcome: "succeeded", TraceID: request.TraceID,
		Metadata: map[string]any{"notice_version": notice.Version},
	}); err != nil {
		return AdministratorOnboardingResult{}, administratorPersistenceError(
			"append administrator privacy audit",
			err,
		)
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return AdministratorOnboardingResult{}, administratorPersistenceError(
			"commit administrator privacy acknowledgement",
			err,
		)
	}
	closed = true
	return AdministratorOnboardingResult{
		User: user, MustChangePassword: passwordChangeRequired(user.CredentialMode),
		MustAcknowledgePrivacy: user.OnboardingCompletedAt == nil,
		SessionToken:           credentials.Token, CSRFToken: credentials.CSRFToken,
	}, nil
}

type normalizedCreateAdministrator struct {
	username        string
	normalizedUser  string
	email           string
	normalizedEmail string
	phone           string
	noticeVersion   string
}

type administratorCreationAggregate struct {
	user        User
	identifiers []IdentityIdentifier
	credential  TemporaryCredential
	declaration AdministratorInformationDeclaration
	auditEvent  audit.Event
	result      CreatedAdministrator
}

func validateCreateAdministratorRequest(
	request CreateAdministratorRequest,
	passwordPolicy RegistrationPasswordPolicy,
) (normalizedCreateAdministrator, error) {
	if !validIdempotencyKey(request.IdempotencyKey) {
		return normalizedCreateAdministrator{}, platform.NewPublicError(
			platform.CodeIdentityIdempotencyKeyRequired,
			errors.New("administrator creation idempotency key is invalid"),
		)
	}
	if !request.InformationProvisionDeclared {
		return normalizedCreateAdministrator{}, platform.NewPublicError(
			platform.CodePrivacyInformationDeclarationRequired,
			errors.New("administrator information provision declaration is required"),
		)
	}
	normalized := normalizeCreateAdministratorRequest(request)
	username := normalized.username
	email := normalized.email
	phone := normalized.phone
	noticeVersion := normalized.noticeVersion
	if !validUsername(username) || !validEmail(email) || phone == "" ||
		noticeVersion == "" {
		return normalizedCreateAdministrator{}, validationError(
			errors.New("administrator creation fields are invalid"),
		)
	}
	if passwordPolicy == nil {
		return normalizedCreateAdministrator{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator password policy is unavailable"),
		)
	}
	if err := passwordPolicy.Validate(request.TemporaryPassword); err != nil {
		return normalizedCreateAdministrator{}, validationError(err)
	}
	return normalized, nil
}

func normalizeCreateAdministratorRequest(
	request CreateAdministratorRequest,
) normalizedCreateAdministrator {
	username := strings.TrimSpace(request.Username)
	email := strings.TrimSpace(request.Email)
	return normalizedCreateAdministrator{
		username: username, normalizedUser: strings.ToLower(username),
		email: email, normalizedEmail: strings.ToLower(email),
		phone:         strings.TrimSpace(request.Phone),
		noticeVersion: strings.TrimSpace(request.InformationProvisionNoticeVersion),
	}
}

func requireInitialAdministrator(principal session.Principal) error {
	if principal.UserID == "" || principal.Role != string(RoleAdmin) ||
		!principal.IsInitialAdmin || principal.Status != string(UserStatusActive) ||
		principal.Restricted() {
		return platform.NewPublicError(
			platform.CodeAuthForbidden,
			errors.New("initial administrator capability is required"),
		)
	}
	return nil
}

func requireCurrentInitialAdministrator(principal session.Principal, user User) error {
	if principal.UserID == "" || user.ID != principal.UserID ||
		user.Role != RoleAdmin || !user.IsInitialAdmin ||
		user.Status != UserStatusActive ||
		passwordChangeRequired(user.CredentialMode) ||
		user.RecoveryRequired || user.DeletedAt != nil {
		return platform.NewPublicError(
			platform.CodeAuthForbidden,
			errors.New("current active initial administrator capability is required"),
		)
	}
	return nil
}

func validIdempotencyKey(value string) bool {
	if len(value) < 16 || len(value) > 128 || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range []byte(value) {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}

func administratorOnboardingComplete(user User, currentNoticeVersion string) bool {
	return user.Role == RoleAdmin &&
		!user.IsInitialAdmin &&
		user.CredentialMode == CredentialModePermanent &&
		user.OnboardingPrivacyAcknowledgedVersion != nil &&
		*user.OnboardingPrivacyAcknowledgedVersion == currentNoticeVersion
}

func validateAdministratorInformationNotice(notice AdministratorInformationNotice) error {
	if strings.TrimSpace(notice.Version) == "" || notice.EffectiveAt.IsZero() ||
		strings.TrimSpace(notice.Content) == "" ||
		strings.TrimSpace(notice.ContentHash) == "" {
		return errors.New("administrator information notice is invalid")
	}
	return nil
}

func (c *AdministratorCreateCoordinator) idempotencyBeginRequest(
	request CreateAdministratorRequest,
	normalized normalizedCreateAdministrator,
) (idempotency.BeginRequest, error) {
	encoded, err := json.Marshal(struct {
		Username          string `json:"username"`
		Email             string `json:"email"`
		Phone             string `json:"phone"`
		TemporaryPassword string `json:"temporary_password"`
		NoticeVersion     string `json:"information_provision_notice_version"`
		Declared          bool   `json:"information_provision_declared"`
	}{
		Username: normalized.normalizedUser, Email: normalized.normalizedEmail,
		Phone: normalized.phone, TemporaryPassword: request.TemporaryPassword,
		NoticeVersion: normalized.noticeVersion,
		Declared:      request.InformationProvisionDeclared,
	})
	if err != nil {
		return idempotency.BeginRequest{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			fmt.Errorf("encode administrator idempotency request: %w", err),
		)
	}
	now := c.dependencies.Clock.Now().UTC()
	scope := []byte(
		request.Principal.UserID + "\x00" + administratorCreateOperation +
			"\x00" + normalized.normalizedUser,
	)
	return idempotency.BeginRequest{
		Scope: scope, Key: request.IdempotencyKey, Request: encoded,
		DigestKeys:  c.dependencies.IdempotencyKeys,
		LockedUntil: now.Add(c.dependencies.IdempotencyLock),
		ExpiresAt:   now.Add(c.dependencies.IdempotencyTTL), Now: now,
	}, nil
}

func (c *AdministratorCreateCoordinator) newAdministratorAggregate(
	request CreateAdministratorRequest,
	normalized normalizedCreateAdministrator,
) (administratorCreationAggregate, error) {
	ids := make([]string, 4)
	for index := range ids {
		id, err := c.dependencies.IDs.NewID()
		if err != nil {
			return administratorCreationAggregate{}, identifierGenerationError(err)
		}
		ids[index] = id
	}
	now := c.dependencies.Clock.Now().UTC()
	credential, err := NewTemporaryCredential(
		c.dependencies.PasswordHasher,
		TemporaryCredentialInput{
			ID: ids[1], UserID: ids[0],
			Purpose: TemporaryCredentialNewAdministrator, Generation: 1,
			Secret: request.TemporaryPassword, DeliveredToEmail: normalized.email,
			IssuedAt: now,
		},
	)
	if err != nil {
		return administratorCreationAggregate{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			fmt.Errorf("create administrator temporary credential: %w", err),
		)
	}
	email, phone := normalized.email, normalized.phone
	user := User{
		ID: ids[0], Username: normalized.username, Email: &email, Phone: &phone,
		Role: RoleAdmin, Status: UserStatusActive,
		CredentialMode: CredentialModeTemporary, CreatedAt: now, UpdatedAt: now,
	}
	actorID, actorRole := request.Principal.UserID, request.Principal.Role
	return administratorCreationAggregate{
		user: user,
		identifiers: []IdentityIdentifier{
			newRegistrationIdentifier(user.ID, IdentifierUsername, normalized.normalizedUser, now),
			newRegistrationIdentifier(user.ID, IdentifierEmail, normalized.normalizedEmail, now),
			newRegistrationIdentifier(user.ID, IdentifierPhone, normalized.phone, now),
		},
		credential: credential,
		declaration: AdministratorInformationDeclaration{
			ID: ids[2], ActorUserID: request.Principal.UserID,
			TargetLocatorDigest: c.administratorLocatorDigest(normalized),
			StatementVersion:    normalized.noticeVersion, DeclaredAt: now,
			Purpose: string(mailer.PurposeNewAdministratorCredential),
		},
		auditEvent: audit.Event{
			ID: ids[3], OccurredAt: now, ActorKind: audit.ActorUser,
			ActorUserID: &actorID, ActorRole: &actorRole,
			Action: administratorCreateOperation, TargetType: "user",
			TargetID: user.ID, Outcome: "succeeded", TraceID: request.TraceID,
			Metadata: map[string]any{
				"notice_version":  normalized.noticeVersion,
				"delivery_status": administratorDeliveryAccepted,
			},
		},
		result: CreatedAdministrator{
			ID: user.ID, Username: user.Username, Role: user.Role, Status: user.Status,
			MustChangePassword: true, MustAcknowledgePrivacy: true,
			TemporaryCredentialExpiresAt: credential.ExpiresAt,
		},
	}, nil
}

func (c *AdministratorCreateCoordinator) administratorLocatorDigest(
	normalized normalizedCreateAdministrator,
) []byte {
	key := c.dependencies.IdempotencyKeys.Keys[c.dependencies.IdempotencyKeys.ActiveVersion]
	digest := hmac.New(sha256.New, key)
	_, _ = digest.Write([]byte(administratorLocatorDomain))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write([]byte(normalized.normalizedUser))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write([]byte(normalized.normalizedEmail))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write([]byte(normalized.phone))
	return digest.Sum(nil)
}

func (c *AdministratorCreateCoordinator) persistPendingAdministrator(
	ctx context.Context,
	tx pgx.Tx,
	aggregate administratorCreationAggregate,
) error {
	if err := c.dependencies.Repository.InsertUser(ctx, tx, aggregate.user); err != nil {
		return administratorCreatePersistenceOrConflict("insert administrator", err)
	}
	for _, identifier := range aggregate.identifiers {
		if err := c.dependencies.Repository.InsertIdentifier(ctx, tx, identifier); err != nil {
			return administratorCreatePersistenceOrConflict(
				"insert administrator identifier",
				err,
			)
		}
	}
	if err := c.dependencies.Repository.InsertTemporaryCredential(
		ctx,
		tx,
		aggregate.credential,
	); err != nil {
		return administratorPersistenceError(
			"insert administrator temporary credential",
			err,
		)
	}
	if err := c.dependencies.Repository.InsertAdministratorInformationDeclaration(
		ctx,
		tx,
		aggregate.declaration,
	); err != nil {
		return administratorPersistenceError(
			"insert administrator information declaration",
			err,
		)
	}
	return nil
}

func (c *AdministratorCreateCoordinator) replayCreateAdministrator(
	record idempotency.Record,
) (CreatedAdministrator, error) {
	if record.ResponseSnapshot == nil {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator creation replay snapshot is missing"),
		)
	}
	snapshot := *record.ResponseSnapshot
	if snapshot.SchemaVersion != idempotency.AdministratorCreateResponseSchemaV1 ||
		snapshot.Body == nil {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator creation replay snapshot version is invalid"),
		)
	}
	if record.Status == idempotency.StatusFailed {
		return CreatedAdministrator{}, newAdministratorResponseError(
			snapshot,
			errors.New("administrator creation replayed stable failure"),
		)
	}
	if record.Status != idempotency.StatusSucceeded || snapshot.HTTPStatus != 201 {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator creation replay state is invalid"),
		)
	}
	expiresAt, err := time.Parse(time.RFC3339, snapshot.Body.TemporaryCredentialExpiresAt)
	if err != nil {
		return CreatedAdministrator{}, platform.NewPublicError(
			platform.CodePlatformInternalError,
			err,
		)
	}
	return CreatedAdministrator{
		ID: snapshot.Body.ID, Username: snapshot.Body.Username,
		Role: Role(snapshot.Body.Role), Status: UserStatus(snapshot.Body.Status),
		MustChangePassword:           snapshot.Body.MustChangePassword,
		MustAcknowledgePrivacy:       snapshot.Body.MustAcknowledgePrivacy,
		TemporaryCredentialExpiresAt: expiresAt,
		DeliveryStatus:               snapshot.Body.DeliveryStatus,
	}, nil
}

func (c *AdministratorCreateCoordinator) persistKnownCreateFailure(
	ctx context.Context,
	beginRequest idempotency.BeginRequest,
	aggregate administratorCreationAggregate,
	cause error,
) error {
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return administratorPersistenceError(
			"begin administrator failure transaction",
			errors.Join(cause, err),
		)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()
	begin, err := c.dependencies.Idempotency.Begin(ctx, tx, beginRequest)
	if err != nil {
		return administratorPersistenceError(
			"reserve administrator failure idempotency",
			errors.Join(cause, err),
		)
	}
	if begin.Action != idempotency.BeginExecute {
		return administratorOperationUncertain(errors.Join(
			cause,
			errors.New("administrator failure idempotency was not executable"),
		))
	}
	now := c.dependencies.Clock.Now().UTC()
	code := platform.PublicCode(cause)
	snapshot, snapshotErr := administratorFailureSnapshot(code)
	if snapshotErr != nil {
		return administratorPersistenceError(
			"create administrator failure snapshot",
			errors.Join(cause, snapshotErr),
		)
	}
	if err := c.dependencies.Idempotency.Complete(
		ctx,
		tx,
		idempotencyCompletion(
			begin.Record,
			idempotency.StatusFailed,
			string(code),
			snapshot,
			now,
		),
	); err != nil {
		return administratorPersistenceError(
			"complete administrator failure idempotency",
			errors.Join(cause, err),
		)
	}
	failureEvent := aggregate.auditEvent
	failureEvent.OccurredAt = now
	failureEvent.Outcome = "failed"
	errorCode := string(code)
	failureEvent.ErrorCode = &errorCode
	failureEvent.Metadata = map[string]any{
		"notice_version": aggregate.declaration.StatementVersion,
	}
	if err := c.dependencies.Audit.Append(ctx, tx, failureEvent); err != nil {
		return administratorPersistenceError(
			"append administrator creation failure audit",
			errors.Join(cause, err),
		)
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return administratorPersistenceError(
			"commit administrator creation failure",
			errors.Join(cause, err),
		)
	}
	closed = true
	return newAdministratorResponseError(snapshot, cause)
}

type administratorResponseError struct {
	snapshot idempotency.ResponseSnapshot
	cause    error
}

func (e *administratorResponseError) Error() string {
	return e.snapshot.ResultCode
}

func (e *administratorResponseError) Unwrap() error {
	return platform.NewPublicError(platform.Code(e.snapshot.ResultCode), e.cause)
}

func newAdministratorResponseError(
	snapshot idempotency.ResponseSnapshot,
	cause error,
) error {
	if snapshot.Body == nil {
		return platform.NewPublicError(platform.CodePlatformInternalError, cause)
	}
	return &administratorResponseError{snapshot: snapshot, cause: cause}
}

func administratorSuccessSnapshot(
	result CreatedAdministrator,
) idempotency.ResponseSnapshot {
	return idempotency.ResponseSnapshot{
		SchemaVersion: idempotency.AdministratorCreateResponseSchemaV1,
		HTTPStatus:    201,
		ResultCode:    administratorCreateOutcome,
		Body: &idempotency.AdministratorCreateResponseBody{
			ID: result.ID, Username: result.Username, Role: string(result.Role),
			Status:                 string(result.Status),
			MustChangePassword:     result.MustChangePassword,
			MustAcknowledgePrivacy: result.MustAcknowledgePrivacy,
			TemporaryCredentialExpiresAt: result.TemporaryCredentialExpiresAt.
				UTC().Format(time.RFC3339),
			DeliveryStatus: result.DeliveryStatus,
		},
	}
}

func administratorFailureSnapshot(
	code platform.Code,
) (idempotency.ResponseSnapshot, error) {
	var status int
	var message string
	switch code {
	case platform.CodeIdentityIdentifierConflict:
		status = 409
		message = "\u8eab\u4efd\u6807\u8bc6\u5df2\u88ab\u4f7f\u7528"
	case platform.CodeIdentityEmailDeliveryFailed:
		status = 503
		message = "\u90ae\u4ef6\u6682\u672a\u53d7\u7406"
	default:
		return idempotency.ResponseSnapshot{}, errors.New(
			"administrator failure is not replayable",
		)
	}
	return idempotency.ResponseSnapshot{
		SchemaVersion: idempotency.AdministratorCreateResponseSchemaV1,
		HTTPStatus:    status,
		ResultCode:    string(code),
		Body: &idempotency.AdministratorCreateResponseBody{
			Code: string(code), Message: message,
			FieldErrors: []idempotency.ResponseFieldError{},
		},
	}, nil
}

func (c *AdministratorCreateCoordinator) markCreateUncertain(
	ctx context.Context,
	beginRequest idempotency.BeginRequest,
	original idempotency.Record,
	cause error,
) error {
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil {
		return administratorOperationUncertain(errors.Join(cause, err))
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()
	begin, err := c.dependencies.Idempotency.Begin(ctx, tx, beginRequest)
	if err != nil {
		return administratorOperationUncertain(errors.Join(cause, err))
	}
	record := begin.Record
	if len(record.ScopeHash) == 0 {
		record = original
	}
	switch begin.Action {
	case idempotency.BeginExecute, idempotency.BeginInProgress:
		if err := c.dependencies.Idempotency.MarkUncertain(
			ctx,
			tx,
			idempotencyTransition(record, c.dependencies.Clock.Now().UTC()),
		); err != nil {
			return administratorOperationUncertain(errors.Join(cause, err))
		}
	case idempotency.BeginOperationUncertain, idempotency.BeginReplay:
	default:
		return administratorOperationUncertain(errors.Join(
			cause,
			errors.New("administrator uncertainty action is invalid"),
		))
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return administratorOperationUncertain(errors.Join(cause, err))
	}
	closed = true
	return administratorOperationUncertain(cause)
}

func idempotencyCompletion(
	record idempotency.Record,
	status idempotency.Status,
	resultCode string,
	snapshot idempotency.ResponseSnapshot,
	now time.Time,
) idempotency.Completion {
	return idempotency.Completion{
		ScopeHash: record.ScopeHash, KeyLocator: record.KeyLocator,
		KeyDigest: record.KeyDigest, RequestDigest: record.RequestDigest,
		DigestKeyVersion:    record.DigestKeyVersion,
		ExecutionGeneration: record.ExecutionGeneration,
		Status:              status, ResultCode: resultCode, ResponseSnapshot: snapshot, Now: now,
	}
}

func idempotencyTransition(
	record idempotency.Record,
	now time.Time,
) idempotency.Transition {
	return idempotency.Transition{
		ScopeHash: record.ScopeHash, KeyLocator: record.KeyLocator,
		KeyDigest: record.KeyDigest, RequestDigest: record.RequestDigest,
		DigestKeyVersion:    record.DigestKeyVersion,
		ExecutionGeneration: record.ExecutionGeneration, Now: now,
	}
}

func administratorIdempotencyError(err error) error {
	switch {
	case errors.Is(err, idempotency.ErrDigestConflict):
		return platform.NewPublicError(platform.CodeIdentityIdempotencyConflict, err)
	case errors.Is(err, idempotency.ErrDigestKeyUnavailable):
		return platform.NewPublicError(platform.CodeIdentityOperationUncertain, err)
	default:
		return administratorPersistenceError("apply administrator idempotency", err)
	}
}

func administratorCreatePersistenceOrConflict(operation string, err error) error {
	if isUniqueViolation(err) {
		return platform.NewPublicError(platform.CodeIdentityIdentifierConflict, err)
	}
	return administratorPersistenceError(operation, err)
}

func administratorPersistenceError(operation string, err error) error {
	return platform.NewPublicError(
		platform.CodePlatformDatabaseUnavailable,
		fmt.Errorf("%s: %w", operation, err),
	)
}

func administratorOperationUncertain(err error) error {
	return platform.NewPublicError(platform.CodeIdentityOperationUncertain, err)
}

// StaticAdministratorInformationNoticeReader exposes one externally approved notice.
type StaticAdministratorInformationNoticeReader struct {
	Notice AdministratorInformationNotice
}

// CurrentAdministratorInformationNotice returns the configured immutable notice.
func (r StaticAdministratorInformationNoticeReader) CurrentAdministratorInformationNotice(
	_ context.Context,
	tx pgx.Tx,
) (AdministratorInformationNotice, error) {
	if tx == nil {
		return AdministratorInformationNotice{}, errors.New(
			"administrator information notice transaction is required",
		)
	}
	if err := validateAdministratorInformationNotice(r.Notice); err != nil {
		return AdministratorInformationNotice{}, err
	}
	return r.Notice, nil
}

// PostgresAdministratorRepository persists administrator creation and onboarding.
type PostgresAdministratorRepository struct{}

// FindAdministratorCreationActor locks the current actor authorization state.
func (PostgresAdministratorRepository) FindAdministratorCreationActor(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (User, error) {
	var user User
	err := tx.QueryRow(ctx, `
		SELECT
			id, role, is_initial_admin, status, credential_mode,
			recovery_required, deleted_at
		FROM users
		WHERE id = $1
		FOR SHARE
	`, userID).Scan(
		&user.ID,
		&user.Role,
		&user.IsInitialAdmin,
		&user.Status,
		&user.CredentialMode,
		&user.RecoveryRequired,
		&user.DeletedAt,
	)
	return user, err
}

func (PostgresAdministratorRepository) InsertUser(
	ctx context.Context,
	tx pgx.Tx,
	user User,
) error {
	return (Repository{}).InsertUser(ctx, tx, user)
}

func (PostgresAdministratorRepository) InsertIdentifier(
	ctx context.Context,
	tx pgx.Tx,
	identifier IdentityIdentifier,
) error {
	return (Repository{}).InsertIdentifier(ctx, tx, identifier)
}

func (PostgresAdministratorRepository) InsertTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	credential TemporaryCredential,
) error {
	return (Repository{}).InsertTemporaryCredential(ctx, tx, credential)
}

func (PostgresAdministratorRepository) InsertAdministratorInformationDeclaration(
	ctx context.Context,
	tx pgx.Tx,
	declaration AdministratorInformationDeclaration,
) error {
	return (Repository{}).InsertAdministratorInformationDeclaration(ctx, tx, declaration)
}

func (PostgresAdministratorRepository) InsertPrivacyAcknowledgement(
	ctx context.Context,
	tx pgx.Tx,
	acknowledgement PrivacyAcknowledgement,
) error {
	return (Repository{}).InsertPrivacyAcknowledgement(ctx, tx, acknowledgement)
}

// ActivateTemporaryCredential makes the delivered pending credential current.
func (PostgresAdministratorRepository) ActivateTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	credentialID string,
	activatedAt time.Time,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE temporary_credentials
		SET state = 'active'
		WHERE id = $1
			AND state = 'pending_delivery'
			AND expires_at > $2
	`, credentialID, activatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("administrator temporary credential state changed")
	}
	return nil
}

// FindCreatedAdministrator reads the safe replay result.
func (PostgresAdministratorRepository) FindCreatedAdministrator(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (CreatedAdministrator, error) {
	var result CreatedAdministrator
	err := tx.QueryRow(ctx, `
		SELECT
			u.id, u.username, u.role, u.status, tc.expires_at
		FROM users u
		JOIN temporary_credentials tc ON tc.user_id = u.id
		WHERE u.id = $1
			AND u.role = 'admin'
			AND NOT u.is_initial_admin
			AND tc.purpose = 'new_administrator'
			AND tc.state = 'active'
		ORDER BY tc.generation DESC
		LIMIT 1
	`, userID).Scan(
		&result.ID, &result.Username, &result.Role, &result.Status,
		&result.TemporaryCredentialExpiresAt,
	)
	if err != nil {
		return CreatedAdministrator{}, err
	}
	result.MustChangePassword = true
	result.MustAcknowledgePrivacy = true
	result.DeliveryStatus = administratorDeliveryAccepted
	return result, nil
}

// FindAdministratorForOnboarding locks the authoritative administrator row.
func (PostgresAdministratorRepository) FindAdministratorForOnboarding(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (User, error) {
	var user User
	err := tx.QueryRow(ctx, `
		SELECT
			id, username, email, phone, permanent_password_digest, role,
			is_initial_admin, status, credential_mode, recovery_required,
			onboarding_privacy_acknowledged_version, onboarding_completed_at,
			retained_email, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, userID).Scan(
		&user.ID, &user.Username, &user.Email, &user.Phone,
		&user.PermanentPasswordDigest, &user.Role, &user.IsInitialAdmin,
		&user.Status, &user.CredentialMode, &user.RecoveryRequired,
		&user.OnboardingPrivacyAcknowledgedVersion, &user.OnboardingCompletedAt,
		&user.RetainedEmail, &user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
	)
	return user, err
}

// SetAdministratorPrivacyAcknowledgement updates the privacy Gate and final marker.
func (PostgresAdministratorRepository) SetAdministratorPrivacyAcknowledgement(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	noticeVersion string,
	complete bool,
	updatedAt time.Time,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE users
		SET
			onboarding_privacy_acknowledged_version = $2,
			onboarding_completed_at = CASE
				WHEN $3 THEN COALESCE(onboarding_completed_at, $4)
				ELSE onboarding_completed_at
			END,
			updated_at = $4
		WHERE id = $1
			AND role = 'admin'
			AND NOT is_initial_admin
			AND status = 'active'
			AND onboarding_completed_at IS NULL
	`, userID, noticeVersion, complete, updatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("administrator privacy Gate state changed")
	}
	return nil
}

// CurrentNotice locks the notice used by password-change finalization.
func (PostgresPasswordChangeRepository) CurrentNotice(
	ctx context.Context,
	tx pgx.Tx,
) (PrivacyNotice, error) {
	return (CurrentNoticeReader{}).CurrentNotice(ctx, tx)
}

// CompleteAdministratorOnboarding writes the final marker only for both current Gates.
func (PostgresPasswordChangeRepository) CompleteAdministratorOnboarding(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	noticeVersion string,
	completedAt time.Time,
) error {
	tag, err := tx.Exec(ctx, `
		UPDATE users
		SET onboarding_completed_at = $3, updated_at = $3
		WHERE id = $1
			AND role = 'admin'
			AND NOT is_initial_admin
			AND status = 'active'
			AND credential_mode = 'permanent'
			AND onboarding_privacy_acknowledged_version = $2
			AND onboarding_completed_at IS NULL
	`, userID, noticeVersion, completedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("administrator onboarding state changed")
	}
	return nil
}
