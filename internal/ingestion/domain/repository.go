package domain

import (
	"context"
	"time"
)

type Repository interface {
	InsertBatch(ctx context.Context, events []IngestionEvent) error
	Insert(ctx context.Context, event IngestionEvent) error
	ExistsDedupe(ctx context.Context, key string, within time.Duration) (bool, error)
	QueryBySource(ctx context.Context, source string, eventType EventType, from, to time.Time, limit int) ([]IngestionEvent, error)
	QueryAggregates(ctx context.Context, source string, eventType EventType, from, to time.Time) ([]Aggregate, error)
}

type Aggregate struct {
	Source string
	Labels map[string]string
	Count  int
	Sum    float64
	Min    float64
	Max    float64
	Avg    float64
	Latest float64
}
