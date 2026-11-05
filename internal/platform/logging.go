package platform

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"log/slog"
	"strings"
)

// TraceHeader is the public HTTP correlation header.
const TraceHeader = "X-Trace-ID"

const traceIDLength = 26

var traceEncoding = base32.NewEncoding("0123456789ABCDEFGHJKMNPQRSTVWXYZ").WithPadding(base32.NoPadding)

type traceContextKey struct{}

// NewTraceID creates a new request correlation identifier.
func NewTraceID() (string, error) {
	var source [16]byte
	if _, err := rand.Read(source[:]); err != nil {
		return "", fmt.Errorf("generate trace ID: %w", err)
	}
	return traceEncoding.EncodeToString(source[:]), nil
}

// ContextWithTraceID stores a validated trusted trace identifier.
func ContextWithTraceID(ctx context.Context, traceID string) (context.Context, error) {
	if ctx == nil {
		return nil, fmt.Errorf("trace context is required")
	}
	if !validTraceID(traceID) {
		return nil, fmt.Errorf("invalid trace ID")
	}
	return context.WithValue(ctx, traceContextKey{}, traceID), nil
}

// TraceIDFromContext returns the request correlation identifier.
func TraceIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	traceID, ok := ctx.Value(traceContextKey{}).(string)
	return traceID, ok && validTraceID(traceID)
}

// LoggerWithTrace returns a structured logger carrying stable correlation fields.
func LoggerWithTrace(logger *slog.Logger, service, module, traceID string) (*slog.Logger, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	service = strings.TrimSpace(service)
	if service == "" {
		return nil, fmt.Errorf("log service is required")
	}
	module = strings.TrimSpace(module)
	if module == "" {
		return nil, fmt.Errorf("log module is required")
	}
	if !validTraceID(traceID) {
		return nil, fmt.Errorf("invalid trace ID")
	}
	return logger.With(
		slog.String("service", service),
		slog.String("module", module),
		slog.String("trace_id", traceID),
	), nil
}

func validTraceID(traceID string) bool {
	if len(traceID) != traceIDLength {
		return false
	}
	for _, character := range traceID {
		if !strings.ContainsRune("0123456789ABCDEFGHJKMNPQRSTVWXYZ", character) {
			return false
		}
	}
	return true
}
