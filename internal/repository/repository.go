// Package repository реализует хранилища метрик: в памяти. в файле и в PostgreSQL.
package repository

import (
	"context"

	"github.com/zheki1/yaprmtrc/internal/models"
)

// Repository — интерфейс хранилища метрик.
// Поддерживает обновление и чтение gauge/counter-метрик,
// пакетное обновление и получение всех метрик.
type Repository interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, delta int64) error
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error

	GetGauge(ctx context.Context, name string) (float64, bool, error)
	GetCounter(ctx context.Context, name string) (int64, bool, error)

	GetAll(ctx context.Context) ([]models.Metrics, error)

	Close() error
}
