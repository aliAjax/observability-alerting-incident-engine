package domain

import (
	"encoding/json"
	"time"
)

type Record struct {
	ID         string          `json:"id"`
	Tenant     string          `json:"tenant"`
	Entity     string          `json:"entity"`
	EntityID   string          `json:"entity_id"`
	Action     string          `json:"action"`
	Actor      string          `json:"actor,omitempty"`
	Detail     json.RawMessage `json:"detail,omitempty"`
	OccurredAt time.Time       `json:"occurred_at"`
	TraceID    string          `json:"trace_id,omitempty"`
}

func (r *Record) Normalize(now time.Time) {
	if r.Tenant == "" {
		r.Tenant = "default"
	}
	if r.OccurredAt.IsZero() {
		r.OccurredAt = now
	}
}
