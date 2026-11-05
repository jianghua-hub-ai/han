package identity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"

	"han/internal/platform"
	"han/internal/recovery"
	"han/internal/session"
)

const (
	recoveryReviewDefaultLimit = 50
	recoveryReviewMaxLimit     = 100
	// recoveryReviewFirstPageBound is the lowest UUID value. The first page
	// compares against it so the empty cursor never reaches a uuid parameter.
	recoveryReviewFirstPageBound = "00000000-0000-0000-0000-000000000000"
)

// RecoveryControlMaintenance is the narrow maintenance seam. It is
// implemented by the same identity executor the recovery runtime drives.
type RecoveryControlMaintenance interface {
	ResolveRestrictedAccountReview(context.Context, recovery.PhasePermit, recovery.RunBinding, string, string, string) error
	ResolveRecoveryIdempotency(context.Context, recovery.PhasePermit, recovery.RunBinding, []byte, []byte, int64, string, string) error
}

// RecoveryRestoredReviewResolver is the optional F002 seam for restored
// decisions. It is absent when no approved restored source is wired.
type RecoveryRestoredReviewResolver interface {
	ResolveRestoredAccountReview(context.Context, recovery.PhasePermit, recovery.RunBinding, string, string, string, string, string) error
}

// RecoveryControlReviewSource lists pending restricted-account reviews.
type RecoveryControlReviewSource interface {
	ListPendingReviews(context.Context, string, int) ([]RecoveryReviewItem, string, error)
}

// RecoveryReviewItem is the closed maintenance projection.
type RecoveryReviewItem struct {
	UserID   string `json:"user_id"`
	Decision string `json:"decision"`
}

// RecoveryCompleteSignaler wakes the root completion controller after the
// HTTP response is actually written. It never authorizes a write.
type RecoveryCompleteSignaler interface {
	RequestComplete() error
}

// RecoveryControlPhase derives the private phase context for one maintenance
// request from the permit the root already holds. A zero permit is invalid.
type RecoveryControlPhase func(context.Context) (context.Context, recovery.PhasePermit, recovery.RunBinding, error)

// RecoveryCompletePreflight is the optional deterministic check the handler
// runs before writing the accepted completion response. A closed signaler
// fails here so the caller receives an error response instead of a 200 that
// the root controller would wait on forever.
type RecoveryCompletePreflight interface {
	CheckComplete() error
}

// RecoveryControlDependencies is the exact maintenance control wiring.
type RecoveryControlDependencies struct {
	Maintenance  RecoveryControlMaintenance
	Reviews      RecoveryControlReviewSource
	TrustChange  RecoveryTrustChange
	Restored     RecoveryRestoredReviewResolver
	Complete     RecoveryCompleteSignaler
	Authorizer   platform.HTTPAuthorizer
	Phase        RecoveryControlPhase
	Session      http.HandlerFunc
	Liveness     http.HandlerFunc
	Readiness    http.HandlerFunc
	MaxBodyBytes int64
}

// RecoveryControlHandlers serves the four recovery control routes and the
// three existing maintenance business routes with one exact router.
type RecoveryControlHandlers struct {
	dependencies RecoveryControlDependencies
}

func NewRecoveryControlHandlers(d RecoveryControlDependencies) (*RecoveryControlHandlers, error) {
	if nilRecoveryControlValue(d.Maintenance) || nilRecoveryControlValue(d.Reviews) || nilRecoveryControlValue(d.Complete) || nilRecoveryControlValue(d.Phase) || d.Session == nil || d.MaxBodyBytes <= 0 || reflect.ValueOf(d.Authorizer).IsZero() {
		return nil, errors.New("recovery control dependencies are incomplete")
	}
	return &RecoveryControlHandlers{dependencies: d}, nil
}

