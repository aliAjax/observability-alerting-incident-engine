package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/observability-alerting/engine/internal/audit/domain"
	"github.com/observability-alerting/engine/internal/common"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Insert(ctx context.Context, record domain.Record) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_records (id, tenant, entity, entity_id, action, actor, detail, occurred_at, trace_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, record.ID, record.Tenant, record.Entity, record.EntityID, record.Action, record.Actor, string(record.Detail), record.OccurredAt, record.TraceID)
	return common.Wrap("insert audit record", err)
}

func (r *PGRepository) List(ctx context.Context, tenant, entity, entityID string, from, to time.Time, limit, offset int) ([]domain.Record, int, error) {
	where := []string{"tenant=$1"}
	args := []any{tenant}
	if entity != "" {
		args = append(args, entity)
		where = append(where, "entity=$"+itoa(len(args)))
	}
	if entityID != "" {
		args = append(args, entityID)
		where = append(where, "entity_id=$"+itoa(len(args)))
	}
	args = append(args, from, to)
	where = append(where, "occurred_at >= $"+itoa(len(args)-1), "occurred_at <= $"+itoa(len(args)))
	args = append(args, limit, offset)
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM audit_records WHERE `+strings.Join(where, " AND "), args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, common.Wrap("audit count", err)
	}
	rows, err := r.pool.Query(ctx, selectAudit+` WHERE `+strings.Join(where, " AND ")+` ORDER BY occurred_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, common.Wrap("audit list", err)
	}
	defer rows.Close()
	items := make([]domain.Record, 0, limit)
	for rows.Next() {
		var rec domain.Record
		var detail []byte
		if err := rows.Scan(&rec.ID, &rec.Tenant, &rec.Entity, &rec.EntityID, &rec.Action, &rec.Actor, &detail, &rec.OccurredAt, &rec.TraceID); err != nil {
			return nil, 0, common.Wrap("scan audit", err)
		}
		rec.Detail = append(json.RawMessage(nil), detail...)
		items = append(items, rec)
	}
	return items, total, rows.Err()
}

const selectAudit = `SELECT id, tenant, entity, entity_id, action, actor, detail, occurred_at, COALESCE(trace_id,'') FROM audit_records`

func scanAuditRow(row pgx.Row) (domain.Record, error) {
	var rec domain.Record
	var detail []byte
	if err := row.Scan(&rec.ID, &rec.Tenant, &rec.Entity, &rec.EntityID, &rec.Action, &rec.Actor, &detail, &rec.OccurredAt, &rec.TraceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Record{}, common.ErrNotFound
		}
		return domain.Record{}, common.Wrap("scan audit row", err)
	}
	rec.Detail = append(json.RawMessage(nil), detail...)
	return rec, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
