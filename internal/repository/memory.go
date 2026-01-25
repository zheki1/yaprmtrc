package repository

import (
	"context"
	"sync"

	"github.com/zheki1/yaprmtrc.git/internal/models"
)

type MemRepository struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemRepository() *MemRepository {
	return &MemRepository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MemRepository) UpdateGauge(
	ctx context.Context,
	name string,
	value float64,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges[name] = value
	return nil
}

func (m *MemRepository) UpdateCounter(
	ctx context.Context,
	name string,
	delta int64,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[name] += delta
	return nil
}

func (m *MemRepository) GetGauge(
	ctx context.Context,
	name string,
) (float64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.gauges[name]
	return val, ok, nil
}

func (m *MemRepository) GetCounter(
	ctx context.Context,
	name string,
) (int64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.counters[name]
	return val, ok, nil
}

func (m *MemRepository) GetAll(
	ctx context.Context,
) ([]models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var res []models.Metrics

	for k, v := range m.gauges {
		val := v
		res = append(res, models.Metrics{
			ID:    k,
			MType: models.Gauge,
			Value: &val,
		})
	}

	for k, v := range m.counters {
		val := v
		res = append(res, models.Metrics{
			ID:    k,
			MType: models.Counter,
			Delta: &val,
		})
	}

	return res, nil
}

func (m *MemRepository) Close() error {
	return nil
}