func nilRecoveryControlValue(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

// Handler returns the maintenance surface with Trace outermost and exact
// method routing before any phase or Session check.
func (h *RecoveryControlHandlers) Handler() http.Handler {
	return platform.TraceMiddleware(http.HandlerFunc(h.route))
}

type recoveryControlKey struct{}

type recoveryControlRequest struct {
	permit  recovery.PhasePermit
	binding recovery.RunBinding
}

func (h *RecoveryControlHandlers) route(response http.ResponseWriter, request *http.Request) {
	if request.Body != nil {
		request.Body = newRecoveryUTF8Body(request.Body)
	}
	path := request.URL.Path
	switch path {
	case "/livez":
		if h.method(response, request, http.MethodGet, "GET") {
			h.probe(response, request, h.dependencies.Liveness)
		}
		return
	case "/readyz":
		if h.method(response, request, http.MethodGet, "GET") {
			h.probe(response, request, h.dependencies.Readiness)
		}
		return
	case "/api/v1/sessions":
		if h.method(response, request, http.MethodPost, "POST") {
			h.withPhase(response, request, false, session.AccessGeneral, func(w http.ResponseWriter, r *http.Request, c recoveryControlRequest) {
				h.dependencies.Session(w, r)
			})
		}
		return
	case "/api/v1/users/me":
		if h.method(response, request, http.MethodGet, "GET") {
			h.withPhase(response, request, true, session.AccessCurrentIdentity, h.currentIdentity)
		}
		return
	case "/api/v1/users/me/password":
		if h.method(response, request, http.MethodPut, "PUT") {
			h.withPhase(response, request, true, session.AccessChangePassword, h.changePassword)
		}
		return
	case "/api/v1/recovery/reviews":
		if h.method(response, request, http.MethodGet, "GET") {
			h.withPhase(response, request, true, session.AccessCurrentIdentity, h.reviews)
		}
		return
	case "/api/v1/recovery/idempotency/resolve":
		if h.method(response, request, http.MethodPost, "POST") {
			h.withPhase(response, request, true, session.AccessGeneral, h.resolveIdempotency)
		}
		return
	case "/api/v1/recovery/complete":
		if h.method(response, request, http.MethodPost, "POST") {
			h.withPhase(response, request, true, session.AccessGeneral, h.complete)
		}
		return
	}
	if strings.HasPrefix(path, "/api/v1/recovery/reviews/") && strings.HasSuffix(path, "/resolve") {
		userID := strings.TrimSuffix(strings.TrimPrefix(path, "/api/v1/recovery/reviews/"), "/resolve")
		if userID == "" || strings.Contains(userID, "/") {
			h.notFound(response, request)
			return
		}
		if h.method(response, request, http.MethodPost, "POST") {
			h.withPhase(response, request, true, session.AccessGeneral, func(w http.ResponseWriter, r *http.Request, c recoveryControlRequest) {
				h.resolveReview(w, r, c, userID)
			})
		}
		return
	}
	h.notFound(response, request)
}

func (h *RecoveryControlHandlers) method(response http.ResponseWriter, request *http.Request, expected, allow string) bool {
	if request.Method == expected {
		return true
	}
	response.Header().Set("Allow", allow)
	_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestMethodNotAllowed, errors.New("method not allowed")))
	return false
}

func (h *RecoveryControlHandlers) notFound(response http.ResponseWriter, request *http.Request) {
	_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestRouteNotFound, errors.New("route not found")))
}

func (h *RecoveryControlHandlers) probe(response http.ResponseWriter, request *http.Request, handler http.HandlerFunc) {
	if handler == nil {
		response.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	handler(response, request)
}

func (h *RecoveryControlHandlers) withPhase(response http.ResponseWriter, request *http.Request, requireSession bool, access session.Access, next func(http.ResponseWriter, *http.Request, recoveryControlRequest)) {
	ctx, permit, binding, err := h.dependencies.Phase(request.Context())
	if err != nil {
		writeRecoveryPhaseError(response, request, err)
		return
	}
	if ctx == nil || ctx.Err() != nil {
		writeRecoveryPhaseError(response, request, recovery.ErrAdmissionClosed)
		return
	}
	control := recoveryControlRequest{permit: permit, binding: binding}
	request = request.WithContext(context.WithValue(ctx, recoveryControlKey{}, control))
	if !requireSession {
		next(response, request, control)
		return
	}
	h.dependencies.Authorizer.Protect(access, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next(w, r, control)
	})).ServeHTTP(response, request)
}

