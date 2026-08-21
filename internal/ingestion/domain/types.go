package domain

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/observability-alerting/engine/internal/common"
)

type EventType string

const (
	EventMetric EventType = "metric"
	EventLog    EventType = "log"
	EventCheck  EventType = "check"
)

type Source struct {
	Name string
	Type string
}

type IngestionEvent struct {
	ID           string        `json:"id"`
	Tenant       string        `json:"tenant"`
	Source       string        `json:"source"`
	Type         EventType     `json:"type"`
	Labels       common.Labels `json:"labels"`
	NumericValue float64       `json:"numeric_value"`
	TextValue    string        `json:"text_value,omitempty"`
	OccurredAt   time.Time     `json:"occurred_at"`
	ReceivedAt   time.Time     `json:"received_at"`
	DedupeKey    string        `json:"dedupe_key,omitempty"`
	TraceID      string        `json:"trace_id,omitempty"`
}

type Batch struct {
	Tenant string           `json:"tenant"`
	Source string           `json:"source"`
	Events []IngestionEvent `json:"events"`
}

type Route struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type ValidationError struct {
	Index int
	Err   error
}

func (e *ValidationError) Error() string {
	return e.Err.Error()
}

func ValidateEvent(e *IngestionEvent) error {
	if strings.TrimSpace(e.Source) == "" {
		return errors.New("source is required")
	}
	switch e.Type {
	case EventMetric:
		if e.NumericValue == 0 && e.TextValue == "" {
			return errors.New("metric requires numeric_value")
		}
	case EventLog:
		if strings.TrimSpace(e.TextValue) == "" {
			return errors.New("log requires text_value")
		}
	case EventCheck:
		if strings.TrimSpace(e.TextValue) == "" {
			return errors.New("check requires text_value")
		}
	default:
		return errors.New("unsupported event type")
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = common.Now()
	}
	if e.Tenant == "" {
		e.Tenant = "default"
	}
	return nil
}

func (e *IngestionEvent) Fill(now time.Time) {
	if e.ID == "" {
		e.ID = common.NewID("evt")
	}
	if e.Tenant == "" {
		e.Tenant = "default"
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = now
	}
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = now
	}
	if e.Labels == nil {
		e.Labels = common.Labels{}
	}
	if e.TraceID == "" {
		e.TraceID = common.NewTraceID()
	}
}

func (e IngestionEvent) DedupeFingerprint() string {
	// Tenant and NumericValue are part of the identity so that events from
	// different tenants or metrics with different values are not collapsed
	// into a single dedupe entry.
	return strings.Join([]string{
		e.Tenant,
		e.Source,
		string(e.Type),
		e.Labels.String(),
		e.TextValue,
		strconv.FormatFloat(e.NumericValue, 'f', -1, 64),
	}, "|")
}
