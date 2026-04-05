package repository

import (
	"context"
	"sync"

	"github.com/zheki1/yaprmtrc/internal/models"
)

// MemRepository — потокобезопасное in-memory хранилище метрик.
// Используется как хранилище по умолчанию, когда не задана база данных.
type MemRepository struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemRepository создаёт новое пустое in-memory хранилище.
func NewMemRepository() *MemRepository {
	return &MemRepository{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge устанавливает значение gauge-метрики.
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

// UpdateCounter добавляет delta к текущему значению counter-метрики.
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

// GetGauge возвращает значение gauge-метрики по имени. Второе значение false, если метрика не найдена.
func (m *MemRepository) GetGauge(
	ctx context.Context,
	name string,
) (float64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.gauges[name]
	return val, ok, nil
}

// GetCounter возвращает значение counter-метрики по имени. Второе значение false, если метрика не найдена.
func (m *MemRepository) GetCounter(
	ctx context.Context,
	name string,
) (int64, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.counters[name]
	return val, ok, nil
}

// GetAll возвращает срез всех хранимых метрик.
func (m *MemRepository) GetAll(
	ctx context.Context,
) ([]models.Metrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]models.Metrics, 0, len(m.gauges)+len(m.counters))

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

// UpdateBatch атомарно обновляет несколько метрик за один вызов.
func (m *MemRepository) UpdateBatch(
	ctx context.Context,
	metrics []models.Metrics,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mt := range metrics {
		switch mt.MType {
		case models.Gauge:
			m.gauges[mt.ID] = *mt.Value

		case models.Counter:
			m.counters[mt.ID] += *mt.Delta
		}
	}

	return nil
}

// Close освобождает ресурсы (для in-memory хранилища ничего не делает).
func (m *MemRepository) Close() error {
	return nil
}
