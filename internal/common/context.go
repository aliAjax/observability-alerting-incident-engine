package common

import "context"

type contextKey string

const (
	TraceIDKey contextKey = "trace_id"
	TenantKey  contextKey = "tenant"
)

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		traceID = NewTraceID()
	}
	return context.WithValue(ctx, TraceIDKey, traceID)
}

func TraceIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok && v != "" {
		return v
	}
	return NewTraceID()
}

func WithTenant(ctx context.Context, tenant string) context.Context {
	if tenant == "" {
		tenant = "default"
	}
	return context.WithValue(ctx, TenantKey, tenant)
}

func TenantFrom(ctx context.Context) string {
	if v, ok := ctx.Value(TenantKey).(string); ok && v != "" {
		return v
	}
	return "default"
}
