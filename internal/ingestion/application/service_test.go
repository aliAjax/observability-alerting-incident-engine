package application

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/ingestion/domain"
	queueapplication "github.com/observability-alerting/engine/internal/queue/application"
	queuedomain "github.com/observability-alerting/engine/internal/queue/domain"
)

type fakeIngestionRepo struct {
	mu      sync.Mutex
	seen    map[string]bool
	inserts []domain.IngestionEvent
}

func newFakeIngestionRepo() *fakeIngestionRepo {
	return &fakeIngestionRepo{seen: map[string]bool{}}
}

func (f *fakeIngestionRepo) InsertBatch(_ context.Context, events []domain.IngestionEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inserts = append(f.inserts, events...)
	for _, event := range events {
		f.seen[event.DedupeKey] = true
	}
	return nil
}

func (f *fakeIngestionRepo) Insert(_ context.Context, _ domain.IngestionEvent) error {
	return nil
}

func (f *fakeIngestionRepo) ExistsDedupe(_ context.Context, key string, _ time.Duration) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.seen[key], nil
}

func (f *fakeIngestionRepo) QueryBySource(_ context.Context, _ string, _ domain.EventType, _, _ time.Time, _ int) ([]domain.IngestionEvent, error) {
	return nil, nil
}

func (f *fakeIngestionRepo) QueryAggregates(_ context.Context, _ string, _ domain.EventType, _, _ time.Time) ([]domain.Aggregate, error) {
	return nil, nil
}

type fakeQueueRepo struct{}

func (fakeQueueRepo) Enqueue(_ context.Context, _ queuedomain.Task) error { return nil }
func (fakeQueueRepo) PollByID(_ context.Context, _ string) (queuedomain.Task, error) {
	return queuedomain.Task{}, common.ErrNotFound
}
func (fakeQueueRepo) Poll(_ context.Context, _ string, _ int, _ time.Time, _ time.Duration, _ string) ([]queuedomain.Task, error) {
	return nil, nil
}
func (fakeQueueRepo) Complete(_ context.Context, _ string, _ string) error { return nil }
func (fakeQueueRepo) Fail(_ context.Context, _ string, _ string, _ error, _ time.Duration) error {
	return nil
}
func (fakeQueueRepo) Requeue(_ context.Context, _ string, _ string, _ time.Duration) error {
	return nil
}
func (fakeQueueRepo) CountPending(_ context.Context, _ string) (int, error) { return 0, nil }

func TestR001ConcurrentIngestKeepsAllKeys(t *testing.T) {
	repo := newFakeIngestionRepo()
	queue := queueapplication.NewQueueService(fakeQueueRepo{}, "worker", time.Minute)
	service := NewService(repo, queue, slog.New(slog.NewTextHandler(io.Discard, nil)), 0, time.Minute)

	events := make([]domain.IngestionEvent, 0, 120)
	for i := 0; i < 120; i++ {
		events = append(events, domain.IngestionEvent{
			Tenant:       "default",
			Source:       "api-gateway",
			Type:         domain.EventMetric,
			Labels:       common.Labels{"instance": "pod-" + strconv.Itoa(i)},
			NumericValue: float64(i + 1),
			OccurredAt:   time.Now().UTC(),
		})
	}

	mid := len(events) / 2
	start := make(chan struct{})
	results := make(chan int, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, part := range [][]domain.IngestionEvent{events[:mid], events[mid:]} {
		wg.Add(1)
		go func(batch []domain.IngestionEvent) {
			defer wg.Done()
			<-start
			got, err := service.IngestBatch(context.Background(), domain.Batch{Tenant: "default", Source: "api-gateway", Events: batch})
			if err != nil {
				errs <- err
				return
			}
			results <- got
		}(part)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatalf("IngestBatch returned error: %v", err)
	}
	total := 0
	for got := range results {
		total += got
	}
	if total != len(events) {
		t.Fatalf("expected %d deduplicated events, got %d", len(events), total)
	}
}


func TestR001FingerprintTenantField(t *testing.T) {
	first := domain.IngestionEvent{Tenant: "alpha", Source: "api", Type: domain.EventMetric, Labels: common.Labels{"a": "1"}}
	second := first
	second.Tenant = "beta"
	if first.DedupeFingerprint() == second.DedupeFingerprint() {
		t.Fatal("dedupe fingerprint must include tenant")
	}
}

func TestR001FingerprintNumericField(t *testing.T) {
	first := domain.IngestionEvent{Tenant: "default", Source: "api", Type: domain.EventMetric, Labels: common.Labels{"a": "1"}, NumericValue: 11}
	second := first
	second.NumericValue = 22
	if first.DedupeFingerprint() == second.DedupeFingerprint() {
		t.Fatal("dedupe fingerprint must include numeric value")
	}
}

func TestR001SamplingExpectedSlots(t *testing.T) {
	repo := newFakeIngestionRepo()
	queue := queueapplication.NewQueueService(fakeQueueRepo{}, "worker", time.Minute)
	service := NewService(repo, queue, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second, time.Minute)
	base := time.Unix(0, 0)

	if !service.shouldSample(domain.IngestionEvent{OccurredAt: base}) {
		t.Fatal("event in the first sampling slot should be accepted")
	}
	if service.shouldSample(domain.IngestionEvent{OccurredAt: base.Add(time.Second)}) {
		t.Fatal("event in the second sampling slot should be dropped")
	}
}

func TestR001SamplingDisabledKeepsAll(t *testing.T) {
	repo := newFakeIngestionRepo()
	queue := queueapplication.NewQueueService(fakeQueueRepo{}, "worker", time.Minute)
	service := NewService(repo, queue, slog.New(slog.NewTextHandler(io.Discard, nil)), 0, time.Minute)

	if !service.shouldSample(domain.IngestionEvent{OccurredAt: time.Now()}) {
		t.Fatal("sampling disabled should accept every event")
	}
}
