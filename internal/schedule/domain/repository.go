package domain

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, schedule Schedule) (Schedule, error)
	Update(ctx context.Context, schedule Schedule) (Schedule, error)
	Get(ctx context.Context, tenant, id string) (Schedule, error)
	List(ctx context.Context, tenant string, limit, offset int) ([]Schedule, int, error)
	FindActive(ctx context.Context, tenant string, at time.Time) ([]Schedule, error)
}
