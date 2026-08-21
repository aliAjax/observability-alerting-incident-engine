package domain

import (
	"context"
	"time"
)

type Repository interface {
	Insert(ctx context.Context, record Record) error
	List(ctx context.Context, tenant, entity, entityID string, from, to time.Time, limit, offset int) ([]Record, int, error)
}
