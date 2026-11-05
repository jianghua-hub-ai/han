package platform

import (
	"errors"
	"net/http"
	"reflect"

	"han/internal/recovery"
)

// RecoveryGuard applies the runtime's ordinary request admission boundary.
type RecoveryGuard struct {
	runtime recovery.Runtime
}

// NewRecoveryGuard binds the guard to the root's shared runtime.
func NewRecoveryGuard(runtime recovery.Runtime) (RecoveryGuard, error) {
	if missingRecoveryHTTPDependency(runtime) {
		return RecoveryGuard{}, errors.New("recovery runtime is required")
	}
	return RecoveryGuard{runtime: runtime}, nil
}

// Protect wraps a selected route's Session and business handler. Exact routing
// must precede this guard. The original writer is passed through, and the lease
// remains owned here until the synchronous handler actually returns or panics.
func (g RecoveryGuard) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if missingRecoveryHTTPDependency(g.runtime) || missingRecoveryHTTPDependency(next) || request == nil {
			writeRecoveryHTTPClosed(response, request, recovery.ErrAdmissionClosed)
			return
		}
		lease, err := g.runtime.Acquire(request.Context(), recovery.WorkRequest)
		if !missingRecoveryHTTPDependency(lease) {
			defer lease.Release()
		}
		if err != nil || missingRecoveryHTTPDependency(lease) {
			writeRecoveryHTTPClosed(response, request, err)
			return
		}
		ctx := lease.Context()
		if missingRecoveryHTTPDependency(ctx) || ctx.Err() != nil || request.Context().Err() != nil {
			writeRecoveryHTTPClosed(response, request, recovery.ErrAdmissionClosed)
			return
		}
		next.ServeHTTP(response, request.WithContext(ctx))
	})
}

// Liveness reports process liveness without authorizing ordinary work.
func (g RecoveryGuard) Liveness(response http.ResponseWriter, _ *http.Request) {
	writeRecoveryHTTPProbe(response, http.StatusOK, "alive")
}

// Readiness reports only the shared runtime's public readiness.
func (g RecoveryGuard) Readiness(response http.ResponseWriter, request *http.Request) {
	if !missingRecoveryHTTPDependency(g.runtime) && request != nil && request.Context().Err() == nil {
		observation, err := g.runtime.Verify(request.Context())
		if err == nil && observation.State == "public_ready" && request.Context().Err() == nil {
			writeRecoveryHTTPProbe(response, http.StatusOK, "ready")
			return
		}
	}
	writeRecoveryHTTPProbe(response, http.StatusServiceUnavailable, "unavailable")
}

func writeRecoveryHTTPProbe(response http.ResponseWriter, status int, value string) {
	if missingRecoveryHTTPDependency(response) {
		return
	}
	response.Header().Set("X-Content-Type-Options", "nosniff")
	_ = WriteJSON(response, status, struct {
		Status string `json:"status"`
	}{Status: value})
}

func writeRecoveryHTTPClosed(response http.ResponseWriter, request *http.Request, cause error) {
	if missingRecoveryHTTPDependency(response) {
		return
	}
	response.Header().Set("X-Content-Type-Options", "nosniff")
	_ = WriteError(response, request, NewPublicError(CodePlatformRecoveryInProgress, cause))
}

func missingRecoveryHTTPDependency(dependency any) bool {
	if dependency == nil {
		return true
	}
	value := reflect.ValueOf(dependency)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
