package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/ingestion/domain"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) InsertBatch(ctx context.Context, events []domain.IngestionEvent) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return common.Wrap("ingestion batch begin", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, evt := range events {
		if err := r.insertTx(ctx, tx, evt); err != nil {
			return err
		}
	}
	return common.Wrap("ingestion batch commit", tx.Commit(ctx))
}

func (r *PGRepository) Insert(ctx context.Context, event domain.IngestionEvent) error {
	return r.insertTx(ctx, r.pool, event)
}

type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (r *PGRepository) insertTx(ctx context.Context, exec execer, evt domain.IngestionEvent) error {
	labels, _ := json.Marshal(evt.Labels)
	_, err := exec.Exec(ctx, `
		INSERT INTO ingestion_events
			(id, tenant, source, event_type, labels, numeric_value, text_value, occurred_at, received_at, dedupe_key, trace_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (dedupe_key) DO NOTHING
	`, evt.ID, evt.Tenant, evt.Source, string(evt.Type), string(labels), evt.NumericValue, evt.TextValue, evt.OccurredAt, evt.ReceivedAt, evt.DedupeKey, evt.TraceID)
	return common.Wrap("ingestion insert", err)
}

func (r *PGRepository) ExistsDedupe(ctx context.Context, key string, within time.Duration) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM ingestion_events WHERE dedupe_key=$1 AND received_at >= $2
		)
	`, key, common.Now().Add(-within)).Scan(&exists)
	if err != nil {
		return false, common.Wrap("ingestion dedupe check", err)
	}
	return exists, nil
}

func (r *PGRepository) QueryBySource(ctx context.Context, source string, eventType domain.EventType, from, to time.Time, limit int) ([]domain.IngestionEvent, error) {
	rows, err := r.pool.Query(ctx, selectEvent+` WHERE source=$1 AND event_type=$2 AND occurred_at >= $3 AND occurred_at <= $4 ORDER BY occurred_at DESC LIMIT $5`, source, string(eventType), from, to, limit)
	if err != nil {
		return nil, common.Wrap("ingestion query", err)
	}
	defer rows.Close()
	items := make([]domain.IngestionEvent, 0, limit)
	for rows.Next() {
		evt, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, evt)
	}
	return items, rows.Err()
}

func (r *PGRepository) QueryAggregates(ctx context.Context, source string, eventType domain.EventType, from, to time.Time) ([]domain.Aggregate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT source, labels,
		       count(*), sum(numeric_value), min(numeric_value), max(numeric_value), avg(numeric_value),
		       (array_agg(numeric_value ORDER BY occurred_at DESC))[1]
		FROM ingestion_events
		WHERE source=$1 AND event_type=$2 AND occurred_at >= $3 AND occurred_at <= $4
		GROUP BY source, labels
	`, source, string(eventType), from, to)
	if err != nil {
		return nil, common.Wrap("ingestion aggregate query", err)
	}
	defer rows.Close()
	var items []domain.Aggregate
	for rows.Next() {
		var agg domain.Aggregate
		var labels []byte
		if err := rows.Scan(&agg.Source, &labels, &agg.Count, &agg.Sum, &agg.Min, &agg.Max, &agg.Avg, &agg.Latest); err != nil {
			return nil, common.Wrap("scan ingestion aggregate", err)
		}
		_ = json.Unmarshal(labels, &agg.Labels)
		items = append(items, agg)
	}
	return items, rows.Err()
}

const selectEvent = `
SELECT id, tenant, source, event_type, labels, numeric_value, text_value, occurred_at, received_at, COALESCE(dedupe_key,''), COALESCE(trace_id,'')
FROM ingestion_events`

func scanEvent(row pgx.Row) (domain.IngestionEvent, error) {
	var evt domain.IngestionEvent
	var labels []byte
	if err := row.Scan(&evt.ID, &evt.Tenant, &evt.Source, &evt.Type, &labels, &evt.NumericValue, &evt.TextValue,
		&evt.OccurredAt, &evt.ReceivedAt, &evt.DedupeKey, &evt.TraceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.IngestionEvent{}, common.ErrNotFound
		}
		return domain.IngestionEvent{}, common.Wrap("scan ingestion event", err)
	}
	_ = json.Unmarshal(labels, &evt.Labels)
	if evt.Labels == nil {
		evt.Labels = common.Labels{}
	}
	return evt, nil
}
