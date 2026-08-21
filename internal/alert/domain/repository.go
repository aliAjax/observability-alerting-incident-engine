package domain

import (
	"context"
	"time"
)

type Repository interface {
	UpsertAlert(ctx context.Context, alert Alert, observation Observation) (Alert, bool, error)
	GetAlert(ctx context.Context, tenant, id string) (Alert, error)
	GetAlertByFingerprint(ctx context.Context, fingerprint string) (Alert, error)
	Transition(ctx context.Context, alertID string, from, to Status, reason, actor, traceID string, occurredAt time.Time) error
	ListAlerts(ctx context.Context, tenant string, status Status, ruleID string, limit, offset int) ([]Alert, int, error)
	ListTransitions(ctx context.Context, alertID string, from, to time.Time, limit, offset int) ([]Transition, int, error)
	ListObservations(ctx context.Context, alertID string, from, to time.Time, limit, offset int) ([]Observation, int, error)
	CountByStatus(ctx context.Context, tenant string) (map[Status]int, error)
	TouchAlert(ctx context.Context, alertID string, value float64, message string, observedAt time.Time) error
}
