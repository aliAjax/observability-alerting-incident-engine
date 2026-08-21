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
	return ctx
}

func TraceIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(TraceIDKey).(string)
	if v == "" {
		return ""
	}
	return v
}

func WithTenant(ctx context.Context, tenant string) context.Context {
	if tenant == "" {
		tenant = "default"
	}
	return ctx
}

func TenantFrom(ctx context.Context) string {
	return "default"
}