func writeRecoveryPhaseError(response http.ResponseWriter, request *http.Request, err error) {
	code := platform.CodePlatformRecoveryInProgress
	if errors.Is(err, recovery.ErrAdmissionBinding) {
		code = platform.CodeAuthForbidden
	}
	_ = platform.WriteError(response, request, platform.NewPublicError(code, err))
}

func (h *RecoveryControlHandlers) currentIdentity(response http.ResponseWriter, request *http.Request, _ recoveryControlRequest) {
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeAuthSessionRequired, errors.New("maintenance principal is missing")))
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, struct {
		ID                     string `json:"id"`
		Role                   string `json:"role"`
		Status                 string `json:"status"`
		MustChangePassword     bool   `json:"must_change_password"`
		MustAcknowledgePrivacy bool   `json:"must_acknowledge_privacy"`
	}{ID: principal.UserID, Role: principal.Role, Status: principal.Status, MustChangePassword: principal.MustChangePassword, MustAcknowledgePrivacy: principal.MustAcknowledgePrivacy})
}

func (h *RecoveryControlHandlers) changePassword(response http.ResponseWriter, request *http.Request, control recoveryControlRequest) {
	if nilRecoveryControlValue(h.dependencies.TrustChange) {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodePlatformRecoveryInProgress, errors.New("recovery trust change is unavailable")))
		return
	}
	principal, ok := platform.PrincipalFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeAuthSessionRequired, errors.New("maintenance principal is missing")))
		return
	}
	traceID, ok := platform.TraceIDFromContext(request.Context())
	if !ok {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodePlatformInternalError, errors.New("trace ID is missing")))
		return
	}
	var input passwordChangeRequest
	if err := platform.DecodeJSON(request, h.dependencies.MaxBodyBytes, &input); err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	cookie, err := request.Cookie(platform.SessionCookieName)
	if err != nil || cookie.Value == "" {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeAuthSessionRequired, errors.New("maintenance session cookie is missing")))
		return
	}
	result, err := h.dependencies.TrustChange.ChangeRecoveryPassword(request.Context(), control.permit, PasswordChangeRequest{Principal: principal, CurrentSessionToken: cookie.Value, CurrentPassword: input.CurrentPassword, NewPassword: input.NewPassword, TraceID: traceID})
	if err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	if err := platform.SetSessionCookie(response, result.SessionToken); err != nil {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodePlatformInternalError, err))
		return
	}
	_ = platform.WriteJSON(response, http.StatusOK, loginResponse{User: loginUserResponse{ID: result.User.ID, Username: result.User.Username, Role: result.User.Role, Status: result.User.Status, IsInitialAdmin: result.User.IsInitialAdmin, MustChangePassword: false, MustAcknowledgePrivacy: result.MustAcknowledgePrivacy}, CSRFToken: result.CSRFToken})
}

func (h *RecoveryControlHandlers) reviews(response http.ResponseWriter, request *http.Request, _ recoveryControlRequest) {
	rawLimit := request.URL.Query().Get("limit")
	limit, err := strconv.Atoi(rawLimit)
	if rawLimit == "" || err != nil || limit < 1 || limit > recoveryReviewMaxLimit {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestInvalidJSON, errors.New("limit must be 1..100")))
		return
	}
	items, next, err := h.dependencies.Reviews.ListPendingReviews(request.Context(), request.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeRecoveryReviewError(response, request, err)
		return
	}
	if items == nil {
		items = []RecoveryReviewItem{}
	}
	var cursor *string
	if next != "" {
		cursor = &next
	}
	_ = platform.WriteJSON(response, http.StatusOK, struct {
		Items      []RecoveryReviewItem `json:"items"`
		NextCursor *string              `json:"next_cursor"`
	}{Items: items, NextCursor: cursor})
}

