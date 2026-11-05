package article

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"han/internal/platform"
)

func TestETag_FormatAndMatch_IsStableStrong(t *testing.T) {
	article := validTestArticle()
	got, err := StrongETag(article)
	if err != nil {
		t.Fatalf("StrongETag() error = %v", err)
	}
	want := `"article-10000000-0000-4000-8000-000000000001-v7"`
	if got != want {
		t.Fatalf("StrongETag() = %q, want %q", got, want)
	}
	if err := RequireMatchingETag(got, article); err != nil {
		t.Fatalf("RequireMatchingETag() error = %v", err)
	}
}

func TestETag_InvalidIfMatch_FailsClosed(t *testing.T) {
	article := validTestArticle()
	tests := []struct {
		name      string
		value     string
		wantError error
	}{
		{name: "missing", wantError: ErrPreconditionRequired},
		{name: "weak", value: `W/"article-10000000-0000-4000-8000-000000000001-v7"`, wantError: ErrInvalidPrecondition},
		{name: "multiple", value: `"article-10000000-0000-4000-8000-000000000001-v7","other"`, wantError: ErrInvalidPrecondition},
		{name: "wildcard", value: `*`, wantError: ErrInvalidPrecondition},
		{name: "unquoted", value: `article-10000000-0000-4000-8000-000000000001-v7`, wantError: ErrInvalidPrecondition},
		{name: "wrong resource format", value: `"resource-10000000-0000-4000-8000-000000000001-v7"`, wantError: ErrInvalidPrecondition},
		{name: "invalid uuid", value: `"article-not-a-uuid-v7"`, wantError: ErrInvalidPrecondition},
		{name: "invalid version", value: `"article-10000000-0000-4000-8000-000000000001-v0"`, wantError: ErrInvalidPrecondition},
		{name: "outer whitespace", value: ` "article-10000000-0000-4000-8000-000000000001-v7"`, wantError: ErrInvalidPrecondition},
		{name: "stale", value: `"article-10000000-0000-4000-8000-000000000001-v6"`, wantError: ErrPreconditionFailed},
		{name: "other article", value: `"article-10000000-0000-4000-8000-000000000009-v7"`, wantError: ErrPreconditionFailed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := RequireMatchingETag(test.value, article); !errors.Is(err, test.wantError) {
				t.Fatalf("RequireMatchingETag() error = %v, want %v", err, test.wantError)
			}
		})
	}

}

func TestETag_HTTPDuplicateHeaders_FailClosed(t *testing.T) {
	article := validTestArticle()
	current, err := StrongETag(article)
	if err != nil {
		t.Fatalf("StrongETag() error = %v", err)
	}
	tests := []struct {
		name   string
		values []string
	}{
		{name: "duplicate strong", values: []string{current, current}},
		{name: "strong and weak", values: []string{current, "W/" + current}},
		{name: "strong and wildcard", values: []string{current, "*"}},
		{name: "strong and stale", values: []string{
			current,
			`"article-10000000-0000-4000-8000-000000000001-v6"`,
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPut, "/api/v1/articles/one", nil)
			for _, value := range test.values {
				request.Header.Add("If-Match", value)
			}
			err := RequireRequestMatchingETag(request, article)
			if !errors.Is(err, ErrInvalidPrecondition) {
				t.Fatalf("RequireRequestMatchingETag() error = %v, want ErrInvalidPrecondition", err)
			}
			if code := platform.PublicCode(MapArticleError(err)); code != CodeRequestInvalidPrecondition {
				t.Fatalf("public code = %q, want %q", code, CodeRequestInvalidPrecondition)
			}
		})
	}

	t.Run("single exact", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPut, "/api/v1/articles/one", nil)
		request.Header.Set("If-Match", current)
		if err := RequireRequestMatchingETag(request, article); err != nil {
			t.Fatalf("RequireRequestMatchingETag() error = %v", err)
		}
	})
	t.Run("missing", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPut, "/api/v1/articles/one", nil)
		if err := RequireRequestMatchingETag(request, article); !errors.Is(err, ErrPreconditionRequired) {
			t.Fatalf("RequireRequestMatchingETag() error = %v, want ErrPreconditionRequired", err)
		}
	})
}

