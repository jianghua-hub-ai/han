package identity

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"han/internal/platform"
	"han/internal/session"
)

// RegistrationService is the registration capability consumed by Handler.
type RegistrationService interface {
	Register(context.Context, RegistrationRequest) (RegisteredUser, error)
}

// LoginService is the authentication capability consumed by Handler.
type LoginService interface {
	Login(context.Context, LoginRequest) (LoginResult, error)
}

// PasswordChangeService is the protected onboarding capability consumed by Handler.
type PasswordChangeService interface {
	ChangePassword(context.Context, PasswordChangeRequest) (PasswordChangeResult, error)
	AuditPasswordChangeFailure(context.Context, PasswordChangeFailureAuditRequest) error
}

// AdministratorCreateService is the initial-admin-only creation capability.
type AdministratorCreateService interface {
	CreateAdministrator(context.Context, CreateAdministratorRequest) (CreatedAdministrator, error)
}

// AdministratorPrivacyService is the current privacy acknowledgement capability.
type AdministratorPrivacyService interface {
	AcknowledgePrivacy(
		context.Context,
		AdministratorPrivacyAcknowledgementRequest,
	) (AdministratorOnboardingResult, error)
}

// AdministratorInformationNoticeService exposes the current declaration.
type AdministratorInformationNoticeService interface {
	CurrentAdministratorInformationNotice(
		context.Context,
		session.Principal,
	) (AdministratorInformationNotice, error)
}

// UserQueryService is the minimum-disclosure governance query capability.
type UserQueryService interface {
	List(context.Context, UserQueryListRequest) (UserQueryPage, error)
	Detail(context.Context, UserQueryDetailRequest) (UserGovernanceDetail, error)
}

// AccountStatusService is the account governance mutation capability.
type AccountStatusService interface {
	SetAccountStatus(context.Context, AccountStatusRequest) (AccountStatusResult, error)
}

// AdministratorGovernanceService is the administrator role revocation capability.
type AdministratorGovernanceService interface {
	RevokeAdministrator(
		context.Context,
		AdministratorRevokeRequest,
	) (AdministratorRevokeResult, error)
}

// CanonicalDeleteService is the irreversible account deletion capability.
type CanonicalDeleteService interface {
	CanonicalDelete(
		context.Context,
		CanonicalDeleteRequest,
	) (CanonicalDeleteResult, error)
}

// PasswordResetService is the administrator credential reset capability.
type PasswordResetService interface {
	ResetPassword(context.Context, PasswordResetRequest) (PasswordResetResult, error)
}

// RegistrationSourceEvidence is the privacy evidence derived at the trusted HTTP boundary.
type RegistrationSourceEvidence struct {
	Kind        string
	Fingerprint string
}

// RegistrationSourceProvider derives privacy evidence without exposing raw request data.
type RegistrationSourceProvider interface {
	Evidence(*http.Request) (RegistrationSourceEvidence, error)
}

const (
	trustedSourceKind       = "network_source_hmac_sha256_v1"
	maxForwardedSourceHops  = 16
	sourceFingerprintDomain = "han.identity.source-fingerprint.v1"
)

// TrustedSourceEvidenceProvider derives one versioned source fingerprint.
type TrustedSourceEvidenceProvider struct {
	key             []byte
	keyVersion      string
	trustedPrefixes []netip.Prefix
}

// NewTrustedSourceEvidenceProvider creates a strict proxy-aware source provider.
func NewTrustedSourceEvidenceProvider(
	key []byte,
	keyVersion string,
	trustedPrefixes []netip.Prefix,
) (*TrustedSourceEvidenceProvider, error) {
	originalVersion := keyVersion
	keyVersion = strings.TrimSpace(keyVersion)
	if originalVersion != keyVersion || len(key) < 32 || len(key) > 64 ||
		!validSourceKeyVersion(keyVersion) || len(trustedPrefixes) > 64 {
		return nil, errors.New("source evidence configuration is invalid")
	}
	prefixes := append([]netip.Prefix(nil), trustedPrefixes...)
	for index, prefix := range prefixes {
		if !prefix.IsValid() || prefix != prefix.Masked() ||
			prefix.Bits() == 0 ||
			(prefix.Addr().Is4In6() && prefix.Bits() < 96) {
			return nil, errors.New("source evidence trusted prefix is invalid")
		}
		prefixes[index] = canonicalSourcePrefix(prefix)
		if prefixes[index].Bits() == 0 {
			return nil, errors.New("source evidence trusted prefix is invalid")
		}
		for previous := 0; previous < index; previous++ {
			if prefixes[previous] == prefixes[index] {
				return nil, errors.New("source evidence trusted prefix is duplicated")
			}
		}
	}
	return &TrustedSourceEvidenceProvider{
		key:             append([]byte(nil), key...),
		keyVersion:      keyVersion,
		trustedPrefixes: prefixes,
	}, nil
}

func validSourceKeyVersion(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if (char < 'A' || char > 'Z') &&
			(char < 'a' || char > 'z') &&
			(char < '0' || char > '9') &&
			char != '_' {
			return false
		}
	}
	return true
}

// Evidence returns only a keyed digest, never a raw address.
func (p *TrustedSourceEvidenceProvider) Evidence(
	request *http.Request,
) (RegistrationSourceEvidence, error) {
	if p == nil || len(p.key) == 0 || request == nil {
		return RegistrationSourceEvidence{}, errors.New("source evidence is unavailable")
	}
	peer, err := parseRemoteSourceAddress(request.RemoteAddr)
	if err != nil {
		return RegistrationSourceEvidence{}, errors.New("source evidence is invalid")
	}
	client := peer
	if p.trusted(peer) {
		forwarded, err := parseForwardedSourceAddresses(request.Header.Values("X-Forwarded-For"))
		if err != nil || len(forwarded) == 0 {
			return RegistrationSourceEvidence{}, errors.New("source evidence is invalid")
		}
		chain := append(forwarded, peer)
		found := false
		for index := len(chain) - 1; index >= 0; index-- {
			if p.trusted(chain[index]) {
				continue
			}
			client = chain[index]
			found = true
			break
		}
		if !found {
			return RegistrationSourceEvidence{}, errors.New("source evidence is invalid")
		}
	}
	mac := hmac.New(sha256.New, p.key)
	writeSourceFingerprintField(mac, []byte(sourceFingerprintDomain))
	writeSourceFingerprintField(mac, client.AsSlice())
	return RegistrationSourceEvidence{
		Kind:        trustedSourceKind,
		Fingerprint: p.keyVersion + ":" + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)),
	}, nil
}

