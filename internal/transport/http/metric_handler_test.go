package http_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metrics-service/internal/models"
	"metrics-service/internal/repository"
	"metrics-service/internal/service"
	transport "metrics-service/internal/transport/http"

	"github.com/stretchr/testify/require"
)

type fakeSaver struct {
	err   error
	saved []models.Metric
}

func (f *fakeSaver) Save(_ context.Context, m models.Metric) error {
	f.saved = append(f.saved, m)
	return f.err
}

func (f *fakeSaver) SaveBatch(_ context.Context, ms []models.Metric) error {
	f.saved = append(f.saved, ms...)
	return f.err
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
	tests := []struct {
		name       string
		body       string
		saveErr    error
		wantStatus int
		wantSaved  int
	}{
		{name: "ok", body: "[" + validBody + "," + validBody + "]", wantStatus: http.StatusCreated, wantSaved: 2},
		{name: "not an array", body: validBody, wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `[{"foo": 1}]`, wantStatus: http.StatusBadRequest},
		{name: "validation error", body: "[" + validBody + "]", saveErr: fmt.Errorf("%w: bad", service.ErrValidation), wantStatus: http.StatusBadRequest, wantSaved: 1},
		{name: "duplicate", body: "[" + validBody + "]", saveErr: repository.ErrDuplicate, wantStatus: http.StatusConflict, wantSaved: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saver := &fakeSaver{err: tt.saveErr}
			router := transport.NewRouter(transport.NewMetricHandler(saver), okPinger{})

			req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/batch", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, rec.Body.String())
			require.Len(t, saver.saved, tt.wantSaved)
		})
	}
}
