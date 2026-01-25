package repository

import (
	"context"

	"github.com/zheki1/yaprmtrc.git/internal/models"
)

type Repository interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, delta int64) error

	GetGauge(ctx context.Context, name string) (float64, bool, error)
	GetCounter(ctx context.Context, name string) (int64, bool, error)

	GetAll(ctx context.Context) ([]models.Metrics, error)

	Close() error
}
