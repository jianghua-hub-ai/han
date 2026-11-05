package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"han/internal/audit"
	"han/internal/idempotency"
	"han/internal/mailer"
	"han/internal/platform"
	"han/internal/session"

	"github.com/jackc/pgx/v5"
)

const temporaryCredentialResendOperation = "temporary_credential.resend"

var errTemporaryCredentialResendContextUnavailable = errors.New(
	"temporary credential resend context is unavailable",
)

// TemporaryCredentialResendPurpose is selected only by the trusted HTTP route.
type TemporaryCredentialResendPurpose string

const (
	// TemporaryCredentialResendPasswordReset uses the durable reset authority.
	TemporaryCredentialResendPasswordReset TemporaryCredentialResendPurpose = "password_reset"
	// TemporaryCredentialResendNewAdministrator preserves the ADR-0011 path.
	TemporaryCredentialResendNewAdministrator TemporaryCredentialResendPurpose = "new_administrator"
)

// TemporaryCredentialResendRequest contains only trusted route and session evidence.
type TemporaryCredentialResendRequest struct {
	Actor          session.Principal
	TargetID       string
	TargetRole     Role
	Purpose        TemporaryCredentialResendPurpose
	IdempotencyKey string
	TraceID        string
}

// TemporaryCredentialResendService is the route-selected resend capability.
type TemporaryCredentialResendService interface {
	ResendTemporaryCredential(
		context.Context,
		TemporaryCredentialResendRequest,
	) (PasswordResetResult, error)
}

// TemporaryCredentialResendRouter selects one existing coordinator by purpose.
type TemporaryCredentialResendRouter struct {
	passwordReset       *PasswordResetCoordinator
	administratorCreate *AdministratorCreateCoordinator
}

// NewTemporaryCredentialResendService wires the two existing authorities.
func NewTemporaryCredentialResendService(
	passwordReset *PasswordResetCoordinator,
	administratorCreate *AdministratorCreateCoordinator,
) (*TemporaryCredentialResendRouter, error) {
	if passwordReset == nil || administratorCreate == nil {
		return nil, errors.New("temporary credential resend dependencies are incomplete")
	}
	return &TemporaryCredentialResendRouter{
		passwordReset:       passwordReset,
		administratorCreate: administratorCreate,
	}, nil
}

// ResendTemporaryCredential dispatches by the server-derived purpose.
func (r *TemporaryCredentialResendRouter) ResendTemporaryCredential(
	ctx context.Context,
	request TemporaryCredentialResendRequest,
) (PasswordResetResult, error) {
	if r == nil || r.passwordReset == nil || r.administratorCreate == nil {
		return PasswordResetResult{}, passwordResetInternalError(
			"validate temporary credential resend service",
			errors.New("service is not initialized"),
		)
	}
	switch request.Purpose {
	case TemporaryCredentialResendPasswordReset:
		if request.TargetRole != RoleMember {
			return PasswordResetResult{}, passwordResetForbiddenError()
		}
		return r.passwordReset.ResendPasswordResetCredential(
			ctx,
			PasswordResetRequest{
				Actor: request.Actor, TargetID: request.TargetID,
				TargetRole: request.TargetRole, IdempotencyKey: request.IdempotencyKey,
				TraceID: request.TraceID,
			},
		)
	case TemporaryCredentialResendNewAdministrator:
		if request.TargetRole != RoleAdmin {
			return PasswordResetResult{}, passwordResetForbiddenError()
		}
		return r.administratorCreate.resendNewAdministratorCredential(
			ctx,
			request,
			r.passwordReset.dependencies.Secrets,
		)
	default:
		return PasswordResetResult{}, passwordResetForbiddenError()
	}
}

type passwordResetResendAuthority struct {
	delegate passwordResetCandidateAuthority
	store    *PasswordResetStore
	request  PasswordResetRequest
}

func (a passwordResetResendAuthority) ClassifyExistingCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	userID string,
	request idempotency.BeginRequest,
	newHoldID func() (string, error),
) (PasswordResetCandidateResult, bool, error) {
	request, err := a.resendBeginRequest(request)
	if err != nil {
		return PasswordResetCandidateResult{}, false, err
	}
	return a.delegate.ClassifyExistingCandidate(
		ctx, tx, userID, request, newHoldID,
	)
}

