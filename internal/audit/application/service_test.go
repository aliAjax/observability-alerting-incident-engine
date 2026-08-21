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
	insertErr error
	listErr   error
}

func (f fakeAuditRepo) Insert(context.Context, domain.Record) error { return f.insertErr }
func (f fakeAuditRepo) List(context.Context, string, string, string, time.Time, time.Time, int, int) ([]domain.Record, int, error) {
	return nil, 0, f.listErr
}

func TestR009AuditListMissingMaps404(t *testing.T) {
	repo := fakeAuditRepo{listErr: fmt.Errorf("audit repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.List(context.Background(), "default", "", "", time.Time{}, time.Now(), 10, 0); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("List should preserve ErrNotFound, got %v", err)
	}
}

func TestR009AuditRecordMissingSentinel(t *testing.T) {
	repo := fakeAuditRepo{insertErr: fmt.Errorf("audit insert miss: %w", common.ErrConflict)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := service.Record(context.Background(), "default", "rule", "rule-1", "update", "alice", map[string]string{"field": "mode"}); !errors.Is(err, common.ErrConflict) {
		t.Fatalf("Record should preserve ErrConflict, got %v", err)
	}
}

func TestR009AuditRecordAutoID(t *testing.T) {
	record := domain.Record{}
	record.Normalize(time.Now())
	if record.ID == "" {
		t.Fatal("Normalize should create an ID")
	}
}

func TestR009AuditRecordAutoTrace(t *testing.T) {
	record := domain.Record{}
	record.Normalize(time.Now())
	if record.TraceID == "" {
		t.Fatal("Normalize should create a trace ID")
	}
}
