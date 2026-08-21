package domain

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, silence Silence) (Silence, error)
	Update(ctx context.Context, silence Silence) (Silence, error)
	Get(ctx context.Context, tenant, id string) (Silence, error)
	List(ctx context.Context, tenant string, limit, offset int) ([]Silence, int, error)
	ListActive(ctx context.Context, tenant string, at time.Time) ([]Silence, error)
}
