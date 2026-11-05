package platform

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"han/internal/session"
)

func TestOrigin_MissingMaliciousOrCrossSiteEvidence_IsRejected(t *testing.T) {
	guard, err := NewOriginGuard(
		"https://community.example.invalid",
		&originObserverStub{},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewOriginGuard() error = %v", err)
	}
	tests := []struct {
		name      string
		origin    string
		fetchSite string
	}{
		{name: "missing origin"},
		{name: "malicious origin", origin: "https://attacker.example.invalid", fetchSite: "cross-site"},
		{name: "cross-site metadata", origin: "https://community.example.invalid", fetchSite: "cross-site"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := guard.Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			}))
			request := httptest.NewRequest(http.MethodPost, "https://community.example.invalid/api/v1/sessions", nil)
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			if test.fetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", test.fetchSite)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if called {
				t.Fatal("protected handler was called")
			}
			requireEnvelopeCode(t, response, http.StatusForbidden, CodeAuthOriginRejected)
		})
	}
}

func TestOriginObserver_ReportsFourReasonsAfterSingleResponseWrite(t *testing.T) {
	tests := []struct {
		name       string
		request    *http.Request
		wantReason OriginRejectionReason
	}{
		{name: "request nil", wantReason: OriginRejectionRequestNil},
		{
			name:       "origin missing",
			request:    httptest.NewRequest(http.MethodPost, "https://community.example.invalid/api/v1/sessions", nil),
			wantReason: OriginRejectionOriginMissing,
		},
		{
			name: "origin mismatch",
			request: func() *http.Request {
				request := httptest.NewRequest(http.MethodPost, "https://community.example.invalid/api/v1/sessions", nil)
				request.Header.Set("Origin", "https://attacker.example.invalid")
				return request
			}(),
			wantReason: OriginRejectionOriginMismatch,
		},
		{
			name: "fetch site rejected",
			request: func() *http.Request {
				request := httptest.NewRequest(http.MethodPost, "https://community.example.invalid/api/v1/sessions", nil)
				request.Header.Set("Origin", "https://community.example.invalid")
				request.Header.Set("Sec-Fetch-Site", "cross-site")
				return request
			}(),
			wantReason: OriginRejectionFetchSiteRejected,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			order := make([]string, 0, 2)
			observer := &originObserverStub{onObserve: func(
				_ context.Context,
				observation OriginRejectionObservation,
			) error {
				order = append(order, "observe")
				if observation.Reason != test.wantReason ||
					observation.ResponseWrite != OriginResponseWriteOK {
					t.Fatalf("observation = %#v", observation)
				}
				return nil
			}}
			guard, err := NewObservedOriginGuard(
				"https://community.example.invalid",
				observer,
				time.Second,
			)
			if err != nil {
				t.Fatalf("NewObservedOriginGuard() error = %v", err)
			}
			response := &originOrderedWriter{
				ResponseRecorder: httptest.NewRecorder(),
				onWrite: func() {
					order = append(order, "write")
				},
			}
			called := false

			guard.Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				called = true
			})).ServeHTTP(response, test.request)

			if called {
				t.Fatal("protected handler was called")
			}
			if strings.Join(order, ",") != "write,observe" {
				t.Fatalf("order = %v, want [write observe]", order)
			}
			if observer.calls != 1 {
				t.Fatalf("observer calls = %d, want 1", observer.calls)
			}
			requireEnvelopeCode(t, response.ResponseRecorder, http.StatusForbidden, CodeAuthOriginRejected)
		})
	}
}

