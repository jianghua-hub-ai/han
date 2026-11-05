package platform

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"han/internal/session"
)

const (
	// SessionCookieName is the host-only browser session cookie.
	SessionCookieName = "__Host-han_session"
	// CSRFHeaderName carries the session-bound CSRF token.
	CSRFHeaderName            = "X-CSRF-Token"
	originRejectionEvent      = "platform.http.origin_rejected"
	originRejectionMetricName = "han_platform_origin_rejections_total"
)

// OriginGuard enforces the configured canonical browser origin.
type OriginGuard struct {
	publicOrigin string
	observer     OriginRejectionObserver
	observeFor   time.Duration
}

// OriginRejectionReason identifies one stable same-origin rejection branch.
type OriginRejectionReason string

const (
	OriginRejectionRequestNil        OriginRejectionReason = "request_nil"
	OriginRejectionOriginMissing     OriginRejectionReason = "origin_missing"
	OriginRejectionOriginMismatch    OriginRejectionReason = "origin_mismatch"
	OriginRejectionFetchSiteRejected OriginRejectionReason = "fetch_site_rejected"
)

// OriginResponseWrite is the low-cardinality result of the single 403 write attempt.
type OriginResponseWrite string

const (
	OriginResponseWriteOK     OriginResponseWrite = "ok"
	OriginResponseWriteFailed OriginResponseWrite = "failed"
)

// OriginRejectionObservation contains no rejected request material.
type OriginRejectionObservation struct {
	Reason        OriginRejectionReason
	ResponseWrite OriginResponseWrite
}

// OriginRejectionObserver observes one rejected request after its response write attempt.
type OriginRejectionObserver interface {
	ObserveRejected(context.Context, OriginRejectionObservation) error
}

// OriginRejectionLogger emits the bounded structured warning.
type OriginRejectionLogger interface {
	LogOriginRejection(context.Context, OriginRejectionObservation) error
}

// SlogOriginRejectionLogger emits the fixed same-origin warning through slog.
type SlogOriginRejectionLogger struct {
	logger  *slog.Logger
	service string
}

// NewSlogOriginRejectionLogger creates the bounded structured Origin logger.
func NewSlogOriginRejectionLogger(
	logger *slog.Logger,
	service string,
) (*SlogOriginRejectionLogger, error) {
	service = strings.TrimSpace(service)
	if logger == nil || service == "" {
		return nil, errors.New("origin rejection logger dependencies are incomplete")
	}
	return &SlogOriginRejectionLogger{logger: logger, service: service}, nil
}

// LogOriginRejection emits only stable low-cardinality rejection fields.
func (l *SlogOriginRejectionLogger) LogOriginRejection(
	ctx context.Context,
	observation OriginRejectionObservation,
) error {
	if l == nil || l.logger == nil || strings.TrimSpace(l.service) == "" {
		return errors.New("origin rejection logger is not initialized")
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	logger := l.logger.With(
		slog.String("service", l.service),
		slog.String("module", "platform"),
	)
	if traceID, ok := TraceIDFromContext(ctx); ok {
		traced, err := LoggerWithTrace(l.logger, l.service, "platform", traceID)
		if err != nil {
			return err
		}
		logger = traced
	}
	record := slog.NewRecord(
		time.Now(),
		slog.LevelWarn,
		originRejectionEvent,
		0,
	)
	record.AddAttrs(
		slog.String("event", originRejectionEvent),
		slog.String("reason", string(observation.Reason)),
		slog.String("response_write", string(observation.ResponseWrite)),
	)
	if !logger.Enabled(ctx, slog.LevelWarn) {
		return contextError(ctx)
	}
	if err := logger.Handler().Handle(ctx, record); err != nil {
		return err
	}
	return contextError(ctx)
}

// OriginSecurityMetrics records the bounded low-cardinality rejection counter.
type OriginSecurityMetrics interface {
	RecordOriginRejection(context.Context, OriginRejectionObservation) error
}

type originRejectionObserver struct {
	logger  OriginRejectionLogger
	metrics OriginSecurityMetrics
}

// NewOriginRejectionObserver composes the fixed logger/metrics fan-out.
func NewOriginRejectionObserver(
	logger OriginRejectionLogger,
	metrics OriginSecurityMetrics,
) (OriginRejectionObserver, error) {
	if logger == nil || metrics == nil {
		return nil, errors.New("origin rejection observer dependencies are incomplete")
	}
	return &originRejectionObserver{logger: logger, metrics: metrics}, nil
}

func (o *originRejectionObserver) ObserveRejected(
	ctx context.Context,
	observation OriginRejectionObservation,
) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if err := o.logger.LogOriginRejection(ctx, observation); err != nil {
		return err
	}
	if err := contextError(ctx); err != nil {
		return err
	}
	return o.metrics.RecordOriginRejection(ctx, observation)
}

// NewOriginGuard creates a same-origin browser request guard.
func NewOriginGuard(
	publicOrigin string,
	observer OriginRejectionObserver,
	observationBudget time.Duration,
) (OriginGuard, error) {
	normalized, err := normalizePublicOrigin(publicOrigin)
	if err != nil || normalized != publicOrigin {
		return OriginGuard{}, errors.New("canonical public origin is required")
	}
	if observer == nil {
		return OriginGuard{}, errors.New("origin rejection observer is required")
	}
	if observationBudget <= 0 {
		return OriginGuard{}, errors.New("origin rejection observation budget must be positive")
	}
	return OriginGuard{
		publicOrigin: normalized,
		observer:     observer,
		observeFor:   observationBudget,
	}, nil
}

