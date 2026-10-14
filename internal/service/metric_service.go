package service

import (
	"context"
	"errors"
	"fmt"

	"metrics-service/internal/models"
)

var ErrValidation = errors.New("validation error")

type MetricRepository interface {
	Insert(ctx context.Context, metric models.Metric) error
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
