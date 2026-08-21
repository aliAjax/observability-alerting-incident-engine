package adapter

import (
	"net/http"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/ingestion/application"
	"github.com/observability-alerting/engine/internal/ingestion/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	var evt domain.IngestionEvent
	if err := common.DecodeJSON(r, &evt); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode ingestion", err))
		return
	}
	evt.Tenant = common.TenantFrom(r.Context())
	created, err := h.service.Ingest(r.Context(), evt)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteCreated(w, created)
}

func (h *Handler) IngestBatch(w http.ResponseWriter, r *http.Request) {
	var batch domain.Batch
	if err := common.DecodeJSON(r, &batch); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode batch", err))
		return
	}
	batch.Tenant = common.TenantFrom(r.Context())
	n, err := h.service.IngestBatch(r.Context(), batch)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteCreated(w, map[string]int{"accepted": n})
}

func (h *Handler) Query(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	eventType := domain.EventType(r.URL.Query().Get("type"))
	from := common.QueryTime(r, "from", time.Now().Add(-time.Hour))
	to := common.QueryTime(r, "to", time.Now())
	limit := common.QueryInt(r, "limit", 100)
	items, err := h.service.Query(r.Context(), source, eventType, from, to, limit)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, items)
}

func (h *Handler) Aggregates(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	eventType := domain.EventType(r.URL.Query().Get("type"))
	from := common.QueryTime(r, "from", time.Now().Add(-time.Hour))
	to := common.QueryTime(r, "to", time.Now())
	items, err := h.service.Aggregates(r.Context(), source, eventType, from, to)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, items)
}