func writeSourceFingerprintField(
	mac interface{ Write([]byte) (int, error) },
	value []byte,
) {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(value)))
	_, _ = mac.Write(length[:])
	_, _ = mac.Write(value)
}

func (p *TrustedSourceEvidenceProvider) trusted(address netip.Addr) bool {
	for _, prefix := range p.trustedPrefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func parseRemoteSourceAddress(value string) (netip.Addr, error) {
	host, port, err := net.SplitHostPort(value)
	if err != nil || port == "" || strings.Contains(host, "%") {
		return netip.Addr{}, errors.New("invalid remote address")
	}
	parsedPort, err := strconv.ParseUint(port, 10, 16)
	if err != nil || parsedPort == 0 {
		return netip.Addr{}, errors.New("invalid remote address")
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, errors.New("invalid remote address")
	}
	return address.Unmap(), nil
}

func parseForwardedSourceAddresses(values []string) ([]netip.Addr, error) {
	var addresses []netip.Addr
	for _, value := range values {
		for _, raw := range strings.Split(value, ",") {
			raw = strings.TrimSpace(raw)
			if raw == "" || strings.ContainsAny(raw, "%[]") {
				return nil, errors.New("invalid forwarded address")
			}
			address, err := netip.ParseAddr(raw)
			if err != nil {
				return nil, errors.New("invalid forwarded address")
			}
			addresses = append(addresses, address.Unmap())
			if len(addresses) > maxForwardedSourceHops {
				return nil, errors.New("forwarded address chain is too long")
			}
		}
	}
	return addresses, nil
}

func canonicalSourcePrefix(prefix netip.Prefix) netip.Prefix {
	address := prefix.Addr()
	if address.Is4In6() && prefix.Bits() >= 96 {
		return netip.PrefixFrom(address.Unmap(), prefix.Bits()-96).Masked()
	}
	return prefix.Masked()
}

// Handler adapts identity operations to HTTP.
type Handler struct {
	registration              RegistrationService
	login                     LoginService
	source                    RegistrationSourceProvider
	maxBodyBytes              int64
	loginMaxBodyBytes         int64
	loginCoordinator          *LoginAttemptCoordinator
	loginObserver             LoginOutcomeObserver
	loginObserveFor           time.Duration
	passwordChange            PasswordChangeService
	passwordMaxBytes          int64
	userQuery                 UserQueryService
	accountStatus             AccountStatusService
	administratorGovernance   AdministratorGovernanceService
	canonicalDelete           CanonicalDeleteService
	passwordReset             PasswordResetService
	temporaryCredentialResend TemporaryCredentialResendService
	administratorCreate       AdministratorCreateService
	administratorPrivacy      AdministratorPrivacyService
	administratorNotices      AdministratorInformationNoticeService
	administratorSource       RegistrationSourceProvider
	administratorCreateBytes  int64
	administratorPrivacyBytes int64
}

// LoginHandlerConfig contains the mandatory bounded login observation wiring.
type LoginHandlerConfig struct {
	Coordinator       *LoginAttemptCoordinator
	Observer          LoginOutcomeObserver
	ObservationBudget time.Duration
}

// NewHandler creates an identity HTTP handler with an explicit request body limit.
func NewHandler(
	registration RegistrationService,
	source RegistrationSourceProvider,
	maxBodyBytes int64,
) (*Handler, error) {
	if registration == nil {
		return nil, errors.New("registration service is required")
	}
	if source == nil {
		return nil, errors.New("registration source provider is required")
	}
	if maxBodyBytes <= 0 {
		return nil, errors.New("registration request body limit must be positive")
	}
	return &Handler{
		registration: registration,
		source:       source,
		maxBodyBytes: maxBodyBytes,
	}, nil
}

// NewLoginHandler creates the login HTTP boundary with an explicit request body limit.
func NewLoginHandler(
	login LoginService,
	source RegistrationSourceProvider,
	maxBodyBytes int64,
	config LoginHandlerConfig,
) (*Handler, error) {
	if login == nil {
		return nil, errors.New("login service is required")
	}
	if source == nil {
		return nil, errors.New("login source provider is required")
	}
	if maxBodyBytes <= 0 {
		return nil, errors.New("login request body limit must be positive")
	}
	if config.Coordinator == nil || config.Observer == nil ||
		config.ObservationBudget <= 0 {
		return nil, errors.New("login observation dependencies are incomplete")
	}
	return &Handler{
		login:             login,
		source:            source,
		loginMaxBodyBytes: maxBodyBytes,
		loginCoordinator:  config.Coordinator,
		loginObserver:     config.Observer,
		loginObserveFor:   config.ObservationBudget,
	}, nil
}

// NewPasswordChangeHandler creates the protected password-change HTTP boundary.
func NewPasswordChangeHandler(
	service PasswordChangeService,
	maxBodyBytes int64,
) (*Handler, error) {
	if service == nil {
		return nil, errors.New("password change service is required")
	}
	if maxBodyBytes <= 0 {
		return nil, errors.New("password change request body limit must be positive")
	}
	return &Handler{
		passwordChange:   service,
		passwordMaxBytes: maxBodyBytes,
	}, nil
}

// NewUserQueryHandler creates the protected governance query HTTP boundary.
func NewUserQueryHandler(service UserQueryService) (*Handler, error) {
	if service == nil {
		return nil, errors.New("identity user query service is required")
	}
	return &Handler{userQuery: service}, nil
}

// NewAccountStatusHandler creates the protected account governance boundary.
func NewAccountStatusHandler(service AccountStatusService) (*Handler, error) {
	if service == nil {
		return nil, errors.New("identity account status service is required")
	}
	return &Handler{accountStatus: service}, nil
}

// NewCanonicalDeleteHandler creates the protected canonical deletion boundary.
func NewCanonicalDeleteHandler(service CanonicalDeleteService) (*Handler, error) {
	if service == nil {
		return nil, errors.New("identity canonical delete service is required")
	}
	return &Handler{canonicalDelete: service}, nil
}

// NewAdministratorGovernanceHandler creates the administrator revoke boundary.
func NewAdministratorGovernanceHandler(
	service AdministratorGovernanceService,
) (*Handler, error) {
	if service == nil {
		return nil, errors.New("identity administrator governance service is required")
	}
	return &Handler{administratorGovernance: service}, nil
}

// NewPasswordResetHandler creates the protected password reset boundary.
func NewPasswordResetHandler(service PasswordResetService) (*Handler, error) {
	if service == nil {
		return nil, errors.New("identity password reset service is required")
	}
	return &Handler{passwordReset: service}, nil
}

// NewTemporaryCredentialResendHandler creates the protected resend boundary.
func NewTemporaryCredentialResendHandler(
	service TemporaryCredentialResendService,
) (*Handler, error) {
	if service == nil {
		return nil, errors.New("identity temporary credential resend service is required")
	}
	return &Handler{temporaryCredentialResend: service}, nil
}

// NewAdministratorHandler creates the administrator creation and onboarding boundary.
func NewAdministratorHandler(
	create AdministratorCreateService,
	privacy AdministratorPrivacyService,
	notices AdministratorInformationNoticeService,
	source RegistrationSourceProvider,
	createMaxBodyBytes int64,
	privacyMaxBodyBytes int64,
) (*Handler, error) {
	if create == nil || privacy == nil || notices == nil || source == nil {
		return nil, errors.New("administrator handler dependencies are incomplete")
	}
	if createMaxBodyBytes <= 0 || privacyMaxBodyBytes <= 0 {
		return nil, errors.New("administrator request body limits must be positive")
	}
	return &Handler{
		administratorCreate: create, administratorPrivacy: privacy,
		administratorNotices: notices, administratorSource: source,
		administratorCreateBytes:  createMaxBodyBytes,
		administratorPrivacyBytes: privacyMaxBodyBytes,
	}, nil
}

// Register handles POST /api/v1/users.
func (h *Handler) Register(response http.ResponseWriter, request *http.Request) {
	var input RegistrationRequest
	if err := platform.DecodeJSON(request, h.maxBodyBytes, &input); err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}

	source, err := h.source.Evidence(request)
	if err != nil {
		_ = platform.WriteError(
			response,
			request,
			platform.NewPublicError(platform.CodePlatformInternalError, err),
		)
		return
	}
	input.SourceKind = source.Kind
	input.SourceFingerprint = source.Fingerprint

	user, err := h.registration.Register(request.Context(), input)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}

	_ = platform.WriteJSON(response, http.StatusCreated, registrationResponse{
		ID:                 user.ID,
		Username:           user.Username,
		Role:               user.Role,
		Status:             user.Status,
		MustChangePassword: false,
		CreatedAt:          user.CreatedAt.UTC().Format(time.RFC3339),
	})
}