func TestArticleErrors_Mapping_IsStableAndFailClosed(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		code    platform.Code
		status  int
		message string
		retry   platform.RetryPolicy
	}{
		{name: "invalid cursor", err: ErrInvalidCursor, code: platform.CodeRequestInvalidCursor, status: http.StatusBadRequest, message: "分页游标无效", retry: platform.RetryAfterCorrection},
		{name: "invalid precondition", err: ErrInvalidPrecondition, code: CodeRequestInvalidPrecondition, status: http.StatusBadRequest, message: "资源版本条件格式无效", retry: platform.RetryAfterCorrection},
		{name: "forbidden", err: ErrArticleForbidden, code: platform.CodeAuthForbidden, status: http.StatusForbidden, message: "无权执行该操作", retry: platform.RetryNever},
		{name: "unavailable", err: ErrArticleUnavailable, code: CodeArticleUnavailable, status: http.StatusNotFound, message: "文章不存在或不可用", retry: platform.RetryNever},
		{name: "invalid state", err: ErrArticleInvalidState, code: CodeArticleInvalidState, status: http.StatusConflict, message: "当前文章状态不允许该操作", retry: platform.RetryAfterStateChange},
		{name: "stale version", err: ErrPreconditionFailed, code: CodeArticlePreconditionFailed, status: http.StatusPreconditionFailed, message: "文章已变化，请刷新后重试", retry: platform.RetryAllowed},
		{name: "validation", err: ErrArticleValidationFailed, code: CodeArticleValidationFailed, status: http.StatusUnprocessableEntity, message: "标题或正文不符合要求", retry: platform.RetryAfterCorrection},
		{name: "invalid transfer target", err: ErrTransferTargetInvalid, code: CodeArticleTransferTargetInvalid, status: http.StatusUnprocessableEntity, message: "目标用户不能成为新作者", retry: platform.RetryAfterCorrection},
		{name: "missing precondition", err: ErrPreconditionRequired, code: CodeRequestPreconditionRequired, status: http.StatusPreconditionRequired, message: "必须提供当前资源版本", retry: platform.RetryAllowed},
		{name: "deletion pending", err: ErrDeletionInProgress, code: CodeArticleDeletionInProgress, status: http.StatusServiceUnavailable, message: "删除正在完成，请稍后核对", retry: platform.RetryAllowed},
		{name: "unknown", err: errors.New("sql detail"), code: platform.CodePlatformInternalError, status: http.StatusInternalServerError, message: "服务内部错误", retry: platform.RetryConditional},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mapped := MapArticleError(test.err)
			if code := platform.PublicCode(mapped); code != test.code {
				t.Fatalf("PublicCode() = %q, want %q", code, test.code)
			}
			definition, ok := ArticleErrorDefinition(test.code)
			if !ok {
				t.Fatalf("ArticleErrorDefinition(%q) missing", test.code)
			}
			if definition.HTTPStatus != test.status ||
				definition.Message != test.message ||
				definition.Retry != test.retry {
				t.Fatalf(
					"definition = %#v, want status=%d message=%q retry=%q",
					definition,
					test.status,
					test.message,
					test.retry,
				)
			}
			if !errors.Is(mapped, test.err) {
				t.Fatalf("mapped error did not preserve cause %v", test.err)
			}
		})
	}
}

func TestArticleErrors_CrossDomainIdentityCodes_FailClosed(t *testing.T) {
	tests := []platform.Code{
		platform.CodeIdentityIdempotencyKeyRequired,
		platform.CodeAuthLoginFailed,
		platform.CodeAuthOriginRejected,
		platform.CodeAuthMustCompleteOnboarding,
		platform.CodeIdentityInitialAdminProtected,
		platform.CodePrivacyAcknowledgementRequired,
		platform.CodePrivacyInformationDeclarationRequired,
		platform.CodeIdentityUserNotFound,
		platform.CodeIdentityIdentifierConflict,
		platform.CodeIdentityInvalidState,
		platform.CodePrivacyNoticeOutdated,
		platform.CodeIdentityIdempotencyConflict,
		platform.CodeIdentityIdempotencyInProgress,
		platform.CodeIdentityValidationFailed,
		platform.CodeIdentityEmailDeliveryFailed,
		platform.CodeIdentityOperationUncertain,
	}

	for _, sourceCode := range tests {
		t.Run(string(sourceCode), func(t *testing.T) {
			internalCause := errors.New("identity detail must stay internal")
			crossDomain := platform.NewPublicError(sourceCode, internalCause)
			mapped := MapArticleError(crossDomain)
			if code := platform.PublicCode(mapped); code != platform.CodePlatformInternalError {
				t.Fatalf("PublicCode() = %q, want %q", code, platform.CodePlatformInternalError)
			}
			if !errors.Is(mapped, internalCause) {
				t.Fatal("mapped error did not preserve the identity cause")
			}

			request := httptest.NewRequest(http.MethodGet, "/api/v1/articles/one", nil)
			recorder := httptest.NewRecorder()
			if err := WriteArticleError(recorder, request, crossDomain); err != nil {
				t.Fatalf("WriteArticleError() error = %v", err)
			}
			var envelope platform.Envelope
			if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if envelope.Code != platform.CodePlatformInternalError {
				t.Fatalf("envelope.Code = %q, want %q", envelope.Code, platform.CodePlatformInternalError)
			}
			if strings.Contains(recorder.Body.String(), string(sourceCode)) ||
				strings.Contains(recorder.Body.String(), internalCause.Error()) {
				t.Fatalf("response leaked cross-domain identity detail: %s", recorder.Body.String())
			}
		})
	}
}

