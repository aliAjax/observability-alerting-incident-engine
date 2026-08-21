package adapter

import (
	"net/http"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/rule/application"
	"github.com/observability-alerting/engine/internal/rule/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var rule domain.Rule
	if err := common.DecodeJSON(r, &rule); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode rule", err))
		return
	}
	rule.Tenant = common.TenantFrom(r.Context())
	created, err := h.service.Create(r.Context(), rule)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteCreated(w, created)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := common.PathID(r)
	var patch domain.Rule
	if err := common.DecodeJSON(r, &patch); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode rule patch", err))
		return
	}
	patch.Tenant = common.TenantFrom(r.Context())
	updated, err := h.service.Update(r.Context(), id, patch)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, updated)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	rule, err := h.service.Get(r.Context(), common.TenantFrom(r.Context()), common.PathID(r))
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, rule)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	filters := map[string]string{
		"data_source": r.URL.Query().Get("data_source"),
		"rule_type":   r.URL.Query().Get("rule_type"),
		"mode":        r.URL.Query().Get("mode"),
	}
	result, err := h.service.List(r.Context(), common.TenantFrom(r.Context()), limit, offset, filters)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}

func (h *Handler) Pause(w http.ResponseWriter, r *http.Request) {
	rule, err := h.service.Pause(r.Context(), common.TenantFrom(r.Context()), common.PathID(r))
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, rule)
}

func (h *Handler) Enable(w http.ResponseWriter, r *http.Request) {
	rule, err := h.service.Enable(r.Context(), common.TenantFrom(r.Context()), common.PathID(r))
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, rule)
}

func (h *Handler) SetMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode domain.Mode `json:"mode"`
	}
	if err := common.DecodeJSON(r, &body); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode mode", err))
		return
	}
	rule, err := h.service.SetMode(r.Context(), common.TenantFrom(r.Context()), common.PathID(r), body.Mode)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, rule)
}
