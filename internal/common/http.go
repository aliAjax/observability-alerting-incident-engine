package common

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func WriteError(w http.ResponseWriter, status int, err error) {
	WriteJSON(w, status, Envelope{Error: err.Error()})
}

func WriteOK(w http.ResponseWriter, value any) {
	WriteJSON(w, http.StatusOK, Envelope{Data: value})
}

func WriteCreated(w http.ResponseWriter, value any) {
	WriteJSON(w, http.StatusCreated, Envelope{Data: value})
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func DecodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func PathID(r *http.Request) string {
	if id := r.PathValue("id"); id != "" {
		return id
	}
	id := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(id, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func QueryInt(r *http.Request, name string, fallback int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func QueryTime(r *http.Request, name string, fallback time.Time) time.Time {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return fallback
	}
	return t
}

func ErrorStatus(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, ErrAlreadyExists):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