func (a passwordResetResendAuthority) AuthorizeNewCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	candidate PasswordResetCandidate,
) error {
	if candidate.Operation.RequestKind != PasswordResetResend ||
		candidate.Operation.PredecessorOperationID == nil ||
		candidate.Operation.ReplacedCredentialID == nil {
		return errTemporaryCredentialResendContextUnavailable
	}
	if a.store != nil {
		pgxTx, ok := tx.(pgx.Tx)
		if !ok {
			return ErrPasswordResetCandidateInvalid
		}
		lineage, err := passwordResetResendLineageStatus(ctx, tx, candidate)
		if err != nil {
			return err
		}
		switch lineage.status {
		case passwordResetResendLineageIneligible:
			return errTemporaryCredentialResendContextUnavailable
		case passwordResetResendLineageDrift:
			return administratorOperationUncertain(
				errors.New("password reset resend lineage integrity drift"),
			)
		}
		if err := a.store.CheckLockedUser(
			ctx,
			pgxTx,
			candidate.Operation.UserID,
		); err != nil {
			if errors.Is(err, ErrPasswordResetGateClosed) {
				return administratorOperationUncertain(
					fmt.Errorf("password reset resend Gate is unresolved: %w", err),
				)
			}
			return err
		}
	}
	if authorizer, ok := a.delegate.(passwordResetNewCandidateAuthorizer); ok {
		return authorizer.AuthorizeNewCandidate(ctx, tx, candidate)
	}
	return nil
}

func (a passwordResetResendAuthority) BeginOrClassifyCandidate(
	ctx context.Context,
	tx idempotency.Transaction,
	candidate PasswordResetCandidate,
) (PasswordResetCandidateResult, error) {
	if candidate.Operation.RequestKind != PasswordResetResend {
		return PasswordResetCandidateResult{},
			errTemporaryCredentialResendContextUnavailable
	}
	request, err := a.resendBeginRequest(candidate.IdempotencyRequest)
	if err != nil {
		return PasswordResetCandidateResult{}, err
	}
	candidate.IdempotencyRequest = request
	if a.store != nil {
		pgxTx, ok := tx.(pgx.Tx)
		if !ok {
			return PasswordResetCandidateResult{}, ErrPasswordResetCandidateInvalid
		}
		if err := a.store.CheckLockedUser(
			ctx,
			pgxTx,
			candidate.Operation.UserID,
		); err != nil {
			if errors.Is(err, ErrPasswordResetGateClosed) {
				return PasswordResetCandidateResult{},
					administratorOperationUncertain(
						fmt.Errorf(
							"password reset resend Gate became unresolved: %w",
							err,
						),
					)
			}
			return PasswordResetCandidateResult{}, err
		}
	}
	result, err := a.delegate.BeginOrClassifyCandidate(ctx, tx, candidate)
	if errors.Is(err, ErrPasswordResetGateClosed) {
		return PasswordResetCandidateResult{},
			administratorOperationUncertain(
				fmt.Errorf("password reset resend authority became unresolved: %w", err),
			)
	}
	return result, err
}

func (a passwordResetResendAuthority) resendBeginRequest(
	request idempotency.BeginRequest,
) (idempotency.BeginRequest, error) {
	encoded, err := json.Marshal(struct {
		TargetID   string                           `json:"target_id"`
		TargetRole Role                             `json:"target_role"`
		Purpose    TemporaryCredentialResendPurpose `json:"purpose"`
	}{
		TargetID: a.request.TargetID, TargetRole: a.request.TargetRole,
		Purpose: TemporaryCredentialResendPasswordReset,
	})
	if err != nil {
		return idempotency.BeginRequest{}, err
	}
	request.Scope = []byte(
		a.request.Actor.UserID + "\x00" + temporaryCredentialResendOperation +
			"\x00" + a.request.TargetID + "\x00" +
			string(TemporaryCredentialResendPasswordReset),
	)
	request.Request = encoded
	return request, nil
}