func TestOriginRejectionObserverAdapter_StopsBeforeMetricsWhenCanceled(t *testing.T) {
	logger := &originLoggerStub{}
	metrics := &originMetricsStub{}
	observer, err := NewOriginRejectionObserver(logger, metrics)
	if err != nil {
		t.Fatalf("NewOriginRejectionObserver() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = observer.ObserveRejected(ctx, OriginRejectionObservation{
		Reason:        OriginRejectionOriginMissing,
		ResponseWrite: OriginResponseWriteOK,
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ObserveRejected() error = %v, want context canceled", err)
	}
	if logger.calls != 0 || metrics.calls != 0 {
		t.Fatalf("logger/metrics calls = %d/%d, want 0/0", logger.calls, metrics.calls)
	}
}

func TestOrigin_F005FailsClosedWithoutBoundedComposition(t *testing.T) {
	if originRejectionMetricName != "han_platform_origin_rejections_total" {
		t.Fatal("Origin rejection metric name changed")
	}
	if _, err := NewOriginRejectionObserver(nil, &originMetricsStub{}); err == nil {
		t.Fatal("NewOriginRejectionObserver() accepted a missing logger")
	}
	if _, err := NewOriginRejectionObserver(&originLoggerStub{}, nil); err == nil {
		t.Fatal("NewOriginRejectionObserver() accepted missing metrics")
	}
	if _, err := NewSlogOriginRejectionLogger(nil, "han-server"); err == nil {
		t.Fatal("NewSlogOriginRejectionLogger() accepted a missing base logger")
	}
	if _, err := NewSlogOriginRejectionLogger(slog.Default(), " "); err == nil {
		t.Fatal("NewSlogOriginRejectionLogger() accepted a blank service")
	}
	tests := []struct {
		name     string
		observer OriginRejectionObserver
		budget   time.Duration
	}{
		{name: "missing observer", budget: time.Second},
		{name: "zero budget", observer: &originObserverStub{}},
		{name: "negative budget", observer: &originObserverStub{}, budget: -time.Second},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewOriginGuard(
				"https://community.example.invalid",
				test.observer,
				test.budget,
			); err == nil {
				t.Fatal("NewOriginGuard() error = nil, want fail-closed composition")
			}
		})
	}
}

func TestOrigin_F005WriterFailureIsObservedWithoutRetryOrHandlerCall(t *testing.T) {
	observer := &originObserverStub{}
	guard, err := NewOriginGuard(
		"https://community.example.invalid",
		observer,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewOriginGuard() error = %v", err)
	}
	response := &originFailingWriter{header: make(http.Header)}
	handlerCalls := 0
	request := httptest.NewRequest(
		http.MethodPost,
		"https://community.example.invalid/api/v1/sessions",
		nil,
	)

	guard.Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalls++
	})).ServeHTTP(response, request)

	if handlerCalls != 0 {
		t.Fatalf("protected handler calls = %d, want 0", handlerCalls)
	}
	if response.writeCalls != 1 {
		t.Fatalf("response body write calls = %d, want 1", response.writeCalls)
	}
	if observer.calls != 1 ||
		observer.observed[0] != (OriginRejectionObservation{
			Reason:        OriginRejectionOriginMissing,
			ResponseWrite: OriginResponseWriteFailed,
		}) {
		t.Fatalf("observations = %#v", observer.observed)
	}
}

func TestOrigin_F005UsesEarlierDeadlineAndBoundsBlockingObserver(t *testing.T) {
	t.Run("request deadline wins", func(t *testing.T) {
		var observedDeadline time.Time
		observer := &originObserverStub{onObserve: func(
			ctx context.Context,
			_ OriginRejectionObservation,
		) error {
			observedDeadline, _ = ctx.Deadline()
			return nil
		}}
		guard, err := NewOriginGuard(
			"https://community.example.invalid",
			observer,
			time.Hour,
		)
		if err != nil {
			t.Fatalf("NewOriginGuard() error = %v", err)
		}
		requestDeadline := time.Now().Add(time.Minute)
		request := httptest.NewRequest(
			http.MethodPost,
			"https://community.example.invalid/api/v1/sessions",
			nil,
		)
		ctx, cancel := context.WithDeadline(request.Context(), requestDeadline)
		defer cancel()

		guard.Protect(http.NotFoundHandler()).ServeHTTP(
			httptest.NewRecorder(),
			request.WithContext(ctx),
		)

		if !observedDeadline.Equal(requestDeadline) {
			t.Fatalf("observer deadline = %v, want request deadline %v", observedDeadline, requestDeadline)
		}
	})

	t.Run("composition budget cancels blocking observer", func(t *testing.T) {
		var observedErr error
		observer := &originObserverStub{onObserve: func(
			ctx context.Context,
			_ OriginRejectionObservation,
		) error {
			<-ctx.Done()
			observedErr = ctx.Err()
			return observedErr
		}}
		guard, err := NewOriginGuard(
			"https://community.example.invalid",
			observer,
			20*time.Millisecond,
		)
		if err != nil {
			t.Fatalf("NewOriginGuard() error = %v", err)
		}
		returned := make(chan struct{})
		go func() {
			guard.Protect(http.NotFoundHandler()).ServeHTTP(
				httptest.NewRecorder(),
				httptest.NewRequest(
					http.MethodPost,
					"https://community.example.invalid/api/v1/sessions",
					nil,
				),
			)
			close(returned)
		}()

		select {
		case <-returned:
		case <-time.After(time.Second):
			t.Fatal("OriginGuard did not return within the bounded observation budget")
		}
		if !errors.Is(observedErr, context.DeadlineExceeded) || observer.calls != 1 {
			t.Fatalf("observer calls/error = %d/%v", observer.calls, observedErr)
		}
	})
}

