package domain

import (
	"context"
	"encoding/json"
	"time"
)

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

type Task struct {
	ID          string          `json:"id"`
	Topic       string          `json:"topic"`
	Payload     json.RawMessage `json:"payload"`
	Status      TaskStatus      `json:"status"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	AvailableAt time.Time       `json:"available_at"`
	CreatedAt   time.Time       `json:"created_at"`
	LockedBy    string          `json:"locked_by,omitempty"`
	LockedUntil time.Time       `json:"locked_until,omitempty"`
	LastError   string          `json:"last_error,omitempty"`
}

func (t Task) Available(now time.Time) bool {
	if t.Status != TaskPending {
		return false
	}
	return t.AvailableAt.Before(now)
}

type Repository interface {
	Enqueue(ctx context.Context, task Task) error
	PollByID(ctx context.Context, id string) (Task, error)
	Poll(ctx context.Context, topic string, limit int, now time.Time, lockTTL time.Duration, owner string) ([]Task, error)
	Complete(ctx context.Context, id, owner string) error
	Fail(ctx context.Context, id, owner string, err error, retryDelay time.Duration) error
	Requeue(ctx context.Context, id, owner string, delay time.Duration) error
	CountPending(ctx context.Context, topic string) (int, error)
}
