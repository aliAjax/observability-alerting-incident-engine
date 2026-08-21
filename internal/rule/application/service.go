package application

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/rule/domain"
)

type Service struct {
	repo   domain.Repository
	logger *slog.Logger
}

func NewService(repo domain.Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Create(ctx context.Context, rule domain.Rule) (domain.Rule, error) {
	if err := rule.Normalize(common.Now()); err != nil {
		return domain.Rule{}, common.Wrap("normalize rule", err)
	}
	if err := validateRule(rule); err != nil {
		return domain.Rule{}, common.Wrap("validate rule", err)
	}
	created, err := s.repo.Create(ctx, rule)
	if err != nil {
		return domain.Rule{}, common.Wrap("create rule", err)
	}
	s.logger.Info("rule created", "rule_id", created.ID, "tenant", created.Tenant, "trace_id", common.TraceIDFrom(ctx))
	return created, nil
}

func (s *Service) Update(ctx context.Context, id string, patch domain.Rule) (domain.Rule, error) {
	existing, err := s.repo.Get(ctx, patch.Tenant, id)
	if err != nil {
		return domain.Rule{}, common.Wrap("get rule", err)
	}
	existing.Name = firstNonEmpty(patch.Name, existing.Name)
	existing.Description = firstNonEmpty(patch.Description, existing.Description)
	existing.DataSource = firstNonEmpty(patch.DataSource, existing.DataSource)
	existing.EventType = firstNonEmpty(patch.EventType, existing.EventType)
	existing.Severity = firstNonEmpty(patch.Severity, existing.Severity)
	if patch.Type != "" {
		existing.Type = patch.Type
	}
	if patch.Mode != "" {
		existing.Mode = patch.Mode
	}
	if patch.Enabled {
		existing.Enabled = true
	}
	if !conditionEmpty(patch.Condition) {
		existing.Condition = patch.Condition
	}
	if patch.Window > 0 {
		existing.Window = patch.Window
	}
	if patch.Interval > 0 {
		existing.Interval = patch.Interval
	}
	if len(patch.Labels) > 0 {
		existing.Labels = patch.Labels.Clone()
	}
	if len(patch.GroupBy) > 0 {
		existing.GroupBy = append([]string(nil), patch.GroupBy...)
	}
	if len(patch.SuppressBy) > 0 {
		existing.SuppressBy = append([]string(nil), patch.SuppressBy...)
	}
	if len(patch.Channels) > 0 {
		existing.Channels = append([]string(nil), patch.Channels...)
	}
	if len(patch.Escalation) > 0 {
		existing.Escalation = append([]domain.EscalationStep(nil), patch.Escalation...)
	}
	existing.Version++
	existing.UpdatedAt = common.Now()
	if err := validateRule(existing); err != nil {
		return domain.Rule{}, common.Wrap("validate updated rule", err)
	}
	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return domain.Rule{}, common.Wrap("update rule", err)
	}
	s.logger.Info("rule updated", "rule_id", updated.ID, "version", updated.Version, "trace_id", common.TraceIDFrom(ctx))
	return updated, nil
}

func (s *Service) Get(ctx context.Context, tenant, id string) (domain.Rule, error) {
	rule, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Rule{}, common.Wrap("get rule", err)
	}
	return rule, nil
}

func (s *Service) List(ctx context.Context, tenant string, limit, offset int, filters map[string]string) (common.PageResult[domain.Rule], error) {
	items, total, err := s.repo.List(ctx, tenant, limit, offset, filters)
	if err != nil {
		return common.PageResult[domain.Rule]{}, common.Wrap("list rules", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}

func (s *Service) Pause(ctx context.Context, tenant, id string) (domain.Rule, error) {
	rule, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Rule{}, common.Wrap("get rule", err)
	}
	rule.Mode = domain.ModePaused
	rule.Enabled = false
	rule.Version++
	rule.UpdatedAt = common.Now()
	updated, err := s.repo.Update(ctx, rule)
	if err != nil {
		return domain.Rule{}, common.Wrap("pause rule", err)
	}
	return updated, nil
}

func (s *Service) Enable(ctx context.Context, tenant, id string) (domain.Rule, error) {
	rule, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Rule{}, common.Wrap("get rule", err)
	}
	rule.Enabled = true
	rule.Mode = domain.ModeActive
	rule.Version++
	rule.UpdatedAt = common.Now()
	updated, err := s.repo.Update(ctx, rule)
	if err != nil {
		return domain.Rule{}, common.Wrap("enable rule", err)
	}
	return updated, nil
}

func (s *Service) SetMode(ctx context.Context, tenant, id string, mode domain.Mode) (domain.Rule, error) {
	rule, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Rule{}, common.Wrap("get rule", err)
	}
	switch mode {
	case domain.ModeActive:
		rule.Enabled = true
	case domain.ModePaused:
		rule.Enabled = false
	case domain.ModeDryRun:
		rule.Enabled = true
	default:
		return domain.Rule{}, common.Wrap("set rule mode", common.ErrInvalid)
	}
	rule.Mode = mode
	rule.Version++
	rule.UpdatedAt = common.Now()
	return s.repo.Update(ctx, rule)
}

func (s *Service) ListForEvaluation(ctx context.Context, now time.Time, limit int) ([]domain.Rule, error) {
	if limit <= 0 {
		limit = 500
	}
	return s.repo.ListForEvaluation(ctx, now, limit)
}

func validateRule(rule domain.Rule) error {
	switch rule.Type {
	case domain.TypeThreshold:
		if rule.Condition.Field == "" || rule.Condition.Operator == "" {
			return common.ErrInvalid
		}
	case domain.TypePercent:
		if rule.Condition.Percent == nil {
			return common.ErrInvalid
		}
	case domain.TypeChange:
		if rule.Condition.Change == nil {
			return common.ErrInvalid
		}
	case domain.TypeWindow:
		if rule.Condition.Window == nil {
			return common.ErrInvalid
		}
	case domain.TypeCompound:
		if len(rule.Condition.Children) == 0 {
			return common.ErrInvalid
		}
	}
	if strings.TrimSpace(rule.Name) == "" {
		return common.ErrInvalid
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func conditionEmpty(cond domain.Condition) bool {
	return cond.Metric == "" && cond.Field == "" && cond.Operator == "" && cond.Value == 0 &&
		cond.Percent == nil && cond.Change == nil && cond.Window == nil &&
		cond.Logic == "" && len(cond.Children) == 0
}