// Login handles POST /api/v1/sessions.
func (h *Handler) Login(response http.ResponseWriter, request *http.Request) {
	startedAt := time.Now()
	attempt, beginErr := h.loginCoordinator.Begin()
	if beginErr != nil {
		publicErr := platform.NewPublicError(platform.CodePlatformInternalError, beginErr)
		writeErr := platform.WriteError(response, request, publicErr)
		observation := (*LoginAttemptContext)(nil).observationSnapshot()
		observation.FailureStage = LoginFailureStageTransactionBegin
		observation.ReasonBucket = "request_context"
		observation.ErrorCode = string(platform.CodePlatformInternalError)
		h.observeLogin(request, startedAt, observation, writeErr)
		return
	}

	var input LoginRequest
	if err := platform.DecodeJSON(request, h.loginMaxBodyBytes, &input); err != nil {
		writeErr := platform.WriteError(response, request, err)
		observation := attempt.observationSnapshot()
		observation.Outcome = LoginOutcomeInvalidRequest
		observation.ErrorCode = string(platform.PublicCode(err))
		h.observeLogin(request, startedAt, observation, writeErr)
		return
	}

	source, err := h.source.Evidence(request)
	if err != nil {
		publicErr := platform.NewPublicError(platform.CodePlatformInternalError, err)
		traceID, _ := platform.TraceIDFromContext(request.Context())
		event := loginFailureAuditEvent(
			attempt,
			traceID,
			nil,
			"",
			LoginFailureStageSourceEvidence,
			publicErr,
		)
		persisted, _ := attempt.recordOnce(
			request.Context(),
			loginAttemptPreService,
			LoginFailureAuditRequest{
				Mode: LoginFailureAuditInsertFailure, Event: event,
			},
		)
		attempt.updateObservation(func(observation *LoginObservation) {
			observation.Outcome = LoginOutcomeInternalFailure
			observation.FailureStage = LoginFailureStageSourceEvidence
			observation.ReasonBucket = "request_context"
			observation.ErrorCode = string(platform.CodePlatformInternalError)
			observation.AuditPersisted = persisted
		})
		writeErr := platform.WriteError(response, request, publicErr)
		h.observeLogin(request, startedAt, attempt.observationSnapshot(), writeErr)
		return
	}
	input.SourceFingerprint = source.Fingerprint
	input.Attempt = attempt
	if cookie, cookieErr := request.Cookie(platform.SessionCookieName); cookieErr == nil {
		input.ExistingSessionToken = cookie.Value
	}
	if err := attempt.enterService(); err != nil {
		publicErr := platform.NewPublicError(platform.CodePlatformInternalError, err)
		attempt.updateObservation(func(observation *LoginObservation) {
			observation.Outcome = LoginOutcomeInternalFailure
			observation.FailureStage = LoginFailureStageUnknown
			observation.ReasonBucket = "request_context"
			observation.ErrorCode = string(platform.CodePlatformInternalError)
			observation.AuditPersisted = LoginAuditPersistedUnknown
		})
		writeErr := platform.WriteError(response, request, publicErr)
		h.observeLogin(request, startedAt, attempt.observationSnapshot(), writeErr)
		return
	}

	result, err := h.login.Login(request.Context(), input)
	if err != nil {
		if retryAfter, ok := LoginRateLimitRetryAfter(err); ok {
			seconds := (retryAfter + time.Second - 1) / time.Second
			response.Header().Set("Retry-After", strconv.FormatInt(int64(seconds), 10))
		}
		writeErr := platform.WriteError(response, request, err)
		observation := attempt.observationSnapshot()
		observation.Outcome = loginOutcomeForError(err)
		observation.ErrorCode = string(platform.PublicCode(err))
		h.observeLogin(request, startedAt, observation, writeErr)
		return
	}
	if err := platform.SetSessionCookie(response, result.SessionToken); err != nil {
		publicErr := platform.NewPublicError(platform.CodePlatformInternalError, err)
		attempt.updateObservation(func(observation *LoginObservation) {
			observation.Outcome = LoginOutcomeInternalFailure
			observation.FailureStage = LoginFailureStageCookieOutput
			observation.ReasonBucket = "response"
			observation.ErrorCode = string(platform.CodePlatformInternalError)
			observation.TransactionOutcome = LoginTransactionCommitted
			observation.StateChanged = LoginStateSessionCommitted
			observation.AuditPersisted = LoginAuditPersistedYes
		})
		writeErr := platform.WriteError(
			response,
			request,
			publicErr,
		)
		h.observeLogin(request, startedAt, attempt.observationSnapshot(), writeErr)
		return
	}

	writeErr := platform.WriteJSON(response, http.StatusOK, loginResponse{
		User: loginUserResponse{
			ID:                     result.User.ID,
			Username:               result.User.Username,
			Role:                   result.User.Role,
			Status:                 result.User.Status,
			IsInitialAdmin:         result.User.IsInitialAdmin,
			MustChangePassword:     result.User.MustChangePassword,
			MustAcknowledgePrivacy: result.User.MustAcknowledgePrivacy,
		},
		CSRFToken: result.CSRFToken,
	})
	observation := attempt.observationSnapshot()
	observation.Outcome = LoginOutcomeSuccess
	observation.ErrorCode = "none"
	h.observeLogin(request, startedAt, observation, writeErr)
}

