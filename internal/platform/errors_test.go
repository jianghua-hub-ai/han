package platform

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

type customDecodedDetails struct {
	Label string `json:"label"`
}

type customDecodedObject struct {
	Name    string               `json:"name"`
	Details customDecodedDetails `json:"details"`
}

func (value *customDecodedObject) UnmarshalJSON(data []byte) error {
	type plain customDecodedObject
	return json.Unmarshal(data, (*plain)(value))
}

type customDecodedArray []customDecodedObject

func (value *customDecodedArray) UnmarshalJSON(data []byte) error {
	type plain customDecodedArray
	return json.Unmarshal(data, (*plain)(value))
}

func TestDefaultRegistry_AcceptedMappings_AreStableAndUnique(t *testing.T) {
	if got, want := DefaultRegistry.Len(), 29; got != want {
		t.Fatalf("DefaultRegistry.Len() = %d, want %d", got, want)
	}

	tests := []struct {
		code        Code
		status      int
		message     string
		retry       RetryPolicy
		logLevel    ErrorLogLevel
		responsible string
	}{
		{CodeRequestRouteNotFound, http.StatusNotFound, "请求的资源不存在", RetryNever, LogNone, "HTTP Router"},
		{CodeRequestMethodNotAllowed, http.StatusMethodNotAllowed, "请求方法不受支持", RetryAfterCorrection, LogNone, "HTTP Router"},
		{CodeRequestInvalidJSON, http.StatusBadRequest, "请求格式不正确", RetryAfterCorrection, LogNone, "HTTP Decoder"},
		{CodeRequestInvalidCursor, http.StatusBadRequest, "分页游标无效", RetryAfterCorrection, LogNone, "Pagination Decoder"},
		{CodeIdentityIdempotencyKeyRequired, http.StatusBadRequest, "缺少幂等键", RetryAfterCorrection, LogNone, "Idempotency Middleware"},
		{CodeAuthLoginFailed, http.StatusUnauthorized, "登录失败", RetryAllowed, LogNone, "IdentityService"},
		{CodeAuthSessionRequired, http.StatusUnauthorized, "需要重新登录", RetryAllowed, LogNone, "Session Middleware"},
		{CodeAuthOriginRejected, http.StatusForbidden, "请求来源不受信任", RetryNever, LogWarn, "Origin Middleware"},
		{CodeAuthCSRFFailed, http.StatusForbidden, "请求安全校验失败", RetryNever, LogWarn, "CSRF Middleware"},
		{CodeAuthMustCompleteOnboarding, http.StatusForbidden, "请先完成必要的账号确认", RetryAllowed, LogNone, "Onboarding Guard"},
		{CodeAuthForbidden, http.StatusForbidden, "无权执行该操作", RetryNever, LogWarn, "AuthorizationPolicy"},
		{CodeIdentityInitialAdminProtected, http.StatusForbidden, "初始管理员账号受保护", RetryNever, LogWarn, "AuthorizationPolicy"},
		{CodePrivacyAcknowledgementRequired, http.StatusForbidden, "必须确认当前隐私声明", RetryAllowed, LogNone, "PrivacyService"},
		{CodePrivacyInformationDeclarationRequired, http.StatusForbidden, "必须确认有权提供目标管理员信息", RetryAfterCorrection, LogNone, "PrivacyService"},
		{CodeIdentityUserNotFound, http.StatusNotFound, "用户不存在", RetryNever, LogNone, "IdentityQueryService"},
		{CodeIdentityIdentifierConflict, http.StatusConflict, "身份标识已被使用", RetryAfterCorrection, LogNone, "IdentityRepository"},
		{CodeIdentityInvalidState, http.StatusConflict, "当前状态不允许该操作", RetryAfterStateChange, LogNone, "IdentityService"},
		{CodePrivacyNoticeOutdated, http.StatusConflict, "隐私声明已更新，请重新确认", RetryAllowed, LogNone, "PrivacyService"},
		{CodeIdentityIdempotencyConflict, http.StatusConflict, "幂等键与原请求不一致", RetryWithNewKey, LogNone, "IdempotencyStore"},
		{CodeIdentityIdempotencyInProgress, http.StatusConflict, "原请求仍在处理中", RetryAllowed, LogNone, "IdempotencyStore"},
		{CodeIdentityValidationFailed, http.StatusUnprocessableEntity, "请求字段不符合要求", RetryAfterCorrection, LogNone, "Request Validator"},
		{CodeRequestRateLimited, http.StatusTooManyRequests, "请求过于频繁，请稍后重试", RetryAllowed, LogNone, "RateLimitResponseBoundary"},
		{CodePlatformProcessingGateClosed, http.StatusServiceUnavailable, "该能力尚未开放", RetryAfterConfiguration, LogNone, "ProcessingGate"},
		{CodePlatformDatabaseUnavailable, http.StatusServiceUnavailable, "服务暂时不可用", RetryAllowed, LogError, "PersistenceBoundary"},
		{CodeIdentityEmailDeliveryFailed, http.StatusServiceUnavailable, "邮件暂未受理", RetryAllowed, LogError, "Mailer 最终尝试"},
		{CodeIdentityOperationUncertain, http.StatusServiceUnavailable, "操作结果未确认，请等待受控核对", RetryAllowed, LogError, "IdentityOperationCoordinator"},
		{CodePlatformRecoveryInProgress, http.StatusServiceUnavailable, "服务正在恢复，暂不可用", RetryAllowed, LogNone, "RecoveryGate"},
		{CodePlatformDependencyUnavailable, http.StatusServiceUnavailable, "依赖服务暂时不可用", RetryAllowed, LogError, "对应 Adapter 最终边界"},
		{CodePlatformInternalError, http.StatusInternalServerError, "服务内部错误", RetryConditional, LogError, "Top-level Error Boundary"},
	}

	for _, test := range tests {
		t.Run(string(test.code), func(t *testing.T) {
			definition, ok := DefaultRegistry.Definition(test.code)
			if !ok {
				t.Fatalf("DefaultRegistry.Definition(%q) missing", test.code)
			}
			if definition.HTTPStatus != test.status ||
				definition.Message != test.message ||
				definition.Retry != test.retry ||
				definition.LogLevel != test.logLevel ||
				definition.Responsibility != test.responsible {
				t.Fatalf("definition = %#v, want status=%d message=%q retry=%q log=%q responsibility=%q",
					definition, test.status, test.message, test.retry, test.logLevel, test.responsible)
			}
		})
	}

	_, err := NewRegistry(
		Definition{Code: CodeRequestInvalidJSON, HTTPStatus: http.StatusBadRequest, Message: "one"},
		Definition{Code: CodeRequestInvalidJSON, HTTPStatus: http.StatusBadRequest, Message: "two"},
	)
	if err == nil {
		t.Fatal("NewRegistry() duplicate error = nil")
	}
}

