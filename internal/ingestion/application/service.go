package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/ingestion/domain"
	"github.com/observability-alerting/engine/internal/queue/application"
)

type Service struct {
	repo         domain.Repository
	queue        *application.QueueService
	logger       *slog.Logger
	sampleEvery  time.Duration
	dedupeWindow time.Duration
}

func NewService(repo domain.Repository, queue *application.QueueService, logger *slog.Logger, sampleEvery, dedupeWindow time.Duration) *Service {
	if sampleEvery < 0 {
		sampleEvery = 0
	}
	if dedupeWindow <= 0 {
		dedupeWindow = 5 * time.Minute
	}
	return &Service{repo: repo, queue: queue, logger: logger, sampleEvery: sampleEvery, dedupeWindow: dedupeWindow}
}

func (s *Service) IngestBatch(ctx context.Context, batch domain.Batch) (int, error) {
	now := common.Now()
	accepted := make([]domain.IngestionEvent, 0, len(batch.Events))
	for i := range batch.Events {
		evt := batch.Events[i]
		evt.Tenant = firstNonEmpty(evt.Tenant, batch.Tenant, "default")
		evt.Source = firstNonEmpty(evt.Source, batch.Source)
		evt.Fill(now)
		if err := domain.ValidateEvent(&evt); err != nil {
			s.logger.Warn("invalid ingestion event", "index", i, "error", err, "trace_id", evt.TraceID)
			continue
		}
		if s.shouldSample(evt) {
			evt.DedupeKey = evt.DedupeFingerprint()
			accepted = append(accepted, evt)
		}
	}
	if len(accepted) == 0 {
		return 0, nil
	}
	deduped, err := s.dedupe(ctx, accepted)
	if err != nil {
		return 0, common.Wrap("dedupe ingestion", err)
	}
	if err := s.repo.InsertBatch(ctx, deduped); err != nil {
		return 0, common.Wrap("persist ingestion", err)
	}
	for _, evt := range deduped {
		if err := s.queue.Enqueue(ctx, "evaluation", domain.Batch{Tenant: evt.Tenant, Source: evt.Source}); err != nil {
			s.logger.Error("enqueue evaluation failed", "trace_id", evt.TraceID, "error", err)
		}
	}
	s.logger.Info("ingested events", "accepted", len(accepted), "persisted", len(deduped))
	return len(deduped), nil
}

func (s *Service) Ingest(ctx context.Context, evt domain.IngestionEvent) (domain.IngestionEvent, error) {
	evt.Fill(common.Now())
	if err := domain.ValidateEvent(&evt); err != nil {
		return domain.IngestionEvent{}, common.Wrap("validate ingestion", err)
	}
	evt.DedupeKey = evt.DedupeFingerprint()
	exists, err := s.repo.ExistsDedupe(ctx, evt.DedupeKey, s.dedupeWindow)
	if err != nil {
		return domain.IngestionEvent{}, common.Wrap("check dedupe", err)
	}
	if exists {
		return evt, nil
	}
	if err := s.repo.Insert(ctx, evt); err != nil {
		return domain.IngestionEvent{}, common.Wrap("persist ingestion", err)
	}
	_ = s.queue.Enqueue(ctx, "evaluation", domain.Batch{Tenant: evt.Tenant, Source: evt.Source})
	return evt, nil
}

func (s *Service) shouldSample(evt domain.IngestionEvent) bool {
	// Sampling disabled: keep every event.
	if s.sampleEvery == 0 {
		return true
	}
	// Keep the first (even) slot of every sampling window, drop the odd one.
	// UnixNano and the duration are both in nanoseconds, so the slot index
	// advances by one for each elapsed sampling window.
	slot := evt.OccurredAt.UnixNano() / int64(s.sampleEvery)
	return slot%2 == 0
}

func (s *Service) dedupe(ctx context.Context, events []domain.IngestionEvent) ([]domain.IngestionEvent, error) {
	seen := make(map[string]struct{}, len(events))
	out := make([]domain.IngestionEvent, 0, len(events))
	var mu sync.Mutex
	var wg sync.WaitGroup
	errCh := make(chan error, len(events))
	for _, evt := range events {
		wg.Add(1)
		go func(e domain.IngestionEvent) {
			defer wg.Done()
			// Reserve the key under the lock so a concurrent duplicate stops
			// here; the map and the output slice are both only ever touched
			// while holding the lock, which keeps the dedupe map race-free.
			mu.Lock()
			if _, ok := seen[e.DedupeKey]; ok {
				mu.Unlock()
				return
			}
			seen[e.DedupeKey] = struct{}{}
			mu.Unlock()

			exists, err := s.repo.ExistsDedupe(ctx, e.DedupeKey, s.dedupeWindow)
			if err != nil {
				errCh <- err
				return
			}
			if exists {
				return
			}
			mu.Lock()
			out = append(out, e)
			mu.Unlock()
		}(evt)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		return nil, err
	}
	return out, nil
}

func (s *Service) Query(ctx context.Context, source string, eventType domain.EventType, from, to time.Time, limit int) ([]domain.IngestionEvent, error) {
	return s.repo.QueryBySource(ctx, source, eventType, from, to, limit)
}

func (s *Service) Aggregates(ctx context.Context, source string, eventType domain.EventType, from, to time.Time) ([]domain.Aggregate, error) {
	return s.repo.QueryAggregates(ctx, source, eventType, from, to)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (s *Service) String() string {
	return fmt.Sprintf("ingestion service sample=%s dedupe=%s", s.sampleEvery, s.dedupeWindow)
}
