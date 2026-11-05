package identity

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"han/internal/idempotency"
	"han/internal/platform"
	"han/internal/session"
)

func TestCreateAdministratorMaterializesStoredFailureThroughHandler(t *testing.T) {
	const traceID = "01ARZ3NDEKTSV4RRFFQ69G5FAB"
	snapshot := idempotency.ResponseSnapshot{
		SchemaVersion: idempotency.AdministratorCreateResponseSchemaV1,
		HTTPStatus:    http.StatusConflict,
		ResultCode:    string(platform.CodeIdentityIdentifierConflict),
		Body: &idempotency.AdministratorCreateResponseBody{
			Code:        string(platform.CodeIdentityIdentifierConflict),
			Message:     "\u8eab\u4efd\u6807\u8bc6\u5df2\u88ab\u4f7f\u7528",
			FieldErrors: []idempotency.ResponseFieldError{},
		},
	}
	create := administratorCreateHandlerStub{
		err: newAdministratorResponseError(snapshot, errors.New("stored failure")),
	}
	handler := &Handler{
		administratorCreate:      &create,
		administratorCreateBytes: 4096,
	}
	resolver := administratorHandlerPrincipalResolver{principal: session.Principal{
		UserID:         "00000000-0000-4000-8000-000000000101",
		Role:           string(RoleAdmin),
		IsInitialAdmin: true,
		Status:         string(UserStatusActive),
	}}
	authorizer, err := platform.NewHTTPAuthorizer(
		resolver,
		func() time.Time { return time.Date(2026, time.October, 23, 0, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	protected := authorizer.Protect(
		session.AccessGeneral,
		http.HandlerFunc(handler.CreateAdministrator),
	)
	ctx, err := platform.ContextWithTraceID(context.Background(), traceID)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"https://community.example.invalid/api/v1/admin/administrators",
		strings.NewReader(`{
			"username":"new-admin",
			"email":"new-admin@example.invalid",
			"phone":"+12025550123",
			"temporary_password":"not-a-real-secret",
			"information_provision_notice_version":"admin-info-v1",
			"information_provision_declared":true
		}`),
	).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "handler-test-key")
	request.Header.Set(platform.CSRFHeaderName, "handler-test-csrf")
	request.AddCookie(&http.Cookie{
		Name:  platform.SessionCookieName,
		Value: "handler-test-session",
	})
	response := httptest.NewRecorder()

	protected.ServeHTTP(response, request)

	if create.calls != 1 {
		t.Fatalf("CreateAdministrator() calls = %d, want 1", create.calls)
	}
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	if got := response.Header().Get(platform.TraceHeader); got != traceID {
		t.Fatalf("%s = %q, want %q", platform.TraceHeader, got, traceID)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 4 {
		t.Fatalf("response fields = %v, want strict four-field error envelope", body)
	}
	fieldErrors, ok := body["field_errors"]
	if !ok || string(fieldErrors) != "[]" {
		t.Fatalf("field_errors = %s, present = %t, want []", fieldErrors, ok)
	}
}

func TestAdministratorCreateErrorMaterializesCurrentTraceAndStableSnapshot(t *testing.T) {
	const (
		traceA = "01ARZ3NDEKTSV4RRFFQ69G5FAA"
		traceB = "01ARZ3NDEKTSV4RRFFQ69G5FAB"
	)
	snapshot := idempotency.ResponseSnapshot{
		SchemaVersion: idempotency.AdministratorCreateResponseSchemaV1,
		HTTPStatus:    http.StatusConflict,
		ResultCode:    string(platform.CodeIdentityIdentifierConflict),
		Body: &idempotency.AdministratorCreateResponseBody{
			Code:        string(platform.CodeIdentityIdentifierConflict),
			Message:     "\u8eab\u4efd\u6807\u8bc6\u5df2\u88ab\u4f7f\u7528",
			FieldErrors: []idempotency.ResponseFieldError{},
		},
	}
	ctx, err := platform.ContextWithTraceID(context.Background(), traceB)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/administrators", nil).
		WithContext(ctx)
	response := httptest.NewRecorder()

	if !writeAdministratorCreateError(
		response,
		request,
		newAdministratorResponseError(snapshot, errors.New("stored trace "+traceA)),
	) {
		t.Fatal("writeAdministratorCreateError() did not handle snapshot")
	}
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	if got := response.Header().Get(platform.TraceHeader); got != traceB {
		t.Fatalf("%s = %q, want %q", platform.TraceHeader, got, traceB)
	}
	if strings.Contains(response.Body.String(), traceA) {
		t.Fatalf("response leaked historical trace: %s", response.Body.String())
	}
	var body struct {
		Code        string                `json:"code"`
		Message     string                `json:"message"`
		FieldErrors []platform.FieldError `json:"field_errors"`
		TraceID     string                `json:"trace_id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != string(platform.CodeIdentityIdentifierConflict) ||
		body.Message != snapshot.Body.Message ||
		body.FieldErrors == nil || len(body.FieldErrors) != 0 ||
		body.TraceID != traceB {
		t.Fatalf("response body = %#v", body)
	}
}

type administratorCreateHandlerStub struct {
	calls int
	err   error
}

func (s *administratorCreateHandlerStub) CreateAdministrator(
	context.Context,
	CreateAdministratorRequest,
) (CreatedAdministrator, error) {
	s.calls++
	return CreatedAdministrator{}, s.err
}

type administratorHandlerPrincipalResolver struct {
	principal session.Principal
}

func (r administratorHandlerPrincipalResolver) ResolvePrincipal(
	context.Context,
	session.ResolveInput,
) (session.Principal, error) {
	return r.principal, nil
}
