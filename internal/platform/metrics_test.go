package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

func TestParseHistogramBuckets(t *testing.T) {
	baseline := "0.005,0.01,0.025,0.05,0.1,0.25,0.5,1,2.5,5"
	thirtyTwo := make([]string, 32)
	for index := range thirtyTwo {
		thirtyTwo[index] = strconv.Itoa(index + 1)
	}
	tests := []struct {
		name    string
		raw     string
		wantLen int
		wantErr bool
	}{
		{name: "baseline", raw: baseline, wantLen: 10},
		{name: "legal non baseline", raw: "0.001,0.2,60", wantLen: 3},
		{name: "one item", raw: "60", wantLen: 1},
		{name: "thirty two items", raw: strings.Join(thirtyTwo, ","), wantLen: 32},
		{name: "missing", raw: "", wantErr: true},
		{name: "blank", raw: " \t ", wantErr: true},
		{name: "empty item", raw: "1,,2", wantErr: true},
		{name: "thirty three items", raw: strings.Join(append(thirtyTwo, "33"), ","), wantErr: true},
		{name: "not a number", raw: "one", wantErr: true},
		{name: "nan", raw: "NaN", wantErr: true},
		{name: "positive infinity", raw: "+Inf", wantErr: true},
		{name: "overflow", raw: "1e999", wantErr: true},
		{name: "zero", raw: "0", wantErr: true},
		{name: "negative", raw: "-1", wantErr: true},
		{name: "descending", raw: "2,1", wantErr: true},
		{name: "duplicate", raw: "1,1", wantErr: true},
		{name: "over maximum", raw: "60.0001", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseHistogramBuckets(test.raw)
			if test.wantErr {
				if err == nil {
					t.Fatalf("ParseHistogramBuckets(%q) error = nil", test.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseHistogramBuckets(%q) error = %v", test.raw, err)
			}
			if len(got) != test.wantLen {
				t.Fatalf("bucket count = %d, want %d", len(got), test.wantLen)
			}
		})
	}
}

func TestMetricsRegistry_RecordsFixedLowCardinalityFamiliesConcurrently(t *testing.T) {
	registry, err := NewMetricsRegistry([]float64{0.1, 1})
	if err != nil {
		t.Fatalf("NewMetricsRegistry() error = %v", err)
	}
	const workers = 24
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			ctx := context.Background()
			if err := registry.RecordLoginAttempt(ctx, "POST /api/v1/sessions", "success", "none", "none"); err != nil {
				t.Errorf("RecordLoginAttempt() error = %v", err)
			}
			if err := registry.RecordLoginDuration(ctx, "POST /api/v1/sessions", "success", "none", "none", 0.25); err != nil {
				t.Errorf("RecordLoginDuration() error = %v", err)
			}
			if err := registry.RecordOriginRejection(ctx, "origin_missing", "ok"); err != nil {
				t.Errorf("RecordOriginRejection() error = %v", err)
			}
		}()
	}
	group.Wait()

	body, err := registry.encodeText()
	if err != nil {
		t.Fatalf("encodeText() error = %v", err)
	}
	text := string(body)
	for _, want := range []string{
		`han_identity_login_attempts_total{endpoint="POST /api/v1/sessions",outcome="success",reason_bucket="none",error_code="none"} 24`,
		`han_identity_login_duration_seconds_bucket{endpoint="POST /api/v1/sessions",outcome="success",reason_bucket="none",error_code="none",le="0.1"} 0`,
		`han_identity_login_duration_seconds_bucket{endpoint="POST /api/v1/sessions",outcome="success",reason_bucket="none",error_code="none",le="1"} 24`,
		`han_identity_login_duration_seconds_bucket{endpoint="POST /api/v1/sessions",outcome="success",reason_bucket="none",error_code="none",le="+Inf"} 24`,
		`han_identity_login_duration_seconds_sum{endpoint="POST /api/v1/sessions",outcome="success",reason_bucket="none",error_code="none"} 6`,
		`han_identity_login_duration_seconds_count{endpoint="POST /api/v1/sessions",outcome="success",reason_bucket="none",error_code="none"} 24`,
		`han_platform_origin_rejections_total{reason="origin_missing",response_write="ok"} 24`,
	} {
		if !strings.Contains(text, want+"\n") {
			t.Fatalf("exposition missing %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"trace_id=", "username=", "user_id=", "origin=", "url="} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("exposition contains forbidden label %q", forbidden)
		}
	}
}