// ChangePassword handles PUT /api/v1/users/me/password.
func (h *Handler) ChangePassword(
	response http.ResponseWriter,
	request *http.Request,
) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("password change principal is missing"),
		))
		return
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("password change trace ID is missing"),
		))
		return
	}
	if !principal.MustChangePassword && !principal.IsInitialAdmin {
		h.writeAuditedPasswordChangeFailure(
			response,
			request,
			principal,
			traceID,
			platform.NewPublicError(
				platform.CodeAuthForbidden,
				errors.New("principal is not allowed to change password"),
			),
		)
		return
	}

	var input passwordChangeRequest
	if err := platform.DecodeJSON(request, h.passwordMaxBytes, &input); err != nil {
		h.writeAuditedPasswordChangeFailure(response, request, principal, traceID, err)
		return
	}
	cookie, err := request.Cookie(platform.SessionCookieName)
	if err != nil || cookie.Value == "" {
		h.writeAuditedPasswordChangeFailure(
			response,
			request,
			principal,
			traceID,
			platform.NewPublicError(
				platform.CodeAuthSessionRequired,
				errors.New("password change session cookie is missing"),
			),
		)
		return
	}
	result, err := h.passwordChange.ChangePassword(
		request.Context(),
		PasswordChangeRequest{
			Principal: principal, CurrentSessionToken: cookie.Value,
			CurrentPassword: input.CurrentPassword, NewPassword: input.NewPassword,
			TraceID: traceID,
		},
	)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	if err := platform.SetSessionCookie(response, result.SessionToken); err != nil {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			err,
		))
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, loginResponse{
		User: loginUserResponse{
			ID: result.User.ID, Username: result.User.Username,
			Role: result.User.Role, Status: result.User.Status,
			IsInitialAdmin:         result.User.IsInitialAdmin,
			MustChangePassword:     false,
			MustAcknowledgePrivacy: result.MustAcknowledgePrivacy,
		},
		CSRFToken: result.CSRFToken,
	})
}

func writeAdministratorCreateError(
	response http.ResponseWriter,
	request *http.Request,
	err error,
) bool {
	var responseErr *administratorResponseError
	if !errors.As(err, &responseErr) || responseErr.snapshot.Body == nil {
		return false
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		return false
	}
	fieldErrors := make([]platform.FieldError, len(responseErr.snapshot.Body.FieldErrors))
	for index, fieldError := range responseErr.snapshot.Body.FieldErrors {
		fieldErrors[index] = platform.FieldError{
			Field: fieldError.Field,
			Code:  fieldError.Code,
		}
	}
	response.Header().Set(platform.TraceHeader, traceID)
	_ = platform.WriteJSON(response, responseErr.snapshot.HTTPStatus, struct {
		Code        string                `json:"code"`
		Message     string                `json:"message"`
		FieldErrors []platform.FieldError `json:"field_errors"`
		TraceID     string                `json:"trace_id"`
	}{
		Code: responseErr.snapshot.Body.Code, Message: responseErr.snapshot.Body.Message,
		FieldErrors: fieldErrors, TraceID: traceID,
	})
	return true
}

