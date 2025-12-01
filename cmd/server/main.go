package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Storage interface {
	UpdateGauge(name string, val float64)
	UpdateCounter(name string, val int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
}

type MemStorage struct {
	mu       sync.RWMutex
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
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = val
}

func (m *MemStorage) UpdateCounter(name string, val int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += val
}

func (m *MemStorage) GetGauge(name string) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.gauges[name]
	return val, ok
}

func (m *MemStorage) GetCounter(name string) (int64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.counters[name]
	return val, ok
}

func updateHandler(storage *MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path
		if !strings.Contains(path, "/update/") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		pathMetrics := strings.TrimPrefix(path, "/update/")
		metrics := strings.Split(pathMetrics, "/")

		if len(metrics) >= 2 && metrics[1] == "" {
			http.Error(w, "Metric name not found", http.StatusNotFound)
			return
		}

		if len(metrics) != 3 {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		metricType, name, rawValue := metrics[0], metrics[1], metrics[2]

		switch metricType {
		case "gauge":
			val, err := strconv.ParseFloat(rawValue, 64)
			if err != nil {
				http.Error(w, "Invalid gauge value", http.StatusBadRequest)
				return
			}
			storage.UpdateGauge(name, val)
			w.WriteHeader(http.StatusOK)
		case "counter":
			val, err := strconv.ParseInt(rawValue, 10, 64)
			if err != nil {
				http.Error(w, "Invalid counter value", http.StatusBadRequest)
				return
			}
			storage.UpdateCounter(name, val)
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "Unknown metric type", http.StatusBadRequest)
		}
	}
}

func main() {
	storage := NewMemStorage()
	http.HandleFunc("/update/", updateHandler(storage))
	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