// NewObservedOriginGuard is the explicit observed guard constructor.
func NewObservedOriginGuard(
	publicOrigin string,
	observer OriginRejectionObserver,
	observationBudget time.Duration,
) (OriginGuard, error) {
	return NewOriginGuard(publicOrigin, observer, observationBudget)
}

// Protect rejects requests without trusted same-origin evidence.
func (g OriginGuard) Protect(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if reason, rejected := g.rejectionReason(request); rejected {
			clearCORSHeaders(response.Header())
			writeErr := WriteError(
				response,
				request,
				NewPublicError(CodeAuthOriginRejected, errors.New("request origin rejected")),
			)
			responseWrite := OriginResponseWriteOK
			if writeErr != nil {
				responseWrite = OriginResponseWriteFailed
			}
			ctx := context.Background()
			if request != nil {
				ctx = request.Context()
			}
			observationCtx, cancel := context.WithTimeout(ctx, g.observeFor)
			_ = g.observer.ObserveRejected(observationCtx, OriginRejectionObservation{
				Reason:        reason,
				ResponseWrite: responseWrite,
			})
			cancel()
			return
		}

		protectedResponse := &sameOriginResponseWriter{ResponseWriter: response}
		next.ServeHTTP(protectedResponse, request)
		clearCORSHeaders(response.Header())
	})
}

func (g OriginGuard) rejectionReason(
	request *http.Request,
) (OriginRejectionReason, bool) {
	if request == nil {
		return OriginRejectionRequestNil, true
	}
	origin := request.Header.Get("Origin")
	if origin == "" {
		return OriginRejectionOriginMissing, true
	}
	if origin != g.publicOrigin {
		return OriginRejectionOriginMismatch, true
	}
	if !trustedFetchSite(request.Header.Get("Sec-Fetch-Site")) {
		return OriginRejectionFetchSiteRejected, true
	}
	return "", false
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("observation context is required")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// SetSessionCookie writes the accepted host-only session cookie contract.
func SetSessionCookie(response http.ResponseWriter, token string) error {
	if response == nil {
		return errors.New("HTTP response writer is required")
	}
	if token == "" {
		return errors.New("session token is required")
	}
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if err := cookie.Valid(); err != nil {
		return errors.New("session token is not a valid cookie value")
	}
	http.SetCookie(response, cookie)
	return nil
}

// PrincipalResolver resolves one current session into a trusted principal.
type PrincipalResolver interface {
	ResolvePrincipal(context.Context, session.ResolveInput) (session.Principal, error)
}

// Clock supplies explicit request time to session expiry checks.
type Clock func() time.Time

// HTTPAuthorizer resolves Cookie/CSRF evidence and attaches the principal.
type HTTPAuthorizer struct {
	resolver PrincipalResolver
	clock    Clock
}

// NewHTTPAuthorizer creates the protected request boundary.
func NewHTTPAuthorizer(resolver PrincipalResolver, clock Clock) (HTTPAuthorizer, error) {
	if resolver == nil {
		return HTTPAuthorizer{}, errors.New("principal resolver is required")
	}
	if clock == nil {
		return HTTPAuthorizer{}, errors.New("session clock is required")
	}
	return HTTPAuthorizer{resolver: resolver, clock: clock}, nil
}

// Protect requires a current principal, CSRF for writes, and the requested access.
func (a HTTPAuthorizer) Protect(access session.Access, next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			writeSessionSecurityError(response, request, session.ErrSessionRequired)
			return
		}

		principal, err := a.resolver.ResolvePrincipal(request.Context(), session.ResolveInput{
			Token:       cookie.Value,
			CSRFToken:   request.Header.Get(CSRFHeaderName),
			RequireCSRF: requiresCSRF(request.Method),
			Now:         a.clock(),
		})
		if err != nil {
			writeSessionSecurityError(response, request, err)
			return
		}
		if !principal.Allows(access) {
			code := CodeAuthForbidden
			if principal.Restricted() {
				code = CodeAuthMustCompleteOnboarding
			}
			_ = WriteError(response, request, NewPublicError(code, errors.New("principal access rejected")))
			return
		}

		next.ServeHTTP(response, request.WithContext(contextWithPrincipal(request.Context(), principal)))
	})
}

// PrincipalFromContext returns the trusted principal attached by HTTPAuthorizer.
func PrincipalFromContext(ctx context.Context) (session.Principal, bool) {
	if ctx == nil {
		return session.Principal{}, false
	}
	principal, ok := ctx.Value(principalContextKey{}).(session.Principal)
	return principal, ok
}

type principalContextKey struct{}

func contextWithPrincipal(ctx context.Context, principal session.Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func trustedFetchSite(value string) bool {
	return value == "" || value == "same-origin"
}

func requiresCSRF(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	default:
		return true
	}
}

func writeSessionSecurityError(response http.ResponseWriter, request *http.Request, err error) {
	code := CodePlatformInternalError
	switch {
	case errors.Is(err, session.ErrSessionRequired):
		code = CodeAuthSessionRequired
	case errors.Is(err, session.ErrCSRFFailed):
		code = CodeAuthCSRFFailed
	case errors.Is(err, session.ErrStoreUnavailable):
		code = CodePlatformDatabaseUnavailable
	}
	_ = WriteError(response, request, NewPublicError(code, err))
}

type sameOriginResponseWriter struct {
	http.ResponseWriter
}

func (w *sameOriginResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *sameOriginResponseWriter) WriteHeader(statusCode int) {
	clearCORSHeaders(w.Header())
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *sameOriginResponseWriter) Write(body []byte) (int, error) {
	clearCORSHeaders(w.Header())
	return w.ResponseWriter.Write(body)
}

func clearCORSHeaders(header http.Header) {
	header.Del("Access-Control-Allow-Origin")
	header.Del("Access-Control-Allow-Credentials")
}
