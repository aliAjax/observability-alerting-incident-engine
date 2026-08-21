package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/observability-alerting/engine/internal/common"
)

type ChannelType string

const (
	ChannelWebhook   ChannelType = "webhook"
	ChannelEmail     ChannelType = "email"
	ChannelSlack     ChannelType = "slack"
	ChannelPagerDuty ChannelType = "pagerduty"
)

type Channel struct {
	ID        string          `json:"id"`
	Tenant    string          `json:"tenant"`
	Name      string          `json:"name"`
	Type      ChannelType     `json:"type"`
	Config    json.RawMessage `json:"config"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type Template struct {
	ID          string      `json:"id"`
	Tenant      string      `json:"tenant"`
	Name        string      `json:"name"`
	ChannelType ChannelType `json:"channel_type"`
	Subject     string      `json:"subject"`
	Body        string      `json:"body"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

type Task struct {
	ID             string          `json:"id"`
	Tenant         string          `json:"tenant"`
	AlertID        string          `json:"alert_id"`
	RuleID         string          `json:"rule_id"`
	ChannelID      string          `json:"channel_id"`
	TemplateID     string          `json:"template_id,omitempty"`
	Receivers      []string        `json:"receivers"`
	Payload        json.RawMessage `json:"payload"`
	Status         TaskStatus      `json:"status"`
	Attempts       int             `json:"attempts"`
	MaxAttempts    int             `json:"max_attempts"`
	AvailableAt    time.Time       `json:"available_at"`
	LastError      string          `json:"last_error,omitempty"`
	LastAttemptAt  time.Time       `json:"last_attempt_at,omitempty"`
	CooldownUntil  time.Time       `json:"cooldown_until,omitempty"`
	EscalationStep int             `json:"escalation_step"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (t *Task) Normalize(now time.Time) {
	if t.ID == "" {
		t.ID = common.NewID("notify")
	}
	if t.Tenant == "" {
		t.Tenant = "default"
	}
	if t.Status == "" {
		t.Status = TaskPending
	}
	if t.MaxAttempts <= 0 {
		t.MaxAttempts = 5
	}
	if t.AvailableAt.IsZero() {
		t.AvailableAt = now
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
}

type Result struct {
	Delivered bool
	Message   string
}

type Sender interface {
	Send(ctx context.Context, channel Channel, template Template, task Task) (Result, error)
}

type Repository interface {
	CreateChannel(ctx context.Context, channel Channel) (Channel, error)
	UpdateChannel(ctx context.Context, channel Channel) (Channel, error)
	GetChannel(ctx context.Context, tenant, id string) (Channel, error)
	ListChannels(ctx context.Context, tenant string, limit, offset int) ([]Channel, int, error)
	CreateTemplate(ctx context.Context, template Template) (Template, error)
	GetTemplate(ctx context.Context, tenant, id string) (Template, error)
	CreateTask(ctx context.Context, task Task) (Task, error)
	PollTasks(ctx context.Context, limit int, now time.Time, lockTTL time.Duration, owner string) ([]Task, error)
	GetTask(ctx context.Context, id string) (Task, error)
	CompleteTask(ctx context.Context, id, owner string) error
	FailTask(ctx context.Context, id, owner string, cause string, retryDelay time.Duration, escalationStep int) error
	RequeueTask(ctx context.Context, id, owner string, delay time.Duration) error
	ListTasks(ctx context.Context, tenant string, status TaskStatus, limit, offset int) ([]Task, int, error)
}
