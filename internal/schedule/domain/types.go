package domain

import (
	"errors"
	"time"

	"github.com/observability-alerting/engine/internal/common"
)

type Schedule struct {
	ID          string    `json:"id"`
	Tenant      string    `json:"tenant"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Timezone    string    `json:"timezone"`
	Enabled     bool      `json:"enabled"`
	Shifts      []Shift   `json:"shifts"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Shift struct {
	ID         string    `json:"id"`
	ScheduleID string    `json:"schedule_id"`
	Assignee   string    `json:"assignee"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
}

type OnCall struct {
	ScheduleID string    `json:"schedule_id"`
	Schedule   string    `json:"schedule"`
	Assignee   string    `json:"assignee"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
}

func (s *Schedule) Normalize(now time.Time) error {
	if s.ID == "" {
		s.ID = common.NewID("sched")
	}
	if s.Tenant == "" {
		s.Tenant = "default"
	}
	if s.Name == "" {
		return errors.New("schedule name is required")
	}
	if s.Timezone == "" {
		s.Timezone = "UTC"
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	return nil
}

func (s Schedule) OnCallAt(at time.Time) *OnCall {
	if !s.Enabled {
		return nil
	}
	for _, shift := range s.Shifts {
		if !shift.StartsAt.After(at) && shift.EndsAt.After(at) {
			return &OnCall{ScheduleID: s.ID, Schedule: s.Name, Assignee: shift.Assignee, StartsAt: shift.StartsAt, EndsAt: shift.EndsAt}
		}
	}
	return nil
}
