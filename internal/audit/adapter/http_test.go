package adapter

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/audit/application"
	"github.com/observability-alerting/engine/internal/audit/domain"
	"github.com/observability-alerting/engine/internal/common"
)

type fakeAuditHTTPRepo struct {
	listErr error
	got     context.Context
}

func (f *fakeAuditHTTPRepo) Insert(context.Context, domain.Record) error { return nil }
func (f *fakeAuditHTTPRepo) List(got context.Context, _ string, _, _ string, _, _ time.Time, _, _ int) ([]domain.Record, int, error) {
	f.got = got
	return nil, 0, f.listErr
}

func TestR009AuditListHttp404(t *testing.T) {
	repo := &fakeAuditHTTPRepo{listErr: fmt.Errorf("audit repository miss: %w", common.ErrNotFound)}
	service := application.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	rec := httptest.NewRecorder()
	handler.List(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for not found, got %d", rec.Code)
	}
}

func TestR009AuditListCancelledQuery(t *testing.T) {
	repo := &fakeAuditHTTPRepo{}
	service := application.NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler := NewHandler(service)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.List(rec, req)
	if repo.got == nil || repo.got.Err() == nil {
		t.Fatal("expected List to pass the canceled request context")
	}
}
