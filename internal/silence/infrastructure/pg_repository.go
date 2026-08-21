package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/silence/domain"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Create(ctx context.Context, silence domain.Silence) (domain.Silence, error) {
	matchers, _ := json.Marshal(silence.Matchers)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO silences (id, tenant, rule_id, scope, matchers, starts_at, ends_at, created_by, comment, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`, silence.ID, silence.Tenant, silence.RuleID, silence.Scope, string(matchers), silence.StartsAt, silence.EndsAt,
		silence.CreatedBy, silence.Comment, silence.Active, silence.CreatedAt, silence.UpdatedAt)
	if err != nil {
		return domain.Silence{}, common.Wrap("create silence", err)
	}
	return silence, nil
}

func (r *PGRepository) Update(ctx context.Context, silence domain.Silence) (domain.Silence, error) {
	matchers, _ := json.Marshal(silence.Matchers)
	tag, err := r.pool.Exec(ctx, `
		UPDATE silences SET rule_id=$3, scope=$4, matchers=$5, starts_at=$6, ends_at=$7, created_by=$8, comment=$9, active=$10, updated_at=$11
		WHERE id=$1 AND tenant=$2
	`, silence.ID, silence.Tenant, silence.RuleID, silence.Scope, string(matchers), silence.StartsAt, silence.EndsAt,
		silence.CreatedBy, silence.Comment, silence.Active, silence.UpdatedAt)
	if err != nil {
		return domain.Silence{}, common.Wrap("update silence", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Silence{}, common.ErrNotFound
	}
	return silence, nil
}

func (r *PGRepository) Get(ctx context.Context, tenant, id string) (domain.Silence, error) {
	row := r.pool.QueryRow(ctx, selectSilence+` WHERE id=$1 AND tenant=$2`, id, tenant)
	return scanSilence(row)
}

func (r *PGRepository) List(ctx context.Context, tenant string, limit, offset int) ([]domain.Silence, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM silences WHERE tenant=$1`, tenant).Scan(&total); err != nil {
		return nil, 0, common.Wrap("silence count", err)
	}
	rows, err := r.pool.Query(ctx, selectSilence+` WHERE tenant=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, tenant, limit, offset)
	if err != nil {
		return nil, 0, common.Wrap("silence list", err)
	}
	defer rows.Close()
	items := make([]domain.Silence, 0, limit)
	for rows.Next() {
		s, err := scanSilence(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, s)
	}
	return items, total, rows.Err()
}

func (r *PGRepository) ListActive(ctx context.Context, tenant string, at time.Time) ([]domain.Silence, error) {
	rows, err := r.pool.Query(ctx, selectSilence+` WHERE tenant=$1 AND active=true AND starts_at <= $2 AND ends_at > $2 ORDER BY starts_at`, tenant, at)
	if err != nil {
		return nil, common.Wrap("silence active list", err)
	}
	defer rows.Close()
	var items []domain.Silence
	for rows.Next() {
		s, err := scanSilence(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

const selectSilence = `
SELECT id, tenant, rule_id, scope, matchers, starts_at, ends_at, created_by, COALESCE(comment,''), active, created_at, updated_at
FROM silences`

func scanSilence(row pgx.Row) (domain.Silence, error) {
	var s domain.Silence
	var matchers []byte
	if err := row.Scan(&s.ID, &s.Tenant, &s.RuleID, &s.Scope, &matchers, &s.StartsAt, &s.EndsAt, &s.CreatedBy,
		&s.Comment, &s.Active, &s.CreatedAt, &s.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Silence{}, common.ErrNotFound
		}
		return domain.Silence{}, common.Wrap("scan silence", err)
	}
	_ = json.Unmarshal(matchers, &s.Matchers)
	if s.Matchers == nil {
		s.Matchers = common.Labels{}
	}
	return s, nil
}
