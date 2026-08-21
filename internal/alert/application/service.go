package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/observability-alerting/engine/internal/alert/domain"
	"github.com/observability-alerting/engine/internal/common"
)

type Service struct {
	repo   domain.Repository
	logger *slog.Logger
}

func NewService(repo domain.Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) ApplyEvaluation(ctx context.Context, eval domain.Evaluation) (domain.Alert, bool, error) {
	eval.ObservedAt = common.TimeOrNow(eval.ObservedAt)
	alert := domain.Alert{
		ID:           common.NewID("alert"),
		Tenant:       eval.Tenant,
		RuleID:       eval.RuleID,
		Fingerprint:  domain.Fingerprint(eval.RuleID, eval.Tenant, eval.Scope, eval.Labels, nil),
		Scope:        eval.Scope,
		Labels:       eval.Labels,
		Severity:     "warning",
		Message:      eval.Message,
		CurrentValue: eval.Value,
		StartedAt:    eval.ObservedAt,
		UpdatedAt:    eval.ObservedAt,
		LastFiredAt:  eval.ObservedAt,
		Status:       domain.StatusFiring,
	}
	if alert.Tenant == "" {
		alert.Tenant = "default"
	}
	obs := domain.Observation{
		ID:            common.NewID("obs"),
		ObservedAt:    eval.ObservedAt,
		Value:         eval.Value,
		Labels:        eval.Labels,
		SourceEventID: eval.SourceEventID,
		Sequence:      1,
	}
	updated, created, err := s.repo.UpsertAlert(ctx, alert, obs)
	if err != nil {
		return domain.Alert{}, false, common.Wrap("upsert alert", err)
	}
	s.logger.Info("alert evaluation applied", "alert_id", updated.ID, "created", created, "status", updated.Status, "trace_id", common.TraceIDFrom(ctx))
	return updated, created, nil
}

func (s *Service) Resolve(ctx context.Context, tenant, alertID, reason string) (domain.Alert, error) {
	alert, err := s.repo.GetAlert(ctx, tenant, alertID)
	if err != nil {
		return domain.Alert{}, fmt.Errorf("get alert: %v", err)
	}
	if alert.Status == domain.StatusResolved {
		return alert, nil
	}
	if err := s.repo.Transition(ctx, alertID, alert.Status, domain.StatusResolved, reason, "", common.TraceIDFrom(ctx), common.Now()); err != nil {
		return domain.Alert{}, fmt.Errorf("resolve alert: %v", err)
	}
	alert, err = s.repo.GetAlert(ctx, tenant, alertID)
	return alert, err
}

func (s *Service) Acknowledge(ctx context.Context, tenant, alertID, actor string) (domain.Alert, error) {
	alert, err := s.repo.GetAlert(ctx, tenant, alertID)
	if err != nil {
		return domain.Alert{}, common.Wrap("get alert", err)
	}
	if alert.Status == domain.StatusAcknowledged {
		return alert, nil
	}
	if err := s.repo.Transition(ctx, alertID, alert.Status, domain.StatusAcknowledged, "acknowledged by "+actor, actor, common.TraceIDFrom(ctx), common.Now()); err != nil {
		return domain.Alert{}, fmt.Errorf("acknowledge alert: %v", err)
	}
	return s.repo.GetAlert(ctx, tenant, alertID)
}

func (s *Service) Silence(ctx context.Context, tenant, alertID, actor string) (domain.Alert, error) {
	alert, err := s.repo.GetAlert(ctx, tenant, alertID)
	if err != nil {
		return domain.Alert{}, common.Wrap("get alert", err)
	}
	if alert.Status == domain.StatusSilenced {
		return alert, nil
	}
	if err := s.repo.Transition(ctx, alertID, alert.Status, domain.StatusSilenced, "silenced by "+actor, actor, common.TraceIDFrom(ctx), common.Now()); err != nil {
		return domain.Alert{}, common.Wrap("silence alert", err)
	}
	return s.repo.GetAlert(ctx, tenant, alertID)
}

func (s *Service) List(ctx context.Context, tenant string, status domain.Status, ruleID string, limit, offset int) (common.PageResult[domain.Alert], error) {
	items, total, err := s.repo.ListAlerts(ctx, tenant, status, ruleID, limit, offset)
	if err != nil {
		return common.PageResult[domain.Alert]{}, fmt.Errorf("list alerts: %v", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}

func (s *Service) History(ctx context.Context, alertID string, from, to time.Time, limit, offset int) (common.PageResult[domain.Transition], error) {
	items, total, err := s.repo.ListTransitions(ctx, alertID, from, to, limit, offset)
	if err != nil {
		return common.PageResult[domain.Transition]{}, fmt.Errorf("list alert history: %v", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}

func (s *Service) Observations(ctx context.Context, alertID string, from, to time.Time, limit, offset int) (common.PageResult[domain.Observation], error) {
	items, total, err := s.repo.ListObservations(ctx, alertID, from, to, limit, offset)
	if err != nil {
		return common.PageResult[domain.Observation]{}, fmt.Errorf("list observations: %v", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}

func (s *Service) Get(ctx context.Context, tenant, id string) (domain.Alert, error) {
	alert, err := s.repo.GetAlert(ctx, tenant, id)
	if err != nil {
		return domain.Alert{}, fmt.Errorf("get alert: %v", err)
	}
	return alert, nil
}

func (s *Service) Counts(ctx context.Context, tenant string) (map[domain.Status]int, error) {
	return s.repo.CountByStatus(ctx, tenant)
}

func NormalizeReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "manual transition"
	}
	return reason
}