// ResendPasswordResetCredential creates only a durable resend successor.
func (c *PasswordResetCoordinator) ResendPasswordResetCredential(
	ctx context.Context,
	request PasswordResetRequest,
) (PasswordResetResult, error) {
	if c == nil {
		return PasswordResetResult{}, passwordResetInternalError(
			"validate password reset resend service",
			errors.New("service is not initialized"),
		)
	}
	if err := validatePasswordResetRequest(request); err != nil {
		return PasswordResetResult{}, err
	}
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil || tx == nil {
		return PasswordResetResult{}, passwordResetDatabaseError(
			"begin password reset resend candidate transaction",
			err,
		)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()
	resend := *c
	resend.authority = passwordResetResendAuthority{
		delegate: c.authority, request: request,
	}
	if c.authority == c.dependencies.Store {
		resend.authority = passwordResetResendAuthority{
			delegate: c.authority, store: c.dependencies.Store, request: request,
		}
	}
	prepared, err := resend.prepareDurableCandidate(ctx, tx, request)
	if err != nil {
		if errors.Is(err, errTemporaryCredentialResendContextUnavailable) {
			return PasswordResetResult{}, accountStatusInvalidStateError()
		}
		return PasswordResetResult{}, err
	}
	if prepared.operation.RequestKind != PasswordResetResend {
		return PasswordResetResult{}, accountStatusInvalidStateError()
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return PasswordResetResult{}, administratorOperationUncertain(
			fmt.Errorf("commit durable password reset resend candidate: %w", err),
		)
	}
	closed = true
	if !prepared.created {
		return classifyPasswordResetReplay(prepared)
	}
	return resend.dispatchPasswordReset(ctx, prepared)
}

type administratorTemporaryCredentialResendRepository interface {
	AdministratorCreateRepository
	LockTemporaryCredentialResendTarget(
		context.Context,
		pgx.Tx,
		string,
	) (User, TemporaryCredential, error)
	UpdateTemporaryCredential(
		context.Context,
		pgx.Tx,
		TemporaryCredential,
	) error
}

type administratorResendPrepared struct {
	beginRequest idempotency.BeginRequest
	beginRecord  idempotency.Record
	target       User
	current      TemporaryCredential
	pending      TemporaryCredential
	auditEvent   audit.Event
	secret       string
	result       PasswordResetResult
}

func (c *AdministratorCreateCoordinator) resendNewAdministratorCredential(
	ctx context.Context,
	request TemporaryCredentialResendRequest,
	secrets PasswordResetSecretGenerator,
) (PasswordResetResult, error) {
	if c == nil || secrets == nil {
		return PasswordResetResult{}, passwordResetInternalError(
			"validate administrator credential resend service",
			errors.New("service is not initialized"),
		)
	}
	if err := validateAdministratorCredentialResendRequest(request); err != nil {
		return PasswordResetResult{}, err
	}
	repository, ok := c.dependencies.Repository.(administratorTemporaryCredentialResendRepository)
	if !ok {
		return PasswordResetResult{}, passwordResetInternalError(
			"validate administrator credential resend repository",
			errors.New("repository is incomplete"),
		)
	}
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil || tx == nil {
		return PasswordResetResult{}, administratorPersistenceError(
			"begin administrator credential resend transaction",
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
		request.Actor.UserID,
	)
	if err != nil {
		return PasswordResetResult{}, administratorPersistenceError(
			"read administrator credential resend actor",
			err,
		)
	}
	if err := requireCurrentInitialAdministrator(request.Actor, actor); err != nil {
		return PasswordResetResult{}, err
	}
	beginRequest, err := c.administratorResendBeginRequest(request)
	if err != nil {
		return PasswordResetResult{}, err
	}
	existing, found, err := c.dependencies.Idempotency.ResolveExisting(
		ctx,
		tx,
		beginRequest,
	)
	if err != nil {
		return PasswordResetResult{}, administratorIdempotencyError(err)
	}
	if found {
		result, replayErr := replayAdministratorCredentialResend(
			existing,
			request.TargetID,
		)
		if replayErr != nil &&
			existing.Action != idempotency.BeginReplay {
			return PasswordResetResult{}, replayErr
		}
		if err := tx.Commit(ctx); err != nil {
			closed = true
			return PasswordResetResult{}, administratorPersistenceError(
				"commit administrator credential resend replay",
				err,
			)
		}
		closed = true
		return result, replayErr
	}
	if err := c.dependencies.Gate.RequireOpen(ctx); err != nil {
		return PasswordResetResult{}, processingGateError(err)
	}
	target, current, err := repository.LockTemporaryCredentialResendTarget(
		ctx,
		tx,
		request.TargetID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PasswordResetResult{}, accountStatusInvalidStateError()
	}
	if err != nil {
		return PasswordResetResult{}, administratorPersistenceError(
			"read administrator credential resend context",
			err,
		)
	}
	if !administratorCredentialResendEligible(target, current) {
		return PasswordResetResult{}, accountStatusInvalidStateError()
	}
	begin, err := c.dependencies.Idempotency.Begin(ctx, tx, beginRequest)
	if err != nil {
		return PasswordResetResult{}, administratorIdempotencyError(err)
	}
	if begin.Action != idempotency.BeginExecute {
		result, replayErr := replayAdministratorCredentialResend(
			begin,
			request.TargetID,
		)
		if replayErr != nil {
			return PasswordResetResult{}, replayErr
		}
		if err := tx.Commit(ctx); err != nil {
			closed = true
			return PasswordResetResult{}, administratorPersistenceError(
				"commit administrator credential resend replay",
				err,
			)
		}
		closed = true
		return result, nil
	}
	prepared, err := c.prepareAdministratorCredentialResend(
		request,
		target,
		current,
		beginRequest,
		begin.Record,
		secrets,
	)
	if err != nil {
		return PasswordResetResult{}, err
	}
	if err := repository.InsertTemporaryCredential(ctx, tx, prepared.pending); err != nil {
		return PasswordResetResult{}, administratorPersistenceError(
			"insert administrator resend credential",
			err,
		)
	}
	mailCtx, cancelMail := context.WithTimeout(
		ctx,
		c.dependencies.SMTPOperationBudget,
	)
	mailErr := c.dependencies.Mailer.Send(mailCtx, mailer.Message{
		Purpose: mailer.PurposeTemporaryCredentialResend,
		To:      strings.TrimSpace(*target.Email),
		Subject: "Your replacement administrator temporary credential",
		Body:    passwordResetSecretPrefix + prepared.secret,
	})
	cancelMail()
	prepared.secret = ""
	if mailErr != nil {
		reconcileCtx, cancel := c.administratorReconciliationContext(ctx)
		defer cancel()
		closed = true
		_ = tx.Rollback(reconcileCtx)
		return PasswordResetResult{}, c.persistAdministratorCredentialResendFailure(
			reconcileCtx,
			prepared,
			platform.NewPublicError(
				platform.CodeIdentityEmailDeliveryFailed,
				fmt.Errorf("deliver replacement administrator credential: %w", mailErr),
			),
		)
	}
	now := c.dependencies.Clock.Now().UTC()
	superseded, _, err := ActivateTemporaryCredentialReplacement(
		current,
		prepared.pending,
		now,
	)
	if err != nil {
		return PasswordResetResult{}, c.markAdministratorCredentialResendUncertain(
			ctx, tx, &closed, prepared, err,
		)
	}
	if err := repository.UpdateTemporaryCredential(ctx, tx, superseded); err != nil {
		return PasswordResetResult{}, c.markAdministratorCredentialResendUncertain(
			ctx, tx, &closed, prepared,
			fmt.Errorf("supersede administrator credential: %w", err),
		)
	}
	if err := repository.ActivateTemporaryCredential(
		ctx,
		tx,
		prepared.pending.ID,
		now,
	); err != nil {
		return PasswordResetResult{}, c.markAdministratorCredentialResendUncertain(
			ctx, tx, &closed, prepared,
			fmt.Errorf("activate administrator resend credential: %w", err),
		)
	}
	prepared.result.DeliveryStatus = passwordResetDeliveryAccepted
	if err := c.dependencies.Audit.Append(ctx, tx, prepared.auditEvent); err != nil {
		return PasswordResetResult{}, c.markAdministratorCredentialResendUncertain(
			ctx, tx, &closed, prepared,
			fmt.Errorf("append administrator credential resend audit: %w", err),
		)
	}
	if err := c.dependencies.Idempotency.Complete(
		ctx,
		tx,
		idempotencyCompletion(
			prepared.beginRecord,
			idempotency.StatusSucceeded,
			passwordResetOutcomeSucceeded,
			temporaryCredentialResendSuccessSnapshot(prepared.result),
			now,
		),
	); err != nil {
		return PasswordResetResult{}, c.markAdministratorCredentialResendUncertain(
			ctx, tx, &closed, prepared,
			fmt.Errorf("complete administrator credential resend idempotency: %w", err),
		)
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		reconcileCtx, cancel := c.administratorReconciliationContext(ctx)
		defer cancel()
		return PasswordResetResult{}, c.markCreateUncertain(
			reconcileCtx,
			prepared.beginRequest,
			prepared.beginRecord,
			fmt.Errorf("commit administrator credential resend: %w", err),
		)
	}
	closed = true
	return prepared.result, nil
}

func validateAdministratorCredentialResendRequest(
	request TemporaryCredentialResendRequest,
) error {
	if request.Purpose != TemporaryCredentialResendNewAdministrator ||
		request.TargetRole != RoleAdmin {
		return passwordResetForbiddenError()
	}
	if err := requireInitialAdministrator(request.Actor); err != nil {
		return err
	}
	if !validAccountStatusUserID(request.TargetID) {
		return userNotFoundError()
	}
	if strings.TrimSpace(request.TraceID) == "" {
		return passwordResetInternalError(
			"validate administrator credential resend trace ID",
			errors.New("trace ID is required"),
		)
	}
	if !validIdempotencyKey(request.IdempotencyKey) {
		return platform.NewPublicError(
			platform.CodeIdentityIdempotencyKeyRequired,
			errors.New("administrator credential resend idempotency key is invalid"),
		)
	}
	return nil
}

func administratorCredentialResendEligible(
	target User,
	current TemporaryCredential,
) bool {
	return target.ID == current.UserID &&
		target.Role == RoleAdmin &&
		!target.IsInitialAdmin &&
		target.Status == UserStatusActive &&
		target.CredentialMode == CredentialModeTemporary &&
		!target.RecoveryRequired &&
		target.DeletedAt == nil &&
		target.Email != nil &&
		strings.TrimSpace(*target.Email) != "" &&
		current.Purpose == TemporaryCredentialNewAdministrator &&
		current.State == TemporaryCredentialActive &&
		TemporaryCredentialCurrent(current)
}

func (c *AdministratorCreateCoordinator) administratorResendBeginRequest(
	request TemporaryCredentialResendRequest,
) (idempotency.BeginRequest, error) {
	encoded, err := json.Marshal(struct {
		TargetID string                           `json:"target_id"`
		Purpose  TemporaryCredentialResendPurpose `json:"purpose"`
	}{
		TargetID: request.TargetID,
		Purpose:  TemporaryCredentialResendNewAdministrator,
	})
	if err != nil {
		return idempotency.BeginRequest{}, passwordResetInternalError(
			"encode administrator credential resend request",
			err,
		)
	}
	now := c.dependencies.Clock.Now().UTC()
	return idempotency.BeginRequest{
		Scope: []byte(
			request.Actor.UserID + "\x00" + temporaryCredentialResendOperation +
				"\x00" + request.TargetID + "\x00" +
				string(TemporaryCredentialResendNewAdministrator),
		),
		Key: request.IdempotencyKey, Request: encoded,
		DigestKeys:  c.dependencies.IdempotencyKeys,
		LockedUntil: now.Add(c.dependencies.IdempotencyLock),
		ExpiresAt:   now.Add(c.dependencies.IdempotencyTTL),
		Now:         now,
	}, nil
}

func replayAdministratorCredentialResend(
	begin idempotency.BeginResult,
	targetID string,
) (PasswordResetResult, error) {
	switch begin.Action {
	case idempotency.BeginReplay:
		return replayPasswordReset(begin.Record, targetID)
	case idempotency.BeginInProgress:
		return PasswordResetResult{}, platform.NewPublicError(
			platform.CodeIdentityIdempotencyInProgress,
			errors.New("administrator credential resend is already processing"),
		)
	case idempotency.BeginOperationUncertain:
		return PasswordResetResult{}, administratorOperationUncertain(
			errors.New("administrator credential resend requires controlled review"),
		)
	default:
		return PasswordResetResult{}, passwordResetInternalError(
			"classify administrator credential resend",
			errors.New("idempotency action is invalid"),
		)
	}
}

func (c *AdministratorCreateCoordinator) prepareAdministratorCredentialResend(
	request TemporaryCredentialResendRequest,
	target User,
	current TemporaryCredential,
	beginRequest idempotency.BeginRequest,
	beginRecord idempotency.Record,
	secrets PasswordResetSecretGenerator,
) (administratorResendPrepared, error) {
	secret, err := secrets.Generate()
	if err != nil {
		return administratorResendPrepared{}, passwordResetInternalError(
			"generate administrator resend credential",
			err,
		)
	}
	if err := c.dependencies.PasswordPolicy.Validate(secret); err != nil {
		return administratorResendPrepared{}, passwordResetInternalError(
			"validate administrator resend credential",
			err,
		)
	}
	credentialID, err := c.dependencies.IDs.NewID()
	if err != nil {
		return administratorResendPrepared{}, identifierGenerationError(err)
	}
	auditID, err := c.dependencies.IDs.NewID()
	if err != nil {
		return administratorResendPrepared{}, identifierGenerationError(err)
	}
	now := beginRecord.CreatedAt.UTC()
	if now.IsZero() {
		return administratorResendPrepared{}, passwordResetInternalError(
			"prepare administrator resend credential",
			errors.New("durable claim creation time is missing"),
		)
	}
	pending, err := NewTemporaryCredential(
		c.dependencies.PasswordHasher,
		TemporaryCredentialInput{
			ID: credentialID, UserID: target.ID,
			Purpose:    TemporaryCredentialNewAdministrator,
			Generation: current.Generation + 1, Secret: secret,
			DeliveredToEmail: strings.TrimSpace(*target.Email),
			IssuedAt:         now,
		},
	)
	if err != nil {
		return administratorResendPrepared{}, passwordResetInternalError(
			"create administrator resend credential",
			err,
		)
	}
	actorID, actorRole := request.Actor.UserID, request.Actor.Role
	return administratorResendPrepared{
		beginRequest: beginRequest, beginRecord: beginRecord,
		target: target, current: current, pending: pending, secret: secret,
		result: PasswordResetResult{
			UserID: target.ID, TemporaryCredentialExpiresAt: pending.ExpiresAt,
		},
		auditEvent: audit.Event{
			ID: auditID, OccurredAt: now, ActorKind: audit.ActorUser,
			ActorUserID: &actorID, ActorRole: &actorRole,
			Action:     temporaryCredentialResendOperation,
			TargetType: "user", TargetID: target.ID,
			Outcome: passwordResetOutcomeSucceeded, TraceID: request.TraceID,
			Metadata: map[string]any{
				"credential_purpose": TemporaryCredentialNewAdministrator,
				"delivery_status":    passwordResetDeliveryAccepted,
			},
		},
	}, nil
}

func (c *AdministratorCreateCoordinator) persistAdministratorCredentialResendFailure(
	ctx context.Context,
	prepared administratorResendPrepared,
	cause error,
) error {
	tx, err := c.dependencies.Transactions.Begin(ctx)
	if err != nil || tx == nil {
		return administratorPersistenceError(
			"begin administrator credential resend failure",
			errors.Join(cause, err),
		)
	}
	closed := false
	defer func() {
		if !closed {
			_ = tx.Rollback(ctx)
		}
	}()
	begin, err := c.dependencies.Idempotency.Begin(ctx, tx, prepared.beginRequest)
	if err != nil {
		return administratorIdempotencyError(err)
	}
	if begin.Action != idempotency.BeginExecute {
		return administratorOperationUncertain(errors.Join(
			cause,
			errors.New("administrator resend failure was not executable"),
		))
	}
	now := c.dependencies.Clock.Now().UTC()
	if err := c.dependencies.Idempotency.Complete(
		ctx,
		tx,
		idempotencyCompletion(
			begin.Record,
			idempotency.StatusFailed,
			string(platform.CodeIdentityEmailDeliveryFailed),
			temporaryCredentialResendFailureSnapshot(),
			now,
		),
	); err != nil {
		return administratorPersistenceError(
			"complete administrator credential resend failure",
			errors.Join(cause, err),
		)
	}
	event := prepared.auditEvent
	event.OccurredAt = now
	event.Outcome = passwordResetOutcomeFailed
	code := string(platform.CodeIdentityEmailDeliveryFailed)
	event.ErrorCode = &code
	delete(event.Metadata, "delivery_status")
	if err := c.dependencies.Audit.Append(ctx, tx, event); err != nil {
		return administratorPersistenceError(
			"append administrator credential resend failure audit",
			errors.Join(cause, err),
		)
	}
	if err := tx.Commit(ctx); err != nil {
		closed = true
		return administratorPersistenceError(
			"commit administrator credential resend failure",
			errors.Join(cause, err),
		)
	}
	closed = true
	return platform.NewPublicError(platform.CodeIdentityEmailDeliveryFailed, cause)
}

func (c *AdministratorCreateCoordinator) markAdministratorCredentialResendUncertain(
	ctx context.Context,
	tx pgx.Tx,
	closed *bool,
	prepared administratorResendPrepared,
	cause error,
) error {
	reconcileCtx, cancel := c.administratorReconciliationContext(ctx)
	defer cancel()
	*closed = true
	_ = tx.Rollback(reconcileCtx)
	return c.markCreateUncertain(
		reconcileCtx,
		prepared.beginRequest,
		prepared.beginRecord,
		cause,
	)
}

func temporaryCredentialResendSuccessSnapshot(
	result PasswordResetResult,
) idempotency.ResponseSnapshot {
	return idempotency.ResponseSnapshot{
		Outcome:    passwordResetOutcomeSucceeded,
		ResourceID: result.UserID,
	}
}

func temporaryCredentialResendFailureSnapshot() idempotency.ResponseSnapshot {
	return idempotency.ResponseSnapshot{
		Outcome:   passwordResetOutcomeFailed,
		ErrorCode: string(platform.CodeIdentityEmailDeliveryFailed),
	}
}

// LockTemporaryCredentialResendTarget locks the administrator and its latest credential.
func (PostgresAdministratorRepository) LockTemporaryCredentialResendTarget(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
) (User, TemporaryCredential, error) {
	if tx == nil {
		return User{}, TemporaryCredential{}, errors.New(
			"administrator credential resend transaction is required",
		)
	}
	var user User
	if err := tx.QueryRow(ctx, `
		SELECT id,email,role,is_initial_admin,status,credential_mode,
			recovery_required,deleted_at
		FROM users
		WHERE id=$1
		FOR UPDATE
	`, userID).Scan(
		&user.ID, &user.Email, &user.Role, &user.IsInitialAdmin,
		&user.Status, &user.CredentialMode, &user.RecoveryRequired,
		&user.DeletedAt,
	); err != nil {
		return User{}, TemporaryCredential{}, err
	}
	var credential TemporaryCredential
	err := tx.QueryRow(ctx, `
		SELECT id,user_id,purpose,generation,auth_generation,source_operation_id,
			state,secret_digest,delivered_to_email,issued_at,expires_at,
			superseded_at,consumed_at,invalidated_at
		FROM temporary_credentials
		WHERE user_id=$1 AND purpose='new_administrator'
		ORDER BY generation DESC
		LIMIT 1
		FOR UPDATE
	`, userID).Scan(
		&credential.ID, &credential.UserID, &credential.Purpose,
		&credential.Generation, &credential.AuthGeneration,
		&credential.SourceOperationID, &credential.State,
		&credential.SecretDigest, &credential.DeliveredToEmail,
		&credential.IssuedAt, &credential.ExpiresAt,
		&credential.SupersededAt, &credential.ConsumedAt,
		&credential.InvalidatedAt,
	)
	return user, credential, err
}

// UpdateTemporaryCredential persists the exact superseded administrator credential.
func (PostgresAdministratorRepository) UpdateTemporaryCredential(
	ctx context.Context,
	tx pgx.Tx,
	credential TemporaryCredential,
) error {
	if tx == nil || credential.State != TemporaryCredentialSuperseded ||
		credential.SupersededAt == nil {
		return errors.New("administrator credential replacement is invalid")
	}
	tag, err := tx.Exec(ctx, `
		UPDATE temporary_credentials
		SET state='superseded',superseded_at=$1
		WHERE id=$2 AND user_id=$3 AND purpose='new_administrator'
			AND generation=$4 AND state='active'
			AND superseded_at IS NULL AND consumed_at IS NULL
			AND invalidated_at IS NULL
	`,
		*credential.SupersededAt,
		credential.ID,
		credential.UserID,
		credential.Generation,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("administrator temporary credential state changed")
	}
	return nil
}
