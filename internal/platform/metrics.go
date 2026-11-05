package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	loginAttemptsMetric         = "han_identity_login_attempts_total"
	loginInternalFailuresMetric = "han_identity_login_internal_failures_total"
	loginDurationMetric         = "han_identity_login_duration_seconds"
	originRejectionsMetric      = "han_platform_origin_rejections_total"
)

var (
	metricNameFirst = newASCIISet("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_:")
	metricNameRest  = newASCIISet("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_0123456789:")
	labelNameFirst  = newASCIISet("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_")
	labelNameRest   = newASCIISet("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_0123456789")

	loginOutcomes = stringSet(
		"success",
		"authentication_failed",
		"rate_limited",
		"invalid_request",
		"internal_failure",
	)
	loginReasonBuckets = stringSet(
		"none",
		"unknown",
		"persistence",
		"credential_processing",
		"session",
		"audit",
		"response",
		"request_context",
	)
	loginFailureStages = stringSet(
		"source_evidence",
		"transaction_begin",
		"abuse_check",
		"user_query",
		"temporary_credential_query",
		"credential_verify",
		"credential_rehash",
		"recovery_mode",
		"abuse_record",
		"session_issue",
		"session_persist",
		"audit_append",
		"failure_audit",
		"transaction_commit",
		"cookie_output",
		"response_write",
		"unknown",
	)
	originReasons = stringSet(
		"request_nil",
		"origin_missing",
		"origin_mismatch",
		"fetch_site_rejected",
	)
	originResponseWrites = stringSet("ok", "failed")
)

// MetricsRegistry owns the fixed in-process identity and platform metric set.
type MetricsRegistry struct {
	mu sync.RWMutex

	loginAttempts         map[string]*counterSeries
	loginInternalFailures map[string]*counterSeries
	loginDurations        map[string]*histogramSeries
	originRejections      map[string]*counterSeries
	durationBuckets       []float64
}

type counterSeries struct {
	labels []metricLabel
	value  uint64
}

type histogramSeries struct {
	labels []metricLabel
	counts []uint64
	sum    float64
	count  uint64
}

type metricLabel struct {
	name  string
	value string
}

// NewMetricsRegistry creates the fixed concurrent registry.
func NewMetricsRegistry(loginDurationBuckets []float64) (*MetricsRegistry, error) {
	if err := validateHistogramBuckets(loginDurationBuckets); err != nil {
		return nil, err
	}
	return &MetricsRegistry{
		loginAttempts:         make(map[string]*counterSeries),
		loginInternalFailures: make(map[string]*counterSeries),
		loginDurations:        make(map[string]*histogramSeries),
		originRejections:      make(map[string]*counterSeries),
		durationBuckets:       append([]float64(nil), loginDurationBuckets...),
	}, nil
}

// ParseHistogramBuckets parses the mandatory seconds list.
func ParseHistogramBuckets(raw string) ([]float64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("histogram buckets are required")
	}
	parts := strings.Split(raw, ",")
	if len(parts) < 1 || len(parts) > 32 {
		return nil, errors.New("histogram bucket count must be between 1 and 32")
	}
	buckets := make([]float64, len(parts))
	for index, part := range parts {
		if strings.TrimSpace(part) == "" {
			return nil, errors.New("histogram bucket is empty")
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return nil, errors.New("histogram bucket is invalid")
		}
		buckets[index] = value
	}
	if err := validateHistogramBuckets(buckets); err != nil {
		return nil, err
	}
	return buckets, nil
}

func validateHistogramBuckets(buckets []float64) error {
	if len(buckets) < 1 || len(buckets) > 32 {
		return errors.New("histogram bucket count must be between 1 and 32")
	}
	var previous float64
	for index, bucket := range buckets {
		if math.IsNaN(bucket) || math.IsInf(bucket, 0) || bucket <= 0 || bucket > 60 {
			return errors.New("histogram bucket is outside the accepted range")
		}
		if index > 0 && bucket <= previous {
			return errors.New("histogram buckets must be strictly increasing")
		}
		previous = bucket
	}
	return nil
}

