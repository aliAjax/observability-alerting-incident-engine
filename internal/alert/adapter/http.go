package adapter

import (
	"net/http"
	"time"

	"github.com/observability-alerting/engine/internal/alert/application"
	"github.com/observability-alerting/engine/internal/alert/domain"
	"github.com/observability-alerting/engine/internal/common"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := domain.Status(r.URL.Query().Get("status"))
	ruleID := r.URL.Query().Get("rule_id")
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	result, err := h.service.List(r.Context(), common.TenantFrom(r.Context()), status, ruleID, limit, offset)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	alert, err := h.service.Get(r.Context(), common.TenantFrom(r.Context()), common.PathID(r))
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, alert)
}

func (h *Handler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Actor string `json:"actor"`
	}
	_ = common.DecodeJSON(r, &body)
	if body.Actor == "" {
		body.Actor = "api"
	}
	alert, err := h.service.Acknowledge(r.Context(), common.TenantFrom(r.Context()), common.PathID(r), body.Actor)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, alert)
}

func (h *Handler) Silence(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Actor string `json:"actor"`
	}
	_ = common.DecodeJSON(r, &body)
	if body.Actor == "" {
		body.Actor = "api"
	}
	alert, err := h.service.Silence(r.Context(), common.TenantFrom(r.Context()), common.PathID(r), body.Actor)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, alert)
}

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason string `json:"reason"`
	}
	_ = common.DecodeJSON(r, &body)
	alert, err := h.service.Resolve(r.Context(), common.TenantFrom(r.Context()), common.PathID(r), application.NormalizeReason(body.Reason))
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, alert)
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	from := common.QueryTime(r, "from", time.Now().Add(-24*time.Hour))
	to := common.QueryTime(r, "to", time.Now())
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	result, err := h.service.History(r.Context(), common.PathID(r), from, to, limit, offset)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}

func (h *Handler) Observations(w http.ResponseWriter, r *http.Request) {
	from := common.QueryTime(r, "from", time.Now().Add(-24*time.Hour))
	to := common.QueryTime(r, "to", time.Now())
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	result, err := h.service.Observations(r.Context(), common.PathID(r), from, to, limit, offset)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}

func (h *Handler) Counts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.service.Counts(r.Context(), common.TenantFrom(r.Context()))
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, counts)
}
