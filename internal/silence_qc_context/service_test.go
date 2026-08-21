package qc_silence_context

import (
    "context"
    "io"
    "log/slog"
    "testing"
    "time"

    "github.com/observability-alerting/engine/internal/common"
    "github.com/observability-alerting/engine/internal/silence/application"
    "github.com/observability-alerting/engine/internal/silence/domain"
)

type fakeSilenceRepo struct {
    listCtx       context.Context
    listActiveCtx context.Context
}

func (f *fakeSilenceRepo) Create(context.Context, domain.Silence) (domain.Silence, error) { return domain.Silence{}, nil }
func (f *fakeSilenceRepo) Update(context.Context, domain.Silence) (domain.Silence, error) { return domain.Silence{}, nil }
func (f *fakeSilenceRepo) Get(context.Context, string, string) (domain.Silence, error) { return domain.Silence{}, nil }
func (f *fakeSilenceRepo) List(got context.Context, _ string, _, _ int) ([]domain.Silence, int, error) { f.listCtx = got; return nil, 0, nil }
func (f *fakeSilenceRepo) ListActive(got context.Context, _ string, _ time.Time) ([]domain.Silence, error) { f.listActiveCtx = got; return nil, nil }

func TestR010TenantContextWrite(t *testing.T) {
    ctx := common.WithTenant(context.Background(), "acme")
    if got := common.TenantFrom(ctx); got != "acme" {
        t.Fatalf("expected tenant acme, got %q", got)
    }
}

func TestR010TraceContextWrite(t *testing.T) {
    ctx := common.WithTraceID(context.Background(), "trace-1")
    if got := common.TraceIDFrom(ctx); got != "trace-1" {
        t.Fatalf("expected trace id trace-1, got %q", got)
    }
}

func TestR010TraceMissingGenerate(t *testing.T) {
    if got := common.TraceIDFrom(context.Background()); got == "" {
        t.Fatal("expected a generated trace id for missing value")
    }
}

func TestR010SilencedCheckContext(t *testing.T) {
    repo := &fakeSilenceRepo{}
    service := application.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    if _, err := service.IsSilenced(ctx, "default", "rule-1", "", common.Labels{}, time.Now()); err != nil {
        t.Fatalf("IsSilenced returned error: %v", err)
    }
    if repo.listActiveCtx.Err() == nil {
        t.Fatal("expected repository to receive the canceled context")
    }
}

func TestR010SilenceListContext(t *testing.T) {
    repo := &fakeSilenceRepo{}
    service := application.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    if _, err := service.List(ctx, "default", 10, 0); err != nil {
        t.Fatalf("List returned error: %v", err)
    }
    if repo.listCtx.Err() == nil {
        t.Fatal("expected List to pass the canceled request context")
    }
}

func TestR010TenantContextRead(t *testing.T) {
    ctx := common.WithTenant(context.Background(), "acme")
    if got := common.TenantFrom(ctx); got != "acme" {
        t.Fatalf("expected tenant acme, got %q", got)
    }
}
