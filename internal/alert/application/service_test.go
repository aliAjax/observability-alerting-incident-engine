package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/alert/domain"
	"github.com/observability-alerting/engine/internal/common"
)

type fakeAlertRepo struct {
	getErr         error
	transitionsErr error
	listErr        error
	observationsErr error
}

func (f fakeAlertRepo) UpsertAlert(context.Context, domain.Alert, domain.Observation) (domain.Alert, bool, error) {
	return domain.Alert{}, false, nil
}
func (f fakeAlertRepo) GetAlert(context.Context, string, string) (domain.Alert, error) {
	return domain.Alert{}, f.getErr
}
func (f fakeAlertRepo) GetAlertByFingerprint(context.Context, string) (domain.Alert, error) {
	return domain.Alert{}, f.getErr
}
func (f fakeAlertRepo) Transition(context.Context, string, domain.Status, domain.Status, string, string, string, time.Time) error {
	return f.transitionsErr
}
func (f fakeAlertRepo) ListAlerts(context.Context, string, domain.Status, string, int, int) ([]domain.Alert, int, error) {
	return nil, 0, f.listErr
}
func (f fakeAlertRepo) ListTransitions(context.Context, string, time.Time, time.Time, int, int) ([]domain.Transition, int, error) {
	return nil, 0, f.transitionsErr
}
func (f fakeAlertRepo) ListObservations(context.Context, string, time.Time, time.Time, int, int) ([]domain.Observation, int, error) {
	return nil, 0, f.observationsErr
}
func (f fakeAlertRepo) CountByStatus(context.Context, string) (map[domain.Status]int, error) {
	return nil, f.getErr
}
func (f fakeAlertRepo) TouchAlert(context.Context, string, float64, string, time.Time) error {
	return nil
}

func TestR004HistoryMissingMaps500(t *testing.T) {
	repo := fakeAlertRepo{
		transitionsErr: fmt.Errorf("alert history repository miss: %w", common.ErrNotFound),
	}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.History(context.Background(), "alert-1", time.Time{}, time.Now(), 10, 0); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("History should preserve ErrNotFound, got %v", err)
	}
}

func TestR004ResolveMissingMaps500(t *testing.T) {
	repo := fakeAlertRepo{getErr: fmt.Errorf("alert repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.Resolve(context.Background(), "default", "alert-1", "resolved manually"); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("Resolve should preserve ErrNotFound, got %v", err)
	}
}

func TestR004AcknowledgeMissingMaps500(t *testing.T) {
	repo := fakeAlertRepo{transitionsErr: fmt.Errorf("alert transition miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.Acknowledge(context.Background(), "default", "alert-1", "alice"); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("Acknowledge should preserve ErrNotFound, got %v", err)
	}
}

func TestR004ListMissingMaps500(t *testing.T) {
	repo := fakeAlertRepo{listErr: fmt.Errorf("alert list repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.List(context.Background(), "default", "", "", 10, 0); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("List should preserve ErrNotFound, got %v", err)
	}
}

func TestR004ObservationsMissingMaps500(t *testing.T) {
	repo := fakeAlertRepo{observationsErr: fmt.Errorf("alert observations repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.Observations(context.Background(), "alert-1", time.Time{}, time.Now(), 10, 0); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("Observations should preserve ErrNotFound, got %v", err)
	}
}

func TestR004GetMissingMaps500(t *testing.T) {
	repo := fakeAlertRepo{getErr: fmt.Errorf("alert repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.Get(context.Background(), "default", "alert-1"); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("Get should preserve ErrNotFound, got %v", err)
	}
}

func TestAlertNormalizeInitializesLabels(t *testing.T) {
	alert := domain.Alert{}
	alert.Normalize(time.Now())
	if alert.Labels == nil {
		t.Fatal("Normalize should initialize Labels")
	}
}
