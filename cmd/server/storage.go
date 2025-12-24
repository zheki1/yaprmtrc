package main

import "github.com/zheki1/yaprmtrc.git/internal/models"

type Storage interface {
	UpdateGauge(name string, val float64)
	UpdateCounter(name string, val int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAll() (map[string]float64, map[string]int64)
	Export() []models.Metrics
	Import(metrics []models.Metrics)
}

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MemStorage) UpdateGauge(name string, val float64) {
	m.gauges[name] = val
}

func (m *MemStorage) UpdateCounter(name string, val int64) {
	m.counters[name] += val
}

func (m *MemStorage) GetGauge(name string) (float64, bool) {
	v, ok := m.gauges[name]
	return v, ok
}

func (m *MemStorage) GetCounter(name string) (int64, bool) {
	v, ok := m.counters[name]
	return v, ok
}

func (m *MemStorage) GetAll() (map[string]float64, map[string]int64) {
	return m.gauges, m.counters
}

func (m *MemStorage) Export() []models.Metrics {
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

	return res
}

func (m *MemStorage) Import(metrics []models.Metrics) {
	for _, ms := range metrics {
		switch ms.MType {
		case models.Gauge:
			if ms.Value != nil {
				m.gauges[ms.ID] = *ms.Value
			}
		case models.Counter:
			if ms.Delta != nil {
				m.counters[ms.ID] = *ms.Delta
			}
		}
	}
}
