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

type recordingQueueRepo struct {
	enqueueCtx   context.Context
	pollCtx      context.Context
	completeCtx  context.Context
	pollByIDCtx  context.Context
	tasks        []domain.Task
}

func (r *recordingQueueRepo) Enqueue(got context.Context, _ domain.Task) error {
	r.enqueueCtx = got
	return nil
}
func (r *recordingQueueRepo) PollByID(got context.Context, _ string) (domain.Task, error) {
	r.pollByIDCtx = got
	return domain.Task{}, nil
}
func (r *recordingQueueRepo) Poll(got context.Context, _ string, _ int, _ time.Time, _ time.Duration, _ string) ([]domain.Task, error) {
	r.pollCtx = got
	return r.tasks, nil
}
func (r *recordingQueueRepo) Complete(got context.Context, _ string, _ string) error {
	r.completeCtx = got
	return nil
}
func (r *recordingQueueRepo) Fail(context.Context, string, string, error, time.Duration) error {
	return nil
}
func (r *recordingQueueRepo) Requeue(context.Context, string, string, time.Duration) error {
	return nil
}
func (r *recordingQueueRepo) CountPending(context.Context, string) (int, error) { return 0, nil }

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

func TestR008HandleCancelledCallback(t *testing.T) {
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

func TestR008HandleCompleteContext(t *testing.T) {
	repo := &recordingQueueRepo{tasks: []domain.Task{{ID: "task-1", Status: domain.TaskRunning}}}
	service := NewQueueService(repo, "worker", time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := service.Handle(ctx, "evaluation", 1, func(context.Context, domain.Task) error { return nil }); err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if repo.completeCtx == nil || repo.completeCtx.Err() == nil {
		t.Fatal("expected Handle to complete with the canceled request context")
	}
}

func TestR008EnqueueRequestContext(t *testing.T) {
	repo := &recordingQueueRepo{}
	service := NewQueueService(repo, "worker", time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.Enqueue(ctx, "evaluation", map[string]string{"a": "b"}); err != nil {
		t.Fatalf("Enqueue returned error: %v", err)
	}
	if repo.enqueueCtx == nil || repo.enqueueCtx.Err() == nil {
		t.Fatal("expected Enqueue to pass the canceled context")
	}
}

func TestR008PollRequestContext(t *testing.T) {
	repo := &recordingQueueRepo{}
	service := NewQueueService(repo, "worker", time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Poll(ctx, "evaluation", 1); err != nil {
		t.Fatalf("Poll returned error: %v", err)
	}
	if repo.pollCtx == nil || repo.pollCtx.Err() == nil {
		t.Fatal("expected Poll to pass the canceled context")
	}
}

func TestR008CompleteRequestContext(t *testing.T) {
	repo := &recordingQueueRepo{}
	service := NewQueueService(repo, "worker", time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.Complete(ctx, "task-1"); err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if repo.completeCtx == nil || repo.completeCtx.Err() == nil {
		t.Fatal("expected Complete to pass the canceled context")
	}
}

func TestR008FailRequestContext(t *testing.T) {
	repo := &recordingQueueRepo{}
	service := NewQueueService(repo, "worker", time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := service.Fail(ctx, "task-1", errors.New("boom")); err != nil {
		t.Fatalf("Fail returned error: %v", err)
	}
	if repo.pollByIDCtx == nil || repo.pollByIDCtx.Err() == nil {
		t.Fatal("expected Fail to pass the canceled context")
	}
}

func TestQueueTaskAvailableRespectsScheduledTime(t *testing.T) {
	future := time.Now().Add(time.Hour)
	task := domain.Task{Status: domain.TaskPending, AvailableAt: future}
	if task.Available(time.Now()) {
		t.Fatal("task scheduled in the future should not be available")
	}
}

func TestR008TaskReadyAtBoundary(t *testing.T) {
	now := time.Now()
	task := domain.Task{Status: domain.TaskPending, AvailableAt: now}
	if !task.Available(now) {
		t.Fatal("task scheduled exactly now should be available")
	}
}
