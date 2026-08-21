package domain

import (
	"errors"
	"time"

	"github.com/observability-alerting/engine/internal/common"
)

type Silence struct {
	ID        string        `json:"id"`
	Tenant    string        `json:"tenant"`
	RuleID    string        `json:"rule_id,omitempty"`
	Scope     string        `json:"scope,omitempty"`
	Matchers  common.Labels `json:"matchers"`
	StartsAt  time.Time     `json:"starts_at"`
	EndsAt    time.Time     `json:"ends_at"`
	CreatedBy string        `json:"created_by"`
	Comment   string        `json:"comment,omitempty"`
	Active    bool          `json:"active"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func (s *Silence) Normalize(now time.Time) error {
	if s.ID == "" {
		s.ID = common.NewID("silence")
	}
	if s.Tenant == "" {
		s.Tenant = "default"
	}
	if s.StartsAt.IsZero() {
		s.StartsAt = now
	}
	if s.EndsAt.IsZero() || !s.EndsAt.After(s.StartsAt) {
		return errors.New("silence end must be after start")
	}
	if s.Matchers == nil {
		s.Matchers = common.Labels{}
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	s.Active = true
	return nil
}

func (s Silence) Matches(ruleID, scope string, labels common.Labels, at time.Time) bool {
	if s.Active && at.Before(s.EndsAt) && at.After(s.StartsAt) {
		if s.RuleID != "" && s.RuleID != ruleID {
			return false
		}
		if s.Scope != "" && s.Scope != scope {
			return false
		}
		if len(s.Matchers) > 0 && !labels.Match(s.Matchers) {
			return false
		}
		return true
	}
	return false
}
