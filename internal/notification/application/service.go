package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/notification/domain"
)

type Service struct {
	repo     domain.Repository
	sender   domain.Sender
	logger   *slog.Logger
	workerID string
	lockTTL  time.Duration
}

func NewService(repo domain.Repository, sender domain.Sender, logger *slog.Logger, workerID string, lockTTL time.Duration) *Service {
	if lockTTL <= 0 {
		lockTTL = time.Minute
	}
	return &Service{repo: repo, sender: sender, logger: logger, workerID: workerID, lockTTL: lockTTL}
}

func (s *Service) CreateChannel(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	now := common.Now()
	if channel.ID == "" {
		channel.ID = common.NewID("chan")
	}
	if channel.Tenant == "" {
		channel.Tenant = "default"
	}
	if channel.Name == "" {
		return domain.Channel{}, common.Wrap("create channel", common.ErrInvalid)
	}
	if channel.CreatedAt.IsZero() {
		channel.CreatedAt = now
	}
	channel.UpdatedAt = now
	return s.repo.CreateChannel(ctx, channel)
}

func (s *Service) CreateTemplate(ctx context.Context, template domain.Template) (domain.Template, error) {
	now := common.Now()
	if template.ID == "" {
		template.ID = common.NewID("tpl")
	}
	if template.Tenant == "" {
		template.Tenant = "default"
	}
	if template.Name == "" || template.Body == "" {
		return domain.Template{}, common.Wrap("create template", common.ErrInvalid)
	}
	if template.CreatedAt.IsZero() {
		template.CreatedAt = now
	}
	template.UpdatedAt = now
	return s.repo.CreateTemplate(ctx, template)
}

func (s *Service) EnqueueAlert(ctx context.Context, alertID, ruleID, tenant string, channels []string, payload any) ([]domain.Task, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, common.Wrap("marshal notification payload", err)
	}
	tasks := make([]domain.Task, 0, len(channels))
	for _, channelID := range channels {
		task := domain.Task{
			Tenant:      tenant,
			AlertID:     alertID,
			RuleID:      ruleID,
			ChannelID:   channelID,
			Payload:     raw,
			Status:      domain.TaskPending,
			MaxAttempts: 5,
			AvailableAt: common.Now(),
		}
		task.Normalize(common.Now())
		created, err := s.repo.CreateTask(ctx, task)
		if err != nil {
			return tasks, common.Wrap("enqueue notification", err)
		}
		tasks = append(tasks, created)
	}
	return tasks, nil
}

func (s *Service) ProcessBatch(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 20
	}
	tasks, err := s.repo.PollTasks(ctx, limit, common.Now(), s.lockTTL, s.workerID)
	if err != nil {
		return 0, common.Wrap("poll notification tasks", err)
	}
	processed := 0
	for _, task := range tasks {
		if err := s.processTask(ctx, task); err != nil {
			_ = s.repo.FailTask(ctx, task.ID, s.workerID, err.Error(), backoffDelay(task.Attempts), task.EscalationStep)
			continue
		}
		if err := s.repo.CompleteTask(ctx, task.ID, s.workerID); err != nil {
			s.logger.Error("complete notification task", "task_id", task.ID, "error", err)
			continue
		}
		processed++
	}
	return processed, nil
}

func (s *Service) processTask(ctx context.Context, task domain.Task) error {
	if task.AvailableAt.After(common.Now()) {
		_ = s.repo.RequeueTask(ctx, task.ID, s.workerID, time.Until(task.AvailableAt))
		return fmt.Errorf("task not available yet")
	}
	channel, err := s.repo.GetChannel(ctx, task.Tenant, task.ChannelID)
	if err != nil {
		return common.Wrap("get channel", err)
	}
	if !channel.Enabled {
		return common.ErrUnavailable
	}
	template := domain.Template{Subject: "Observability alert", Body: "{{.message}}"}
	if task.TemplateID != "" {
		tmpl, err := s.repo.GetTemplate(ctx, task.Tenant, task.TemplateID)
		if err == nil {
			template = tmpl
		}
	}
	result, err := s.sender.Send(ctx, channel, template, task)
	if err != nil {
		return common.Wrap("send notification", err)
	}
	if !result.Delivered {
		return common.Wrap("send notification", fmt.Errorf("%s", result.Message))
	}
	return nil
}

func (s *Service) ListTasks(ctx context.Context, tenant string, status domain.TaskStatus, limit, offset int) (common.PageResult[domain.Task], error) {
	items, total, err := s.repo.ListTasks(ctx, tenant, status, limit, offset)
	if err != nil {
		return common.PageResult[domain.Task]{}, common.Wrap("list notification tasks", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}

func (s *Service) ListChannels(ctx context.Context, tenant string, limit, offset int) (common.PageResult[domain.Channel], error) {
	items, total, err := s.repo.ListChannels(ctx, tenant, limit, offset)
	if err != nil {
		return common.PageResult[domain.Channel]{}, common.Wrap("list notification channels", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}

func backoffDelay(attempts int) time.Duration {
	delay := time.Duration(1<<attempts) * time.Second
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	return delay
}

func RenderTemplate(template domain.Template, task domain.Task) (string, string) {
	subject := template.Subject
	body := template.Body
	var payload map[string]any
	_ = json.Unmarshal(task.Payload, &payload)
	for k, v := range payload {
		subject = strings.ReplaceAll(subject, "{{."+k+"}}", fmt.Sprint(v))
		body = strings.ReplaceAll(body, "{{."+k+"}}", fmt.Sprint(v))
	}
	return subject, body
}
