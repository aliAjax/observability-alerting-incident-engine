package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/notification/domain"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) CreateChannel(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notification_channels (id, tenant, name, type, config, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, channel.ID, channel.Tenant, channel.Name, string(channel.Type), string(channel.Config), channel.Enabled, channel.CreatedAt, channel.UpdatedAt)
	if err != nil {
		return domain.Channel{}, common.Wrap("create notification channel", err)
	}
	return channel, nil
}

func (r *PGRepository) UpdateChannel(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notification_channels SET name=$3, type=$4, config=$5, enabled=$6, updated_at=$7
		WHERE id=$1 AND tenant=$2
	`, channel.ID, channel.Tenant, channel.Name, string(channel.Type), string(channel.Config), channel.Enabled, channel.UpdatedAt)
	if err != nil {
		return domain.Channel{}, common.Wrap("update notification channel", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Channel{}, common.ErrNotFound
	}
	return channel, nil
}

func (r *PGRepository) GetChannel(ctx context.Context, tenant, id string) (domain.Channel, error) {
	row := r.pool.QueryRow(ctx, selectChannels+` WHERE id=$1 AND tenant=$2`, id, tenant)
	return scanChannel(row)
}

func (r *PGRepository) ListChannels(ctx context.Context, tenant string, limit, offset int) ([]domain.Channel, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notification_channels WHERE tenant=$1`, tenant).Scan(&total); err != nil {
		return nil, 0, common.Wrap("notification channel count", err)
	}
	rows, err := r.pool.Query(ctx, selectChannels+` WHERE tenant=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, tenant, limit, offset)
	if err != nil {
		return nil, 0, common.Wrap("notification channel list", err)
	}
	defer rows.Close()
	items := make([]domain.Channel, 0, limit)
	for rows.Next() {
		ch, err := scanChannel(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, ch)
	}
	return items, total, rows.Err()
}

func (r *PGRepository) CreateTemplate(ctx context.Context, template domain.Template) (domain.Template, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notification_templates (id, tenant, name, channel_type, subject, body, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, template.ID, template.Tenant, template.Name, string(template.ChannelType), template.Subject, template.Body, template.CreatedAt, template.UpdatedAt)
	if err != nil {
		return domain.Template{}, common.Wrap("create notification template", err)
	}
	return template, nil
}

func (r *PGRepository) GetTemplate(ctx context.Context, tenant, id string) (domain.Template, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, tenant, name, channel_type, subject, body, created_at, updated_at
		FROM notification_templates WHERE id=$1 AND tenant=$2
	`, id, tenant)
	return scanTemplate(row)
}

func (r *PGRepository) CreateTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	receivers, _ := json.Marshal(task.Receivers)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notification_tasks
			(id, tenant, alert_id, rule_id, channel_id, template_id, receivers, payload, status, attempts,
			 max_attempts, available_at, last_error, last_attempt_at, cooldown_until, escalation_step, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
	`, task.ID, task.Tenant, task.AlertID, task.RuleID, task.ChannelID, task.TemplateID, string(receivers), string(task.Payload),
		string(task.Status), task.Attempts, task.MaxAttempts, task.AvailableAt, task.LastError, task.LastAttemptAt,
		task.CooldownUntil, task.EscalationStep, task.CreatedAt, task.UpdatedAt)
	if err != nil {
		return domain.Task{}, common.Wrap("create notification task", err)
	}
	return task, nil
}

func (r *PGRepository) PollTasks(ctx context.Context, limit int, now time.Time, lockTTL time.Duration, owner string) ([]domain.Task, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, common.Wrap("notification poll begin", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, selectTasks+`
		WHERE status='pending' AND available_at <= $1
		  AND (cooldown_until IS NULL OR cooldown_until <= $1)
		  AND (locked_until IS NULL OR locked_until <= $1)
		ORDER BY available_at, created_at
		FOR UPDATE SKIP LOCKED
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, common.Wrap("notification poll select", err)
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
		return nil, common.Wrap("notification poll rows", err)
	}
	if len(ids) > 0 {
		_, err = tx.Exec(ctx, `UPDATE notification_tasks SET status='running', locked_by=$1, locked_until=$2 WHERE id = ANY($3)`, owner, now.Add(lockTTL), ids)
		if err != nil {
			return nil, common.Wrap("notification poll lock", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, common.Wrap("notification poll commit", err)
	}
	for i := range tasks {
		tasks[i].Status = domain.TaskRunning
	}
	return tasks, nil
}

func (r *PGRepository) GetTask(ctx context.Context, id string) (domain.Task, error) {
	row := r.pool.QueryRow(ctx, selectTasks+` WHERE id=$1`, id)
	return scanTask(row)
}

func (r *PGRepository) CompleteTask(ctx context.Context, id, owner string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notification_tasks SET status='completed', locked_by='', locked_until=NULL, updated_at=$3
		WHERE id=$1 AND locked_by=$2
	`, id, owner, common.Now())
	if err != nil {
		return common.Wrap("complete notification task", err)
	}
	if tag.RowsAffected() == 0 {
		return common.ErrNotFound
	}
	return nil
}

func (r *PGRepository) FailTask(ctx context.Context, id, owner, cause string, retryDelay time.Duration, escalationStep int) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notification_tasks
		SET attempts=attempts+1,
		    status=CASE WHEN attempts+1 >= max_attempts THEN 'failed' ELSE 'pending' END,
		    available_at=$4,
		    last_error=$3,
		    last_attempt_at=$5,
		    escalation_step=$6,
		    locked_by='',
		    locked_until=NULL,
		    updated_at=$5
		WHERE id=$1 AND locked_by=$2
	`, id, owner, cause, common.Now().Add(retryDelay), common.Now(), escalationStep)
	if err != nil {
		return common.Wrap("fail notification task", err)
	}
	if tag.RowsAffected() == 0 {
		return common.ErrNotFound
	}
	return nil
}

