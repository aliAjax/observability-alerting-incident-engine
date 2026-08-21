package application

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/observability-alerting/engine/internal/audit/domain"
	"github.com/observability-alerting/engine/internal/common"
)

type Service struct {
	repo   domain.Repository
	logger *slog.Logger
}

func NewService(repo domain.Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) Record(ctx context.Context, tenant, entity, entityID, action, actor string, detail any) error {
	raw, _ := json.Marshal(detail)
	record := domain.Record{
		Tenant:     tenant,
		Entity:     entity,
		EntityID:   entityID,
		Action:     action,
		Actor:      actor,
		Detail:     raw,
		OccurredAt: common.Now(),
		TraceID:    common.TraceIDFrom(ctx),
	}
	record.Normalize(record.OccurredAt)
	if err := s.repo.Insert(ctx, record); err != nil {
		return common.Wrap("insert audit record", err)
	}
	return nil
}

func (s *Service) List(ctx context.Context, tenant, entity, entityID string, from, to time.Time, limit, offset int) (common.PageResult[domain.Record], error) {
	items, total, err := s.repo.List(ctx, tenant, entity, entityID, from, to, limit, offset)
	if err != nil {
		return common.PageResult[domain.Record]{}, common.Wrap("list audit records", err)
	}
	return common.NewPageResult(items, total, limit, offset), nil
}