func TestOrigin_F005ObserverFailureDoesNotChangePublicResponse(t *testing.T) {
	observerErr := errors.New("synthetic Origin observer failure")
	observer := &originObserverStub{onObserve: func(
		context.Context,
		OriginRejectionObservation,
	) error {
		return observerErr
	}}
	guard, err := NewOriginGuard(
		"https://community.example.invalid",
		observer,
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewOriginGuard() error = %v", err)
	}
	response := httptest.NewRecorder()
	handlerCalls := 0

	guard.Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		handlerCalls++
	})).ServeHTTP(
		response,
		httptest.NewRequest(
			http.MethodPost,
			"https://community.example.invalid/api/v1/sessions",
			nil,
		),
	)

	if handlerCalls != 0 {
		t.Fatalf("protected handler calls = %d, want 0", handlerCalls)
	}
	requireEnvelopeCode(t, response, http.StatusForbidden, CodeAuthOriginRejected)
	if observer.calls != 1 {
		t.Fatalf("observer calls = %d, want 1", observer.calls)
	}
	if strings.Contains(response.Body.String(), observerErr.Error()) {
		t.Fatal("public response exposed the Origin observer failure")
	}
}

func TestOriginRejectionObserverAdapter_StopsFanoutAfterSinkFailure(t *testing.T) {
	loggerErr := errors.New("synthetic origin logger failure")
	logger := &originLoggerStub{err: loggerErr}
	metrics := &originMetricsStub{}
	observer, err := NewOriginRejectionObserver(logger, metrics)
	if err != nil {
		t.Fatalf("NewOriginRejectionObserver() error = %v", err)
	}

	err = observer.ObserveRejected(context.Background(), OriginRejectionObservation{
		Reason:        OriginRejectionOriginMissing,
		ResponseWrite: OriginResponseWriteOK,
	})

	if !errors.Is(err, loggerErr) {
		t.Fatalf("ObserveRejected() error = %v, want logger failure", err)
	}
	if logger.calls != 1 || metrics.calls != 0 {
		t.Fatalf("logger/metrics calls = %d/%d, want 1/0", logger.calls, metrics.calls)
	}
}

func TestOriginRejectionObserverAdapter_SinksObeyCancellation(t *testing.T) {
	t.Run("logger blocks metrics", func(t *testing.T) {
		logger := &originLoggerStub{blockUntilCanceled: true}
		metrics := &originMetricsStub{}
		observer, err := NewOriginRejectionObserver(logger, metrics)
		if err != nil {
			t.Fatalf("NewOriginRejectionObserver() error = %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		err = observer.ObserveRejected(ctx, OriginRejectionObservation{
			Reason:        OriginRejectionOriginMissing,
			ResponseWrite: OriginResponseWriteOK,
		})

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("ObserveRejected() error = %v, want deadline exceeded", err)
		}
		if logger.calls != 1 || metrics.calls != 0 {
			t.Fatalf("logger/metrics calls = %d/%d, want 1/0", logger.calls, metrics.calls)
		}
	})

	t.Run("metrics returns on deadline", func(t *testing.T) {
		logger := &originLoggerStub{}
		metrics := &originMetricsStub{blockUntilCanceled: true}
		observer, err := NewOriginRejectionObserver(logger, metrics)
		if err != nil {
			t.Fatalf("NewOriginRejectionObserver() error = %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		err = observer.ObserveRejected(ctx, OriginRejectionObservation{
			Reason:        OriginRejectionOriginMissing,
			ResponseWrite: OriginResponseWriteOK,
		})

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("ObserveRejected() error = %v, want deadline exceeded", err)
		}
		if logger.calls != 1 || metrics.calls != 1 {
			t.Fatalf("logger/metrics calls = %d/%d, want 1/1", logger.calls, metrics.calls)
		}
	})
}

