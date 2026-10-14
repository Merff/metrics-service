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
