package common

import (
	"context"
	"testing"
)

func TestWithTenantSetsValue(t *testing.T) {
	ctx := WithTenant(context.Background(), "acme")
	if got := ctx.Value(TenantKey); got != "acme" {
		t.Fatalf("expected tenant value acme, got %v", got)
	}
}

func TestTenantFromReturnsDefaultWhenMissing(t *testing.T) {
	if got := TenantFrom(context.Background()); got != "default" {
		t.Fatalf("expected default tenant, got %q", got)
	}
}

func TestWithTraceIDSetsValue(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-1")
	if got := ctx.Value(TraceIDKey); got != "trace-1" {
		t.Fatalf("expected trace id trace-1, got %v", got)
	}
}

func TestTraceIDFromGeneratesMissingValue(t *testing.T) {
	if got := TraceIDFrom(context.Background()); got == "" {
		t.Fatal("expected a generated trace id for missing value")
	}
}