func TestSlogOriginRejectionLogger_EmitsOnlyStableWarningFields(t *testing.T) {
	capture := &originSlogCapture{}
	logger, err := NewSlogOriginRejectionLogger(
		slog.New(&originSlogHandler{capture: capture}),
		"han-server",
	)
	if err != nil {
		t.Fatalf("NewSlogOriginRejectionLogger() error = %v", err)
	}
	ctx, err := ContextWithTraceID(
		context.Background(),
		"01ARZ3NDEKTSV4RRFFQ69G5FAV",
	)
	if err != nil {
		t.Fatalf("ContextWithTraceID() error = %v", err)
	}
	observation := OriginRejectionObservation{
		Reason:        OriginRejectionOriginMismatch,
		ResponseWrite: OriginResponseWriteFailed,
	}

	if err := logger.LogOriginRejection(ctx, observation); err != nil {
		t.Fatalf("LogOriginRejection() error = %v", err)
	}

	if len(capture.records) != 1 {
		t.Fatalf("log records = %d, want 1", len(capture.records))
	}
	record := capture.records[0]
	if record.level != slog.LevelWarn {
		t.Fatalf("log level = %v, want WARN", record.level)
	}
	want := map[string]string{
		"service":        "han-server",
		"module":         "platform",
		"trace_id":       "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"event":          "platform.http.origin_rejected",
		"reason":         string(observation.Reason),
		"response_write": string(observation.ResponseWrite),
	}
	if len(record.attributes) != len(want) {
		t.Fatalf("log attribute keys = %v", record.attributes)
	}
	for key, value := range want {
		if record.attributes[key] != value {
			t.Fatalf("log attribute %q = %q, want %q", key, record.attributes[key], value)
		}
	}
	encoded, err := json.Marshal(record.attributes)
	if err != nil {
		t.Fatalf("marshal captured origin log fields: %v", err)
	}
	for _, forbidden := range []string{
		"https://attacker.example.invalid",
		"cross-site",
		"192.0.2.1",
		"Origin",
		"Sec-Fetch-Site",
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatal("origin log fields contained forbidden request material")
		}
	}
}

func TestSlogOriginRejectionLogger_ObeysCancellationWithoutRetry(t *testing.T) {
	capture := &originSlogCapture{blockUntilCanceled: true}
	logger, err := NewSlogOriginRejectionLogger(
		slog.New(&originSlogHandler{capture: capture}),
		"han-server",
	)
	if err != nil {
		t.Fatalf("NewSlogOriginRejectionLogger() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err = logger.LogOriginRejection(ctx, OriginRejectionObservation{
		Reason:        OriginRejectionOriginMissing,
		ResponseWrite: OriginResponseWriteOK,
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("LogOriginRejection() error = %v, want deadline exceeded", err)
	}
	if capture.handleCalls != 1 {
		t.Fatalf("slog handler calls = %d, want 1", capture.handleCalls)
	}
}

func TestOrigin_ExactOriginAndSameOriginMetadata_IsAllowed(t *testing.T) {
	guard, err := NewOriginGuard(
		"https://community.example.invalid",
		&originObserverStub{},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewOriginGuard() error = %v", err)
	}
	handler := guard.Protect(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "https://community.example.invalid/api/v1/sessions", nil)
	request.Header.Set("Origin", "https://community.example.invalid")
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestCORS_CrossOriginCredentialsAndInnerHeaders_AreNotExposed(t *testing.T) {
	guard, err := NewOriginGuard(
		"https://community.example.invalid",
		&originObserverStub{},
		time.Second,
	)
	if err != nil {
		t.Fatalf("NewOriginGuard() error = %v", err)
	}
	handler := guard.Protect(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Access-Control-Allow-Origin", "https://attacker.example.invalid")
		response.Header().Set("Access-Control-Allow-Credentials", "true")
		response.WriteHeader(http.StatusNoContent)
	}))

	for _, origin := range []string{
		"https://attacker.example.invalid",
		"https://community.example.invalid",
	} {
		t.Run(origin, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodOptions, "https://community.example.invalid/api/v1/sessions", nil)
			request.Header.Set("Origin", origin)
			request.Header.Set("Sec-Fetch-Site", map[bool]string{true: "same-origin", false: "cross-site"}[origin == "https://community.example.invalid"])
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if value := response.Header().Get("Access-Control-Allow-Origin"); value != "" {
				t.Fatalf("Access-Control-Allow-Origin = %q, want empty", value)
			}
			if value := response.Header().Get("Access-Control-Allow-Credentials"); value != "" {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", value)
			}
		})
	}
}