// RecordLoginAttempt increments the Handler-scope login attempt counter.
func (registry *MetricsRegistry) RecordLoginAttempt(
	ctx context.Context,
	endpoint string,
	outcome string,
	reasonBucket string,
	errorCode string,
) error {
	if err := checkMetricContext(ctx); err != nil {
		return err
	}
	if endpoint != "POST /api/v1/sessions" || !loginOutcomes[outcome] ||
		!loginReasonBuckets[reasonBucket] || !validLoginErrorCode(errorCode) {
		return errors.New("login attempt metric labels are invalid")
	}
	labels := loginAttemptLabels(endpoint, outcome, reasonBucket, errorCode)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	incrementCounter(registry.loginAttempts, labels)
	return nil
}

// RecordLoginInternalFailure increments the fixed internal failure counter.
func (registry *MetricsRegistry) RecordLoginInternalFailure(
	ctx context.Context,
	endpoint string,
	reasonBucket string,
	errorCode string,
	failureStage string,
) error {
	if err := checkMetricContext(ctx); err != nil {
		return err
	}
	if endpoint != "POST /api/v1/sessions" || !loginReasonBuckets[reasonBucket] ||
		!validLoginErrorCode(errorCode) || !loginFailureStages[failureStage] {
		return errors.New("login internal failure metric labels are invalid")
	}
	labels := []metricLabel{
		{name: "endpoint", value: endpoint},
		{name: "reason_bucket", value: reasonBucket},
		{name: "error_code", value: errorCode},
		{name: "failure_stage", value: failureStage},
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	incrementCounter(registry.loginInternalFailures, labels)
	return nil
}

// RecordLoginDuration observes one Handler-scope duration in seconds.
func (registry *MetricsRegistry) RecordLoginDuration(
	ctx context.Context,
	endpoint string,
	outcome string,
	reasonBucket string,
	errorCode string,
	seconds float64,
) error {
	if err := checkMetricContext(ctx); err != nil {
		return err
	}
	if endpoint != "POST /api/v1/sessions" || !loginOutcomes[outcome] ||
		!loginReasonBuckets[reasonBucket] || !validLoginErrorCode(errorCode) ||
		math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 {
		return errors.New("login duration metric sample is invalid")
	}
	labels := loginAttemptLabels(endpoint, outcome, reasonBucket, errorCode)
	key := metricLabelKey(labels)
	registry.mu.Lock()
	defer registry.mu.Unlock()
	series := registry.loginDurations[key]
	if series == nil {
		series = &histogramSeries{
			labels: append([]metricLabel(nil), labels...),
			counts: make([]uint64, len(registry.durationBuckets)),
		}
		registry.loginDurations[key] = series
	}
	for index, boundary := range registry.durationBuckets {
		if seconds <= boundary {
			series.counts[index]++
		}
	}
	series.sum += seconds
	series.count++
	return nil
}

// RecordOriginRejection increments the platform-owned security counter.
func (registry *MetricsRegistry) RecordOriginRejection(
	ctx context.Context,
	reason string,
	responseWrite string,
) error {
	if err := checkMetricContext(ctx); err != nil {
		return err
	}
	if !originReasons[reason] || !originResponseWrites[responseWrite] {
		return errors.New("origin rejection metric labels are invalid")
	}
	labels := []metricLabel{
		{name: "reason", value: reason},
		{name: "response_write", value: responseWrite},
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	incrementCounter(registry.originRejections, labels)
	return nil
}

func checkMetricContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("metric context is required")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func validLoginErrorCode(code string) bool {
	switch code {
	case "none",
		string(CodeRequestInvalidJSON),
		string(CodeAuthLoginFailed),
		string(CodeRequestRateLimited),
		string(CodePlatformDatabaseUnavailable),
		string(CodePlatformInternalError):
		return true
	default:
		return false
	}
}

func loginAttemptLabels(endpoint, outcome, reasonBucket, errorCode string) []metricLabel {
	return []metricLabel{
		{name: "endpoint", value: endpoint},
		{name: "outcome", value: outcome},
		{name: "reason_bucket", value: reasonBucket},
		{name: "error_code", value: errorCode},
	}
}

func incrementCounter(series map[string]*counterSeries, labels []metricLabel) {
	key := metricLabelKey(labels)
	sample := series[key]
	if sample == nil {
		sample = &counterSeries{labels: append([]metricLabel(nil), labels...)}
		series[key] = sample
	}
	sample.value++
}

func metricLabelKey(labels []metricLabel) string {
	var builder strings.Builder
	for _, label := range labels {
		builder.WriteString(label.name)
		builder.WriteByte(0)
		builder.WriteString(label.value)
		builder.WriteByte(0)
	}
	return builder.String()
}

// NewMetricsHandler exposes only GET /metrics.
func NewMetricsHandler(registry *MetricsRegistry) http.Handler {
	return newMetricsHandler(registry, registryMetricsEncoder{})
}

type metricsEncoder interface {
	Encode(*MetricsRegistry) ([]byte, error)
}

type registryMetricsEncoder struct{}

func (registryMetricsEncoder) Encode(registry *MetricsRegistry) ([]byte, error) {
	return registry.encodeText()
}

func newMetricsHandler(registry *MetricsRegistry, encoder metricsEncoder) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request == nil || request.Method != http.MethodGet || request.URL.Path != "/metrics" {
			http.NotFound(response, request)
			return
		}
		if registry == nil || encoder == nil {
			writeMetricsEncodingFailure(response)
			return
		}
		body, err := encoder.Encode(registry)
		if err != nil || validateMetricsText(body) != nil {
			writeMetricsEncodingFailure(response)
			return
		}
		response.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_, _ = response.Write(body)
	})
}

