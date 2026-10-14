package http_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"metrics-service/internal/models"
	"metrics-service/internal/repository"
	"metrics-service/internal/service"
	transport "metrics-service/internal/transport/http"

	"github.com/stretchr/testify/require"
)

type fakeSaver struct {
	err      error
	inserted int64
	saved    []models.Metric
	daily    []models.DailyMetric
	filter   models.DailyFilter
}

func (f *fakeSaver) Save(_ context.Context, m models.Metric) error {
	f.saved = append(f.saved, m)
	return f.err
}

func (f *fakeSaver) SaveBatch(_ context.Context, ms []models.Metric) (int64, error) {
	f.saved = append(f.saved, ms...)
	if f.err != nil {
		return 0, f.err
	}
	return f.inserted, nil
}

func (f *fakeSaver) GetDaily(_ context.Context, filter models.DailyFilter) ([]models.DailyMetric, error) {
	f.filter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.daily, nil
}

type okPinger struct{}

func (okPinger) Ping(context.Context) error { return nil }

const validBody = `{
	"client_id": 1,
	"app_id": 2,
	"requests": 200,
	"landings": 50,
	"api_register_total": 100,
	"api_register_scam": 10,
	"timestamp": "2026-10-14T12:00:00Z"
}`

func TestCreateMetric(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		saveErr    error
		wantStatus int
	}{
		{name: "ok", body: validBody, wantStatus: http.StatusCreated},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `{"foo": 1}`, wantStatus: http.StatusBadRequest},
		{name: "validation error", body: validBody, saveErr: fmt.Errorf("%w: bad", service.ErrValidation), wantStatus: http.StatusBadRequest},
		{name: "duplicate", body: validBody, saveErr: repository.ErrDuplicate, wantStatus: http.StatusConflict},
		{name: "internal error", body: validBody, saveErr: fmt.Errorf("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &fakeSaver{err: tt.saveErr}
			router := transport.NewRouter(transport.NewMetricHandler(saver), okPinger{})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
		})
	}
}

func TestCreateMetricsBatch(t *testing.T) {
	twoItems := "[" + validBody + "," + validBody + "]"

	tests := []struct {
		name       string
		body       string
		inserted   int64
		saveErr    error
		wantStatus int
		wantBody   string
		wantSaved  int
	}{
		{name: "all inserted", body: twoItems, inserted: 2, wantStatus: http.StatusOK, wantBody: `{"inserted":2,"skipped":0}`, wantSaved: 2},
		{name: "duplicates skipped", body: twoItems, inserted: 1, wantStatus: http.StatusOK, wantBody: `{"inserted":1,"skipped":1}`, wantSaved: 2},
		{name: "not an array", body: validBody, wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `[{"foo": 1}]`, wantStatus: http.StatusBadRequest},
		{name: "validation error", body: "[" + validBody + "]", saveErr: fmt.Errorf("%w: bad", service.ErrValidation), wantStatus: http.StatusBadRequest, wantSaved: 1},
		{name: "internal error", body: "[" + validBody + "]", saveErr: fmt.Errorf("boom"), wantStatus: http.StatusInternalServerError, wantSaved: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &fakeSaver{err: tt.saveErr, inserted: tt.inserted}
			router := transport.NewRouter(transport.NewMetricHandler(saver), okPinger{})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/batch", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			require.Len(t, saver.saved, tt.wantSaved)
			if tt.wantBody != "" {
				require.JSONEq(t, tt.wantBody, rec.Body.String())
			}
		})
	}
}

func TestDailyMetrics(t *testing.T) {
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		query      string
		daily      []models.DailyMetric
		saveErr    error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "ok",
			query:      "client_id=1&app_id=2&from=2026-10-01&to=2026-10-01",
			daily:      []models.DailyMetric{{Date: day, Requests: 10, Landings: 5, ApiRegisterTotal: 3, ApiRegisterScam: 1}},
			wantStatus: http.StatusOK,
			wantBody:   `{"items":[{"date":"2026-10-01","requests":10,"landings":5,"api_register_total":3,"api_register_scam":1}]}`,
		},
		{name: "missing client_id", query: "from=2026-10-01&to=2026-10-02", wantStatus: http.StatusBadRequest},
		{name: "bad app_id", query: "client_id=1&app_id=x&from=2026-10-01&to=2026-10-02", wantStatus: http.StatusBadRequest},
		{name: "bad date", query: "client_id=1&from=01.10.2026&to=2026-10-02", wantStatus: http.StatusBadRequest},
		{name: "missing to", query: "client_id=1&from=2026-10-01", wantStatus: http.StatusBadRequest},
		{name: "validation error", query: "client_id=1&from=2026-10-02&to=2026-10-01", saveErr: fmt.Errorf("%w: bad", service.ErrValidation), wantStatus: http.StatusBadRequest},
		{name: "internal error", query: "client_id=1&from=2026-10-01&to=2026-10-02", saveErr: fmt.Errorf("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &fakeSaver{err: tt.saveErr, daily: tt.daily}
			router := transport.NewRouter(transport.NewMetricHandler(saver), okPinger{})

			req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/daily?"+tt.query, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			if tt.wantBody != "" {
				require.JSONEq(t, tt.wantBody, rec.Body.String())
				require.EqualValues(t, 1, saver.filter.ClientID)
				require.NotNil(t, saver.filter.AppID)
				require.EqualValues(t, 2, *saver.filter.AppID)
			}
		})
	}
}
