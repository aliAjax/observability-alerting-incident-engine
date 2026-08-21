package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/observability-alerting/engine/internal/alert/domain"
	"github.com/observability-alerting/engine/internal/common"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) UpsertAlert(ctx context.Context, alert domain.Alert, observation domain.Observation) (domain.Alert, bool, error) {
	labels, _ := json.Marshal(alert.Labels)
	var id string
	var inserted bool
	err := r.pool.QueryRow(ctx, `
		INSERT INTO alerts
			(id, tenant, rule_id, fingerprint, scope, labels, status, severity, message, current_value,
			 started_at, updated_at, last_fired_at, resolved_at, acknowledged_at, silenced_at,
			 fired_count, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NULL,NULL,NULL,1,1)
		ON CONFLICT (fingerprint) DO UPDATE SET
			current_value=EXCLUDED.current_value,
			message=EXCLUDED.message,
			updated_at=EXCLUDED.updated_at,
			last_fired_at=EXCLUDED.last_fired_at,
			status=CASE WHEN alerts.status IN ('resolved','silenced') THEN 'firing' ELSE alerts.status END,
			resolved_at=CASE WHEN alerts.status IN ('resolved','silenced') THEN NULL ELSE alerts.resolved_at END,
			fired_count=alerts.fired_count+1,
			version=alerts.version+1
		RETURNING id, (xmax = 0) AS inserted
	`, alert.ID, alert.Tenant, alert.RuleID, alert.Fingerprint, alert.Scope, string(labels), string(alert.Status),
		alert.Severity, alert.Message, alert.CurrentValue, alert.StartedAt, alert.UpdatedAt, alert.LastFiredAt).Scan(&id, &inserted)
	if err != nil {
		return domain.Alert{}, false, common.Wrap("upsert alert", err)
	}
	_ = id
	var updated domain.Alert
	var resolvedAt, acknowledgedAt, silencedAt *time.Time
	if err := r.pool.QueryRow(ctx, selectAlertByFingerprint, alert.Fingerprint).Scan(
		&updated.ID, &updated.Tenant, &updated.RuleID, &updated.Fingerprint, &updated.Scope, &labels,
		&updated.Status, &updated.Severity, &updated.Message, &updated.CurrentValue, &updated.StartedAt,
		&updated.UpdatedAt, &updated.LastFiredAt, &resolvedAt, &acknowledgedAt,
		&silencedAt, &updated.FiredCount, &updated.Version); err != nil {
		return domain.Alert{}, false, common.Wrap("scan upserted alert", err)
	}
	updated.ResolvedAt = nullTime(resolvedAt)
	updated.AcknowledgedAt = nullTime(acknowledgedAt)
	updated.SilencedAt = nullTime(silencedAt)
	_ = json.Unmarshal(labels, &updated.Labels)
	if updated.Labels == nil {
		updated.Labels = common.Labels{}
	}
	observation.AlertID = updated.ID
	if observation.ID == "" {
		observation.ID = common.NewID("obs")
	}
	if observation.ObservedAt.IsZero() {
		observation.ObservedAt = common.Now()
	}
	obsLabels, _ := json.Marshal(observation.Labels)
	_, err = r.pool.Exec(ctx, `
		INSERT INTO alert_observations
			(id, alert_id, observed_at, value, labels, source_event_id, sequence)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (id) DO NOTHING
	`, observation.ID, updated.ID, observation.ObservedAt, observation.Value, string(obsLabels), observation.SourceEventID, observation.Sequence)
	if err != nil {
		return domain.Alert{}, false, common.Wrap("insert observation", err)
	}
	if updated.Status == domain.StatusPending || (updated.Status == domain.StatusFiring && updated.FiredCount == 1) {
		_ = r.insertTransition(ctx, updated.ID, domain.StatusPending, domain.StatusFiring, "evaluation threshold matched", "", common.TraceIDFrom(ctx), common.Now())
	}
	return updated, inserted, nil
}

