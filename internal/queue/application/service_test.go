package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/queue/domain"
)

type fakeContextQueueRepo struct {
	tasks []domain.Task
}

func (f fakeContextQueueRepo) Enqueue(context.Context, domain.Task) error { return nil }
func (f fakeContextQueueRepo) PollByID(context.Context, string) (domain.Task, error) {
	return domain.Task{}, common.ErrNotFound
}
func (f fakeContextQueueRepo) Poll(context.Context, string, int, time.Time, time.Duration, string) ([]domain.Task, error) {
	return f.tasks, nil
}
func (f fakeContextQueueRepo) Complete(context.Context, string, string) error { return nil }
func (f fakeContextQueueRepo) Fail(context.Context, string, string, error, time.Duration) error {
	return nil
}
func (f fakeContextQueueRepo) Requeue(context.Context, string, string, time.Duration) error {
	return nil
}
func (f fakeContextQueueRepo) CountPending(context.Context, string) (int, error) { return 0, nil }

func TestHandlePropagatesContextCancellation(t *testing.T) {
	repo := fakeContextQueueRepo{tasks: []domain.Task{{ID: "task-1", Status: domain.TaskRunning}}}
	service := NewQueueService(repo, "worker", time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	handlerErr := errors.New("context was not propagated")
	propagated := false
	_, err := service.Handle(ctx, "evaluation", 1, func(got context.Context, _ domain.Task) error {
		propagated = got.Err() != nil
		return nil
	})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if !propagated {
		t.Fatal("expected handler to receive the canceled context")
	}
	_ = handlerErr
}

func TestQueueTaskAvailableRespectsScheduledTime(t *testing.T) {
	future := time.Now().Add(time.Hour)
	task := domain.Task{Status: domain.TaskPending, AvailableAt: future}
	if task.Available(time.Now()) {
		t.Fatal("task scheduled in the future should not be available")
	}
}
