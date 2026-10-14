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
	batches  [][]models.Metric
	daily    []models.DailyMetric
	from, to time.Time
	calls    int
}

func (f *fakeRepo) DailyStats(_ context.Context, _ int32, _ *int32, from, to time.Time) ([]models.DailyMetric, error) {
	f.calls++
	f.from, f.to = from, to
	return f.daily, nil
}

func (f *fakeRepo) Insert(context.Context, models.Metric) error { return nil }

func (f *fakeRepo) InsertBatch(_ context.Context, ms []models.Metric) (int64, error) {
	f.batches = append(f.batches, ms)
	return int64(len(ms)), nil
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
			inserted, err := service.NewMetricService(repo).SaveBatch(context.Background(), tt.metrics)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.EqualValues(t, len(tt.metrics), inserted)
				require.Len(t, repo.batches, 1)
				return
			}

			require.ErrorIs(t, err, service.ErrValidation)
			require.ErrorContains(t, err, tt.wantErr)
			require.Empty(t, repo.batches)
		})
	}
}

func date(day int) time.Time {
	return time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC)
}

func TestGetDailyFillsGaps(t *testing.T) {
	repo := &fakeRepo{daily: []models.DailyMetric{
		{Date: date(2), Requests: 10, ApiRegisterTotal: 3, ApiRegisterScam: 1},
	}}

	got, err := service.NewMetricService(repo).GetDaily(context.Background(), models.DailyFilter{
		ClientID: 1,
		From:     date(1),
		To:       date(3),
	})
	require.NoError(t, err)

	require.Equal(t, date(1), repo.from)
	require.Equal(t, date(4), repo.to)

	require.Equal(t, []models.DailyMetric{
		{Date: date(1)},
		{Date: date(2), Requests: 10, ApiRegisterTotal: 3, ApiRegisterScam: 1},
		{Date: date(3)},
	}, got)
}

func TestGetDailyValidation(t *testing.T) {
	zero := int32(0)

	tests := []struct {
		name    string
		filter  models.DailyFilter
		wantErr string
	}{
		{name: "no client", filter: models.DailyFilter{From: date(1), To: date(2)}, wantErr: "client_id"},
		{name: "bad app", filter: models.DailyFilter{ClientID: 1, AppID: &zero, From: date(1), To: date(2)}, wantErr: "app_id"},
		{name: "no dates", filter: models.DailyFilter{ClientID: 1}, wantErr: "required"},
		{name: "reversed", filter: models.DailyFilter{ClientID: 1, From: date(3), To: date(1)}, wantErr: "after"},
		{name: "too long", filter: models.DailyFilter{ClientID: 1, From: date(1), To: date(1).AddDate(1, 0, 1)}, wantErr: "exceed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeRepo{}
			_, err := service.NewMetricService(repo).GetDaily(context.Background(), tt.filter)

			require.ErrorIs(t, err, service.ErrValidation)
			require.ErrorContains(t, err, tt.wantErr)
			require.Zero(t, repo.calls)
		})
	}
}
