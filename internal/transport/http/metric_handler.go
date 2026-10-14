package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"metrics-service/internal/models"
	"metrics-service/internal/repository"
	"metrics-service/internal/service"
)

const dateLayout = "2006-01-02"

type MetricService interface {
	Save(ctx context.Context, metric models.Metric) error
	SaveBatch(ctx context.Context, metrics []models.Metric) (int64, error)
	GetDaily(ctx context.Context, filter models.DailyFilter) ([]models.DailyMetric, error)
}

type MetricHandler struct {
	svc MetricService
}

func NewMetricHandler(svc MetricService) *MetricHandler {
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
		writeServiceError(w, err)
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

	inserted, err := h.svc.SaveBatch(r.Context(), metrics)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, batchResponse{
		Inserted: inserted,
		Skipped:  int64(len(metrics)) - inserted,
	})
}

type dailyMetricResponse struct {
	Date             string `json:"date"`
	Requests         int64  `json:"requests"`
	Landings         int64  `json:"landings"`
	ApiRegisterTotal int64  `json:"api_register_total"`
	ApiRegisterScam  int64  `json:"api_register_scam"`
}

type dailyResponse struct {
	Items []dailyMetricResponse `json:"items"`
}

func (h *MetricHandler) Daily(w http.ResponseWriter, r *http.Request) {
	filter, err := parseDailyFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	stats, err := h.svc.GetDaily(r.Context(), filter)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	resp := dailyResponse{Items: make([]dailyMetricResponse, len(stats))}
	for i, st := range stats {
		resp.Items[i] = dailyMetricResponse{
			Date:             st.Date.Format(dateLayout),
			Requests:         st.Requests,
			Landings:         st.Landings,
			ApiRegisterTotal: st.ApiRegisterTotal,
			ApiRegisterScam:  st.ApiRegisterScam,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func parseDailyFilter(r *http.Request) (models.DailyFilter, error) {
	q := r.URL.Query()
	var f models.DailyFilter

	clientID, err := parseInt32(q.Get("client_id"))
	if err != nil {
		return f, errors.New("client_id: " + err.Error())
	}
	f.ClientID = clientID

	if raw := q.Get("app_id"); raw != "" {
		appID, err := parseInt32(raw)
		if err != nil {
			return f, errors.New("app_id: " + err.Error())
		}
		f.AppID = &appID
	}

	if f.From, err = time.Parse(dateLayout, q.Get("from")); err != nil {
		return f, errors.New("from: expected date in YYYY-MM-DD format")
	}
	if f.To, err = time.Parse(dateLayout, q.Get("to")); err != nil {
		return f, errors.New("to: expected date in YYYY-MM-DD format")
	}

	return f, nil
}

func parseInt32(s string) (int32, error) {
	if s == "" {
		return 0, errors.New("is required")
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, errors.New("must be an integer")
	}
	return int32(v), nil
}

type batchResponse struct {
	Inserted int64 `json:"inserted"`
	Skipped  int64 `json:"skipped"`
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
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid json: unexpected data after top-level value")
		return false
	}

	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrDuplicate):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("metric service: %v", err)
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
