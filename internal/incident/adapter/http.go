package adapter

import (
	"net/http"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/incident/application"
	"github.com/observability-alerting/engine/internal/incident/domain"
)

type Handler struct {
	service *application.Service
}

func NewHandler(service *application.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var incident domain.Incident
	if err := common.DecodeJSON(r, &incident); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode incident", err))
		return
	}
	incident.Tenant = common.TenantFrom(r.Context())
	created, err := h.service.Create(r.Context(), incident)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteCreated(w, created)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	incident, err := h.service.Get(r.Context(), common.TenantFrom(r.Context()), common.PathID(r))
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, incident)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := domain.Status(r.URL.Query().Get("status"))
	assignee := r.URL.Query().Get("assignee")
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	result, err := h.service.List(r.Context(), common.TenantFrom(r.Context()), status, assignee, limit, offset)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}

func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Assignee string `json:"assignee"`
		Actor    string `json:"actor"`
	}
	_ = common.DecodeJSON(r, &body)
	incident, err := h.service.Assign(r.Context(), common.TenantFrom(r.Context()), common.PathID(r), body.Assignee, body.Actor)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, incident)
}

func (h *Handler) Action(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action  string `json:"action"`
		Actor   string `json:"actor"`
		Comment string `json:"comment"`
	}
	if err := common.DecodeJSON(r, &body); err != nil {
		common.WriteError(w, http.StatusBadRequest, common.Wrap("decode incident action", err))
		return
	}
	incident, err := h.service.Act(r.Context(), common.TenantFrom(r.Context()), common.PathID(r), body.Action, body.Actor, body.Comment)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, incident)
}

func (h *Handler) Close(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Actor  string `json:"actor"`
		Reason string `json:"reason"`
	}
	_ = common.DecodeJSON(r, &body)
	incident, err := h.service.Close(r.Context(), common.TenantFrom(r.Context()), common.PathID(r), body.Actor, body.Reason)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, incident)
}

func (h *Handler) Escalate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Actor  string `json:"actor"`
		Reason string `json:"reason"`
	}
	_ = common.DecodeJSON(r, &body)
	incident, err := h.service.Escalate(r.Context(), common.TenantFrom(r.Context()), common.PathID(r), body.Actor, body.Reason)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, incident)
}

func (h *Handler) Actions(w http.ResponseWriter, r *http.Request) {
	limit := common.QueryInt(r, "limit", 100)
	offset := common.QueryInt(r, "offset", 0)
	result, err := h.service.Actions(r.Context(), common.PathID(r), limit, offset)
	if err != nil {
		common.WriteError(w, common.ErrorStatus(err), err)
		return
	}
	common.WriteOK(w, result)
}
