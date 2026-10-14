package repository

import (
	"context"
	"errors"
	"fmt"

	"metrics-service/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolation = "23505"

var ErrDuplicate = errors.New("metric already exists")

type MetricRepository struct {
	db *pgxpool.Pool
}

func NewMetricRepository(
	db *pgxpool.Pool,
) *MetricRepository {
	return &MetricRepository{
		db: db,
	}
}

func (r *MetricRepository) Insert(
	ctx context.Context,
	metric models.Metric,
) error {

	query := `
    INSERT INTO metrics (
      client_id,
      app_id,
      requests,
      landings,
      api_register_total,
      api_register_scam,
      timestamp
    )
    VALUES ($1, $2, $3, $4, $5, $6, $7)
  `

	_, err := r.db.Exec(
		ctx,
		query,
		metric.ClientID,
		metric.AppID,
		metric.Requests,
		metric.Landings,
		metric.ApiRegisterTotal,
		metric.ApiRegisterScam,
		metric.Timestamp,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("insert metric: %w", err)
	}

	return nil
}
