package platform

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Code is a stable public REST error identifier.
type Code string

const (
	CodeRequestInvalidJSON                    Code = "request.invalid_json"
	CodeRequestInvalidCursor                  Code = "request.invalid_cursor"
	CodeRequestRouteNotFound                  Code = "request.route_not_found"
	CodeRequestMethodNotAllowed               Code = "request.method_not_allowed"
	CodeIdentityIdempotencyKeyRequired        Code = "identity.idempotency_key_required"
	CodeAuthLoginFailed                       Code = "auth.login_failed"
	CodeAuthSessionRequired                   Code = "auth.session_required"
	CodeAuthOriginRejected                    Code = "auth.origin_rejected"
	CodeAuthCSRFFailed                        Code = "auth.csrf_failed"
	CodeAuthMustCompleteOnboarding            Code = "auth.must_complete_onboarding"
	CodeAuthForbidden                         Code = "auth.forbidden"
	CodeIdentityInitialAdminProtected         Code = "identity.initial_admin_protected"
	CodePrivacyAcknowledgementRequired        Code = "privacy.acknowledgement_required"
	CodePrivacyInformationDeclarationRequired Code = "privacy.information_declaration_required"
	CodeIdentityUserNotFound                  Code = "identity.user_not_found"
	CodeIdentityIdentifierConflict            Code = "identity.identifier_conflict"
	CodeIdentityInvalidState                  Code = "identity.invalid_state"
	CodePrivacyNoticeOutdated                 Code = "privacy.notice_outdated"
	CodeIdentityIdempotencyConflict           Code = "identity.idempotency_conflict"
	CodeIdentityIdempotencyInProgress         Code = "identity.idempotency_in_progress"
	CodeIdentityValidationFailed              Code = "identity.validation_failed"
	CodeRequestRateLimited                    Code = "request.rate_limited"
	CodePlatformProcessingGateClosed          Code = "platform.processing_gate_closed"
	CodePlatformDatabaseUnavailable           Code = "platform.database_unavailable"
	CodeIdentityEmailDeliveryFailed           Code = "identity.email_delivery_failed"
	CodeIdentityOperationUncertain            Code = "identity.operation_uncertain"
	CodePlatformRecoveryInProgress            Code = "platform.recovery_in_progress"
	CodePlatformDependencyUnavailable         Code = "platform.dependency_unavailable"
	CodePlatformInternalError                 Code = "platform.internal_error"
)

// RetryPolicy describes the stable retry semantics of a public error.
type RetryPolicy string

const (
	RetryNever              RetryPolicy = "never"
	RetryAllowed            RetryPolicy = "allowed"
	RetryAfterCorrection    RetryPolicy = "after_correction"
	RetryAfterConfiguration RetryPolicy = "after_configuration"
	RetryAfterStateChange   RetryPolicy = "after_state_change"
	RetryWithNewKey         RetryPolicy = "with_new_key"
	RetryConditional        RetryPolicy = "conditional"
)

// ErrorLogLevel is the final responsibility-boundary log level.
type ErrorLogLevel string

const (
	LogNone  ErrorLogLevel = "NONE"
	LogWarn  ErrorLogLevel = "WARN"
	LogError ErrorLogLevel = "ERROR"
)

// Definition is the stable HTTP mapping for one public error code.
type Definition struct {
	Code           Code
	HTTPStatus     int
	Message        string
	Retry          RetryPolicy
	Responsibility string
	LogLevel       ErrorLogLevel
}

// Registry rejects duplicate public error mappings.
type Registry struct {
	definitions map[Code]Definition
}