func (r *PGRepository) RequeueTask(ctx context.Context, id, owner string, delay time.Duration) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notification_tasks
		SET status='pending', available_at=$3, locked_by='', locked_until=NULL, updated_at=$4
		WHERE id=$1 AND locked_by=$2
	`, id, owner, common.Now().Add(delay), common.Now())
	if err != nil {
		return common.Wrap("requeue notification task", err)
	}
	if tag.RowsAffected() == 0 {
		return common.ErrNotFound
	}
	return nil
}

func (r *PGRepository) ListTasks(ctx context.Context, tenant string, status domain.TaskStatus, limit, offset int) ([]domain.Task, int, error) {
	where := []string{"tenant=$1"}
	args := []any{tenant}
	if status != "" {
		args = append(args, status)
		where = append(where, "status=$"+itoa(len(args)))
	}
	args = append(args, limit, offset)
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM notification_tasks WHERE `+strings.Join(where, " AND "), args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, common.Wrap("notification task count", err)
	}
	rows, err := r.pool.Query(ctx, selectTasks+` WHERE `+strings.Join(where, " AND ")+` ORDER BY created_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, common.Wrap("notification task list", err)
	}
	defer rows.Close()
	items := make([]domain.Task, 0, limit)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, task)
	}
	return items, total, rows.Err()
}

const selectChannels = `SELECT id, tenant, name, type, config, enabled, created_at, updated_at FROM notification_channels`
const selectTasks = `
SELECT id, tenant, alert_id, rule_id, channel_id, COALESCE(template_id,''), receivers, payload, status, attempts,
       max_attempts, available_at, COALESCE(last_error,''), COALESCE(last_attempt_at, to_timestamp(0)),
       COALESCE(cooldown_until, to_timestamp(0)), escalation_step, created_at, updated_at
FROM notification_tasks`

func scanChannel(row pgx.Row) (domain.Channel, error) {
	var ch domain.Channel
	var config []byte
	if err := row.Scan(&ch.ID, &ch.Tenant, &ch.Name, &ch.Type, &config, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Channel{}, common.ErrNotFound
		}
		return domain.Channel{}, common.Wrap("scan notification channel", err)
	}
	ch.Config = append(json.RawMessage(nil), config...)
	return ch, nil
}

func scanTemplate(row pgx.Row) (domain.Template, error) {
	var t domain.Template
	if err := row.Scan(&t.ID, &t.Tenant, &t.Name, &t.ChannelType, &t.Subject, &t.Body, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Template{}, common.ErrNotFound
		}
		return domain.Template{}, common.Wrap("scan notification template", err)
	}
	return t, nil
}

func scanTask(row pgx.Row) (domain.Task, error) {
	var task domain.Task
	var receivers, payload []byte
	if err := row.Scan(&task.ID, &task.Tenant, &task.AlertID, &task.RuleID, &task.ChannelID, &task.TemplateID, &receivers, &payload,
		&task.Status, &task.Attempts, &task.MaxAttempts, &task.AvailableAt, &task.LastError, &task.LastAttemptAt,
		&task.CooldownUntil, &task.EscalationStep, &task.CreatedAt, &task.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Task{}, common.ErrNotFound
		}
		return domain.Task{}, common.Wrap("scan notification task", err)
	}
	_ = json.Unmarshal(receivers, &task.Receivers)
	task.Payload = append(json.RawMessage(nil), payload...)
	return task, nil
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