// CurrentAdministratorInformationNotice handles the current declaration read.
func (h *Handler) CurrentAdministratorInformationNotice(
	response http.ResponseWriter,
	request *http.Request,
) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("administrator information notice principal is missing"),
		))
		return
	}
	notice, err := h.administratorNotices.CurrentAdministratorInformationNotice(
		request.Context(),
		principal,
	)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, administratorInformationNoticeResponse{
		Version: notice.Version, EffectiveAt: notice.EffectiveAt.UTC().Format(time.RFC3339),
		Content: notice.Content, ContentHash: notice.ContentHash,
	})
}

// CreateAdministrator handles POST /api/v1/admin/administrators.
func (h *Handler) CreateAdministrator(
	response http.ResponseWriter,
	request *http.Request,
) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("administrator creation principal is missing"),
		))
		return
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("administrator creation trace ID is missing"),
		))
		return
	}
	keys := request.Header.Values("Idempotency-Key")
	if len(keys) != 1 {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeIdentityIdempotencyKeyRequired,
			errors.New("administrator creation requires one idempotency key"),
		))
		return
	}
	var input createAdministratorRequest
	if err := platform.DecodeJSON(
		request,
		h.administratorCreateBytes,
		&input,
	); err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	result, err := h.administratorCreate.CreateAdministrator(
		request.Context(),
		CreateAdministratorRequest{
			Principal: principal, IdempotencyKey: keys[0],
			Username: input.Username, Email: input.Email, Phone: input.Phone,
			TemporaryPassword:                 input.TemporaryPassword,
			InformationProvisionNoticeVersion: input.InformationProvisionNoticeVersion,
			InformationProvisionDeclared:      input.InformationProvisionDeclared,
			TraceID:                           traceID,
		},
	)
	if err != nil {
		if writeAdministratorCreateError(response, request, err) {
			return
		}
		_ = platform.WriteError(response, request, err)
		return
	}
	_ = platform.WriteJSON(response, http.StatusCreated, administratorCreateResponse{
		ID: result.ID, Username: result.Username, Role: result.Role,
		Status: result.Status, MustChangePassword: result.MustChangePassword,
		MustAcknowledgePrivacy: result.MustAcknowledgePrivacy,
		TemporaryCredentialExpiresAt: result.TemporaryCredentialExpiresAt.
			UTC().Format(time.RFC3339),
		DeliveryStatus: result.DeliveryStatus,
	})
}

// AcknowledgePrivacy handles POST /api/v1/users/me/privacy-acknowledgements.
func (h *Handler) AcknowledgePrivacy(
	response http.ResponseWriter,
	request *http.Request,
) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("privacy acknowledgement principal is missing"),
		))
		return
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("privacy acknowledgement trace ID is missing"),
		))
		return
	}
	var input privacyAcknowledgementRequest
	if err := platform.DecodeJSON(
		request,
		h.administratorPrivacyBytes,
		&input,
	); err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	source, err := h.administratorSource.Evidence(request)
	if err != nil {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			err,
		))
		return
	}
	cookie, err := request.Cookie(platform.SessionCookieName)
	if err != nil || cookie.Value == "" {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("privacy acknowledgement session cookie is missing"),
		))
		return
	}
	result, err := h.administratorPrivacy.AcknowledgePrivacy(
		request.Context(),
		AdministratorPrivacyAcknowledgementRequest{
			Principal: principal, CurrentSessionToken: cookie.Value,
			PrivacyNoticeVersion: input.PrivacyNoticeVersion,
			Acknowledged:         input.Acknowledged, SourceKind: source.Kind,
			SourceFingerprint: source.Fingerprint, TraceID: traceID,
		},
	)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	if err := platform.SetSessionCookie(response, result.SessionToken); err != nil {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			err,
		))
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, loginResponse{
		User: loginUserResponse{
			ID: result.User.ID, Username: result.User.Username,
			Role: result.User.Role, Status: result.User.Status,
			IsInitialAdmin:         result.User.IsInitialAdmin,
			MustChangePassword:     result.MustChangePassword,
			MustAcknowledgePrivacy: result.MustAcknowledgePrivacy,
		},
		CSRFToken: result.CSRFToken,
	})
}

// ListUsers handles GET /api/v1/admin/users.
func (h *Handler) ListUsers(response http.ResponseWriter, request *http.Request) {
	h.listGovernanceUsers(response, request, UserQueryCategoryUsers)
}

// GetUser handles GET /api/v1/admin/users/{id}.
func (h *Handler) GetUser(response http.ResponseWriter, request *http.Request) {
	h.getGovernanceUser(response, request, UserQueryCategoryUsers)
}

// ListAdministrators handles GET /api/v1/admin/administrators.
func (h *Handler) ListAdministrators(response http.ResponseWriter, request *http.Request) {
	h.listGovernanceUsers(response, request, UserQueryCategoryAdministrators)
}

// GetAdministrator handles GET /api/v1/admin/administrators/{id}.
func (h *Handler) GetAdministrator(response http.ResponseWriter, request *http.Request) {
	h.getGovernanceUser(response, request, UserQueryCategoryAdministrators)
}

// SuspendUser handles POST /api/v1/admin/users/{id}/suspend.
func (h *Handler) SuspendUser(response http.ResponseWriter, request *http.Request) {
	h.setAccountStatus(response, request, RoleMember, UserStatusSuspended)
}

// RestoreUser handles POST /api/v1/admin/users/{id}/restore.
func (h *Handler) RestoreUser(response http.ResponseWriter, request *http.Request) {
	h.setAccountStatus(response, request, RoleMember, UserStatusActive)
}

// SuspendAdministrator handles POST /api/v1/admin/administrators/{id}/suspend.
func (h *Handler) SuspendAdministrator(response http.ResponseWriter, request *http.Request) {
	h.setAccountStatus(response, request, RoleAdmin, UserStatusSuspended)
}

// RestoreAdministrator handles POST /api/v1/admin/administrators/{id}/restore.
func (h *Handler) RestoreAdministrator(response http.ResponseWriter, request *http.Request) {
	h.setAccountStatus(response, request, RoleAdmin, UserStatusActive)
}

