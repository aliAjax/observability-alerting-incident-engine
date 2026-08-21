package httpapi

import (
	"net/http"
	"time"

	"github.com/observability-alerting/engine/internal/alert/adapter"
	auditadapter "github.com/observability-alerting/engine/internal/audit/adapter"
	incidentadapter "github.com/observability-alerting/engine/internal/incident/adapter"
	ingestionadapter "github.com/observability-alerting/engine/internal/ingestion/adapter"
	notificationadapter "github.com/observability-alerting/engine/internal/notification/adapter"
	ruleadapter "github.com/observability-alerting/engine/internal/rule/adapter"
	scheduleadapter "github.com/observability-alerting/engine/internal/schedule/adapter"
	silenceadapter "github.com/observability-alerting/engine/internal/silence/adapter"
)

type Dependencies struct {
	Ingestion    *ingestionadapter.Handler
	Rule         *ruleadapter.Handler
	Alert        *adapter.Handler
	Notification *notificationadapter.Handler
	Schedule     *scheduleadapter.Handler
	Silence      *silenceadapter.Handler
	Incident     *incidentadapter.Handler
	Audit        *auditadapter.Handler
	Evaluator    EvaluatorHandler
}

type EvaluatorHandler interface {
	Run(w http.ResponseWriter, r *http.Request)
}

func (d Dependencies) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", readyz)
	mux.HandleFunc("GET /metrics", metricsHandler)
	mux.HandleFunc("POST /api/v1/ingest", d.Ingestion.Ingest)
	mux.HandleFunc("POST /api/v1/ingest/batch", d.Ingestion.IngestBatch)
	mux.HandleFunc("GET /api/v1/events", d.Ingestion.Query)
	mux.HandleFunc("GET /api/v1/aggregates", d.Ingestion.Aggregates)
	mux.HandleFunc("POST /api/v1/rules", d.Rule.Create)
	mux.HandleFunc("GET /api/v1/rules", d.Rule.List)
	mux.HandleFunc("GET /api/v1/rules/{id}", d.Rule.Get)
	mux.HandleFunc("PUT /api/v1/rules/{id}", d.Rule.Update)
	mux.HandleFunc("POST /api/v1/rules/{id}/pause", d.Rule.Pause)
	mux.HandleFunc("POST /api/v1/rules/{id}/enable", d.Rule.Enable)
	mux.HandleFunc("POST /api/v1/rules/{id}/mode", d.Rule.SetMode)
	mux.HandleFunc("GET /api/v1/alerts", d.Alert.List)
	mux.HandleFunc("GET /api/v1/alerts/counts", d.Alert.Counts)
	mux.HandleFunc("GET /api/v1/alerts/{id}", d.Alert.Get)
	mux.HandleFunc("POST /api/v1/alerts/{id}/acknowledge", d.Alert.Acknowledge)
	mux.HandleFunc("POST /api/v1/alerts/{id}/silence", d.Alert.Silence)
	mux.HandleFunc("POST /api/v1/alerts/{id}/resolve", d.Alert.Resolve)
	mux.HandleFunc("GET /api/v1/alerts/{id}/history", d.Alert.History)
	mux.HandleFunc("GET /api/v1/alerts/{id}/observations", d.Alert.Observations)
	mux.HandleFunc("POST /api/v1/notification/channels", d.Notification.CreateChannel)
	mux.HandleFunc("GET /api/v1/notification/channels", d.Notification.ListChannels)
	mux.HandleFunc("POST /api/v1/notification/templates", d.Notification.CreateTemplate)
	mux.HandleFunc("GET /api/v1/notification/tasks", d.Notification.ListTasks)
	mux.HandleFunc("POST /api/v1/schedules", d.Schedule.Create)
	mux.HandleFunc("GET /api/v1/schedules", d.Schedule.List)
	mux.HandleFunc("GET /api/v1/schedules/on-call", d.Schedule.OnCall)
	mux.HandleFunc("GET /api/v1/schedules/{id}", d.Schedule.Get)
	mux.HandleFunc("PUT /api/v1/schedules/{id}", d.Schedule.Update)
	mux.HandleFunc("POST /api/v1/silences", d.Silence.Create)
	mux.HandleFunc("GET /api/v1/silences", d.Silence.List)
	mux.HandleFunc("GET /api/v1/silences/{id}", d.Silence.Get)
	mux.HandleFunc("PUT /api/v1/silences/{id}", d.Silence.Update)
	mux.HandleFunc("POST /api/v1/incidents", d.Incident.Create)
	mux.HandleFunc("GET /api/v1/incidents", d.Incident.List)
	mux.HandleFunc("GET /api/v1/incidents/{id}", d.Incident.Get)
	mux.HandleFunc("POST /api/v1/incidents/{id}/assign", d.Incident.Assign)
	mux.HandleFunc("POST /api/v1/incidents/{id}/actions", d.Incident.Action)
	mux.HandleFunc("POST /api/v1/incidents/{id}/close", d.Incident.Close)
	mux.HandleFunc("POST /api/v1/incidents/{id}/escalate", d.Incident.Escalate)
	mux.HandleFunc("GET /api/v1/incidents/{id}/actions", d.Incident.Actions)
	mux.HandleFunc("GET /api/v1/audit", d.Audit.List)
	mux.HandleFunc("POST /api/v1/evaluate", d.Evaluator.Run)
	return mux
}

func healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "time": time.Now().UTC().Format(time.RFC3339)})
}

func readyz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func WriteOK(w http.ResponseWriter, value any) {
	writeJSON(w, http.StatusOK, value)
}

func WriteError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