func (r *PGRepository) GetAlert(ctx context.Context, tenant, id string) (domain.Alert, error) {
	row := r.pool.QueryRow(ctx, selectAlertByID, id, tenant)
	return scanAlert(row)
}

func (r *PGRepository) GetAlertByFingerprint(ctx context.Context, fingerprint string) (domain.Alert, error) {
	row := r.pool.QueryRow(ctx, selectAlertByFingerprint, fingerprint)
	return scanAlert(row)
}

func (r *PGRepository) Transition(ctx context.Context, alertID string, from, to domain.Status, reason, actor, traceID string, occurredAt time.Time) error {
	if err := domain.ValidateTransition(from, to); err != nil {
		return common.Wrap("validate transition", err)
	}
	fields := map[domain.Status]string{
		domain.StatusFiring:       "resolved_at=NULL",
		domain.StatusResolved:     "resolved_at=$4",
		domain.StatusAcknowledged: "acknowledged_at=$4",
		domain.StatusSilenced:     "silenced_at=$4",
	}
	updates := "status=$3, updated_at=$4, version=version+1"
	if clause := fields[to]; clause != "" {
		updates += ", " + clause
	}
	tag, err := r.pool.Exec(ctx, `UPDATE alerts SET `+updates+` WHERE id=$1 AND status=$2`, alertID, from, to, occurredAt)
	if err != nil {
		return common.Wrap("transition alert", err)
	}
	if tag.RowsAffected() == 0 {
		return common.ErrConflict
	}
	return r.insertTransition(ctx, alertID, from, to, reason, actor, traceID, occurredAt)
}

func (r *PGRepository) insertTransition(ctx context.Context, alertID string, from, to domain.Status, reason, actor, traceID string, occurredAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO alert_transitions (id, alert_id, from_status, to_status, reason, actor, occurred_at, trace_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO NOTHING
	`, common.NewID("trx"), alertID, from, to, reason, actor, occurredAt, traceID)
	return common.Wrap("insert alert transition", err)
}

func (r *PGRepository) ListAlerts(ctx context.Context, tenant string, status domain.Status, ruleID string, limit, offset int) ([]domain.Alert, int, error) {
	where := []string{"tenant=$1"}
	args := []any{tenant}
	if status != "" {
		args = append(args, status)
		where = append(where, "status=$"+itoa(len(args)))
	}
	if ruleID != "" {
		args = append(args, ruleID)
		where = append(where, "rule_id=$"+itoa(len(args)))
	}
	args = append(args, limit, offset)
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM alerts WHERE `+strings.Join(where, " AND "), args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, common.Wrap("alert count", err)
	}
	rows, err := r.pool.Query(ctx, selectAlerts+` WHERE `+strings.Join(where, " AND ")+` ORDER BY updated_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, common.Wrap("alert list", err)
	}
	defer rows.Close()
	items := make([]domain.Alert, 0, limit)
	for rows.Next() {
		a, err := scanAlert(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, a)
	}
	return items, total, rows.Err()
}

func (r *PGRepository) ListTransitions(ctx context.Context, alertID string, from, to time.Time, limit, offset int) ([]domain.Transition, int, error) {
	where := []string{"alert_id=$1", "occurred_at >= $2", "occurred_at <= $3"}
	args := []any{alertID, from, to, limit, offset}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM alert_transitions WHERE `+strings.Join(where, " AND "), args[:3]...).Scan(&total); err != nil {
		return nil, 0, common.Wrap("transition count", err)
	}
	rows, err := r.pool.Query(ctx, selectTransitions+` WHERE `+strings.Join(where, " AND ")+` ORDER BY occurred_at DESC LIMIT $4 OFFSET $5`, args...)
	if err != nil {
		return nil, 0, common.Wrap("transition list", err)
	}
	defer rows.Close()
	items := make([]domain.Transition, 0, limit)
	for rows.Next() {
		var t domain.Transition
		if err := rows.Scan(&t.ID, &t.AlertID, &t.FromStatus, &t.ToStatus, &t.Reason, &t.Actor, &t.OccurredAt, &t.TraceID); err != nil {
			return nil, 0, common.Wrap("scan transition", err)
		}
		items = append(items, t)
	}
	return items, total, rows.Err()
}