// RevokeAdministrator handles POST /api/v1/admin/administrators/{id}/revoke.
func (h *Handler) RevokeAdministrator(
	response http.ResponseWriter,
	request *http.Request,
) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("identity administrator governance principal is missing"),
		))
		return
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("identity administrator governance trace ID is missing"),
		))
		return
	}
	targetID, err := parseAdministratorRevokePath(request.URL.Path)
	if err != nil {
		_ = platform.WriteError(response, request, userNotFoundError())
		return
	}
	result, err := h.administratorGovernance.RevokeAdministrator(
		request.Context(),
		AdministratorRevokeRequest{
			Actor: principal, TargetID: targetID, TraceID: traceID,
		},
	)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, result)
}

func parseAdministratorRevokePath(path string) (string, error) {
	const (
		prefix = "/api/v1/admin/administrators/"
		suffix = "/revoke"
	)
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", errors.New("identity administrator revoke path is invalid")
	}
	targetID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if targetID == "" || strings.Contains(targetID, "/") ||
		!validAccountStatusUserID(targetID) {
		return "", errors.New("identity administrator revoke path is invalid")
	}
	return targetID, nil
}

// DeleteUser handles DELETE /api/v1/admin/users/{id}.
func (h *Handler) DeleteUser(response http.ResponseWriter, request *http.Request) {
	h.deleteCanonicalUser(response, request, RoleMember)
}

// DeleteAdministrator handles DELETE /api/v1/admin/administrators/{id}.
func (h *Handler) DeleteAdministrator(
	response http.ResponseWriter,
	request *http.Request,
) {
	h.deleteCanonicalUser(response, request, RoleAdmin)
}

// ResetPassword handles both administrator password-reset routes.
func (h *Handler) ResetPassword(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", http.MethodPost)
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeRequestMethodNotAllowed,
			errors.New("identity password reset method is not allowed"),
		))
		return
	}
	input, err := passwordResetRequestFromHTTP(request)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	result, err := h.passwordReset.ResetPassword(request.Context(), input)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, passwordResetResponse{
		UserID: result.UserID,
		TemporaryCredentialExpiresAt: result.TemporaryCredentialExpiresAt.
			UTC().Format(time.RFC3339),
		DeliveryStatus: result.DeliveryStatus,
	})
}

// ResendTemporaryCredential handles both administrator resend routes.
func (h *Handler) ResendTemporaryCredential(
	response http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", http.MethodPost)
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeRequestMethodNotAllowed,
			errors.New("identity temporary credential resend method is not allowed"),
		))
		return
	}
	input, err := temporaryCredentialResendRequestFromHTTP(request)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	result, err := h.temporaryCredentialResend.ResendTemporaryCredential(
		request.Context(),
		input,
	)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, passwordResetResponse{
		UserID: result.UserID,
		TemporaryCredentialExpiresAt: result.TemporaryCredentialExpiresAt.
			UTC().Format(time.RFC3339),
		DeliveryStatus: result.DeliveryStatus,
	})
}

func temporaryCredentialResendRequestFromHTTP(
	request *http.Request,
) (TemporaryCredentialResendRequest, error) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		return TemporaryCredentialResendRequest{}, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("identity temporary credential resend principal is missing"),
		)
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		return TemporaryCredentialResendRequest{}, passwordResetInternalError(
			"read temporary credential resend trace ID",
			errors.New("trace ID is missing"),
		)
	}
	targetID, targetRole, purpose, err := parseTemporaryCredentialResendTarget(
		request.URL.Path,
	)
	if err != nil {
		return TemporaryCredentialResendRequest{}, userNotFoundError()
	}
	keys := request.Header.Values("Idempotency-Key")
	if len(keys) != 1 {
		return TemporaryCredentialResendRequest{}, platform.NewPublicError(
			platform.CodeIdentityIdempotencyKeyRequired,
			errors.New("temporary credential resend requires one idempotency key"),
		)
	}
	return TemporaryCredentialResendRequest{
		Actor: principal, TargetID: targetID, TargetRole: targetRole,
		Purpose: purpose, IdempotencyKey: keys[0], TraceID: traceID,
	}, nil
}

func parseTemporaryCredentialResendTarget(
	path string,
) (string, Role, TemporaryCredentialResendPurpose, error) {
	for _, candidate := range []struct {
		prefix  string
		suffix  string
		role    Role
		purpose TemporaryCredentialResendPurpose
	}{
		{
			prefix: endpointForUserQueryCategory(UserQueryCategoryUsers) + "/",
			suffix: "/password-reset/resend", role: RoleMember,
			purpose: TemporaryCredentialResendPasswordReset,
		},
		{
			prefix: endpointForUserQueryCategory(UserQueryCategoryAdministrators) + "/",
			suffix: "/temporary-password/resend", role: RoleAdmin,
			purpose: TemporaryCredentialResendNewAdministrator,
		},
	} {
		targetID := strings.TrimSuffix(
			strings.TrimPrefix(path, candidate.prefix),
			candidate.suffix,
		)
		if targetID != path && targetID != "" && !strings.Contains(targetID, "/") &&
			path == candidate.prefix+targetID+candidate.suffix &&
			validAccountStatusUserID(targetID) {
			return targetID, candidate.role, candidate.purpose, nil
		}
	}
	return "", "", "", errors.New("identity temporary credential resend path is invalid")
}

func passwordResetRequestFromHTTP(request *http.Request) (PasswordResetRequest, error) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		return PasswordResetRequest{}, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("identity password reset principal is missing"),
		)
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		return PasswordResetRequest{}, passwordResetInternalError(
			"read password reset trace ID",
			errors.New("trace ID is missing"),
		)
	}
	targetID, targetRole, err := parsePasswordResetTarget(request.URL.Path)
	if err != nil {
		return PasswordResetRequest{}, userNotFoundError()
	}
	keys := request.Header.Values("Idempotency-Key")
	if len(keys) != 1 {
		return PasswordResetRequest{}, platform.NewPublicError(
			platform.CodeIdentityIdempotencyKeyRequired,
			errors.New("password reset requires one idempotency key"),
		)
	}
	return PasswordResetRequest{
		Actor: principal, TargetID: targetID, TargetRole: targetRole,
		IdempotencyKey: keys[0], TraceID: traceID,
	}, nil
}

