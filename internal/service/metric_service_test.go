package service_test

import (
	"context"
	"testing"
	"time"

	"metrics-service/internal/models"
	"metrics-service/internal/service"

	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	batches [][]models.Metric
}

func (f *fakeRepo) Insert(context.Context, models.Metric) error { return nil }

func (f *fakeRepo) InsertBatch(_ context.Context, ms []models.Metric) error {
	f.batches = append(f.batches, ms)
	return nil
}

func validMetric() models.Metric {
	return models.Metric{
		ClientID:         1,
		AppID:            1,
		Requests:         10,
		ApiRegisterTotal: 5,
		ApiRegisterScam:  1,
		Timestamp:        time.Now(),
	}
}

func TestSaveBatch(t *testing.T) {
	invalid := validMetric()
	invalid.ApiRegisterScam = 10

	tests := []struct {
		name    string
		metrics []models.Metric
		wantErr string
	}{
		{name: "ok", metrics: []models.Metric{validMetric(), validMetric()}},
		{name: "empty", metrics: nil, wantErr: "batch is empty"},
		{name: "too large", metrics: make([]models.Metric, service.MaxBatchSize+1), wantErr: "exceeds limit"},
		{name: "invalid item", metrics: []models.Metric{validMetric(), invalid}, wantErr: "metrics[1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			err := service.NewMetricService(repo).SaveBatch(context.Background(), tt.metrics)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Len(t, repo.batches, 1)
				return
			}

			require.ErrorIs(t, err, service.ErrValidation)
			require.ErrorContains(t, err, tt.wantErr)
			require.Empty(t, repo.batches)
		})
	}
}
