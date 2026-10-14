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
	SaveBatch(ctx context.Context, metrics []models.Metric) error
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
	var req createMetricRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if err := h.svc.Save(r.Context(), req.toModel()); err != nil {
		writeSaveError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *MetricHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req []createMetricRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	metrics := make([]models.Metric, len(req))
	for i, m := range req {
		metrics[i] = m.toModel()
	}

	if err := h.svc.SaveBatch(r.Context(), metrics); err != nil {
		writeSaveError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int{"inserted": len(metrics)})
}

func (req createMetricRequest) toModel() models.Metric {
	return models.Metric{
		ClientID:         req.ClientID,
		AppID:            req.AppID,
		Requests:         req.Requests,
		Landings:         req.Landings,
		ApiRegisterTotal: req.ApiRegisterTotal,
		ApiRegisterScam:  req.ApiRegisterScam,
		Timestamp:        req.Timestamp,
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}

	return true
}

func writeSaveError(w http.ResponseWriter, err error) {
	switch {
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
