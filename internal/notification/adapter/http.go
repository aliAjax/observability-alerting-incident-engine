package adapter

import (
	"net/http"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/notification/application"
	"github.com/observability-alerting/engine/internal/notification/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var channel domain.Channel
	if err := common.DecodeJSON(r, &channel); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode channel", err))
		return
	}
	channel.Tenant = common.TenantFrom(r.Context())
	created, err := h.service.CreateChannel(r.Context(), channel)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteCreated(w, created)
}

func (h *Handler) ListChannels(w http.ResponseWriter, r *http.Request) {
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	result, err := h.service.ListChannels(r.Context(), common.TenantFrom(r.Context()), limit, offset)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}

func (h *Handler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var template domain.Template
	if err := common.DecodeJSON(r, &template); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode template", err))
		return
	}
	template.Tenant = common.TenantFrom(r.Context())
	created, err := h.service.CreateTemplate(r.Context(), template)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteCreated(w, created)
}

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	status := domain.TaskStatus(r.URL.Query().Get("status"))
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	result, err := h.service.ListTasks(r.Context(), common.TenantFrom(r.Context()), status, limit, offset)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}
