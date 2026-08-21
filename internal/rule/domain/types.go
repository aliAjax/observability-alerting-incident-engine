package domain

import (
	"errors"
	"time"

	"github.com/observability-alerting/engine/internal/common"
)

type Type string

const (
	TypeThreshold Type = "threshold"
	TypePercent   Type = "percent"
	TypeChange    Type = "change"
	TypeWindow    Type = "window"
	TypeCompound  Type = "compound"
	TypeSuppress  Type = "suppress"
)

type Mode string

const (
	ModeActive Mode = "active"
	ModePaused Mode = "paused"
	ModeDryRun Mode = "dry_run"
)

type Condition struct {
	Metric   string            `json:"metric,omitempty"`
	Field    string            `json:"field,omitempty"`
	Operator string            `json:"operator,omitempty"`
	Value    float64           `json:"value,omitempty"`
	Percent  *PercentCondition `json:"percent,omitempty"`
	Change   *ChangeCondition  `json:"change,omitempty"`
	Window   *WindowCondition  `json:"window,omitempty"`
	Logic    string            `json:"logic,omitempty"`
	Children []Condition       `json:"children,omitempty"`
}

type PercentCondition struct {
	Field           string  `json:"field"`
	Operator        string  `json:"operator"`
	Percent         float64 `json:"percent"`
	BaselineSeconds int64   `json:"baseline_seconds"`
}

type ChangeCondition struct {
	Field           string  `json:"field"`
	Mode            string  `json:"mode"`
	Percent         float64 `json:"percent"`
	Absolute        float64 `json:"absolute"`
	BaselineSeconds int64   `json:"baseline_seconds"`
}

type WindowCondition struct {
	Field    string  `json:"field"`
	Operator string  `json:"operator"`
	Seconds  int64   `json:"seconds"`
	Value    float64 `json:"value"`
}

type EscalationStep struct {
	DelaySeconds int64    `json:"delay_seconds"`
	Channels     []string `json:"channels"`
	Receivers    []string `json:"receivers,omitempty"`
}

type Rule struct {
	ID          string           `json:"id"`
	Tenant      string           `json:"tenant"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	DataSource  string           `json:"data_source"`
	EventType   string           `json:"event_type"`
	Type        Type             `json:"type"`
	Condition   Condition        `json:"condition"`
	Severity    string           `json:"severity"`
	Enabled     bool             `json:"enabled"`
	Mode        Mode             `json:"mode"`
	Window      time.Duration    `json:"window"`
	Interval    time.Duration    `json:"interval"`
	Labels      common.Labels    `json:"labels"`
	GroupBy     []string         `json:"group_by"`
	SuppressBy  []string         `json:"suppress_by"`
	Channels    []string         `json:"channels"`
	Escalation  []EscalationStep `json:"escalation"`
	Version     int              `json:"version"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func (r *Rule) Normalize(now time.Time) error {
	if r.ID == "" {
		r.ID = common.NewID("rule")
	}
	if r.Tenant == "" {
		r.Tenant = "default"
	}
	if r.Name == "" {
		return errors.New("rule name is required")
	}
	if r.DataSource == "" {
		return errors.New("data source is required")
	}
	if r.EventType == "" {
		r.EventType = "metric"
	}
	if r.Type == "" {
		r.Type = TypeThreshold
	}
	if r.Mode == "" {
		r.Mode = ModeActive
	}
	if !r.Enabled && r.Mode != ModePaused {
		r.Enabled = true
	}
	if r.Severity == "" {
		r.Severity = "warning"
	}
	if r.Window <= 0 {
		r.Window = 5 * time.Minute
	}
	if r.Interval <= 0 {
		r.Interval = time.Minute
	}
	if r.Labels == nil {
		r.Labels = common.Labels{}
	}
	if r.Version == 0 {
		r.Version = 1
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	return nil
}

func (r Rule) Active() bool {
	return r.Enabled && r.Mode != ModePaused
}

func (r Rule) ShouldEvaluate(now time.Time, last time.Time) bool {
	if !r.Enabled || r.Mode == ModePaused {
		return false
	}
	if last.IsZero() {
		return true
	}
	return now.Sub(last) >= r.Interval
}
