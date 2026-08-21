package application

import (
	"context"

	"github.com/observability-alerting/engine/internal/queue/domain"
)

type TaskRepository interface {
	domain.Repository
	PollByID(ctx context.Context, id string) (domain.Task, error)
}