func TestMetricsHandler_PrometheusText004(t *testing.T) {
	registry, err := NewMetricsRegistry([]float64{0.005, 1})
	if err != nil {
		t.Fatalf("NewMetricsRegistry() error = %v", err)
	}
	if err := registry.RecordLoginAttempt(context.Background(), "POST /api/v1/sessions", "internal_failure", "persistence", "platform.database_unavailable"); err != nil {
		t.Fatalf("RecordLoginAttempt() error = %v", err)
	}
	if err := registry.RecordLoginInternalFailure(context.Background(), "POST /api/v1/sessions", "persistence", "platform.database_unavailable", "transaction_commit"); err != nil {
		t.Fatalf("RecordLoginInternalFailure() error = %v", err)
	}
	if err := registry.RecordLoginDuration(context.Background(), "POST /api/v1/sessions", "internal_failure", "persistence", "platform.database_unavailable", 0.25); err != nil {
		t.Fatalf("RecordLoginDuration() error = %v", err)
	}

	handler := NewMetricsHandler(registry)
	for _, accept := range []string{"", "application/openmetrics-text", "application/json"} {
		request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		request.Header.Set("Accept", accept)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("Accept %q status = %d, want 200", accept, response.Code)
		}
		if got := response.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
			t.Fatalf("Content-Type = %q", got)
		}
		if !utf8.Valid(response.Body.Bytes()) || bytes.Contains(response.Body.Bytes(), []byte("\r")) ||
			!bytes.HasSuffix(response.Body.Bytes(), []byte("\n")) {
			t.Fatalf("invalid text 0.0.4 body %q", response.Body.Bytes())
		}
		if err := parsePrometheusText004(response.Body.Bytes()); err != nil {
			t.Fatalf("independent parser rejected exposition: %v\n%s", err, response.Body)
		}
	}
}

func TestMetricsHandler_OnlyExposesGETMetrics(t *testing.T) {
	handler := NewMetricsHandler(mustMetricsRegistry(t))
	for _, test := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/"},
		{method: http.MethodGet, path: "/api/v1/sessions"},
		{method: http.MethodPost, path: "/metrics"},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(test.method, test.path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, want 404", test.method, test.path, response.Code)
		}
		if strings.Contains(response.Body.String(), "han_") {
			t.Fatalf("%s %s leaked metrics", test.method, test.path)
		}
	}
}

func TestMetricsHandler_EncodingFailureIsAtomicAndRedacted(t *testing.T) {
	internalCause := errors.New("synthetic-secret-encoder-cause")
	handler := newMetricsHandler(mustMetricsRegistry(t), metricsEncoderFunc(func(*MetricsRegistry) ([]byte, error) {
		return []byte("partial-sensitive-canary"), internalCause
	}))
	response := &commitTrackingWriter{header: make(http.Header)}

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if response.status != http.StatusInternalServerError || response.commits != 1 {
		t.Fatalf("status/commits = %d/%d, want 500/1", response.status, response.commits)
	}
	if got := response.header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := response.body.String(); got != "metrics encoding failed\n" {
		t.Fatalf("body = %q", got)
	}
	if strings.Contains(response.body.String(), internalCause.Error()) ||
		strings.Contains(response.body.String(), "partial-sensitive-canary") {
		t.Fatal("failure response exposed encoder state")
	}
}

func TestMetricsTextEscapingNamesAndFloats(t *testing.T) {
	for _, test := range []struct {
		value float64
		want  string
	}{
		{value: 1.25, want: "1.25"},
		{value: math.NaN(), want: "NaN"},
		{value: math.Inf(1), want: "+Inf"},
		{value: math.Inf(-1), want: "-Inf"},
	} {
		if got := formatMetricFloat(test.value); got != test.want {
			t.Fatalf("formatMetricFloat(%v) = %q, want %q", test.value, got, test.want)
		}
	}
	if got := escapeMetricHelp("slash\\line\nnext"); got != `slash\\line\nnext` {
		t.Fatalf("escaped HELP = %q", got)
	}
	if got := escapeMetricLabel("slash\\quote\"\nnext"); got != `slash\\quote\"\nnext` {
		t.Fatalf("escaped label = %q", got)
	}
	for _, name := range []string{"a", "_a", ":a", "a:b", "a_1"} {
		if !validMetricName(name) {
			t.Fatalf("valid metric name rejected: %q", name)
		}
	}
	for _, name := range []string{"", "1a", "a-b", "a b"} {
		if validMetricName(name) {
			t.Fatalf("invalid metric name accepted: %q", name)
		}
	}
	for _, name := range []string{"a", "_a", "a_1"} {
		if !validMetricLabelName(name) {
			t.Fatalf("valid label name rejected: %q", name)
		}
	}
	for _, name := range []string{"", "1a", ":a", "a-b"} {
		if validMetricLabelName(name) {
			t.Fatalf("invalid label name accepted: %q", name)
		}
	}
}

