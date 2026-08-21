package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/queue/domain"
)

type QueueService struct {
	repo    domain.Repository
	owner   string
	lockTTL time.Duration
}

func NewQueueService(repo domain.Repository, owner string, lockTTL time.Duration) *QueueService {
	if owner == "" {
		owner = common.NewID("worker")
	}
	if lockTTL <= 0 {
		lockTTL = time.Minute
	}
	return &QueueService{repo: repo, owner: owner, lockTTL: lockTTL}
}

func (s *QueueService) Enqueue(ctx context.Context, topic string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return common.Wrap("marshal queue payload", err)
	}
	task := domain.Task{
		ID:          common.NewID("task"),
		Topic:       topic,
		Payload:     raw,
		Status:      domain.TaskPending,
		MaxAttempts: 5,
		AvailableAt: common.Now(),
		CreatedAt:   common.Now(),
	}
	if err := s.repo.Enqueue(ctx, task); err != nil {
		return common.Wrap("enqueue task", err)
	}
	return nil
}

func (s *QueueService) Poll(ctx context.Context, topic string, limit int) ([]domain.Task, error) {
	if limit <= 0 {
		limit = 10
	}
	tasks, err := s.repo.Poll(ctx, topic, limit, common.Now(), s.lockTTL, s.owner)
	if err != nil {
		return nil, common.Wrap("poll tasks", err)
	}
	return tasks, nil
}

func (s *QueueService) Complete(ctx context.Context, id string) error {
	if err := s.repo.Complete(ctx, id, s.owner); err != nil {
		return common.Wrap("complete task", err)
	}
	return nil
}

func (s *QueueService) Fail(ctx context.Context, id string, cause error) error {
	task, err := s.repo.PollByID(ctx, id)
	if err != nil {
		return common.Wrap("load task", err)
	}
	if task.Attempts+1 >= task.MaxAttempts {
		return s.repo.Fail(ctx, id, s.owner, cause, 0)
	}
	delay := time.Duration(1<<task.Attempts) * time.Second
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	return s.repo.Fail(ctx, id, s.owner, cause, delay)
}

func (s *QueueService) Requeue(ctx context.Context, id string, delay time.Duration) error {
	return s.repo.Requeue(ctx, id, s.owner, delay)
}

func (s *QueueService) CountPending(ctx context.Context, topic string) (int, error) {
	return s.repo.CountPending(ctx, topic)
}

func (s *QueueService) Owner() string {
	return s.owner
}

func (s *QueueService) Handle(ctx context.Context, topic string, limit int, fn func(context.Context, domain.Task) error) (int, error) {
	tasks, err := s.Poll(ctx, topic, limit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, task := range tasks {
		if err := fn(ctx, task); err != nil {
			_ = s.Fail(ctx, task.ID, err)
			continue
		}
		if err := s.Complete(ctx, task.ID); err != nil && !errors.Is(err, common.ErrNotFound) {
			continue
		}
		processed++
	}
	return processed, nil
}
