package adapter

import (
	"net/http"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/schedule/application"
	"github.com/observability-alerting/engine/internal/schedule/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var schedule domain.Schedule
	if err := common.DecodeJSON(r, &schedule); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode schedule", err))
		return
	}
	schedule.Tenant = common.TenantFrom(r.Context())
	created, err := h.service.Create(r.Context(), schedule)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteCreated(w, created)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var schedule domain.Schedule
	if err := common.DecodeJSON(r, &schedule); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode schedule", err))
		return
	}
	schedule.ID = common.PathID(r)
	schedule.Tenant = common.TenantFrom(r.Context())
	updated, err := h.service.Update(r.Context(), schedule)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, updated)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	schedule, err := h.service.Get(r.Context(), common.TenantFrom(r.Context()), common.PathID(r))
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, schedule)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	result, err := h.service.List(r.Context(), common.TenantFrom(r.Context()), limit, offset)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}

func (h *Handler) OnCall(w http.ResponseWriter, r *http.Request) {
	at := common.QueryTime(r, "at", time.Now())
	items, err := h.service.OnCall(r.Context(), common.TenantFrom(r.Context()), at)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, items)
}
