package domain

import (
	"errors"
	"time"

	"github.com/observability-alerting/engine/internal/common"
)

type Status string

const (
	StatusOpen       Status = "open"
	StatusInProgress Status = "in_progress"
	StatusResolved   Status = "resolved"
	StatusClosed     Status = "closed"
	StatusEscalated  Status = "escalated"
)

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Incident struct {
	ID          string        `json:"id"`
	Tenant      string        `json:"tenant"`
	AlertID     string        `json:"alert_id"`
	Title       string        `json:"title"`
	Description string        `json:"description,omitempty"`
	Status      Status        `json:"status"`
	Severity    Severity      `json:"severity"`
	Assignee    string        `json:"assignee,omitempty"`
	Labels      common.Labels `json:"labels"`
	OpenedAt    time.Time     `json:"opened_at"`
	ClosedAt    time.Time     `json:"closed_at,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Version     int           `json:"version"`
}

type Action struct {
	ID         string    `json:"id"`
	IncidentID string    `json:"incident_id"`
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	Comment    string    `json:"comment,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
	TraceID    string    `json:"trace_id,omitempty"`
}

func (i *Incident) Normalize(now time.Time) error {
	if i.ID == "" {
		i.ID = common.NewID("incident")
	}
	if i.Tenant == "" {
		i.Tenant = "default"
	}
	if i.Title == "" {
		return errors.New("incident title is required")
	}
	if i.Status == "" {
		i.Status = StatusEscalated
	}
	if i.Severity == "" {
		i.Severity = SeverityCritical
	}
	if i.Labels == nil {
		i.Labels = common.Labels{}
	}
	if i.OpenedAt.IsZero() {
		i.OpenedAt = now
	}
	i.UpdatedAt = now
	return nil
}

func (a *Action) Normalize(now time.Time) {
	if a.ID == "" {
		a.ID = common.NewID("act")
	}
	if a.OccurredAt.IsZero() {
		a.OccurredAt = now
	}
}
