package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"metrics-service/internal/models"
	"metrics-service/internal/repository"
	"metrics-service/internal/service"
)

const maxBodyBytes = 1 << 20

type MetricSaver interface {
	Save(ctx context.Context, metric models.Metric) error
}

type MetricHandler struct {
	svc MetricSaver
}

func NewMetricHandler(svc MetricSaver) *MetricHandler {
	return &MetricHandler{svc: svc}
}

type createMetricRequest struct {
	ClientID         int32     `json:"client_id"`
	AppID            int32     `json:"app_id"`
	Requests         int32     `json:"requests"`
	Landings         int32     `json:"landings"`
	ApiRegisterTotal int32     `json:"api_register_total"`
	ApiRegisterScam  int32     `json:"api_register_scam"`
	Timestamp        time.Time `json:"timestamp"`
}

func (h *MetricHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req createMetricRequest
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}

	err := h.svc.Save(r.Context(), models.Metric{
		ClientID:         req.ClientID,
		AppID:            req.AppID,
		Requests:         req.Requests,
		Landings:         req.Landings,
		ApiRegisterTotal: req.ApiRegisterTotal,
		ApiRegisterScam:  req.ApiRegisterScam,
		Timestamp:        req.Timestamp,
	})

	switch {
	case err == nil:
		w.WriteHeader(http.StatusCreated)
	case errors.Is(err, service.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrDuplicate):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("save metric: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