func writeRecoveryReviewError(response http.ResponseWriter, request *http.Request, err error) {
	code := platform.CodePlatformRecoveryInProgress
	if errors.Is(err, recovery.ErrAdmissionBinding) {
		code = platform.CodeAuthForbidden
	}
	if errors.Is(err, errRecoveryReviewCursor) {
		code = platform.CodeRequestInvalidCursor
	}
	_ = platform.WriteError(response, request, platform.NewPublicError(code, err))
}

type recoveryResolveRequest struct {
	UserID      string  `json:"user_id"`
	Decision    string  `json:"decision"`
	Role        *string `json:"role"`
	Status      *string `json:"status"`
	EvidenceRef string  `json:"evidence_ref"`
}

func (h *RecoveryControlHandlers) resolveReview(response http.ResponseWriter, request *http.Request, control recoveryControlRequest, userID string) {
	var input recoveryResolveRequest
	if err := platform.DecodeJSON(request, h.dependencies.MaxBodyBytes, &input); err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	if input.UserID != userID || strings.TrimSpace(input.EvidenceRef) == "" {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestInvalidJSON, errors.New("review body mismatch")))
		return
	}
	switch input.Decision {
	case "denied", "kept_deleted":
		if input.Role != nil || input.Status != nil {
			_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestInvalidJSON, errors.New("role and status must be null")))
			return
		}
		if err := h.dependencies.Maintenance.ResolveRestrictedAccountReview(request.Context(), control.permit, control.binding, userID, input.Decision, input.EvidenceRef); err != nil {
			writeRecoveryPhaseError(response, request, err)
			return
		}
	case "restored":
		if input.Role == nil || input.Status == nil || (*input.Role != "member" && *input.Role != "admin") || (*input.Status != "active" && *input.Status != "suspended") {
			_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestInvalidJSON, errors.New("restored requires role and status")))
			return
		}
		if request.Header.Get("Idempotency-Key") == "" {
			_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeIdentityIdempotencyKeyRequired, errors.New("idempotency key is required")))
			return
		}
		if nilRecoveryControlValue(h.dependencies.Restored) {
			_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodePlatformRecoveryInProgress, errors.New("restored review source is unavailable")))
			return
		}
		if err := h.dependencies.Restored.ResolveRestoredAccountReview(request.Context(), control.permit, control.binding, userID, *input.Role, *input.Status, input.EvidenceRef, request.Header.Get("Idempotency-Key")); err != nil {
			writeRecoveryPhaseError(response, request, err)
			return
		}
	default:
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestInvalidJSON, errors.New("unknown review decision")))
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

type recoveryIdempotencyRequest struct {
	ScopeHash           string `json:"scope_hash"`
	KeyLocator          string `json:"key_locator"`
	ExecutionGeneration int64  `json:"execution_generation"`
	Decision            string `json:"decision"`
	EvidenceRef         string `json:"evidence_ref"`
}

