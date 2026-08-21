package domain

import (
	"context"
	"time"
)

type Repository interface {
	Create(ctx context.Context, rule Rule) (Rule, error)
	Update(ctx context.Context, rule Rule) (Rule, error)
	Get(ctx context.Context, tenant, id string) (Rule, error)
	List(ctx context.Context, tenant string, limit, offset int, filters map[string]string) ([]Rule, int, error)
	ListForEvaluation(ctx context.Context, now time.Time, limit int) ([]Rule, error)
	GetByName(ctx context.Context, tenant, name string) (Rule, error)
}
