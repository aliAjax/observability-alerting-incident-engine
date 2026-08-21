package application

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/notification/domain"
)

type fakeNotificationRepo struct {
	tasks []domain.Task
}

func (f *fakeNotificationRepo) CreateChannel(context.Context, domain.Channel) (domain.Channel, error) {
	return domain.Channel{}, nil
}
func (f *fakeNotificationRepo) UpdateChannel(context.Context, domain.Channel) (domain.Channel, error) {
	return domain.Channel{}, nil
}
func (f *fakeNotificationRepo) GetChannel(context.Context, string, string) (domain.Channel, error) {
	return domain.Channel{Enabled: true}, nil
}
func (f *fakeNotificationRepo) ListChannels(context.Context, string, int, int) ([]domain.Channel, int, error) {
	return nil, 0, nil
}
func (f *fakeNotificationRepo) CreateTemplate(context.Context, domain.Template) (domain.Template, error) {
	return domain.Template{}, nil
}
func (f *fakeNotificationRepo) GetTemplate(context.Context, string, string) (domain.Template, error) {
	return domain.Template{}, nil
}
func (f *fakeNotificationRepo) CreateTask(context.Context, domain.Task) (domain.Task, error) {
	return domain.Task{}, nil
}
func (f *fakeNotificationRepo) PollTasks(context.Context, int, time.Time, time.Duration, string) ([]domain.Task, error) {
	return f.tasks, nil
}
func (f *fakeNotificationRepo) GetTask(context.Context, string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (f *fakeNotificationRepo) CompleteTask(context.Context, string, string) error { return nil }
func (f *fakeNotificationRepo) FailTask(context.Context, string, string, string, time.Duration, int) error {
	return nil
}
func (f *fakeNotificationRepo) RequeueTask(context.Context, string, string, time.Duration) error {
	return nil
}
func (f *fakeNotificationRepo) ListTasks(context.Context, string, domain.TaskStatus, int, int) ([]domain.Task, int, error) {
	return nil, 0, nil
}

type fakeNotificationSender struct{}

func (fakeNotificationSender) Send(context.Context, domain.Channel, domain.Template, domain.Task) (domain.Result, error) {
	return domain.Result{Delivered: true}, nil
}

func TestProcessBatchConcurrentCompletion(t *testing.T) {
	tasks := make([]domain.Task, 0, 80)
	for i := 0; i < 80; i++ {
		tasks = append(tasks, domain.Task{ID: string(rune('a' + i%26)) + string(rune('0'+i/26)), Status: domain.TaskRunning, Payload: []byte(`{"message":"cpu high"}`)})
	}
	repo := &fakeNotificationRepo{tasks: tasks}
	service := NewService(repo, fakeNotificationSender{}, slog.New(slog.NewTextHandler(io.Discard, nil)), "worker", time.Minute)

	processed, err := service.ProcessBatch(context.Background(), len(tasks))
	if err != nil {
		t.Fatalf("ProcessBatch returned error: %v", err)
	}
	if processed != len(tasks) {
		t.Fatalf("expected %d processed tasks, got %d", len(tasks), processed)
	}
}

func TestNotificationTaskNormalizeDefaultsMaxAttempts(t *testing.T) {
	task := domain.Task{}
	task.Normalize(time.Now())
	if task.MaxAttempts <= 0 {
		t.Fatalf("expected positive MaxAttempts, got %d", task.MaxAttempts)
	}
}

func TestNotificationTaskNormalizeDefaultsStatus(t *testing.T) {
	task := domain.Task{}
	task.Normalize(time.Now())
	if task.Status != domain.TaskPending {
		t.Fatalf("expected pending status, got %q", task.Status)
	}
}

func TestNotificationTaskNormalizeDefaultsAvailableAt(t *testing.T) {
	now := time.Now()
	task := domain.Task{}
	task.Normalize(now)
	if task.AvailableAt.IsZero() || !task.AvailableAt.Equal(now) {
		t.Fatalf("expected AvailableAt to be initialized to %v, got %v", now, task.AvailableAt)
	}
}

func TestNotificationTaskNormalizeDefaultsCreatedAt(t *testing.T) {
	now := time.Now()
	task := domain.Task{}
	task.Normalize(now)
	if task.CreatedAt.IsZero() || !task.CreatedAt.Equal(now) {
		t.Fatalf("expected CreatedAt to be initialized to %v, got %v", now, task.CreatedAt)
	}
}
