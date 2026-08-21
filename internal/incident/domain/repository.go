package domain

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, incident Incident) (Incident, error)
	Update(ctx context.Context, incident Incident) (Incident, error)
	Get(ctx context.Context, tenant, id string) (Incident, error)
	GetByAlert(ctx context.Context, tenant, alertID string) (Incident, error)
	List(ctx context.Context, tenant string, status Status, assignee string, limit, offset int) ([]Incident, int, error)
	AddAction(ctx context.Context, action Action) error
	ListActions(ctx context.Context, incidentID string, limit, offset int) ([]Action, int, error)
}
