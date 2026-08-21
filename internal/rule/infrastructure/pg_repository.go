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
	"github.com/observability-alerting/engine/internal/rule/domain"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Create(ctx context.Context, rule domain.Rule) (domain.Rule, error) {
	condition, _ := json.Marshal(rule.Condition)
	labels, _ := json.Marshal(rule.Labels)
	groupBy := toJSONStringArray(rule.GroupBy)
	suppressBy := toJSONStringArray(rule.SuppressBy)
	channels := toJSONStringArray(rule.Channels)
	escalation, _ := json.Marshal(rule.Escalation)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO rules
			(id, tenant, name, description, data_source, event_type, rule_type, condition,
			 severity, enabled, mode, window_seconds, interval_seconds, labels,
			 group_by, suppress_by, channels, escalation, version, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
	`, rule.ID, rule.Tenant, rule.Name, rule.Description, rule.DataSource, rule.EventType, string(rule.Type),
		string(condition), rule.Severity, rule.Enabled, string(rule.Mode), int64(rule.Window.Seconds()), int64(rule.Interval.Seconds()),
		string(labels), string(groupBy), string(suppressBy), string(channels), string(escalation), rule.Version, rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return domain.Rule{}, common.Wrap("rule create", err)
	}
	return rule, nil
}

func (r *PGRepository) Update(ctx context.Context, rule domain.Rule) (domain.Rule, error) {
	condition, _ := json.Marshal(rule.Condition)
	labels, _ := json.Marshal(rule.Labels)
	groupBy := toJSONStringArray(rule.GroupBy)
	suppressBy := toJSONStringArray(rule.SuppressBy)
	channels := toJSONStringArray(rule.Channels)
	escalation, _ := json.Marshal(rule.Escalation)
	tag, err := r.pool.Exec(ctx, `
		UPDATE rules SET
			name=$3, description=$4, data_source=$5, event_type=$6, rule_type=$7, condition=$8,
			severity=$9, enabled=$10, mode=$11, window_seconds=$12, interval_seconds=$13, labels=$14,
			group_by=$15, suppress_by=$16, channels=$17, escalation=$18, version=$19, updated_at=$20
		WHERE id=$1 AND tenant=$2
	`, rule.ID, rule.Tenant, rule.Name, rule.Description, rule.DataSource, rule.EventType, string(rule.Type),
		string(condition), rule.Severity, rule.Enabled, string(rule.Mode), int64(rule.Window.Seconds()), int64(rule.Interval.Seconds()),
		string(labels), string(groupBy), string(suppressBy), string(channels), string(escalation), rule.Version, rule.UpdatedAt)
	if err != nil {
		return domain.Rule{}, common.Wrap("rule update", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Rule{}, common.ErrNotFound
	}
	return rule, nil
}

func (r *PGRepository) Get(ctx context.Context, tenant, id string) (domain.Rule, error) {
	row := r.pool.QueryRow(ctx, selectRules+` WHERE id=$1 AND tenant=$2`, id, tenant)
	return scanRule(row)
}

func (r *PGRepository) List(ctx context.Context, tenant string, limit, offset int, filters map[string]string) ([]domain.Rule, int, error) {
	where := []string{"tenant=$1"}
	args := []any{tenant}
	if v := filters["data_source"]; v != "" {
		args = append(args, v)
		where = append(where, "data_source=$"+itoa(len(args)))
	}
	if v := filters["rule_type"]; v != "" {
		args = append(args, v)
		where = append(where, "rule_type=$"+itoa(len(args)))
	}
	if v := filters["mode"]; v != "" {
		args = append(args, v)
		where = append(where, "mode=$"+itoa(len(args)))
	}
	args = append(args, limit, offset)
	countArgs := args[:len(args)-2]
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM rules WHERE `+strings.Join(where, " AND "), countArgs...).Scan(&total); err != nil {
		return nil, 0, common.Wrap("rule count", err)
	}
	rows, err := r.pool.Query(ctx, selectRules+` WHERE `+strings.Join(where, " AND ")+` ORDER BY created_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, common.Wrap("rule list", err)
	}
	defer rows.Close()
	items := make([]domain.Rule, 0, limit)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, rule)
	}
	return items, total, rows.Err()
}

func (r *PGRepository) ListForEvaluation(ctx context.Context, now time.Time, limit int) ([]domain.Rule, error) {
	rows, err := r.pool.Query(ctx, selectRules+` WHERE enabled=true AND mode <> 'paused' ORDER BY updated_at LIMIT $1`, limit)
	if err != nil {
		return nil, common.Wrap("list rules for evaluation", err)
	}
	defer rows.Close()
	items := make([]domain.Rule, 0, limit)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, rule)
	}
	return items, rows.Err()
}

func (r *PGRepository) GetByName(ctx context.Context, tenant, name string) (domain.Rule, error) {
	row := r.pool.QueryRow(ctx, selectRules+` WHERE tenant=$1 AND name=$2 ORDER BY version DESC LIMIT 1`, tenant, name)
	return scanRule(row)
}

const selectRules = `
SELECT id, tenant, name, COALESCE(description,''), data_source, event_type, rule_type, condition,
       severity, enabled, mode, window_seconds, interval_seconds, labels,
       group_by, suppress_by, channels, escalation, version, created_at, updated_at
FROM rules`

func scanRule(row pgx.Row) (domain.Rule, error) {
	var r domain.Rule
	var condition, labels, groupBy, suppressBy, channels, escalation []byte
	var windowSeconds, intervalSeconds int64
	if err := row.Scan(&r.ID, &r.Tenant, &r.Name, &r.Description, &r.DataSource, &r.EventType, &r.Type, &condition,
		&r.Severity, &r.Enabled, &r.Mode, &windowSeconds, &intervalSeconds, &labels,
		&groupBy, &suppressBy, &channels, &escalation, &r.Version, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Rule{}, common.ErrNotFound
		}
		return domain.Rule{}, common.Wrap("scan rule", err)
	}
	_ = json.Unmarshal(condition, &r.Condition)
	_ = json.Unmarshal(labels, &r.Labels)
	r.GroupBy = fromJSONStringArray(groupBy)
	r.SuppressBy = fromJSONStringArray(suppressBy)
	r.Channels = fromJSONStringArray(channels)
	_ = json.Unmarshal(escalation, &r.Escalation)
	r.Window = time.Duration(windowSeconds) * time.Second
	r.Interval = time.Duration(intervalSeconds) * time.Second
	if r.Labels == nil {
		r.Labels = common.Labels{}
	}
	return r, nil
}

func toJSONStringArray(values []string) []byte {
	if values == nil {
		values = []string{}
	}
	b, _ := json.Marshal(values)
	return b
}

func fromJSONStringArray(data []byte) []string {
	var values []string
	_ = json.Unmarshal(data, &values)
	if values == nil {
		return []string{}
	}
	return values
}

func itoa(n int) string {
	return strconvItoa(n)
}

func strconvItoa(n int) string {
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
