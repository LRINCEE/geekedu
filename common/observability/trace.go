package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"go.uber.org/zap"
)

const (
	TraceHeader      = "X-Trace-ID"
	TraceMetadataKey = "x-trace-id"
)

type traceContextKey struct{}

func GenerateTraceID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "trace-id-unavailable"
	}
	return hex.EncodeToString(buf[:])
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceContextKey{}, traceID)
}

func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	traceID, _ := ctx.Value(traceContextKey{}).(string)
	return traceID
}

func EnsureTraceID(ctx context.Context) (context.Context, string) {
	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = GenerateTraceID()
		ctx = WithTraceID(ctx, traceID)
	}
	return ctx, traceID
}

func TraceField(ctx context.Context) zap.Field {
	return zap.String("trace_id", TraceIDFromContext(ctx))
}

func Fields(ctx context.Context, fields ...zap.Field) []zap.Field {
	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		return fields
	}
	return append([]zap.Field{zap.String("trace_id", traceID)}, fields...)
}
