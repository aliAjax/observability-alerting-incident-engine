package adapter

import (
	"net/http"
	"time"

	"github.com/observability-alerting/engine/internal/audit/application"
	"github.com/observability-alerting/engine/internal/common"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	entity := r.URL.Query().Get("entity")
	entityID := r.URL.Query().Get("entity_id")
	from := common.QueryTime(r, "from", time.Now().Add(-24*time.Hour))
	to := common.QueryTime(r, "to", time.Now())
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	result, err := h.service.List(r.Context(), common.TenantFrom(r.Context()), entity, entityID, from, to, limit, offset)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}