func parsePasswordResetTarget(path string) (string, Role, error) {
	for _, candidate := range []struct {
		prefix string
		role   Role
	}{
		{prefix: endpointForUserQueryCategory(UserQueryCategoryUsers) + "/", role: RoleMember},
		{
			prefix: endpointForUserQueryCategory(UserQueryCategoryAdministrators) + "/",
			role:   RoleAdmin,
		},
	} {
		suffix := "/password-reset"
		targetID := strings.TrimSuffix(strings.TrimPrefix(path, candidate.prefix), suffix)
		if targetID != path && targetID != "" && !strings.Contains(targetID, "/") &&
			path == candidate.prefix+targetID+suffix &&
			validAccountStatusUserID(targetID) {
			return targetID, candidate.role, nil
		}
	}
	return "", "", errors.New("identity password reset path is invalid")
}

func (h *Handler) deleteCanonicalUser(
	response http.ResponseWriter,
	request *http.Request,
	targetRole Role,
) {
	if request.Method != http.MethodDelete {
		response.Header().Set("Allow", http.MethodDelete)
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeRequestMethodNotAllowed,
			errors.New("identity canonical delete method is not allowed"),
		))
		return
	}
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("identity canonical delete principal is missing"),
		))
		return
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, canonicalDeleteInternalError(
			"read canonical delete trace ID",
			errors.New("trace ID is missing"),
		))
		return
	}
	targetID, err := parseCanonicalDeleteTargetID(request.URL.Path, targetRole)
	if err != nil {
		_ = platform.WriteError(response, request, userNotFoundError())
		return
	}
	reason, err := canonicalDeleteReasonFromHTTP(request)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	_, err = h.canonicalDelete.CanonicalDelete(
		request.Context(),
		CanonicalDeleteRequest{
			Actor: principal, TargetID: targetID, TargetRole: targetRole,
			Mode: CanonicalDeleteDailyGovernance, Reason: reason, TraceID: traceID,
		},
	)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func parseCanonicalDeleteTargetID(path string, targetRole Role) (string, error) {
	category := UserQueryCategoryUsers
	if targetRole == RoleAdmin {
		category = UserQueryCategoryAdministrators
	}
	prefix := endpointForUserQueryCategory(category) + "/"
	targetID := strings.TrimPrefix(path, prefix)
	if targetID == path || targetID == "" || strings.Contains(targetID, "/") ||
		!validAccountStatusUserID(targetID) {
		return "", errors.New("identity canonical delete path is invalid")
	}
	return targetID, nil
}

func canonicalDeleteReasonFromHTTP(
	request *http.Request,
) (AccountDeletionReason, error) {
	const header = "X-Account-Deletion-Reason"
	values := request.Header.Values(header)
	if len(values) == 0 {
		return AccountDeletionGovernance, nil
	}
	if len(values) != 1 {
		return "", platform.NewPublicError(
			platform.CodeIdentityValidationFailed,
			errors.New("canonical delete reason must appear at most once"),
			platform.FieldError{Field: header, Code: "invalid"},
		)
	}
	reason := AccountDeletionReason(values[0])
	switch reason {
	case AccountDeletionGovernance, AccountDeletionPrivacyWithdrawal:
		return reason, nil
	default:
		return "", platform.NewPublicError(
			platform.CodeIdentityValidationFailed,
			errors.New("canonical delete reason is invalid"),
			platform.FieldError{Field: header, Code: "invalid"},
		)
	}
}

func (h *Handler) setAccountStatus(
	response http.ResponseWriter,
	request *http.Request,
	targetRole Role,
	status UserStatus,
) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("identity account status principal is missing"),
		))
		return
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodePlatformInternalError,
			errors.New("identity account status trace ID is missing"),
		))
		return
	}
	targetID, err := parseAccountStatusTargetID(request.URL.Path, targetRole, status)
	if err != nil {
		_ = platform.WriteError(response, request, userNotFoundError())
		return
	}
	result, err := h.accountStatus.SetAccountStatus(
		request.Context(),
		AccountStatusRequest{
			Actor: principal, TargetID: targetID, TargetRole: targetRole,
			Status: status, TraceID: traceID,
		},
	)
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, result)
}

func parseAccountStatusTargetID(path string, targetRole Role, status UserStatus) (string, error) {
	category := UserQueryCategoryUsers
	if targetRole == RoleAdmin {
		category = UserQueryCategoryAdministrators
	}
	action := "suspend"
	if status == UserStatusActive {
		action = "restore"
	}
	prefix := endpointForUserQueryCategory(category) + "/"
	suffix := "/" + action
	targetID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if targetID == path || targetID == "" || strings.Contains(targetID, "/") ||
		path != prefix+targetID+suffix {
		return "", errors.New("identity account status path is invalid")
	}
	return targetID, nil
}

func (h *Handler) listGovernanceUsers(
	response http.ResponseWriter,
	request *http.Request,
	category UserQueryCategory,
) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("identity user query principal is missing"),
		))
		return
	}
	cursor, cursorPresent, err := singleUserQueryParameter(request, "cursor")
	if err != nil {
		_ = platform.WriteError(response, request, invalidIdentityCursor(err))
		return
	}
	if cursorPresent && cursor == "" {
		_ = platform.WriteError(
			response,
			request,
			invalidIdentityCursor(errors.New("identity user query cursor is empty")),
		)
		return
	}
	limitValue, limitPresent, err := singleUserQueryParameter(request, "limit")
	if err != nil {
		_ = platform.WriteError(response, request, invalidUserQueryLimit(err))
		return
	}
	var limit *int
	if limitValue != "" {
		parsed, parseErr := strconv.Atoi(limitValue)
		if parseErr != nil {
			_ = platform.WriteError(response, request, invalidUserQueryLimit(parseErr))
			return
		}
		limit = &parsed
	} else if limitPresent {
		_ = platform.WriteError(
			response,
			request,
			invalidUserQueryLimit(errors.New("identity user query limit is empty")),
		)
		return
	}

	page, err := h.userQuery.List(request.Context(), UserQueryListRequest{
		Actor: principal, Endpoint: endpointForUserQueryCategory(category),
		Category: category, Cursor: cursor, Limit: limit,
		FilterDigest: identityUserQueryFilterDigest,
	})
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, page)
}