func TestArticleErrors_SharedAllowlist_PreservesCode(t *testing.T) {
	tests := []platform.Code{
		platform.CodeRequestInvalidJSON,
		platform.CodeRequestInvalidCursor,
		platform.CodeAuthSessionRequired,
		platform.CodeAuthCSRFFailed,
		platform.CodeAuthForbidden,
		platform.CodeRequestRateLimited,
		platform.CodePlatformProcessingGateClosed,
		platform.CodePlatformDatabaseUnavailable,
		platform.CodePlatformRecoveryInProgress,
		platform.CodePlatformDependencyUnavailable,
		platform.CodePlatformInternalError,
	}

	for _, sourceCode := range tests {
		t.Run(string(sourceCode), func(t *testing.T) {
			internalCause := errors.New("shared boundary cause")
			mapped := MapArticleError(platform.NewPublicError(sourceCode, internalCause))
			if code := platform.PublicCode(mapped); code != sourceCode {
				t.Fatalf("PublicCode() = %q, want %q", code, sourceCode)
			}
			if !errors.Is(mapped, internalCause) {
				t.Fatal("mapped error did not preserve the shared boundary cause")
			}
		})
	}
}

func TestArticleErrors_TransferTarget_UsesArticleCode(t *testing.T) {
	internalCause := errors.New("identity target was not found")
	identityError := platform.NewPublicError(platform.CodeIdentityUserNotFound, internalCause)
	mapped := MapArticleError(errors.Join(ErrTransferTargetInvalid, identityError))

	if code := platform.PublicCode(mapped); code != CodeArticleTransferTargetInvalid {
		t.Fatalf("PublicCode() = %q, want %q", code, CodeArticleTransferTargetInvalid)
	}
	if !errors.Is(mapped, internalCause) {
		t.Fatal("mapped transfer error did not preserve the identity cause")
	}
}

func TestArticleErrors_Response_UsesSharedEnvelopeWithoutLeakage(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/articles/secret", nil)
	ctx, err := platform.ContextWithTraceID(
		request.Context(),
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
	)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()
	internalCause := errors.New("hidden reason: author suspended")

	err = WriteArticleError(
		recorder,
		request,
		errors.Join(ErrArticleUnavailable, internalCause),
	)
	if err != nil {
		t.Fatalf("WriteArticleError() error = %v", err)
	}
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := recorder.Header().Get(platform.TraceHeader); got != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("%s = %q", platform.TraceHeader, got)
	}

	var envelope platform.Envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if envelope.Code != CodeArticleUnavailable ||
		envelope.Message != "文章不存在或不可用" ||
		envelope.TraceID != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Fatalf("envelope = %#v", envelope)
	}
	if strings.Contains(recorder.Body.String(), internalCause.Error()) ||
		strings.Contains(recorder.Body.String(), "suspended") {
		t.Fatalf("response leaked internal cause: %s", recorder.Body.String())
	}
}

func TestArticleErrors_SuccessResponse_SetsETagAndNoSniff(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := WriteArticleJSON(
		recorder,
		http.StatusOK,
		AuthorArticleView{
			Title:            "title",
			Body:             "body",
			ModerationStatus: ModerationNormal,
			Editable:         true,
		},
		`"article-10000000-0000-4000-8000-000000000001-v7"`,
	)
	if err != nil {
		t.Fatalf("WriteArticleJSON() error = %v", err)
	}
	if got := recorder.Header().Get("ETag"); got != `"article-10000000-0000-4000-8000-000000000001-v7"` {
		t.Fatalf("ETag = %q", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}
