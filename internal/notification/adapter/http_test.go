package adapter

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/notification/application"
	"github.com/observability-alerting/engine/internal/notification/domain"
)

type contextNotificationRepo struct {
	got              context.Context
	gotCreateChannel context.Context
	gotListChannels  context.Context
}

func (f *contextNotificationRepo) CreateChannel(got context.Context, _ domain.Channel) (domain.Channel, error) {
	f.gotCreateChannel = got
	return domain.Channel{}, nil
}
func (f *contextNotificationRepo) UpdateChannel(context.Context, domain.Channel) (domain.Channel, error) {
	return domain.Channel{}, nil
}
func (f *contextNotificationRepo) GetChannel(context.Context, string, string) (domain.Channel, error) {
	return domain.Channel{}, nil
}
func (f *contextNotificationRepo) ListChannels(got context.Context, _ string, _, _ int) ([]domain.Channel, int, error) {
	f.gotListChannels = got
	return nil, 0, nil
}
func (f *contextNotificationRepo) CreateTemplate(context.Context, domain.Template) (domain.Template, error) {
	return domain.Template{}, nil
}
func (f *contextNotificationRepo) GetTemplate(context.Context, string, string) (domain.Template, error) {
	return domain.Template{}, nil
}
func (f *contextNotificationRepo) CreateTask(context.Context, domain.Task) (domain.Task, error) {
	return domain.Task{}, nil
}
func (f *contextNotificationRepo) PollTasks(context.Context, int, time.Time, time.Duration, string) ([]domain.Task, error) {
	return nil, nil
}
func (f *contextNotificationRepo) GetTask(context.Context, string) (domain.Task, error) {
	return domain.Task{}, nil
}
func (f *contextNotificationRepo) CompleteTask(context.Context, string, string) error { return nil }
func (f *contextNotificationRepo) FailTask(context.Context, string, string, string, time.Duration, int) error {
	return nil
}
func (f *contextNotificationRepo) RequeueTask(context.Context, string, string, time.Duration) error {
	return nil
}
func (f *contextNotificationRepo) ListTasks(got context.Context, _ string, _ domain.TaskStatus, _, _ int) ([]domain.Task, int, error) {
	f.got = got
	return nil, 0, nil
}

func TestR003TaskListKeepsRequestCtx(t *testing.T) {
	repo := &contextNotificationRepo{}
	service := application.NewService(repo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), "worker", time.Minute)
	handler := NewHandler(service)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification/tasks", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ListTasks(rec, req)
	if repo.got.Err() == nil {
		t.Fatal("expected ListTasks to pass the canceled request context")
	}
}

func TestR003ChannelCreateKeepsRequestCtx(t *testing.T) {
	repo := &contextNotificationRepo{}
	service := application.NewService(repo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), "worker", time.Minute)
	handler := NewHandler(service)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification/channels", strings.NewReader(`{"name":"ops"}`)).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.CreateChannel(rec, req)
	if repo.gotCreateChannel.Err() == nil {
		t.Fatal("expected CreateChannel to pass the canceled request context")
	}
}

func TestR003ChannelListKeepsRequestCtx(t *testing.T) {
	repo := &contextNotificationRepo{}
	service := application.NewService(repo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), "worker", time.Minute)
	handler := NewHandler(service)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification/channels", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ListChannels(rec, req)
	if repo.gotListChannels.Err() == nil {
		t.Fatal("expected ListChannels to pass the canceled request context")
	}
}
