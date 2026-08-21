package application

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/incident/domain"
)

type fakeIncidentRepo struct {
	incident domain.Incident
	actions  []domain.Action
}

func (f *fakeIncidentRepo) Create(_ context.Context, incident domain.Incident) (domain.Incident, error) {
	f.incident = incident
	return incident, nil
}
func (f *fakeIncidentRepo) Update(_ context.Context, incident domain.Incident) (domain.Incident, error) {
	f.incident = incident
	return incident, nil
}
func (f *fakeIncidentRepo) Get(_ context.Context, _, _ string) (domain.Incident, error) {
	return f.incident, nil
}
func (f *fakeIncidentRepo) GetByAlert(_ context.Context, _, _ string) (domain.Incident, error) {
	return f.incident, nil
}
func (f *fakeIncidentRepo) List(_ context.Context, _ string, _ domain.Status, _ string, _, _ int) ([]domain.Incident, int, error) {
	return []domain.Incident{f.incident}, 1, nil
}
func (f *fakeIncidentRepo) AddAction(_ context.Context, action domain.Action) error {
	f.actions = append(f.actions, action)
	return nil
}
func (f *fakeIncidentRepo) ListActions(_ context.Context, _ string, _, _ int) ([]domain.Action, int, error) {
	return f.actions, len(f.actions), nil
}

func TestR006EscalatedThenResolve(t *testing.T) {
	repo := &fakeIncidentRepo{}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := service.Create(context.Background(), domain.Incident{Title: "gateway latency", Severity: domain.SeverityHigh})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if _, err := service.Escalate(context.Background(), created.Tenant, created.ID, "alice", "need platform owner"); err != nil {
		t.Fatalf("Escalate failed: %v", err)
	}
	resolved, err := service.Act(context.Background(), created.Tenant, created.ID, "resolve", "alice", "deploy rolled back")
	if err != nil {
		t.Fatalf("Act failed: %v", err)
	}
	if resolved.Status != domain.StatusResolved {
		t.Fatalf("expected escalated incident to resolve, got %s", resolved.Status)
	}
}

func TestR006AssignInProgress(t *testing.T) {
	repo := &fakeIncidentRepo{}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := service.Create(context.Background(), domain.Incident{Title: "gateway latency", Severity: domain.SeverityHigh})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	updated, err := service.Assign(context.Background(), created.Tenant, created.ID, "alice", "bob")
	if err != nil {
		t.Fatalf("Assign failed: %v", err)
	}
	if updated.Status != domain.StatusInProgress {
		t.Fatalf("expected assigned incident to be in progress, got %s", updated.Status)
	}
}

func TestR006EscalateStatus(t *testing.T) {
	repo := &fakeIncidentRepo{}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := service.Create(context.Background(), domain.Incident{Title: "gateway latency", Severity: domain.SeverityHigh})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	updated, err := service.Escalate(context.Background(), created.Tenant, created.ID, "alice", "need owner")
	if err != nil {
		t.Fatalf("Escalate failed: %v", err)
	}
	if updated.Status != domain.StatusEscalated {
		t.Fatalf("expected escalated status, got %s", updated.Status)
	}
}

func TestR006EscalateClosedAt(t *testing.T) {
	repo := &fakeIncidentRepo{}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := service.Create(context.Background(), domain.Incident{Title: "gateway latency", Severity: domain.SeverityHigh})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	updated, err := service.Escalate(context.Background(), created.Tenant, created.ID, "alice", "need owner")
	if err != nil {
		t.Fatalf("Escalate failed: %v", err)
	}
	if !updated.ClosedAt.IsZero() {
		t.Fatal("escalate should not set ClosedAt")
	}
}

func TestR006CloseResolved(t *testing.T) {
	repo := &fakeIncidentRepo{}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := service.Create(context.Background(), domain.Incident{Title: "gateway latency", Severity: domain.SeverityHigh})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	updated, err := service.Act(context.Background(), created.Tenant, created.ID, "close", "alice", "done")
	if err != nil {
		t.Fatalf("Act failed: %v", err)
	}
	if updated.Status != domain.StatusResolved {
		t.Fatalf("expected close to resolve incident, got %s", updated.Status)
	}
}

func TestR006ResolveStatus(t *testing.T) {
	repo := &fakeIncidentRepo{}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	created, err := service.Create(context.Background(), domain.Incident{Title: "gateway latency", Severity: domain.SeverityHigh})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	updated, err := service.Act(context.Background(), created.Tenant, created.ID, "resolve", "alice", "fixed")
	if err != nil {
		t.Fatalf("Act failed: %v", err)
	}
	if updated.Status != domain.StatusResolved {
		t.Fatalf("expected resolve to mark incident resolved, got %s", updated.Status)
	}
}

func TestR006NormalizeOpen(t *testing.T) {
	incident := domain.Incident{Title: "gateway latency"}
	if err := incident.Normalize(time.Now()); err != nil {
		t.Fatalf("Normalize returned error: %v", err)
	}
	if incident.Status != domain.StatusOpen {
		t.Fatalf("expected default status open, got %s", incident.Status)
	}
}

func TestR006NormalizeSeverity(t *testing.T) {
	incident := domain.Incident{Title: "gateway latency"}
	if err := incident.Normalize(time.Now()); err != nil {
		t.Fatalf("Normalize returned error: %v", err)
	}
	if incident.Severity != domain.SeverityMedium {
		t.Fatalf("expected default severity medium, got %s", incident.Severity)
	}
}
