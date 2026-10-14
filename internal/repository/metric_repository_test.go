package repository_test

import (
	"context"
	"testing"
	"time"

	"metrics-service/internal/config"
	"metrics-service/internal/models"
	"metrics-service/internal/repository"
	"metrics-service/internal/storage"

	"github.com/stretchr/testify/require"
)

func TestInsertMetric(t *testing.T) {

	ctx := context.Background()

	db, err := storage.NewPool(ctx, config.Load().DatabaseURL)

	require.NoError(t, err)

	defer db.Close()

	repo := repository.NewMetricRepository(db)

	err = repo.Insert(
		ctx,
		models.Metric{
			ClientID:         1,
			AppID:            1,
			Requests:         200,
			Landings:         50,
			ApiRegisterTotal: 100,
			ApiRegisterScam:  10,
			Timestamp:        time.Now(),
		},
	)

	require.NoError(t, err)
}

func TestInsertMetricsBatch(t *testing.T) {
	ctx := context.Background()

	db, err := storage.NewPool(ctx, config.Load().DatabaseURL)
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewMetricRepository(db)

	now := time.Now()
	metrics := []models.Metric{
		{ClientID: 1, AppID: 1, Requests: 10, Landings: 5, ApiRegisterTotal: 3, ApiRegisterScam: 1, Timestamp: now},
		{ClientID: 1, AppID: 2, Requests: 20, Landings: 6, ApiRegisterTotal: 4, ApiRegisterScam: 0, Timestamp: now},
		{ClientID: 1, AppID: 2, Requests: 20, Landings: 6, ApiRegisterTotal: 4, ApiRegisterScam: 0, Timestamp: now},
	}

	inserted, err := repo.InsertBatch(ctx, metrics)
	require.NoError(t, err)
	require.EqualValues(t, 2, inserted)

	inserted, err = repo.InsertBatch(ctx, metrics)
	require.NoError(t, err)
	require.EqualValues(t, 0, inserted)
}

func TestDailyStats(t *testing.T) {
	ctx := context.Background()

	db, err := storage.NewPool(ctx, config.Load().DatabaseURL)
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewMetricRepository(db)

	clientID := int32(time.Now().UnixNano()%1_000_000_000) + 1_000
	day1 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	day2 := day1.AddDate(0, 0, 1)

	_, err = repo.InsertBatch(ctx, []models.Metric{
		{ClientID: clientID, AppID: 1, Requests: 10, Landings: 1, ApiRegisterTotal: 5, ApiRegisterScam: 1, Timestamp: day1.Add(1 * time.Hour)},
		{ClientID: clientID, AppID: 1, Requests: 20, Landings: 2, ApiRegisterTotal: 6, ApiRegisterScam: 2, Timestamp: day1.Add(23 * time.Hour)},
		{ClientID: clientID, AppID: 2, Requests: 30, Landings: 3, ApiRegisterTotal: 7, ApiRegisterScam: 3, Timestamp: day1.Add(2 * time.Hour)},
		{ClientID: clientID, AppID: 1, Requests: 40, Landings: 4, ApiRegisterTotal: 8, ApiRegisterScam: 4, Timestamp: day2.Add(1 * time.Hour)},
		{ClientID: clientID, AppID: 1, Requests: 99, Landings: 9, ApiRegisterTotal: 9, ApiRegisterScam: 9, Timestamp: day2.AddDate(0, 0, 1)},
	})
	require.NoError(t, err)

	all, err := repo.DailyStats(ctx, clientID, nil, day1, day2.AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, day1, all[0].Date.UTC())
	require.EqualValues(t, 60, all[0].Requests)
	require.EqualValues(t, 6, all[0].ApiRegisterScam)
	require.EqualValues(t, 40, all[1].Requests)

	appID := int32(1)
	app1, err := repo.DailyStats(ctx, clientID, &appID, day1, day2.AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Len(t, app1, 2)
	require.EqualValues(t, 30, app1[0].Requests)
	require.EqualValues(t, 40, app1[1].Requests)
}