func (h *RecoveryControlHandlers) resolveIdempotency(response http.ResponseWriter, request *http.Request, control recoveryControlRequest) {
	var input recoveryIdempotencyRequest
	if err := platform.DecodeJSON(request, h.dependencies.MaxBodyBytes, &input); err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	scope, err := decodeRecoveryDigest(input.ScopeHash)
	if err != nil {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestInvalidJSON, err))
		return
	}
	locator, err := decodeRecoveryDigest(input.KeyLocator)
	if err != nil {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestInvalidJSON, err))
		return
	}
	if input.ExecutionGeneration <= 0 || input.Decision != "failed" || strings.TrimSpace(input.EvidenceRef) == "" {
		_ = platform.WriteError(response, request, platform.NewPublicError(platform.CodeRequestInvalidJSON, errors.New("invalid idempotency review")))
		return
	}
	if err := h.dependencies.Maintenance.ResolveRecoveryIdempotency(request.Context(), control.permit, control.binding, scope, locator, input.ExecutionGeneration, input.Decision, input.EvidenceRef); err != nil {
		writeRecoveryPhaseError(response, request, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func decodeRecoveryDigest(value string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(raw) != 32 {
		return nil, errors.New("digest must be 32 base64url bytes")
	}
	return raw, nil
}

func (h *RecoveryControlHandlers) complete(response http.ResponseWriter, request *http.Request, _ recoveryControlRequest) {
	var input struct{}
	if err := platform.DecodeJSON(request, h.dependencies.MaxBodyBytes, &input); err != nil {
		_ = platform.WriteError(response, request, err)
		return
	}
	// Preflight is a deterministic refusal, so a closed signaler still
	// produces an error response rather than an accepted 200.
	if preflight, ok := h.dependencies.Complete.(RecoveryCompletePreflight); ok {
		if err := preflight.CheckComplete(); err != nil {
			writeRecoveryPhaseError(response, request, err)
			return
		}
	}
	// The HTTP call only validates and signals. The accepted response is
	// written and flushed first, and the root controller performs the real
	// executor call only after this handler has returned.
	if err := platform.WriteJSON(response, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "identity_ready"}); err != nil {
		slog.Default().ErrorContext(request.Context(), "recovery complete response write failed", "error", err)
		return
	}
	if flusher, ok := response.(http.Flusher); ok {
		flusher.Flush()
	}
	// A post-flush failure cannot change the response anymore; the signaler
	// records it and wakes the root, which surfaces it instead of waiting.
	if err := h.dependencies.Complete.RequestComplete(); err != nil {
		slog.Default().ErrorContext(request.Context(), "recovery complete signal failed", "error", err)
	}
}

// NewRecoveryMaintenanceStore exposes only the maintenance session store for
// the same sealed executor; it never creates a second authority.
func NewRecoveryMaintenanceStore(executor recovery.IdentityRecoveryExecutor, gate session.PasswordResetGate) (session.MaintenanceStore, error) {
	target := executor
	if forwarder, ok := executor.(*recoveryExecutionForwarder); ok {
		sealed, err := forwarder.get()
		if err != nil {
			return session.MaintenanceStore{}, err
		}
		target = sealed
	}
	execution, ok := target.(*recoveryIdentityExecution)
	if !ok {
		return session.MaintenanceStore{}, recovery.ErrAdmissionBinding
	}
	return execution.maintenanceSessions(gate)
}

var errRecoveryReviewCursor = errors.New("recovery review cursor is invalid")

// RecoveryReviewLister is the read-only pending-review projection. It binds
// every page to the current private phase and never returns credentials.
type RecoveryReviewLister struct {
	pool    *pgxpool.Pool
	runtime recovery.Runtime
}

func NewRecoveryReviewLister(pool *pgxpool.Pool, runtime recovery.Runtime) (*RecoveryReviewLister, error) {
	if pool == nil || runtime == nil {
		return nil, errors.New("recovery review lister dependencies are incomplete")
	}
	return &RecoveryReviewLister{pool: pool, runtime: runtime}, nil
}

type recoveryReviewCursor struct {
	RunID          string `json:"run_id"`
	InstallationID string `json:"installation_id"`
	LastUserID     string `json:"last_user_id"`
}

func (l *RecoveryReviewLister) ListPendingReviews(ctx context.Context, cursor string, limit int) ([]RecoveryReviewItem, string, error) {
	if limit < 1 || limit > recoveryReviewMaxLimit {
		return nil, "", recovery.ErrAdmissionBinding
	}
	if deadline, ok := ctx.Deadline(); !ok || !deadline.After(time.Now()) {
		return nil, "", recovery.ErrAdmissionBinding
	}
	binding, err := l.runtime.ReadPermitBinding(ctx)
	if err != nil {
		return nil, "", err
	}
	if binding.Purpose != recovery.PhaseRecoveryIdentity || binding.RunID == "" || binding.InstallationID == "" {
		return nil, "", recovery.ErrAdmissionBinding
	}
	last := recoveryReviewFirstPageBound
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return nil, "", errRecoveryReviewCursor
		}
		var decoded recoveryReviewCursor
		if err := json.Unmarshal(raw, &decoded); err != nil || decoded.RunID != binding.RunID || decoded.InstallationID != binding.InstallationID || !recoveryUUID.MatchString(decoded.LastUserID) {
			return nil, "", errRecoveryReviewCursor
		}
		last = decoded.LastUserID
	}
	rows, err := l.pool.Query(ctx, `SELECT user_id, decision FROM recovery_account_reviews WHERE recovery_run_id=$1 AND decision='pending' AND user_id>$2 ORDER BY user_id LIMIT $3`, binding.RunID, last, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]RecoveryReviewItem, 0, limit)
	for rows.Next() {
		var item RecoveryReviewItem
		if err := rows.Scan(&item.UserID, &item.Decision); err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(items) <= limit {
		return items, "", nil
	}
	items = items[:limit]
	next, err := json.Marshal(recoveryReviewCursor{RunID: binding.RunID, InstallationID: binding.InstallationID, LastUserID: items[len(items)-1].UserID})
	if err != nil {
		return nil, "", err
	}
	return items, base64.RawURLEncoding.EncodeToString(next), nil
}