func (r *PGRepository) ListObservations(ctx context.Context, alertID string, from, to time.Time, limit, offset int) ([]domain.Observation, int, error) {
	where := []string{"alert_id=$1", "observed_at >= $2", "observed_at <= $3"}
	args := []any{alertID, from, to, limit, offset}
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM alert_observations WHERE `+strings.Join(where, " AND "), args[:3]...).Scan(&total); err != nil {
		return nil, 0, common.Wrap("observation count", err)
	}
	rows, err := r.pool.Query(ctx, selectObservations+` WHERE `+strings.Join(where, " AND ")+` ORDER BY observed_at DESC LIMIT $4 OFFSET $5`, args...)
	if err != nil {
		return nil, 0, common.Wrap("observation list", err)
	}
	defer rows.Close()
	items := make([]domain.Observation, 0, limit)
	for rows.Next() {
		var o domain.Observation
		var labels []byte
		if err := rows.Scan(&o.ID, &o.AlertID, &o.ObservedAt, &o.Value, &labels, &o.SourceEventID, &o.Sequence); err != nil {
			return nil, 0, common.Wrap("scan observation", err)
		}
		_ = json.Unmarshal(labels, &o.Labels)
		items = append(items, o)
	}
	return items, total, rows.Err()
}

func (r *PGRepository) CountByStatus(ctx context.Context, tenant string) (map[domain.Status]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT status, count(*) FROM alerts WHERE tenant=$1 GROUP BY status`, tenant)
	if err != nil {
		return nil, common.Wrap("alert status count", err)
	}
	defer rows.Close()
	out := map[domain.Status]int{}
	for rows.Next() {
		var status domain.Status
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, common.Wrap("scan alert status count", err)
		}
		out[status] = n
	}
	return out, rows.Err()
}

func (r *PGRepository) TouchAlert(ctx context.Context, alertID string, value float64, message string, observedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE alerts SET current_value=$2, message=$3, updated_at=$4, last_fired_at=$4, fired_count=fired_count+1, version=version+1
		WHERE id=$1
	`, alertID, value, message, observedAt)
	return common.Wrap("touch alert", err)
}

const selectAlerts = `
SELECT id, tenant, rule_id, fingerprint, scope, labels, status, severity, message, current_value,
       started_at, updated_at, last_fired_at, resolved_at, acknowledged_at, silenced_at, fired_count, version
FROM alerts`

const selectAlertByID = selectAlerts + ` WHERE id=$1 AND tenant=$2`
const selectAlertByFingerprint = selectAlerts + ` WHERE fingerprint=$1`

const selectTransitions = `
SELECT id, alert_id, from_status, to_status, reason, actor, occurred_at, COALESCE(trace_id,'')
FROM alert_transitions`

const selectObservations = `
SELECT id, alert_id, observed_at, value, labels, COALESCE(source_event_id,''), sequence
FROM alert_observations`

func scanAlert(row pgx.Row) (domain.Alert, error) {
	var a domain.Alert
	var labels []byte
	var resolvedAt, acknowledgedAt, silencedAt *time.Time
	if err := row.Scan(&a.ID, &a.Tenant, &a.RuleID, &a.Fingerprint, &a.Scope, &labels, &a.Status, &a.Severity,
		&a.Message, &a.CurrentValue, &a.StartedAt, &a.UpdatedAt, &a.LastFiredAt, &resolvedAt,
		&acknowledgedAt, &silencedAt, &a.FiredCount, &a.Version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Alert{}, common.ErrNotFound
		}
		return domain.Alert{}, common.Wrap("scan alert", err)
	}
	a.ResolvedAt = nullTime(resolvedAt)
	a.AcknowledgedAt = nullTime(acknowledgedAt)
	a.SilencedAt = nullTime(silencedAt)
	_ = json.Unmarshal(labels, &a.Labels)
	if a.Labels == nil {
		a.Labels = common.Labels{}
	}
	return a, nil
}

func nullTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