func TestWriteError_PublicError_WritesEnvelopeWithoutInternalCause(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	ctx, err := ContextWithTraceID(request.Context(), "01ARZ3NDEKTSV4RRFFQ69G5FAV")
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()
	internalCause := errors.New("database detail that must stay internal")

	err = WriteError(recorder, request, NewPublicError(
		CodeIdentityValidationFailed,
		internalCause,
		FieldError{Field: "password", Code: "password_policy"},
	))
	if err != nil {
		t.Fatalf("WriteError() error = %v", err)
	}

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
	var envelope Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != CodeIdentityValidationFailed ||
		envelope.Message != "请求字段不符合要求" ||
		envelope.TraceID != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("envelope = %#v", envelope)
	}
	if len(envelope.FieldErrors) != 1 || envelope.FieldErrors[0].Field != "password" {
		t.Fatalf("field_errors = %#v", envelope.FieldErrors)
	}
	if strings.Contains(recorder.Body.String(), internalCause.Error()) {
		t.Fatalf("response leaked internal cause: %s", recorder.Body.String())
	}

	publicErr := NewPublicError(CodeIdentityValidationFailed, internalCause)
	if !errors.Is(publicErr, internalCause) {
		t.Fatal("NewPublicError() did not preserve internal cause for server-side handling")
	}
}

