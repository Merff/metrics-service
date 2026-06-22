package repository_test

import (
	"context"
	"testing"
	"time"

	"metrics-service/internal/models"
	"metrics-service/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestInsertMetric(t *testing.T) {

	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://postgres:postgres@localhost:5433/metrics",
	)

	require.NoError(t, err)

	defer db.Close()

	repo := repository.NewMetricRepository(db)

	err = repo.Insert(
		ctx,
		models.Metric{
			ClientID:  1,
		  AppID:     1,
		  Requests:  200,
		  Landings:  50,
		  ApiRegisterTotal: 100,
		  ApiRegisterScam:  10,
		  Timestamp: time.Now(),
		},
	)

	require.NoError(t, err)
}