func mustMetricsRegistry(t *testing.T) *MetricsRegistry {
	t.Helper()
	registry, err := NewMetricsRegistry([]float64{0.1, 1})
	if err != nil {
		t.Fatalf("NewMetricsRegistry() error = %v", err)
	}
	return registry
}

func parsePrometheusText004(body []byte) error {
	if !utf8.Valid(body) || len(body) == 0 || body[len(body)-1] != '\n' ||
		bytes.Contains(body, []byte{'\r'}) {
		return errors.New("body framing is invalid")
	}
	types := make(map[string]string)
	for number, line := range strings.Split(strings.TrimSuffix(string(body), "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "# HELP "):
			parts := strings.SplitN(strings.TrimPrefix(line, "# HELP "), " ", 2)
			if len(parts) != 2 || !validMetricName(parts[0]) {
				return fmt.Errorf("line %d invalid HELP", number+1)
			}
		case strings.HasPrefix(line, "# TYPE "):
			parts := strings.Fields(line)
			if len(parts) != 4 || !validMetricName(parts[2]) ||
				(parts[3] != "counter" && parts[3] != "histogram") {
				return fmt.Errorf("line %d invalid TYPE", number+1)
			}
			types[parts[2]] = parts[3]
		case strings.HasPrefix(line, "#"):
			return fmt.Errorf("line %d unknown directive", number+1)
		default:
			if err := parseMetricSample(line); err != nil {
				return fmt.Errorf("line %d: %w", number+1, err)
			}
		}
	}
	for _, name := range []string{
		"han_identity_login_attempts_total",
		"han_identity_login_internal_failures_total",
		"han_identity_login_duration_seconds",
		"han_platform_origin_rejections_total",
	} {
		if types[name] == "" {
			return fmt.Errorf("missing TYPE for %s", name)
		}
	}
	return nil
}

func parseMetricSample(line string) error {
	nameAndLabels, value, found := splitMetricSample(line)
	if !found || value == "" {
		return errors.New("sample value is missing")
	}
	name := nameAndLabels
	if open := strings.IndexByte(nameAndLabels, '{'); open >= 0 {
		if !strings.HasSuffix(nameAndLabels, "}") {
			return errors.New("labels are not closed")
		}
		name = nameAndLabels[:open]
		labels := nameAndLabels[open+1 : len(nameAndLabels)-1]
		for _, label := range splitMetricLabels(labels) {
			key, quoted, ok := strings.Cut(label, "=")
			if !ok || !validMetricLabelName(key) || len(quoted) < 2 ||
				quoted[0] != '"' || quoted[len(quoted)-1] != '"' {
				return errors.New("invalid label")
			}
		}
	}
	if !validMetricName(name) {
		return errors.New("invalid sample name")
	}
	switch value {
	case "NaN", "+Inf", "-Inf":
		return nil
	default:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return errors.New("invalid sample value")
		}
		return nil
	}
}

func splitMetricSample(line string) (string, string, bool) {
	quoted := false
	escaped := false
	for index, character := range line {
		switch {
		case escaped:
			escaped = false
		case character == '\\' && quoted:
			escaped = true
		case character == '"':
			quoted = !quoted
		case character == ' ' && !quoted:
			return line[:index], line[index+1:], true
		}
	}
	return "", "", false
}

func splitMetricLabels(labels string) []string {
	if labels == "" {
		return nil
	}
	var parts []string
	start := 0
	escaped := false
	for index, character := range labels {
		switch {
		case escaped:
			escaped = false
		case character == '\\':
			escaped = true
		case character == ',':
			parts = append(parts, labels[start:index])
			start = index + 1
		}
	}
	return append(parts, labels[start:])
}

type metricsEncoderFunc func(*MetricsRegistry) ([]byte, error)

func (function metricsEncoderFunc) Encode(registry *MetricsRegistry) ([]byte, error) {
	return function(registry)
}

type commitTrackingWriter struct {
	header  http.Header
	status  int
	commits int
	body    bytes.Buffer
}

func (writer *commitTrackingWriter) Header() http.Header {
	return writer.header
}

func (writer *commitTrackingWriter) WriteHeader(status int) {
	if writer.status == 0 {
		writer.status = status
		writer.commits++
	}
}

func (writer *commitTrackingWriter) Write(body []byte) (int, error) {
	if writer.status == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.body.Write(body)
}

var _ io.Writer = (*commitTrackingWriter)(nil)