func TestDecodeJSON_InvalidInputs_ReturnStablePublicError(t *testing.T) {
	type details struct {
		Label string `json:"label"`
	}
	type payload struct {
		Name    string              `json:"name"`
		Details details             `json:"details"`
		Custom  customDecodedObject `json:"custom"`
		Items   customDecodedArray  `json:"items"`
	}
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "wrong content type", contentType: "text/plain", body: `{"name":"one"}`},
		{name: "unknown field", contentType: "application/json", body: `{"name":"one","extra":true}`},
		{name: "duplicate field", contentType: "application/json", body: `{"name":"one","name":"two"}`},
		{name: "case alias", contentType: "application/json", body: `{"Name":"one"}`},
		{name: "case variant conflict", contentType: "application/json", body: `{"name":"one","Name":"two"}`},
		{name: "nested case alias", contentType: "application/json", body: `{"name":"one","details":{"Label":"first"}}`},
		{name: "nested case variant conflict", contentType: "application/json", body: `{"name":"one","details":{"label":"first","Label":"second"}}`},
		{name: "custom decoder case alias", contentType: "application/json", body: `{"custom":{"Name":"first"}}`},
		{name: "custom decoder unknown field", contentType: "application/json", body: `{"custom":{"extra":true}}`},
		{name: "custom decoder nested conflict", contentType: "application/json", body: `{"custom":{"details":{"label":"first","Label":"second"}}}`},
		{name: "custom array decoder case alias", contentType: "application/json", body: `{"items":[{"Name":"first"}]}`},
		{name: "custom array decoder unknown field", contentType: "application/json", body: `{"items":[{"extra":true}]}`},
		{name: "custom array decoder conflict", contentType: "application/json", body: `{"items":[{"name":"first","Name":"second"}]}`},
		{name: "trailing value", contentType: "application/json", body: `{"name":"one"} {}`},
		{name: "empty body", contentType: "application/json", body: ``},
		{name: "top-level null", contentType: "application/json", body: `null`},
		{name: "top-level array", contentType: "application/json", body: `[]`},
		{name: "top-level string", contentType: "application/json", body: `"value"`},
		{name: "top-level number", contentType: "application/json", body: `7`},
		{name: "top-level boolean", contentType: "application/json", body: `true`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(test.body))
			request.Header.Set("Content-Type", test.contentType)
			var decoded payload

			err := DecodeJSON(request, 1024, &decoded)
			if err == nil {
				t.Fatal("DecodeJSON() error = nil")
			}
			if code := PublicCode(err); code != CodeRequestInvalidJSON {
				t.Fatalf("PublicCode() = %q, want %q; error=%v", code, CodeRequestInvalidJSON, err)
			}
		})
	}
}

func TestDecodeJSON_ExactNestedFieldNames_DecodesObject(t *testing.T) {
	type details struct {
		Label string `json:"label"`
	}
	type payload struct {
		Name    string  `json:"name"`
		Details details `json:"details"`
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewBufferString(`{"name":"one","details":{"label":"first"}}`),
	)
	request.Header.Set("Content-Type", "application/json")
	var decoded payload

	if err := DecodeJSON(request, 1024, &decoded); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if decoded.Name != "one" || decoded.Details.Label != "first" {
		t.Fatalf("decoded = %#v", decoded)
	}
}

