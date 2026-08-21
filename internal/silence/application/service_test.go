package application

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/silence/domain"
)

type fakeSilenceRepo struct {
	ctxErr  error
	listCtx context.Context
}

func (f *fakeSilenceRepo) Create(context.Context, domain.Silence) (domain.Silence, error) {
	return domain.Silence{}, nil
}
func (f *fakeSilenceRepo) Update(context.Context, domain.Silence) (domain.Silence, error) {
	return domain.Silence{}, nil
}
func (f *fakeSilenceRepo) Get(context.Context, string, string) (domain.Silence, error) {
	return domain.Silence{}, nil
}
func (f *fakeSilenceRepo) List(got context.Context, _ string, _, _ int) ([]domain.Silence, int, error) {
	f.listCtx = got
	return nil, 0, nil
}
func (f *fakeSilenceRepo) ListActive(got context.Context, _ string, _ time.Time) ([]domain.Silence, error) {
	f.ctxErr = got.Err()
	return nil, nil
}

func TestIsSilencedPropagatesContext(t *testing.T) {
	repo := &fakeSilenceRepo{}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.IsSilenced(ctx, "default", "rule-1", "", common.Labels{}, time.Now()); err != nil {
		t.Fatalf("IsSilenced returned error: %v", err)
	}
	if repo.ctxErr == nil {
		t.Fatal("expected repository to receive the canceled context")
	}
}

func TestListPropagatesContext(t *testing.T) {
	repo := &fakeSilenceRepo{}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.List(ctx, "default", 10, 0); err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if repo.listCtx.Err() == nil {
		t.Fatal("expected List to pass the canceled request context")
	}
}

func TestSilenceNormalizeInitializesMatchers(t *testing.T) {
	now := time.Now()
	silence := domain.Silence{StartsAt: now, EndsAt: now.Add(time.Hour)}
	if err := silence.Normalize(time.Now()); err != nil {
		t.Fatalf("Normalize returned error: %v", err)
	}
	if silence.Matchers == nil {
		t.Fatal("Normalize should initialize Matchers")
	}
}

func TestTenantFromPreservesValue(t *testing.T) {
	ctx := common.WithTenant(context.Background(), "acme")
	if got := common.TenantFrom(ctx); got != "acme" {
		t.Fatalf("expected tenant acme, got %s", got)
	}
}
