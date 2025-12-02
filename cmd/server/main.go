package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Storage interface {
	UpdateGauge(name string, val float64)
	UpdateCounter(name string, val int64)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAll() (map[string]float64, map[string]int64)
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
	g := make(map[string]float64)
	c := make(map[string]int64)
	for k, v := range m.gauges {
		g[k] = v
	}
	for k, v := range m.counters {
		c[k] = v
	}
	return g, c
}

func updateHandler(storage Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		name := chi.URLParam(r, "name")
		valueStr := chi.URLParam(r, "value")
		if name == "" {
			http.Error(w, "Metric name not found", http.StatusNotFound)
			return
		}

		switch metricType {
		case "gauge":
			v, err := strconv.ParseFloat(valueStr, 64)
			if err != nil {
				http.Error(w, "invalid value", http.StatusBadRequest)
				return
			}
			storage.UpdateGauge(name, v)
		case "counter":
			delta, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				http.Error(w, "invalid value", http.StatusBadRequest)
				return
			}
			storage.UpdateCounter(name, delta)
		default:
			http.Error(w, "unknown metric type", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func valueHandler(store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "type")
		name := chi.URLParam(r, "name")

		switch mType {
		case "gauge":
			if v, ok := store.GetGauge(name); ok {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "%f", v)
				return
			}
		case "counter":
			if v, ok := store.GetCounter(name); ok {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "%d", v)
				return
			}
		}

		http.Error(w, "metric not found", http.StatusNotFound)
	}
}

func pageHandler(store Storage) http.HandlerFunc {
	type MetricRow struct {
		Name  string
		Type  string
		Value string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		gauges, counters := store.GetAll()

		var rows []MetricRow
		for name, v := range gauges {
			rows = append(rows, MetricRow{name, "gauge", fmt.Sprintf("%f", v)})
		}
		for name, v := range counters {
			rows = append(rows, MetricRow{name, "counter", fmt.Sprintf("%d", v)})
		}

		tpl := `
		<!DOCTYPE html>
		<html>
		<head><title>Metrics</title></head>
		<body>
			<h1>Metrics</h1>
			<table>
				<tr>
					<th>Name</th>
					<th>Type</th>
					<th>Value</th>
				</tr>

				{{range .}}
				<tr>
					<td>{{.Name}}</td>
					<td>{{.Type}}</td>
					<td>{{.Value}}</td>
				</tr>
				{{end}}
				
			</table>
		</body>
		</html>
		`

		t := template.Must(template.New("index").Parse(tpl))
		w.WriteHeader(http.StatusOK)
		t.Execute(w, rows)
	}
}

func main() {
	storage := NewMemStorage()
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", updateHandler(storage))
	r.Get("/value/{type}/{name}", valueHandler(storage))
	r.Get("/", pageHandler(storage))
	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