// errRecoveryBodyInvalidUTF8 rejects any maintenance body that is not valid
// UTF-8 end to end. The platform decoder normalizes invalid bytes inside JSON
// strings, so the closed maintenance surface validates the raw body first.
var errRecoveryBodyInvalidUTF8 = errors.New("request body must be valid UTF-8")

// newRecoveryUTF8Body wraps a request body with strict UTF-8 validation.
func newRecoveryUTF8Body(source io.ReadCloser) io.ReadCloser {
	return &recoveryUTF8Body{source: source}
}

type recoveryUTF8Body struct {
	source  io.ReadCloser
	carry   []byte
	pending error
}

func (b *recoveryUTF8Body) Close() error {
	return b.source.Close()
}

func (b *recoveryUTF8Body) Read(destination []byte) (int, error) {
	if b.pending != nil {
		return 0, b.pending
	}
	if len(destination) == 0 {
		return 0, nil
	}
	for {
		read, err := b.source.Read(destination)
		delivered := read
		if read > 0 {
			combined := make([]byte, 0, len(b.carry)+read)
			combined = append(combined, b.carry...)
			combined = append(combined, destination[:read]...)
			prefix, ok := completeUTF8Prefix(combined)
			if !ok {
				b.pending = errRecoveryBodyInvalidUTF8
				return 0, b.pending
			}
			if prefix > len(destination) {
				prefix = len(destination)
			}
			copy(destination, combined[:prefix])
			b.carry = append(b.carry[:0], combined[prefix:]...)
			delivered = prefix
		}
		if err == io.EOF && len(b.carry) > 0 {
			b.pending = errRecoveryBodyInvalidUTF8
			return 0, b.pending
		}
		if delivered == 0 && err == nil {
			continue
		}
		return delivered, err
	}
}

// completeUTF8Prefix returns the length of the longest valid UTF-8 prefix and
// whether the remainder is merely a truncated rune. An invalid sequence
// reports ok=false.
func completeUTF8Prefix(buffer []byte) (int, bool) {
	index := 0
	for index < len(buffer) {
		if buffer[index] < utf8.RuneSelf {
			index++
			continue
		}
		r, size := utf8.DecodeRune(buffer[index:])
		if r == utf8.RuneError && size == 1 {
			return index, truncatedUTF8(buffer[index:])
		}
		index += size
	}
	return index, true
}

func truncatedUTF8(rest []byte) bool {
	width := 0
	switch lead := rest[0]; {
	case lead&0xE0 == 0xC0:
		width = 2
	case lead&0xF0 == 0xE0:
		width = 3
	case lead&0xF8 == 0xF0:
		width = 4
	default:
		return false
	}
	if len(rest) >= width {
		return false
	}
	for index := 1; index < len(rest); index++ {
		if rest[index]&0xC0 != 0x80 {
			return false
		}
	}
	return true
}
