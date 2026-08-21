package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/observability-alerting/engine/internal/common"
)

type Status string

const (
	StatusPending      Status = "pending"
	StatusFiring       Status = "firing"
	StatusResolved     Status = "resolved"
	StatusAcknowledged Status = "acknowledged"
	StatusSilenced     Status = "silenced"
)

type Alert struct {
	ID             string        `json:"id"`
	Tenant         string        `json:"tenant"`
	RuleID         string        `json:"rule_id"`
	Fingerprint    string        `json:"fingerprint"`
	Scope          string        `json:"scope"`
	Labels         common.Labels `json:"labels"`
	Status         Status        `json:"status"`
	Severity       string        `json:"severity"`
	Message        string        `json:"message"`
	CurrentValue   float64       `json:"current_value"`
	StartedAt      time.Time     `json:"started_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	LastFiredAt    time.Time     `json:"last_fired_at"`
	ResolvedAt     time.Time     `json:"resolved_at,omitempty"`
	AcknowledgedAt time.Time     `json:"acknowledged_at,omitempty"`
	SilencedAt     time.Time     `json:"silenced_at,omitempty"`
	FiredCount     int           `json:"fired_count"`
	Version        int           `json:"version"`
}

type Observation struct {
	ID            string        `json:"id"`
	AlertID       string        `json:"alert_id"`
	ObservedAt    time.Time     `json:"observed_at"`
	Value         float64       `json:"value"`
	Labels        common.Labels `json:"labels"`
	SourceEventID string        `json:"source_event_id,omitempty"`
	Sequence      int           `json:"sequence"`
}

type Transition struct {
	ID         string    `json:"id"`
	AlertID    string    `json:"alert_id"`
	FromStatus Status    `json:"from_status"`
	ToStatus   Status    `json:"to_status"`
	Reason     string    `json:"reason"`
	Actor      string    `json:"actor,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
	TraceID    string    `json:"trace_id,omitempty"`
}

type Evaluation struct {
	RuleID        string        `json:"rule_id"`
	Tenant        string        `json:"tenant"`
	Scope         string        `json:"scope"`
	Labels        common.Labels `json:"labels"`
	Value         float64       `json:"value"`
	Message       string        `json:"message"`
	ObservedAt    time.Time     `json:"observed_at"`
	SourceEventID string        `json:"source_event_id,omitempty"`
	DryRun        bool          `json:"dry_run,omitempty"`
}

func (a *Alert) Normalize(now time.Time) {
	if a.ID == "" {
		a.ID = common.NewID("alert")
	}
	if a.Tenant == "" {
		a.Tenant = "default"
	}
	if a.Labels == nil {
		a.Labels = common.Labels{}
	}
	if a.Status == "" {
		a.Status = StatusPending
	}
	if a.StartedAt.IsZero() {
		a.StartedAt = now
	}
	if a.UpdatedAt.IsZero() {
		a.UpdatedAt = now
	}
}

func Fingerprint(ruleID, tenant, scope string, labels common.Labels, groupBy []string) string {
	if scope == "" && len(groupBy) > 0 {
		parts := make([]string, 0, len(groupBy))
		for _, k := range groupBy {
			parts = append(parts, k+"="+labels[k])
		}
		scope = strings.Join(parts, "|")
	}
	return strings.Join([]string{tenant, ruleID, scope, labels.String()}, "::")
}

func ValidateTransition(from, to Status) error {
	if from == to {
		return nil
	}
	allowed := map[Status][]Status{
		StatusPending:      {StatusFiring, StatusResolved},
		StatusFiring:       {StatusResolved, StatusAcknowledged, StatusSilenced},
		StatusResolved:     {StatusFiring},
		StatusAcknowledged: {StatusFiring, StatusResolved, StatusSilenced},
		StatusSilenced:     {StatusFiring, StatusResolved, StatusAcknowledged},
	}
	for _, candidate := range allowed[from] {
		if candidate == to {
			return nil
		}
	}
	return errors.New("invalid alert status transition")
}