// DefaultRegistry contains the accepted identity and shared-platform mappings.
var DefaultRegistry = mustNewRegistry(
	Definition{Code: CodeRequestRouteNotFound, HTTPStatus: http.StatusNotFound, Message: "请求的资源不存在", Retry: RetryNever, Responsibility: "HTTP Router", LogLevel: LogNone},
	Definition{Code: CodeRequestMethodNotAllowed, HTTPStatus: http.StatusMethodNotAllowed, Message: "请求方法不受支持", Retry: RetryAfterCorrection, Responsibility: "HTTP Router", LogLevel: LogNone},
	Definition{Code: CodeRequestInvalidJSON, HTTPStatus: http.StatusBadRequest, Message: "请求格式不正确", Retry: RetryAfterCorrection, Responsibility: "HTTP Decoder", LogLevel: LogNone},
	Definition{Code: CodeRequestInvalidCursor, HTTPStatus: http.StatusBadRequest, Message: "分页游标无效", Retry: RetryAfterCorrection, Responsibility: "Pagination Decoder", LogLevel: LogNone},
	Definition{Code: CodeIdentityIdempotencyKeyRequired, HTTPStatus: http.StatusBadRequest, Message: "缺少幂等键", Retry: RetryAfterCorrection, Responsibility: "Idempotency Middleware", LogLevel: LogNone},
	Definition{Code: CodeAuthLoginFailed, HTTPStatus: http.StatusUnauthorized, Message: "登录失败", Retry: RetryAllowed, Responsibility: "IdentityService", LogLevel: LogNone},
	Definition{Code: CodeAuthSessionRequired, HTTPStatus: http.StatusUnauthorized, Message: "需要重新登录", Retry: RetryAllowed, Responsibility: "Session Middleware", LogLevel: LogNone},
	Definition{Code: CodeAuthOriginRejected, HTTPStatus: http.StatusForbidden, Message: "请求来源不受信任", Retry: RetryNever, Responsibility: "Origin Middleware", LogLevel: LogWarn},
	Definition{Code: CodeAuthCSRFFailed, HTTPStatus: http.StatusForbidden, Message: "请求安全校验失败", Retry: RetryNever, Responsibility: "CSRF Middleware", LogLevel: LogWarn},
	Definition{Code: CodeAuthMustCompleteOnboarding, HTTPStatus: http.StatusForbidden, Message: "请先完成必要的账号确认", Retry: RetryAllowed, Responsibility: "Onboarding Guard", LogLevel: LogNone},
	Definition{Code: CodeAuthForbidden, HTTPStatus: http.StatusForbidden, Message: "无权执行该操作", Retry: RetryNever, Responsibility: "AuthorizationPolicy", LogLevel: LogWarn},
	Definition{Code: CodeIdentityInitialAdminProtected, HTTPStatus: http.StatusForbidden, Message: "初始管理员账号受保护", Retry: RetryNever, Responsibility: "AuthorizationPolicy", LogLevel: LogWarn},
	Definition{Code: CodePrivacyAcknowledgementRequired, HTTPStatus: http.StatusForbidden, Message: "必须确认当前隐私声明", Retry: RetryAllowed, Responsibility: "PrivacyService", LogLevel: LogNone},
	Definition{Code: CodePrivacyInformationDeclarationRequired, HTTPStatus: http.StatusForbidden, Message: "必须确认有权提供目标管理员信息", Retry: RetryAfterCorrection, Responsibility: "PrivacyService", LogLevel: LogNone},
	Definition{Code: CodeIdentityUserNotFound, HTTPStatus: http.StatusNotFound, Message: "用户不存在", Retry: RetryNever, Responsibility: "IdentityQueryService", LogLevel: LogNone},
	Definition{Code: CodeIdentityIdentifierConflict, HTTPStatus: http.StatusConflict, Message: "身份标识已被使用", Retry: RetryAfterCorrection, Responsibility: "IdentityRepository", LogLevel: LogNone},
	Definition{Code: CodeIdentityInvalidState, HTTPStatus: http.StatusConflict, Message: "当前状态不允许该操作", Retry: RetryAfterStateChange, Responsibility: "IdentityService", LogLevel: LogNone},
	Definition{Code: CodePrivacyNoticeOutdated, HTTPStatus: http.StatusConflict, Message: "隐私声明已更新，请重新确认", Retry: RetryAllowed, Responsibility: "PrivacyService", LogLevel: LogNone},
	Definition{Code: CodeIdentityIdempotencyConflict, HTTPStatus: http.StatusConflict, Message: "幂等键与原请求不一致", Retry: RetryWithNewKey, Responsibility: "IdempotencyStore", LogLevel: LogNone},
	Definition{Code: CodeIdentityIdempotencyInProgress, HTTPStatus: http.StatusConflict, Message: "原请求仍在处理中", Retry: RetryAllowed, Responsibility: "IdempotencyStore", LogLevel: LogNone},
	Definition{Code: CodeIdentityValidationFailed, HTTPStatus: http.StatusUnprocessableEntity, Message: "请求字段不符合要求", Retry: RetryAfterCorrection, Responsibility: "Request Validator", LogLevel: LogNone},
	Definition{Code: CodeRequestRateLimited, HTTPStatus: http.StatusTooManyRequests, Message: "请求过于频繁，请稍后重试", Retry: RetryAllowed, Responsibility: "RateLimitResponseBoundary", LogLevel: LogNone},
	Definition{Code: CodePlatformProcessingGateClosed, HTTPStatus: http.StatusServiceUnavailable, Message: "该能力尚未开放", Retry: RetryAfterConfiguration, Responsibility: "ProcessingGate", LogLevel: LogNone},
	Definition{Code: CodePlatformDatabaseUnavailable, HTTPStatus: http.StatusServiceUnavailable, Message: "服务暂时不可用", Retry: RetryAllowed, Responsibility: "PersistenceBoundary", LogLevel: LogError},
	Definition{Code: CodeIdentityEmailDeliveryFailed, HTTPStatus: http.StatusServiceUnavailable, Message: "邮件暂未受理", Retry: RetryAllowed, Responsibility: "Mailer 最终尝试", LogLevel: LogError},
	Definition{Code: CodeIdentityOperationUncertain, HTTPStatus: http.StatusServiceUnavailable, Message: "操作结果未确认，请等待受控核对", Retry: RetryAllowed, Responsibility: "IdentityOperationCoordinator", LogLevel: LogError},
	Definition{Code: CodePlatformRecoveryInProgress, HTTPStatus: http.StatusServiceUnavailable, Message: "服务正在恢复，暂不可用", Retry: RetryAllowed, Responsibility: "RecoveryGate", LogLevel: LogNone},
	Definition{Code: CodePlatformDependencyUnavailable, HTTPStatus: http.StatusServiceUnavailable, Message: "依赖服务暂时不可用", Retry: RetryAllowed, Responsibility: "对应 Adapter 最终边界", LogLevel: LogError},
	Definition{Code: CodePlatformInternalError, HTTPStatus: http.StatusInternalServerError, Message: "服务内部错误", Retry: RetryConditional, Responsibility: "Top-level Error Boundary", LogLevel: LogError},
)

