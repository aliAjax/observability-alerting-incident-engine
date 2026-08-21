package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/audit/domain"
	"github.com/observability-alerting/engine/internal/common"
)

type fakeAuditRepo struct {
	listErr error
}

func (fakeAuditRepo) Insert(context.Context, domain.Record) error { return nil }
func (f fakeAuditRepo) List(context.Context, string, string, string, time.Time, time.Time, int, int) ([]domain.Record, int, error) {
	return nil, 0, f.listErr
}

func TestAuditListPreservesSentinel(t *testing.T) {
	repo := fakeAuditRepo{listErr: fmt.Errorf("audit repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.List(context.Background(), "default", "", "", time.Time{}, time.Now(), 10, 0); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("List should preserve ErrNotFound, got %v", err)
	}
}

func TestAuditRecordNormalizeCreatesID(t *testing.T) {
	record := domain.Record{}
	record.Normalize(time.Now())
	if record.ID == "" {
		t.Fatal("Normalize should create an ID")
	}
}
