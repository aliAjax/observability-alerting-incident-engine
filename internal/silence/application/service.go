package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/silence/domain"
)

type Service struct {
	repo   domain.Repository
	logger *slog.Logger
}

func NewService(repo domain.Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Create(ctx context.Context, silence domain.Silence) (domain.Silence, error) {
	if err := silence.Normalize(common.Now()); err != nil {
		return domain.Silence{}, common.Wrap("normalize silence", err)
	}
	created, err := s.repo.Create(ctx, silence)
	if err != nil {
		return domain.Silence{}, common.Wrap("create silence", err)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, silence domain.Silence) (domain.Silence, error) {
	silence.UpdatedAt = common.Now()
	updated, err := s.repo.Update(ctx, silence)
	if err != nil {
		return domain.Silence{}, common.Wrap("update silence", err)
	}
	return updated, nil
}

func (s *Service) Get(ctx context.Context, tenant, id string) (domain.Silence, error) {
	silence, err := s.repo.Get(ctx, tenant, id)
	if err != nil {
		return domain.Silence{}, common.Wrap("get silence", err)
	}
	return silence, nil
}

func (s *Service) List(ctx context.Context, tenant string, limit, offset int) (common.PageResult[domain.Silence], error) {
	items, total, err := s.repo.List(ctx, tenant, limit, offset)
	if err != nil {
		return common.PageResult[domain.Silence]{}, common.Wrap("list silences", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}

func (s *Service) IsSilenced(ctx context.Context, tenant, ruleID, scope string, labels common.Labels, at time.Time) (bool, error) {
	silences, err := s.repo.ListActive(ctx, tenant, at)
	if err != nil {
		return false, common.Wrap("list active silences", err)
	}
	for _, silence := range silences {
		if err := ctx.Err(); err != nil {
			return false, common.Wrap("check silence", err)
		}
		if silence.Matches(ruleID, scope, labels, at) {
			s.logger.Info("notification suppressed by silence", "silence_id", silence.ID, "rule_id", ruleID)
			return true, nil
		}
	}
	return false, nil
}