// NewRegistry creates an immutable public error registry.
func NewRegistry(definitions ...Definition) (Registry, error) {
	registered := make(map[Code]Definition, len(definitions))
	for _, definition := range definitions {
		if definition.Code == "" {
			return Registry{}, errors.New("public error code is required")
		}
		if definition.HTTPStatus < 400 || definition.HTTPStatus > 599 {
			return Registry{}, fmt.Errorf("public error %q has invalid HTTP status", definition.Code)
		}
		if strings.TrimSpace(definition.Message) == "" {
			return Registry{}, fmt.Errorf("public error %q has empty message", definition.Code)
		}
		if !validRetryPolicy(definition.Retry) {
			return Registry{}, fmt.Errorf("public error %q has invalid retry policy", definition.Code)
		}
		if strings.TrimSpace(definition.Responsibility) == "" {
			return Registry{}, fmt.Errorf("public error %q has empty responsibility", definition.Code)
		}
		if !validErrorLogLevel(definition.LogLevel) {
			return Registry{}, fmt.Errorf("public error %q has invalid log level", definition.Code)
		}
		if _, exists := registered[definition.Code]; exists {
			return Registry{}, fmt.Errorf("duplicate public error code %q", definition.Code)
		}
		registered[definition.Code] = definition
	}
	return Registry{definitions: registered}, nil
}

// Definition returns the mapping for code.
func (r Registry) Definition(code Code) (Definition, bool) {
	definition, exists := r.definitions[code]
	return definition, exists
}

// Len returns the number of registered definitions.
func (r Registry) Len() int {
	return len(r.definitions)
}

// FieldError identifies one invalid request field without exposing internal causes.
type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}

// Envelope is the shared REST error response.
type Envelope struct {
	Code        Code         `json:"code"`
	Message     string       `json:"message"`
	FieldErrors []FieldError `json:"field_errors,omitempty"`
	TraceID     string       `json:"trace_id"`
}

// PublicError keeps the internal cause separate from the public mapping.
type PublicError struct {
	code        Code
	fieldErrors []FieldError
	cause       error
}

// NewPublicError maps an internal cause to one registered public code.
func NewPublicError(code Code, cause error, fieldErrors ...FieldError) error {
	return &PublicError{code: code, cause: cause, fieldErrors: fieldErrors}
}

// Error returns a diagnostic string for server-side use.
func (e *PublicError) Error() string {
	if e == nil {
		return ""
	}
	return string(e.code)
}

// Unwrap exposes the internal cause to server-side error handling.
func (e *PublicError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// PublicCode returns the stable public code carried by err.
func PublicCode(err error) Code {
	var publicError *PublicError
	if errors.As(err, &publicError) {
		return publicError.code
	}
	return CodePlatformInternalError
}

// WriteError writes the shared REST error envelope.
func WriteError(response http.ResponseWriter, request *http.Request, err error) error {
	if response == nil {
		return errors.New("HTTP response writer is required")
	}

	traceID, ok := traceIDFromRequest(request)
	if !ok {
		var traceErr error
		traceID, traceErr = NewTraceID()
		if traceErr != nil {
			return fmt.Errorf("create error response trace ID: %w", traceErr)
		}
	}

	code := PublicCode(err)
	definition, registered := DefaultRegistry.Definition(code)
	if !registered {
		code = CodePlatformInternalError
		definition, _ = DefaultRegistry.Definition(code)
	}

	var fieldErrors []FieldError
	var publicError *PublicError
	if errors.As(err, &publicError) && publicError.code == code {
		fieldErrors = append([]FieldError(nil), publicError.fieldErrors...)
	}

	response.Header().Set(TraceHeader, traceID)
	return WriteJSON(response, definition.HTTPStatus, Envelope{
		Code:        code,
		Message:     definition.Message,
		FieldErrors: fieldErrors,
		TraceID:     traceID,
	})
}

func mustNewRegistry(definitions ...Definition) Registry {
	registry, err := NewRegistry(definitions...)
	if err != nil {
		panic(err)
	}
	return registry
}

func validRetryPolicy(policy RetryPolicy) bool {
	switch policy {
	case RetryNever, RetryAllowed, RetryAfterCorrection, RetryAfterConfiguration,
		RetryAfterStateChange, RetryWithNewKey, RetryConditional:
		return true
	default:
		return false
	}
}

func validErrorLogLevel(level ErrorLogLevel) bool {
	switch level {
	case LogNone, LogWarn, LogError:
		return true
	default:
		return false
	}
}

func traceIDFromRequest(request *http.Request) (string, bool) {
	if request == nil {
		return "", false
	}
	return TraceIDFromContext(request.Context())
}
