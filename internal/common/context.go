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
	v, _ := ctx.Value(TraceIDKey).(string)
	if v == "" {
		return NewTraceID()
	}
	return v
}

func WithTenant(ctx context.Context, tenant string) context.Context {
	if tenant == "" {
		tenant = "default"
	}
	return context.WithValue(ctx, TenantKey, tenant)
}

func TenantFrom(ctx context.Context) string {
	v, _ := ctx.Value(TenantKey).(string)
	if v == "" {
		return "default"
	}
	return v
}
