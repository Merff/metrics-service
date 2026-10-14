package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"metrics-service/internal/models"

	"github.com/jackc/pgx/v5"
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

	return mapError(err, "insert metric")
}

func (r *MetricRepository) InsertBatch(
	ctx context.Context,
	metrics []models.Metric,
) (int64, error) {
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
    SELECT * FROM unnest(
      $1::int[],
      $2::int[],
      $3::int[],
      $4::int[],
      $5::int[],
      $6::int[],
      $7::timestamptz[]
    )
    ON CONFLICT (client_id, app_id, timestamp) DO NOTHING
  `

	n := len(metrics)
	clientIDs := make([]int32, n)
	appIDs := make([]int32, n)
	requests := make([]int32, n)
	landings := make([]int32, n)
	registerTotal := make([]int32, n)
	registerScam := make([]int32, n)
	timestamps := make([]time.Time, n)

	for i, m := range metrics {
		clientIDs[i] = m.ClientID
		appIDs[i] = m.AppID
		requests[i] = m.Requests
		landings[i] = m.Landings
		registerTotal[i] = m.ApiRegisterTotal
		registerScam[i] = m.ApiRegisterScam
		timestamps[i] = m.Timestamp
	}

	tag, err := r.db.Exec(
		ctx,
		query,
		clientIDs,
		appIDs,
		requests,
		landings,
		registerTotal,
		registerScam,
		timestamps,
	)
	if err != nil {
		return 0, fmt.Errorf("insert metrics batch: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (r *MetricRepository) DailyStats(
	ctx context.Context,
	clientID int32,
	appID *int32,
	from time.Time,
	to time.Time,
) ([]models.DailyMetric, error) {
	query := `
    SELECT
      (timestamp AT TIME ZONE 'UTC')::date AS day,
      SUM(requests),
      SUM(landings),
      SUM(api_register_total),
      SUM(api_register_scam)
    FROM metrics
    WHERE client_id = $1
      AND ($2::int IS NULL OR app_id = $2)
      AND timestamp >= $3
      AND timestamp < $4
    GROUP BY day
    ORDER BY day
  `

	rows, err := r.db.Query(ctx, query, clientID, appID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query daily stats: %w", err)
	}

	result, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.DailyMetric, error) {
		var m models.DailyMetric
		err := row.Scan(
			&m.Date,
			&m.Requests,
			&m.Landings,
			&m.ApiRegisterTotal,
			&m.ApiRegisterScam,
		)
		return m, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan daily stats: %w", err)
	}

	return result, nil
}

func mapError(err error, op string) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return ErrDuplicate
	}

	return fmt.Errorf("%s: %w", op, err)
}