func (h *Handler) getGovernanceUser(
	response http.ResponseWriter,
	request *http.Request,
	category UserQueryCategory,
) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(
			platform.CodeAuthSessionRequired,
			errors.New("identity user query principal is missing"),
		))
		return
	}
	prefix := endpointForUserQueryCategory(category) + "/"
	id := strings.TrimPrefix(request.URL.Path, prefix)
	if id == request.URL.Path || id == "" || strings.Contains(id, "/") {
		_ = platform.WriteError(response, request, userNotFoundError())
		return
	}
	detail, err := h.userQuery.Detail(request.Context(), UserQueryDetailRequest{
		Actor: principal, Category: category, ID: id,
	})
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, detail)
}

func singleUserQueryParameter(
	request *http.Request,
	name string,
) (string, bool, error) {
	values, exists := request.URL.Query()[name]
	if !exists {
		return "", false, nil
	}
	if len(values) != 1 {
		return "", true, errors.New("identity user query parameter must appear once")
	}
	return values[0], true, nil
}

func invalidUserQueryLimit(cause error) error {
	return platform.NewPublicError(
		platform.CodeIdentityValidationFailed,
		cause,
		platform.FieldError{Field: "limit", Code: "invalid"},
	)
}

func (h *Handler) writeAuditedPasswordChangeFailure(
	response http.ResponseWriter,
	request *http.Request,
	principal session.Principal,
	traceID string,
	publicErr error,
) {
	if err := h.passwordChange.AuditPasswordChangeFailure(
		request.Context(),
		PasswordChangeFailureAuditRequest{
			Principal: principal,
			ErrorCode: platform.PublicCode(publicErr),
			TraceID:   traceID,
		},
	); err != nil {
		publicErr = platform.NewPublicError(
			platform.CodePlatformDatabaseUnavailable,
			fmt.Errorf("append password change failure audit: %w", err),
		)
	}
	_ = platform.WriteError(response, request, publicErr)
}

func (h *Handler) observeLogin(
	request *http.Request,
	startedAt time.Time,
	observation LoginObservation,
	writeErr error,
) {
	observation.Duration = time.Since(startedAt)
	observation.ResponseWrite = LoginResponseWriteOK
	if request != nil {
		observation.TraceID, _ = platform.TraceIDFromContext(request.Context())
	}
	if writeErr != nil {
		observation.Outcome = LoginOutcomeInternalFailure
		observation.FailureStage = LoginFailureStageResponseWrite
		observation.ReasonBucket = "response"
		observation.ErrorCode = string(platform.CodePlatformInternalError)
		observation.ResponseWrite = LoginResponseWriteFailed
	}
	ctx := context.Background()
	if request != nil {
		ctx = request.Context()
	}
	observationCtx, cancel := context.WithTimeout(ctx, h.loginObserveFor)
	_ = h.loginObserver.ObserveFinal(observationCtx, observation)
	cancel()
}

func loginOutcomeForError(err error) LoginOutcome {
	switch platform.PublicCode(err) {
	case platform.CodeAuthLoginFailed:
		return LoginOutcomeAuthenticationFailed
	case platform.CodeRequestRateLimited:
		return LoginOutcomeRateLimited
	case platform.CodeRequestInvalidJSON:
		return LoginOutcomeInvalidRequest
	default:
		return LoginOutcomeInternalFailure
	}
}

type registrationResponse struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	Role               Role       `json:"role"`
	Status             UserStatus `json:"status"`
	MustChangePassword bool       `json:"must_change_password"`
	CreatedAt          string     `json:"created_at"`
}

type loginResponse struct {
	User      loginUserResponse `json:"user"`
	CSRFToken string            `json:"csrf_token"`
}

type passwordChangeRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type createAdministratorRequest struct {
	Username                          string `json:"username"`
	Email                             string `json:"email"`
	Phone                             string `json:"phone"`
	TemporaryPassword                 string `json:"temporary_password"`
	InformationProvisionNoticeVersion string `json:"information_provision_notice_version"`
	InformationProvisionDeclared      bool   `json:"information_provision_declared"`
}

type privacyAcknowledgementRequest struct {
	PrivacyNoticeVersion string `json:"privacy_notice_version"`
	Acknowledged         bool   `json:"acknowledged"`
}

type administratorInformationNoticeResponse struct {
	Version     string `json:"version"`
	EffectiveAt string `json:"effective_at"`
	Content     string `json:"content"`
	ContentHash string `json:"content_hash"`
}

type administratorCreateResponse struct {
	ID                           string     `json:"id"`
	Username                     string     `json:"username"`
	Role                         Role       `json:"role"`
	Status                       UserStatus `json:"status"`
	MustChangePassword           bool       `json:"must_change_password"`
	MustAcknowledgePrivacy       bool       `json:"must_acknowledge_privacy"`
	TemporaryCredentialExpiresAt string     `json:"temporary_credential_expires_at"`
	DeliveryStatus               string     `json:"delivery_status"`
}

type passwordResetResponse struct {
	UserID                       string `json:"user_id"`
	TemporaryCredentialExpiresAt string `json:"temporary_credential_expires_at"`
	DeliveryStatus               string `json:"delivery_status"`
}

type loginUserResponse struct {
	ID                     string     `json:"id"`
	Username               string     `json:"username"`
	Role                   Role       `json:"role"`
	Status                 UserStatus `json:"status"`
	IsInitialAdmin         bool       `json:"is_initial_admin"`
	MustChangePassword     bool       `json:"must_change_password"`
	MustAcknowledgePrivacy bool       `json:"must_acknowledge_privacy"`
}
