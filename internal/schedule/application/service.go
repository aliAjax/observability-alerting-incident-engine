package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/schedule/domain"
)

type Service struct {
	repo   domain.Repository
	logger *slog.Logger
}

func NewService(repo domain.Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Create(ctx context.Context, schedule domain.Schedule) (domain.Schedule, error) {
	if err := schedule.Normalize(common.Now()); err != nil {
		return domain.Schedule{}, common.Wrap("normalize schedule", err)
	}
	created, err := s.repo.Create(ctx, schedule)
	if err != nil {
		return domain.Schedule{}, common.Wrap("create schedule", err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, schedule domain.Schedule) (domain.Schedule, error) {
	if schedule.UpdatedAt.IsZero() {
		schedule.UpdatedAt = common.Now()
	}
	updated, err := s.repo.Update(ctx, schedule)
	if err != nil {
		return domain.Schedule{}, common.Wrap("update schedule", err)
	}
	return updated, nil
}

func (s *Service) Get(ctx context.Context, tenant, id string) (domain.Schedule, error) {
	schedule, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Schedule{}, common.Wrap("get schedule", err)
	}
	return schedule, nil
}

func (s *Service) List(ctx context.Context, tenant string, limit, offset int) (common.PageResult[domain.Schedule], error) {
	items, total, err := s.repo.List(ctx, tenant, limit, offset)
	if err != nil {
		return common.PageResult[domain.Schedule]{}, common.Wrap("list schedules", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}

func (s *Service) OnCall(ctx context.Context, tenant string, at time.Time) ([]domain.OnCall, error) {
	schedules, err := s.repo.FindActive(ctx, tenant, at)
	if err != nil {
		return nil, common.Wrap("find active schedules", err)
	}
	out := make([]domain.OnCall, 0)
	for _, schedule := range schedules {
		if onCall := schedule.OnCallAt(at); onCall != nil {
			out = append(out, *onCall)
		}
	}
	return out, nil
}
