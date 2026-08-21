package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/queue/domain"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Enqueue(ctx context.Context, task domain.Task) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO queue_items
			(id, topic, payload, status, attempts, max_attempts, available_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, task.ID, task.Topic, string(task.Payload), string(task.Status), task.Attempts, task.MaxAttempts, task.AvailableAt, task.CreatedAt)
	return common.Wrap("queue enqueue", err)
}

func (r *PGRepository) PollByID(ctx context.Context, id string) (domain.Task, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, topic, payload, status, attempts, max_attempts, available_at, created_at,
		       COALESCE(locked_by,''), COALESCE(locked_until, to_timestamp(0)), COALESCE(last_error,'')
		FROM queue_items WHERE id=$1
	`, id)
	return scanTask(row)
}

func (r *PGRepository) Poll(ctx context.Context, topic string, limit int, now time.Time, lockTTL time.Duration, owner string) ([]domain.Task, error) {
	if limit <= 0 {
		limit = 10
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, common.Wrap("queue poll begin", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT id, topic, payload, status, attempts, max_attempts, available_at, created_at,
		       COALESCE(locked_by,''), COALESCE(locked_until, to_timestamp(0)), COALESCE(last_error,'')
		FROM queue_items
		WHERE topic=$1 AND status='pending' AND available_at <= $2
		  AND (locked_until IS NULL OR locked_until <= $2)
		ORDER BY available_at, created_at
		FOR UPDATE SKIP LOCKED
		LIMIT $3
	`, topic, now, limit)
	if err != nil {
		return nil, common.Wrap("queue poll select", err)
	}
	tasks := make([]domain.Task, 0, limit)
	ids := make([]string, 0, limit)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		tasks = append(tasks, task)
		ids = append(ids, task.ID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, common.Wrap("queue poll rows", err)
	}
	if len(ids) > 0 {
		lockedUntil := now.Add(lockTTL)
		_, err = tx.Exec(ctx, `UPDATE queue_items SET status='running', locked_by=$1, locked_until=$2 WHERE id = ANY($3)`,
			owner, lockedUntil, ids)
		if err != nil {
			return nil, common.Wrap("queue poll lock", err)
		}
		for i := range tasks {
			tasks[i].Status = domain.TaskRunning
			tasks[i].LockedBy = owner
			tasks[i].LockedUntil = lockedUntil
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, common.Wrap("queue poll commit", err)
	}
	return tasks, nil
}

func (r *PGRepository) Complete(ctx context.Context, id, owner string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE queue_items SET status='completed', locked_by='', locked_until=NULL
		WHERE id=$1 AND locked_by=$2
	`, id, owner)
	if err != nil {
		return common.Wrap("queue complete", err)
	}
	if tag.RowsAffected() == 0 {
		return common.ErrNotFound
	}
	return nil
}

func (r *PGRepository) Fail(ctx context.Context, id, owner string, cause error, retryDelay time.Duration) error {
	errText := ""
	if cause != nil {
		errText = cause.Error()
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE queue_items
		SET attempts=attempts+1,
		    status=CASE WHEN attempts+1 >= max_attempts THEN 'failed' ELSE 'pending' END,
		    available_at=$3,
		    locked_by='',
		    locked_until=NULL,
		    last_error=$4
		WHERE id=$1 AND locked_by=$2
	`, id, owner, time.Now().UTC().Add(retryDelay), errText)
	if err != nil {
		return common.Wrap("queue fail", err)
	}
	if tag.RowsAffected() == 0 {
		return common.ErrNotFound
	}
	return nil
}

func (r *PGRepository) Requeue(ctx context.Context, id, owner string, delay time.Duration) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE queue_items
		SET status='pending', available_at=$3, locked_by='', locked_until=NULL
		WHERE id=$1 AND locked_by=$2
	`, id, owner, time.Now().UTC().Add(delay))
	if err != nil {
		return common.Wrap("queue requeue", err)
	}
	if tag.RowsAffected() == 0 {
		return common.ErrNotFound
	}
	return nil
}

func (r *PGRepository) CountPending(ctx context.Context, topic string) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM queue_items WHERE topic=$1 AND status='pending'`, topic).Scan(&n); err != nil {
		return 0, common.Wrap("queue count", err)
	}
	return n, nil
}

func scanTask(row pgx.Row) (domain.Task, error) {
	var task domain.Task
	var payload []byte
	var lockedUntil time.Time
	if err := row.Scan(&task.ID, &task.Topic, &payload, &task.Status, &task.Attempts, &task.MaxAttempts,
		&task.AvailableAt, &task.CreatedAt, &task.LockedBy, &lockedUntil, &task.LastError); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, common.ErrNotFound
		}
		return domain.Task{}, common.Wrap("scan queue task", err)
	}
	task.LockedUntil = lockedUntil
	task.Payload = append(json.RawMessage(nil), payload...)
	return task, nil
}

func (r *PGRepository) String() string {
	return fmt.Sprintf("postgres queue repository")
}
