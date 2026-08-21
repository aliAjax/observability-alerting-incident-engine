package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/schedule/domain"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Create(ctx context.Context, schedule domain.Schedule) (domain.Schedule, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return domain.Schedule{}, common.Wrap("schedule create begin", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO schedules (id, tenant, name, description, timezone, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, schedule.ID, schedule.Tenant, schedule.Name, schedule.Description, schedule.Timezone, schedule.Enabled, schedule.CreatedAt, schedule.UpdatedAt)
	if err != nil {
		return domain.Schedule{}, common.Wrap("schedule create", err)
	}
	if err := r.replaceShifts(ctx, tx, schedule); err != nil {
		return domain.Schedule{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Schedule{}, common.Wrap("schedule create commit", err)
	}
	return schedule, nil
}

func (r *PGRepository) Update(ctx context.Context, schedule domain.Schedule) (domain.Schedule, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return domain.Schedule{}, common.Wrap("schedule update begin", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE schedules SET name=$3, description=$4, timezone=$5, enabled=$6, updated_at=$7
		WHERE id=$1 AND tenant=$2
	`, schedule.ID, schedule.Tenant, schedule.Name, schedule.Description, schedule.Timezone, schedule.Enabled, schedule.UpdatedAt)
	if err != nil {
		return domain.Schedule{}, common.Wrap("schedule update", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Schedule{}, common.ErrNotFound
	}
	if err := r.replaceShifts(ctx, tx, schedule); err != nil {
		return domain.Schedule{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Schedule{}, common.Wrap("schedule update commit", err)
	}
	return schedule, nil
}

func (r *PGRepository) replaceShifts(ctx context.Context, tx pgx.Tx, schedule domain.Schedule) error {
	if _, err := tx.Exec(ctx, `DELETE FROM schedule_shifts WHERE schedule_id=$1`, schedule.ID); err != nil {
		return common.Wrap("delete schedule shifts", err)
	}
	for _, shift := range schedule.Shifts {
		if shift.ID == "" {
			shift.ID = common.NewID("shift")
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO schedule_shifts (id, schedule_id, assignee, starts_at, ends_at)
			VALUES ($1,$2,$3,$4,$5)
		`, shift.ID, schedule.ID, shift.Assignee, shift.StartsAt, shift.EndsAt); err != nil {
			return common.Wrap("insert schedule shift", err)
		}
	}
	return nil
}

func (r *PGRepository) Get(ctx context.Context, tenant, id string) (domain.Schedule, error) {
	row := r.pool.QueryRow(ctx, selectSchedule+` WHERE id=$1 AND tenant=$2`, id, tenant)
	schedule, err := scanSchedule(row)
	if err != nil {
		return domain.Schedule{}, common.Wrap("get schedule", err)
	}
	return r.loadShifts(ctx, schedule)
}

func (r *PGRepository) List(ctx context.Context, tenant string, limit, offset int) ([]domain.Schedule, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM schedules WHERE tenant=$1`, tenant).Scan(&total); err != nil {
		return nil, 0, common.Wrap("schedule count", err)
	}
	rows, err := r.pool.Query(ctx, selectSchedule+` WHERE tenant=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, tenant, limit, offset)
	if err != nil {
		return nil, 0, common.Wrap("schedule list", err)
	}
	defer rows.Close()
	items := make([]domain.Schedule, 0, limit)
	for rows.Next() {
		schedule, err := scanSchedule(rows)
		if err != nil {
			return nil, 0, err
		}
		schedule, err = r.loadShifts(ctx, schedule)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, schedule)
	}
	return items, total, rows.Err()
}

func (r *PGRepository) FindActive(ctx context.Context, tenant string, at time.Time) ([]domain.Schedule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT s.id, s.tenant, s.name, s.description, s.timezone, s.enabled, s.created_at, s.updated_at
		FROM schedules s
		JOIN schedule_shifts sh ON sh.schedule_id = s.id
		WHERE s.tenant=$1 AND s.enabled=true AND sh.starts_at <= $2 AND sh.ends_at > $2
	`, tenant, at)
	if err != nil {
		return nil, common.Wrap("find active schedules", err)
	}
	defer rows.Close()
	var items []domain.Schedule
	for rows.Next() {
		schedule, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedule, err = r.loadShifts(ctx, schedule)
		if err != nil {
			return nil, err
		}
		items = append(items, schedule)
	}
	return items, rows.Err()
}

func (r *PGRepository) loadShifts(ctx context.Context, schedule domain.Schedule) (domain.Schedule, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, schedule_id, assignee, starts_at, ends_at FROM schedule_shifts WHERE schedule_id=$1 ORDER BY starts_at`, schedule.ID)
	if err != nil {
		return domain.Schedule{}, common.Wrap("load schedule shifts", err)
	}
	defer rows.Close()
	schedule.Shifts = make([]domain.Shift, 0)
	for rows.Next() {
		var shift domain.Shift
		if err := rows.Scan(&shift.ID, &shift.ScheduleID, &shift.Assignee, &shift.StartsAt, &shift.EndsAt); err != nil {
			return domain.Schedule{}, common.Wrap("scan schedule shift", err)
		}
		schedule.Shifts = append(schedule.Shifts, shift)
	}
	return schedule, rows.Err()
}

const selectSchedule = `SELECT id, tenant, name, COALESCE(description,''), timezone, enabled, created_at, updated_at FROM schedules`

func scanSchedule(row pgx.Row) (domain.Schedule, error) {
	var s domain.Schedule
	if err := row.Scan(&s.ID, &s.Tenant, &s.Name, &s.Description, &s.Timezone, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Schedule{}, common.ErrNotFound
		}
		return domain.Schedule{}, common.Wrap("scan schedule", err)
	}
	return s, nil
}