func TestDecodeJSON_CustomScalarDecoder_RemainsAllowed(t *testing.T) {
	type payload struct {
		OccurredAt time.Time `json:"occurred_at"`
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		bytes.NewBufferString(`{"occurred_at":"2026-10-21T10:00:00Z"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	var decoded payload

	if err := DecodeJSON(request, 1024, &decoded); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if decoded.OccurredAt.IsZero() {
		t.Fatal("DecodeJSON() left occurred_at at zero value")
	}
}

func TestDecodeJSON_JSONTagCompatibilityMatchesEncodingJSONV1(t *testing.T) {
	t.Run("only complete dash tag is ignored", func(t *testing.T) {
		type payload struct {
			Ignored string `json:"-"`
		}
		var decoded payload

		if err := decodeJSONTestBody(`{}`, &decoded); err != nil {
			t.Fatalf("DecodeJSON(empty object) error = %v", err)
		}
		err := decodeJSONTestBody(`{"-":"not-consumable"}`, &decoded)
		if err == nil {
			t.Fatal("DecodeJSON(ignored dash field) error = nil")
		}
		if code := PublicCode(err); code != CodeRequestInvalidJSON {
			t.Fatalf("PublicCode() = %q, want %q", code, CodeRequestInvalidJSON)
		}
	})

	for _, test := range []struct {
		name string
		tag  reflect.StructTag
	}{
		{name: "dash with trailing comma", tag: `json:"-,"`},
		{name: "dash with omitempty", tag: `json:"-,omitempty"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			payloadType := reflect.StructOf([]reflect.StructField{{
				Name: "Value",
				Type: reflect.TypeFor[string](),
				Tag:  test.tag,
			}})
			decoded := reflect.New(payloadType)

			if err := decodeJSONTestBody(`{"-":"accepted"}`, decoded.Interface()); err != nil {
				t.Fatalf("DecodeJSON() error = %v", err)
			}
			if got := decoded.Elem().Field(0).String(); got != "accepted" {
				t.Fatalf("decoded value = %q, want accepted", got)
			}
		})
	}

	t.Run("UTF-8 and punctuation names remain valid", func(t *testing.T) {
		type payload struct {
			Unicode     string `json:"色は匂へど"`
			Punctuation string `json:"!#$%&()*+-./:;<=>?@[]^_{|}~ "`
		}
		var decoded payload

		if err := decodeJSONTestBody(
			`{"色は匂へど":"unicode","!#$%&()*+-./:;<=>?@[]^_{|}~ ":"punctuation"}`,
			&decoded,
		); err != nil {
			t.Fatalf("DecodeJSON() error = %v", err)
		}
		if decoded.Unicode != "unicode" || decoded.Punctuation != "punctuation" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	for _, test := range []struct {
		name          string
		tag           reflect.StructTag
		canonicalName string
		malformedName string
		rejectGoField bool
	}{
		{
			name:          "backslash after identifier",
			tag:           `json:"bad\\name"`,
			canonicalName: "bad", malformedName: `bad\name`, rejectGoField: true,
		},
		{
			name:          "single quote after identifier",
			tag:           `json:"bad'name"`,
			canonicalName: "bad", malformedName: "bad'name", rejectGoField: true,
		},
		{
			name:          "double quote after identifier",
			tag:           `json:"bad\"name"`,
			canonicalName: "bad", malformedName: `bad"name`, rejectGoField: true,
		},
		{
			name:          "backtick after identifier",
			tag:           reflect.StructTag("json:\"bad`name\""),
			canonicalName: "bad", malformedName: "bad`name", rejectGoField: true,
		},
		{
			name:          "punctuation terminates identifier prefix",
			tag:           `json:"bad-name\\tail"`,
			canonicalName: "bad", malformedName: `bad-name\tail`, rejectGoField: true,
		},
		{
			name:          "digit cannot start identifier fallback",
			tag:           `json:"123\\tail"`,
			canonicalName: "Value", malformedName: `123\tail`,
		},
		{
			name:          "underscore starts identifier",
			tag:           `json:"_bad\\tail"`,
			canonicalName: "_bad", malformedName: `_bad\tail`, rejectGoField: true,
		},
		{
			name:          "standard library invalid leading character fallback",
			tag:           `json:" !\"#&'()*+,."`,
			canonicalName: "Value", malformedName: ` !"#&'()*+,.`,
		},
	} {
		t.Run("best effort malformed tag "+test.name, func(t *testing.T) {
			payloadType := reflect.StructOf([]reflect.StructField{{
				Name: "Value",
				Type: reflect.TypeFor[string](),
				Tag:  test.tag,
			}})

			body, err := json.Marshal(map[string]string{test.canonicalName: "accepted"})
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			decoded := reflect.New(payloadType)
			if err := decodeJSONTestBody(string(body), decoded.Interface()); err != nil {
				t.Fatalf("DecodeJSON(canonical name %q) error = %v; cause = %v",
					test.canonicalName, err, errors.Unwrap(err))
			}
			if got := decoded.Elem().Field(0).String(); got != "accepted" {
				t.Fatalf("decoded value = %q, want accepted", got)
			}

			rejectedNames := []string{test.malformedName}
			if test.rejectGoField {
				rejectedNames = append(rejectedNames, "Value")
			}
			for _, rejectedName := range rejectedNames {
				body, err := json.Marshal(map[string]string{rejectedName: "rejected"})
				if err != nil {
					t.Fatalf("json.Marshal() error = %v", err)
				}
				rejected := reflect.New(payloadType)
				err = decodeJSONTestBody(string(body), rejected.Interface())
				if err == nil {
					t.Fatalf("DecodeJSON(rejected name %q) error = nil", rejectedName)
				}
				if code := PublicCode(err); code != CodeRequestInvalidJSON {
					t.Fatalf("PublicCode() = %q, want %q", code, CodeRequestInvalidJSON)
				}
			}
		})
	}
}

func TestDecodeJSON_UnexportedAnonymousAndEmbedCompatibilityMatchesEncodingJSONV1(t *testing.T) {
	type hidden struct {
		Value string `json:"value"`
	}

	t.Run("unnamed unexported struct promotes exported fields", func(t *testing.T) {
		type payload struct {
			hidden
		}
		var decoded payload

		if err := decodeJSONTestBody(`{"value":"promoted"}`, &decoded); err != nil {
			t.Fatalf("DecodeJSON() error = %v", err)
		}
		if decoded.Value != "promoted" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("named unexported struct is an object field", func(t *testing.T) {
		type payload struct {
			hidden `json:"named"`
		}
		var decoded payload

		if err := decodeJSONTestBody(`{"named":{"value":"nested"}}`, &decoded); err != nil {
			t.Fatalf("DecodeJSON() error = %v", err)
		}
		if decoded.Value != "nested" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("initialized unexported anonymous pointer is writable", func(t *testing.T) {
		type payload struct {
			*hidden
		}
		decoded := payload{hidden: &hidden{}}

		if err := decodeJSONTestBody(`{"value":"initialized"}`, &decoded); err != nil {
			t.Fatalf("DecodeJSON() error = %v", err)
		}
		if decoded.Value != "initialized" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("nil unexported anonymous pointer returns stable invalid JSON", func(t *testing.T) {
		type payload struct {
			*hidden
		}
		var decoded payload

		err := decodeJSONTestBody(`{"value":"cannot-allocate"}`, &decoded)
		if err == nil {
			t.Fatal("DecodeJSON() error = nil")
		}
		if code := PublicCode(err); code != CodeRequestInvalidJSON {
			t.Fatalf("PublicCode() = %q, want %q", code, CodeRequestInvalidJSON)
		}
	})

	t.Run("explicit embed option flattens non-anonymous struct", func(t *testing.T) {
		type payload struct {
			Embedded hidden `json:",embed"`
		}
		var decoded payload

		if err := decodeJSONTestBody(`{"value":"embedded"}`, &decoded); err != nil {
			t.Fatalf("DecodeJSON() error = %v", err)
		}
		if decoded.Embedded.Value != "embedded" {
			t.Fatalf("decoded = %#v", decoded)
		}

		err := decodeJSONTestBody(`{"Embedded":{"value":"not-an-object-field"}}`, &decoded)
		if err == nil {
			t.Fatal("DecodeJSON(non-embedded field name) error = nil")
		}
		if code := PublicCode(err); code != CodeRequestInvalidJSON {
			t.Fatalf("PublicCode() = %q, want %q", code, CodeRequestInvalidJSON)
		}
	})
}

func TestDecodeJSON_EmbeddedFieldDominanceMatchesEncodingJSON(t *testing.T) {
	t.Run("outer field dominates embedded field", func(t *testing.T) {
		type Embedded struct {
			Name string `json:"name"`
		}
		type payload struct {
			Embedded
			Name string `json:"name"`
		}
		var decoded payload

		if err := decodeJSONTestBody(`{"name":"outer"}`, &decoded); err != nil {
			t.Fatalf("DecodeJSON() error = %v", err)
		}
		if decoded.Name != "outer" || decoded.Embedded.Name != "" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("tagged field dominates untagged field at same depth", func(t *testing.T) {
		type Tagged struct {
			Value string `json:"Name"`
		}
		type Untagged struct {
			Name string
		}
		type payload struct {
			Tagged
			Untagged
		}
		var decoded payload

		if err := decodeJSONTestBody(`{"Name":"tagged"}`, &decoded); err != nil {
			t.Fatalf("DecodeJSON() error = %v", err)
		}
		if decoded.Tagged.Value != "tagged" || decoded.Untagged.Name != "" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("same-depth conflict removes consumable field", func(t *testing.T) {
		type Left struct {
			Name string `json:"name"`
		}
		type Right struct {
			Name string `json:"name"`
		}
		payloadType := reflect.StructOf([]reflect.StructField{
			{Name: "Left", Type: reflect.TypeOf(Left{}), Anonymous: true},
			{Name: "Right", Type: reflect.TypeOf(Right{}), Anonymous: true},
		})

		empty := reflect.New(payloadType).Interface()
		if err := decodeJSONTestBody(`{}`, empty); err != nil {
			t.Fatalf("DecodeJSON(empty object) error = %v", err)
		}

		conflicted := reflect.New(payloadType).Interface()
		err := decodeJSONTestBody(`{"name":"value"}`, conflicted)
		if err == nil {
			t.Fatal("DecodeJSON(conflicted field) error = nil")
		}
		if code := PublicCode(err); code != CodeRequestInvalidJSON {
			t.Fatalf("PublicCode() = %q, want %q", code, CodeRequestInvalidJSON)
		}
	})

	t.Run("anonymous pointer embedding is promoted", func(t *testing.T) {
		type Embedded struct {
			Name string `json:"name"`
		}
		type payload struct {
			*Embedded
		}
		var decoded payload

		if err := decodeJSONTestBody(`{"name":"pointer"}`, &decoded); err != nil {
			t.Fatalf("DecodeJSON() error = %v", err)
		}
		if decoded.Embedded == nil || decoded.Name != "pointer" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("recursive anonymous embedding terminates", func(t *testing.T) {
		type Payload struct {
			*Payload
			Value string `json:"value"`
		}
		var decoded Payload

		if err := decodeJSONTestBody(`{"value":"finite"}`, &decoded); err != nil {
			t.Fatalf("DecodeJSON() error = %v", err)
		}
		if decoded.Value != "finite" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})
}

func decodeJSONTestBody(body string, destination any) error {
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	return DecodeJSON(request, 4096, destination)
}

func TestTraceMiddleware_Request_AddsSameTraceToContextAndResponse(t *testing.T) {
	var contextTrace string
	handler := TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextTrace, _ = TraceIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	headerTrace := recorder.Header().Get(TraceHeader)
	if contextTrace == "" || headerTrace == "" || contextTrace != headerTrace {
		t.Fatalf("context trace = %q, header trace = %q", contextTrace, headerTrace)
	}
	if len(contextTrace) != 26 {
		t.Fatalf("trace length = %d, want 26", len(contextTrace))
	}
}
