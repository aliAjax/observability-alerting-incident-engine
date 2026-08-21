package application

import (
	"context"
	"log/slog"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/incident/domain"
)

type Service struct {
	repo   domain.Repository
	logger *slog.Logger
}

func NewService(repo domain.Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Create(ctx context.Context, incident domain.Incident) (domain.Incident, error) {
	if err := incident.Normalize(common.Now()); err != nil {
		return domain.Incident{}, common.Wrap("normalize incident", err)
	}
	created, err := s.repo.Create(ctx, incident)
	if err != nil {
		return domain.Incident{}, common.Wrap("create incident", err)
	}
	_ = s.repo.AddAction(ctx, domain.Action{
		IncidentID: created.ID,
		Action:     "created",
		Actor:      "system",
		OccurredAt: common.Now(),
		TraceID:    common.TraceIDFrom(ctx),
	})
	return created, nil
}

func (s *Service) Assign(ctx context.Context, tenant, id, assignee, actor string) (domain.Incident, error) {
	incident, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Incident{}, common.Wrap("get incident", err)
	}
	incident.Assignee = assignee
	incident.Status = domain.StatusInProgress
	incident.UpdatedAt = common.Now()
	incident.Version++
	updated, err := s.repo.Update(ctx, incident)
	if err != nil {
		return domain.Incident{}, common.Wrap("assign incident", err)
	}
	_ = s.repo.AddAction(ctx, domain.Action{
		IncidentID: id, Action: "assigned", Actor: actor, Comment: assignee, OccurredAt: common.Now(), TraceID: common.TraceIDFrom(ctx),
	})
	return updated, nil
}

func (s *Service) Act(ctx context.Context, tenant, id, action, actor, comment string) (domain.Incident, error) {
	incident, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Incident{}, common.Wrap("get incident", err)
	}
	incident.UpdatedAt = common.Now()
	incident.Version++
	if action == "resolve" || action == "close" {
		incident.Status = domain.StatusResolved
		incident.ClosedAt = common.Now()
	}
	updated, err := s.repo.Update(ctx, incident)
	if err != nil {
		return domain.Incident{}, common.Wrap("update incident", err)
	}
	_ = s.repo.AddAction(ctx, domain.Action{
		IncidentID: id, Action: action, Actor: actor, Comment: comment, OccurredAt: common.Now(), TraceID: common.TraceIDFrom(ctx),
	})
	return updated, nil
}

func (s *Service) Close(ctx context.Context, tenant, id, actor, reason string) (domain.Incident, error) {
	return s.Act(ctx, tenant, id, "close", actor, reason)
}

func (s *Service) Escalate(ctx context.Context, tenant, id, actor, reason string) (domain.Incident, error) {
	incident, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Incident{}, common.Wrap("get incident", err)
	}
	incident.Status = domain.StatusEscalated
	incident.UpdatedAt = common.Now()
	incident.Version++
	updated, err := s.repo.Update(ctx, incident)
	if err != nil {
		return domain.Incident{}, common.Wrap("escalate incident", err)
	}
	_ = s.repo.AddAction(ctx, domain.Action{
		IncidentID: id, Action: "escalated", Actor: actor, Comment: reason, OccurredAt: common.Now(), TraceID: common.TraceIDFrom(ctx),
	})
	return updated, nil
}

func (s *Service) Get(ctx context.Context, tenant, id string) (domain.Incident, error) {
	incident, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Incident{}, common.Wrap("get incident", err)
	}
	return incident, nil
}

func (s *Service) List(ctx context.Context, tenant string, status domain.Status, assignee string, limit, offset int) (common.PageResult[domain.Incident], error) {
	items, total, err := s.repo.List(ctx, tenant, status, assignee, limit, offset)
	if err != nil {
		return common.PageResult[domain.Incident]{}, common.Wrap("list incidents", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}

func (s *Service) Actions(ctx context.Context, incidentID string, limit, offset int) (common.PageResult[domain.Action], error) {
	items, total, err := s.repo.ListActions(ctx, incidentID, limit, offset)
	if err != nil {
		return common.PageResult[domain.Action]{}, common.Wrap("list incident actions", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}
