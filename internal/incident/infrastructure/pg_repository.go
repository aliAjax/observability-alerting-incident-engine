package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/incident/domain"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository {
	return &PGRepository{pool: pool}
}

func (r *PGRepository) Create(ctx context.Context, incident domain.Incident) (domain.Incident, error) {
	labels, _ := json.Marshal(incident.Labels)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO incidents
			(id, tenant, alert_id, title, description, status, severity, assignee, labels, opened_at, closed_at, updated_at, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, incident.ID, incident.Tenant, incident.AlertID, incident.Title, incident.Description, string(incident.Status),
		string(incident.Severity), incident.Assignee, string(labels), incident.OpenedAt, incident.ClosedAt, incident.UpdatedAt, incident.Version)
	if err != nil {
		return domain.Incident{}, common.Wrap("create incident", err)
	}
	return incident, nil
}

func (r *PGRepository) Update(ctx context.Context, incident domain.Incident) (domain.Incident, error) {
	labels, _ := json.Marshal(incident.Labels)
	tag, err := r.pool.Exec(ctx, `
		UPDATE incidents SET title=$3, description=$4, status=$5, severity=$6, assignee=$7, labels=$8, closed_at=$9, updated_at=$10, version=$11
		WHERE id=$1 AND tenant=$2
	`, incident.ID, incident.Tenant, incident.Title, incident.Description, string(incident.Status), string(incident.Severity),
		incident.Assignee, string(labels), incident.ClosedAt, incident.UpdatedAt, incident.Version)
	if err != nil {
		return domain.Incident{}, common.Wrap("update incident", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.Incident{}, common.ErrNotFound
	}
	return incident, nil
}

func (r *PGRepository) Get(ctx context.Context, tenant, id string) (domain.Incident, error) {
	row := r.pool.QueryRow(ctx, selectIncident+` WHERE id=$1 AND tenant=$2`, id, tenant)
	return scanIncident(row)
}

func (r *PGRepository) GetByAlert(ctx context.Context, tenant, alertID string) (domain.Incident, error) {
	row := r.pool.QueryRow(ctx, selectIncident+` WHERE tenant=$1 AND alert_id=$2 ORDER BY opened_at DESC LIMIT 1`, tenant, alertID)
	return scanIncident(row)
}

func (r *PGRepository) List(ctx context.Context, tenant string, status domain.Status, assignee string, limit, offset int) ([]domain.Incident, int, error) {
	where := []string{"tenant=$1"}
	args := []any{tenant}
	if status != "" {
		args = append(args, status)
		where = append(where, "status=$"+itoa(len(args)))
	}
	if assignee != "" {
		args = append(args, assignee)
		where = append(where, "assignee=$"+itoa(len(args)))
	}
	args = append(args, limit, offset)
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM incidents WHERE `+strings.Join(where, " AND "), args[:len(args)-2]...).Scan(&total); err != nil {
		return nil, 0, common.Wrap("incident count", err)
	}
	rows, err := r.pool.Query(ctx, selectIncident+` WHERE `+strings.Join(where, " AND ")+` ORDER BY opened_at DESC LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, 0, common.Wrap("incident list", err)
	}
	defer rows.Close()
	items := make([]domain.Incident, 0, limit)
	for rows.Next() {
		i, err := scanIncident(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, i)
	}
	return items, total, rows.Err()
}

func (r *PGRepository) AddAction(ctx context.Context, action domain.Action) error {
	action.Normalize(common.Now())
	_, err := r.pool.Exec(ctx, `
		INSERT INTO incident_actions (id, incident_id, action, actor, comment, occurred_at, trace_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`, action.ID, action.IncidentID, action.Action, action.Actor, action.Comment, action.OccurredAt, action.TraceID)
	return common.Wrap("insert incident action", err)
}

func (r *PGRepository) ListActions(ctx context.Context, incidentID string, limit, offset int) ([]domain.Action, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM incident_actions WHERE incident_id=$1`, incidentID).Scan(&total); err != nil {
		return nil, 0, common.Wrap("incident action count", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, incident_id, action, actor, COALESCE(comment,''), occurred_at, COALESCE(trace_id,'')
		FROM incident_actions WHERE incident_id=$1 ORDER BY occurred_at DESC LIMIT $2 OFFSET $3
	`, incidentID, limit, offset)
	if err != nil {
		return nil, 0, common.Wrap("incident action list", err)
	}
	defer rows.Close()
	items := make([]domain.Action, 0, limit)
	for rows.Next() {
		var a domain.Action
		if err := rows.Scan(&a.ID, &a.IncidentID, &a.Action, &a.Actor, &a.Comment, &a.OccurredAt, &a.TraceID); err != nil {
			return nil, 0, common.Wrap("scan incident action", err)
		}
		items = append(items, a)
	}
	return items, total, rows.Err()
}

const selectIncident = `
SELECT id, tenant, alert_id, title, COALESCE(description,''), status, severity, COALESCE(assignee,''), labels,
       opened_at, COALESCE(closed_at, to_timestamp(0)), updated_at, version
FROM incidents`

func scanIncident(row pgx.Row) (domain.Incident, error) {
	var i domain.Incident
	var labels []byte
	if err := row.Scan(&i.ID, &i.Tenant, &i.AlertID, &i.Title, &i.Description, &i.Status, &i.Severity, &i.Assignee,
		&labels, &i.OpenedAt, &i.ClosedAt, &i.UpdatedAt, &i.Version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Incident{}, common.ErrNotFound
		}
		return domain.Incident{}, common.Wrap("scan incident", err)
	}
	_ = json.Unmarshal(labels, &i.Labels)
	if i.Labels == nil {
		i.Labels = common.Labels{}
	}
	return i, nil
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
