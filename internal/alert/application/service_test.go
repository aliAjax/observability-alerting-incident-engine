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
	getErr       error
	transitionsErr error
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
	return nil
}
func (f fakeAlertRepo) ListAlerts(context.Context, string, domain.Status, string, int, int) ([]domain.Alert, int, error) {
	return nil, 0, nil
}
func (f fakeAlertRepo) ListTransitions(context.Context, string, time.Time, time.Time, int, int) ([]domain.Transition, int, error) {
	return nil, 0, f.transitionsErr
}
func (f fakeAlertRepo) ListObservations(context.Context, string, time.Time, time.Time, int, int) ([]domain.Observation, int, error) {
	return nil, 0, nil
}
func (f fakeAlertRepo) CountByStatus(context.Context, string) (map[domain.Status]int, error) {
	return nil, f.getErr
}
func (f fakeAlertRepo) TouchAlert(context.Context, string, float64, string, time.Time) error {
	return nil
}

func TestAlertHistoryPreservesSentinel(t *testing.T) {
	repo := fakeAlertRepo{
		transitionsErr: fmt.Errorf("alert history repository miss: %w", common.ErrNotFound),
	}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.History(context.Background(), "alert-1", time.Time{}, time.Now(), 10, 0); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("History should preserve ErrNotFound, got %v", err)
	}
}

func TestAlertNormalizeInitializesLabels(t *testing.T) {
	alert := domain.Alert{}
	alert.Normalize(time.Now())
	if alert.Labels == nil {
		t.Fatal("Normalize should initialize Labels")
	}
}