func writeMetricsEncodingFailure(response http.ResponseWriter) {
	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	response.WriteHeader(http.StatusInternalServerError)
	_, _ = response.Write([]byte("metrics encoding failed\n"))
}

func (registry *MetricsRegistry) encodeText() ([]byte, error) {
	if registry == nil {
		return nil, errors.New("metrics registry is required")
	}
	registry.mu.RLock()
	snapshot := registry.snapshot()
	registry.mu.RUnlock()

	var buffer bytes.Buffer
	writeCounterFamily(
		&buffer,
		loginAttemptsMetric,
		"Total login attempts entering the identity handler.",
		snapshot.loginAttempts,
	)
	writeCounterFamily(
		&buffer,
		loginInternalFailuresMetric,
		"Total internal login failures by stable failure stage.",
		snapshot.loginInternalFailures,
	)
	writeHistogramFamily(
		&buffer,
		loginDurationMetric,
		"Login handler duration in seconds, excluding observation fan-out.",
		snapshot.durationBuckets,
		snapshot.loginDurations,
	)
	writeCounterFamily(
		&buffer,
		originRejectionsMetric,
		"Total same-origin request rejections after the response write attempt.",
		snapshot.originRejections,
	)
	body := buffer.Bytes()
	if err := validateMetricsText(body); err != nil {
		return nil, err
	}
	return append([]byte(nil), body...), nil
}

type metricsSnapshot struct {
	loginAttempts         []counterSeries
	loginInternalFailures []counterSeries
	loginDurations        []histogramSeries
	originRejections      []counterSeries
	durationBuckets       []float64
}

func (registry *MetricsRegistry) snapshot() metricsSnapshot {
	return metricsSnapshot{
		loginAttempts:         copyCounterSeries(registry.loginAttempts),
		loginInternalFailures: copyCounterSeries(registry.loginInternalFailures),
		loginDurations:        copyHistogramSeries(registry.loginDurations),
		originRejections:      copyCounterSeries(registry.originRejections),
		durationBuckets:       append([]float64(nil), registry.durationBuckets...),
	}
}

func copyCounterSeries(source map[string]*counterSeries) []counterSeries {
	keys := sortedMapKeys(source)
	result := make([]counterSeries, 0, len(keys))
	for _, key := range keys {
		series := source[key]
		result = append(result, counterSeries{
			labels: append([]metricLabel(nil), series.labels...),
			value:  series.value,
		})
	}
	return result
}

func copyHistogramSeries(source map[string]*histogramSeries) []histogramSeries {
	keys := sortedMapKeys(source)
	result := make([]histogramSeries, 0, len(keys))
	for _, key := range keys {
		series := source[key]
		result = append(result, histogramSeries{
			labels: append([]metricLabel(nil), series.labels...),
			counts: append([]uint64(nil), series.counts...),
			sum:    series.sum,
			count:  series.count,
		})
	}
	return result
}

func sortedMapKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func writeCounterFamily(
	buffer *bytes.Buffer,
	name string,
	help string,
	series []counterSeries,
) {
	fmt.Fprintf(buffer, "# HELP %s %s\n", name, escapeMetricHelp(help))
	fmt.Fprintf(buffer, "# TYPE %s counter\n", name)
	for _, sample := range series {
		writeMetricSample(buffer, name, sample.labels, strconv.FormatUint(sample.value, 10))
	}
}

