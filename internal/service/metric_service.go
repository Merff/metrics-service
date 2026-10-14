package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"metrics-service/internal/models"
)

const (
	MaxBatchSize = 1000
	MaxRangeDays = 366
)

var ErrValidation = errors.New("validation error")

type MetricRepository interface {
	Insert(ctx context.Context, metric models.Metric) error
	InsertBatch(ctx context.Context, metrics []models.Metric) (int64, error)
	DailyStats(ctx context.Context, clientID int32, appID *int32, from, to time.Time) ([]models.DailyMetric, error)
}

type MetricService struct {
	repo MetricRepository
}

func NewMetricService(repo MetricRepository) *MetricService {
	return &MetricService{repo: repo}
}

func (s *MetricService) Save(ctx context.Context, metric models.Metric) error {
	if err := validate(metric); err != nil {
		return err
	}

	return s.repo.Insert(ctx, metric)
}

func (s *MetricService) SaveBatch(ctx context.Context, metrics []models.Metric) (int64, error) {
	if len(metrics) == 0 {
		return 0, fmt.Errorf("%w: batch is empty", ErrValidation)
	}
	if len(metrics) > MaxBatchSize {
		return 0, fmt.Errorf("%w: batch size %d exceeds limit %d", ErrValidation, len(metrics), MaxBatchSize)
	}

	for i, m := range metrics {
		if err := validate(m); err != nil {
			return 0, fmt.Errorf("metrics[%d]: %w", i, err)
		}
	}

	return s.repo.InsertBatch(ctx, metrics)
}

// GetDaily returns per-day sums for the inclusive UTC date range,
// with zero-filled entries for days without data.
func (s *MetricService) GetDaily(ctx context.Context, f models.DailyFilter) ([]models.DailyMetric, error) {
	from := truncateDay(f.From)
	to := truncateDay(f.To)

	switch {
	case f.ClientID <= 0:
		return nil, fmt.Errorf("%w: client_id must be positive", ErrValidation)
	case f.AppID != nil && *f.AppID <= 0:
		return nil, fmt.Errorf("%w: app_id must be positive", ErrValidation)
	case from.IsZero() || to.IsZero():
		return nil, fmt.Errorf("%w: from and to are required", ErrValidation)
	case to.Before(from):
		return nil, fmt.Errorf("%w: from must not be after to", ErrValidation)
	case daysBetween(from, to) >= MaxRangeDays:
		return nil, fmt.Errorf("%w: range must not exceed %d days", ErrValidation, MaxRangeDays)
	}

	stats, err := s.repo.DailyStats(ctx, f.ClientID, f.AppID, from, to.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}

	byDay := make(map[time.Time]models.DailyMetric, len(stats))
	for _, st := range stats {
		byDay[truncateDay(st.Date)] = st
	}

	result := make([]models.DailyMetric, 0, daysBetween(from, to)+1)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		st := byDay[d]
		st.Date = d
		result = append(result, st)
	}

	return result, nil
}

func truncateDay(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func daysBetween(from, to time.Time) int {
	return int(to.Sub(from).Hours() / 24)
}

func validate(m models.Metric) error {
	switch {
	case m.ClientID <= 0:
		return fmt.Errorf("%w: client_id must be positive", ErrValidation)
	case m.AppID <= 0:
		return fmt.Errorf("%w: app_id must be positive", ErrValidation)
	case m.Requests < 0, m.Landings < 0, m.ApiRegisterTotal < 0, m.ApiRegisterScam < 0:
		return fmt.Errorf("%w: counters must be non-negative", ErrValidation)
	case m.ApiRegisterScam > m.ApiRegisterTotal:
		return fmt.Errorf("%w: api_register_scam must not exceed api_register_total", ErrValidation)
	case m.Timestamp.IsZero():
		return fmt.Errorf("%w: timestamp is required", ErrValidation)
	}

	return nil
}