func TestSessionCookie_UsesHostOnlyProtectedAttributesWithoutTBDLifetime(t *testing.T) {
	response := httptest.NewRecorder()
	if err := SetSessionCookie(response, "synthetic-session-token"); err != nil {
		t.Fatalf("SetSessionCookie() error = %v", err)
	}

	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != SessionCookieName || cookie.Value != "synthetic-session-token" {
		t.Fatalf("cookie identity = %s/%q", cookie.Name, cookie.Value)
	}
	if cookie.Path != "/" || cookie.Domain != "" || !cookie.Secure || !cookie.HttpOnly ||
		cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie attributes = %#v", cookie)
	}
	if cookie.MaxAge != 0 || !cookie.Expires.IsZero() {
		t.Fatal("cookie contains an unapproved persistent lifetime")
	}
}

func TestSessionCookie_InvalidValue_IsRejected(t *testing.T) {
	response := httptest.NewRecorder()
	if err := SetSessionCookie(response, "invalid\r\ncookie"); err == nil {
		t.Fatal("SetSessionCookie() error = nil, want invalid cookie value rejected")
	}
	if len(response.Header().Values("Set-Cookie")) != 0 {
		t.Fatal("SetSessionCookie() wrote a cookie after rejecting its value")
	}
}

func TestCSRF_ProtectedWriteMissingOrForgedToken_IsRejected(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		csrfToken   string
		resolverErr error
		wantCode    Code
	}{
		{name: "missing", resolverErr: session.ErrCSRFFailed, wantCode: CodeAuthCSRFFailed},
		{name: "forged", csrfToken: "forged-csrf-token", resolverErr: session.ErrCSRFFailed, wantCode: CodeAuthCSRFFailed},
		{name: "old cookie", resolverErr: session.ErrSessionRequired, wantCode: CodeAuthSessionRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := &resolverStub{err: test.resolverErr}
			authorizer, err := NewHTTPAuthorizer(resolver, func() time.Time { return now })
			if err != nil {
				t.Fatalf("NewHTTPAuthorizer() error = %v", err)
			}
			handler := authorizer.Protect(session.AccessGeneral, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("protected handler was called")
			}))
			request := httptest.NewRequest(http.MethodPost, "https://community.example.invalid/api/v1/articles", nil)
			request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "synthetic-session-token"})
			if test.csrfToken != "" {
				request.Header.Set(CSRFHeaderName, test.csrfToken)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			requireEnvelopeCode(t, response, DefaultRegistry.definitions[test.wantCode].HTTPStatus, test.wantCode)
			if resolver.input.Now != now || !resolver.input.RequireCSRF {
				t.Fatalf("ResolveInput = %#v, want explicit now and CSRF requirement", resolver.input)
			}
		})
	}
}

func TestSession_RestrictedPrincipalCannotEnterGeneralCapability(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	resolver := &resolverStub{principal: session.Principal{
		UserID:             "00000000-0000-4000-8000-000000000201",
		Status:             "active",
		MustChangePassword: true,
	}}
	authorizer, err := NewHTTPAuthorizer(resolver, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	handler := authorizer.Protect(session.AccessGeneral, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	}))
	request := httptest.NewRequest(http.MethodGet, "https://community.example.invalid/api/v1/articles", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "synthetic-session-token"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	requireEnvelopeCode(t, response, http.StatusForbidden, CodeAuthMustCompleteOnboarding)
}

func TestSession_AllowedPrincipalIsAttachedToRequestContext(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	want := session.Principal{
		UserID: "00000000-0000-4000-8000-000000000201",
		Status: "active",
	}
	resolver := &resolverStub{principal: want}
	authorizer, err := NewHTTPAuthorizer(resolver, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	handler := authorizer.Protect(session.AccessGeneral, http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		got, ok := PrincipalFromContext(request.Context())
		if !ok || got != want {
			t.Fatalf("PrincipalFromContext() = %#v, %t, want %#v, true", got, ok, want)
		}
	}))
	request := httptest.NewRequest(http.MethodGet, "https://community.example.invalid/api/v1/articles", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "synthetic-session-token"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if resolver.input.RequireCSRF {
		t.Fatal("safe GET unexpectedly required CSRF")
	}
}

