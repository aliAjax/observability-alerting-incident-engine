package adapter

import (
	"net/http"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/silence/application"
	"github.com/observability-alerting/engine/internal/silence/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var silence domain.Silence
	if err := common.DecodeJSON(r, &silence); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode silence", err))
		return
	}
	silence.Tenant = common.TenantFrom(r.Context())
	created, err := h.service.Create(r.Context(), silence)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteCreated(w, created)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var silence domain.Silence
	if err := common.DecodeJSON(r, &silence); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode silence", err))
		return
	}
	silence.ID = common.PathID(r)
	silence.Tenant = common.TenantFrom(r.Context())
	updated, err := h.service.Update(r.Context(), silence)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, updated)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	silence, err := h.service.Get(r.Context(), common.TenantFrom(r.Context()), common.PathID(r))
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, silence)
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