func writeHistogramFamily(
	buffer *bytes.Buffer,
	name string,
	help string,
	buckets []float64,
	series []histogramSeries,
) {
	fmt.Fprintf(buffer, "# HELP %s %s\n", name, escapeMetricHelp(help))
	fmt.Fprintf(buffer, "# TYPE %s histogram\n", name)
	for _, sample := range series {
		for index, boundary := range buckets {
			labels := appendMetricLabel(sample.labels, metricLabel{
				name: "le", value: formatMetricFloat(boundary),
			})
			writeMetricSample(
				buffer,
				name+"_bucket",
				labels,
				strconv.FormatUint(sample.counts[index], 10),
			)
		}
		writeMetricSample(
			buffer,
			name+"_bucket",
			appendMetricLabel(sample.labels, metricLabel{name: "le", value: "+Inf"}),
			strconv.FormatUint(sample.count, 10),
		)
		writeMetricSample(buffer, name+"_sum", sample.labels, formatMetricFloat(sample.sum))
		writeMetricSample(
			buffer,
			name+"_count",
			sample.labels,
			strconv.FormatUint(sample.count, 10),
		)
	}
}

func appendMetricLabel(labels []metricLabel, label metricLabel) []metricLabel {
	combined := make([]metricLabel, 0, len(labels)+1)
	combined = append(combined, labels...)
	return append(combined, label)
}

func writeMetricSample(
	buffer *bytes.Buffer,
	name string,
	labels []metricLabel,
	value string,
) {
	buffer.WriteString(name)
	if len(labels) > 0 {
		buffer.WriteByte('{')
		for index, label := range labels {
			if index > 0 {
				buffer.WriteByte(',')
			}
			buffer.WriteString(label.name)
			buffer.WriteString(`="`)
			buffer.WriteString(escapeMetricLabel(label.value))
			buffer.WriteByte('"')
		}
		buffer.WriteByte('}')
	}
	buffer.WriteByte(' ')
	buffer.WriteString(value)
	buffer.WriteByte('\n')
}

func validateMetricsText(body []byte) error {
	if len(body) == 0 || body[len(body)-1] != '\n' || !utf8.Valid(body) ||
		bytes.ContainsRune(body, '\r') {
		return errors.New("metrics text framing is invalid")
	}
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "# HELP "):
			parts := strings.SplitN(strings.TrimPrefix(line, "# HELP "), " ", 2)
			if len(parts) != 2 || !validMetricName(parts[0]) {
				return errors.New("metrics HELP is invalid")
			}
		case strings.HasPrefix(line, "# TYPE "):
			parts := strings.Fields(line)
			if len(parts) != 4 || !validMetricName(parts[2]) ||
				(parts[3] != "counter" && parts[3] != "histogram") {
				return errors.New("metrics TYPE is invalid")
			}
		case strings.HasPrefix(line, "#"):
			return errors.New("metrics directive is invalid")
		default:
			name, _, found := strings.Cut(line, " ")
			if !found {
				return errors.New("metrics sample is invalid")
			}
			if open := strings.IndexByte(name, '{'); open >= 0 {
				name = name[:open]
			}
			if !validMetricName(name) {
				return errors.New("metrics sample name is invalid")
			}
		}
	}
	return nil
}

func escapeMetricHelp(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, "\n", `\n`)
}

func escapeMetricLabel(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func formatMetricFloat(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "+Inf"
	case math.IsInf(value, -1):
		return "-Inf"
	default:
		return strconv.FormatFloat(value, 'g', -1, 64)
	}
}

func validMetricName(name string) bool {
	return validPrometheusName(name, metricNameFirst, metricNameRest)
}

func validMetricLabelName(name string) bool {
	return validPrometheusName(name, labelNameFirst, labelNameRest)
}

func validPrometheusName(name string, first, rest [256]bool) bool {
	if name == "" || name[0] >= 128 || !first[name[0]] {
		return false
	}
	for index := 1; index < len(name); index++ {
		if name[index] >= 128 || !rest[name[index]] {
			return false
		}
	}
	return true
}

func newASCIISet(characters string) [256]bool {
	var result [256]bool
	for index := 0; index < len(characters); index++ {
		result[characters[index]] = true
	}
	return result
}

func stringSet(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}