func TestSessionStoreError_PublicResponseDoesNotLeakDatabaseCause(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	databaseCause := errors.New("synthetic private database detail")
	resolver := &resolverStub{err: errors.Join(session.ErrStoreUnavailable, databaseCause)}
	authorizer, err := NewHTTPAuthorizer(resolver, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewHTTPAuthorizer() error = %v", err)
	}
	handler := authorizer.Protect(session.AccessGeneral, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called")
	}))
	request := httptest.NewRequest(http.MethodGet, "https://community.example.invalid/api/v1/articles", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "synthetic-session-token"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	requireEnvelopeCode(
		t,
		response,
		DefaultRegistry.definitions[CodePlatformDatabaseUnavailable].HTTPStatus,
		CodePlatformDatabaseUnavailable,
	)
	if strings.Contains(response.Body.String(), databaseCause.Error()) {
		t.Fatal("public response leaked the database cause")
	}
}

type resolverStub struct {
	principal session.Principal
	err       error
	input     session.ResolveInput
}

type originObserverStub struct {
	calls     int
	observed  []OriginRejectionObservation
	onObserve func(context.Context, OriginRejectionObservation) error
}

func (s *originObserverStub) ObserveRejected(
	ctx context.Context,
	observation OriginRejectionObservation,
) error {
	s.calls++
	s.observed = append(s.observed, observation)
	if s.onObserve != nil {
		return s.onObserve(ctx, observation)
	}
	return nil
}

type originLoggerStub struct {
	calls              int
	err                error
	blockUntilCanceled bool
}

func (s *originLoggerStub) LogOriginRejection(
	ctx context.Context,
	_ OriginRejectionObservation,
) error {
	s.calls++
	if s.blockUntilCanceled {
		<-ctx.Done()
		return ctx.Err()
	}
	return s.err
}

type originMetricsStub struct {
	calls              int
	err                error
	blockUntilCanceled bool
}

func (s *originMetricsStub) RecordOriginRejection(
	ctx context.Context,
	_ OriginRejectionObservation,
) error {
	s.calls++
	if s.blockUntilCanceled {
		<-ctx.Done()
		return ctx.Err()
	}
	return s.err
}

type originOrderedWriter struct {
	*httptest.ResponseRecorder
	onWrite func()
}

func (w *originOrderedWriter) WriteHeader(statusCode int) {
	if w.onWrite != nil {
		w.onWrite()
		w.onWrite = nil
	}
	w.ResponseRecorder.WriteHeader(statusCode)
}

func (w *originOrderedWriter) Write(body []byte) (int, error) {
	if w.onWrite != nil {
		w.onWrite()
		w.onWrite = nil
	}
	return w.ResponseRecorder.Write(body)
}

type originFailingWriter struct {
	header     http.Header
	writeCalls int
}

func (w *originFailingWriter) Header() http.Header {
	return w.header
}

func (*originFailingWriter) WriteHeader(int) {}

func (w *originFailingWriter) Write([]byte) (int, error) {
	w.writeCalls++
	return 0, errors.New("synthetic origin response write failure")
}

type capturedOriginLog struct {
	level      slog.Level
	attributes map[string]string
}

type originSlogCapture struct {
	handleCalls        int
	blockUntilCanceled bool
	records            []capturedOriginLog
}

type originSlogHandler struct {
	capture    *originSlogCapture
	attributes []slog.Attr
}

func (*originSlogHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *originSlogHandler) Handle(ctx context.Context, record slog.Record) error {
	h.capture.handleCalls++
	if h.capture.blockUntilCanceled {
		<-ctx.Done()
		return ctx.Err()
	}
	attributes := make(map[string]string, len(h.attributes)+record.NumAttrs())
	for _, attribute := range h.attributes {
		attributes[attribute.Key] = attribute.Value.String()
	}
	record.Attrs(func(attribute slog.Attr) bool {
		attributes[attribute.Key] = attribute.Value.String()
		return true
	})
	h.capture.records = append(h.capture.records, capturedOriginLog{
		level:      record.Level,
		attributes: attributes,
	})
	return nil
}

func (h *originSlogHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	combined := append([]slog.Attr(nil), h.attributes...)
	combined = append(combined, attributes...)
	return &originSlogHandler{capture: h.capture, attributes: combined}
}

func (h *originSlogHandler) WithGroup(string) slog.Handler {
	return h
}

func (r *resolverStub) ResolvePrincipal(_ context.Context, input session.ResolveInput) (session.Principal, error) {
	r.input = input
	return r.principal, r.err
}

func requireEnvelopeCode(t *testing.T, response *httptest.ResponseRecorder, status int, code Code) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d", response.Code, status)
	}
	var envelope Envelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Code != code {
		t.Fatalf("envelope code = %q, want %q", envelope.Code, code)
	}
	if envelope.TraceID == "" {
		t.Fatal("envelope trace_id is empty")
	}
}
