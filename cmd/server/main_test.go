package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type StorageMock struct {
	GaugeUpdates   map[string]float64
	CounterUpdates map[string]int64
}

func NewStorageMock() *StorageMock {
	return &StorageMock{
		GaugeUpdates:   make(map[string]float64),
		CounterUpdates: make(map[string]int64),
	}
}

func (m *StorageMock) UpdateGauge(name string, value float64) {
	m.GaugeUpdates[name] = value
}

func (m *StorageMock) UpdateCounter(name string, value int64) {
	m.CounterUpdates[name] += value
}

func (m *StorageMock) GetMetrics() map[string]float64 {
	metrics := make(map[string]float64)
	for key, value := range m.GaugeUpdates {
		metrics[key+"_gauges"] = value
	}
	for key, value := range m.CounterUpdates {
		metrics[key+"_counter"] = float64(value)
	}
	return metrics
}

func TestUpdateHandler(t *testing.T) {
	storage := NewStorageMock()
	handler := updateHandler(storage)

	tests := []struct {
		name            string
		method          string
		urlPath         string
		expectedCode    int
		expectedGauge   map[string]float64
		expectedCounter map[string]int64
	}{
		{
			name:          "Valid gauge",
			method:        http.MethodPost,
			urlPath:       "/update/gauge/temp/36.6",
			expectedCode:  http.StatusOK,
			expectedGauge: map[string]float64{"temp": 36.6},
		},
		{
			name:            "Valid counter",
			method:          http.MethodPost,
			urlPath:         "/update/counter/hits/10",
			expectedCode:    http.StatusOK,
			expectedCounter: map[string]int64{"hits": 10},
		},
		{
			name:         "Invalid method",
			method:       http.MethodGet,
			urlPath:      "/update/gauge/temp/36.6",
			expectedCode: http.StatusMethodNotAllowed,
		},
		{
			name:         "Invalid metric type",
			method:       http.MethodPost,
			urlPath:      "/update/unknown/metric/123",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Empty metric name",
			method:       http.MethodPost,
			urlPath:      "/update/gauge//123",
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Bad gauge value",
			method:       http.MethodPost,
			urlPath:      "/update/gauge/pressure/abc",
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.urlPath, nil)
			rr := httptest.NewRecorder()

			handler(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, rr.Code)
			}

			for key, value := range tt.expectedGauge {
				if existedValue, ok := storage.GaugeUpdates[key]; !ok || existedValue != value {
					t.Errorf("expected gauge %s=%v, got %v", key, value, existedValue)
				}
			}

			for key, value := range tt.expectedCounter {
				if existedValue, ok := storage.CounterUpdates[key]; !ok || existedValue != value {
					t.Errorf("expected counter %s=%v, got %v", key, value, existedValue)
				}
			}
		})
	}
}
