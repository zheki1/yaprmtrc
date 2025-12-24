package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/zheki1/yaprmtrc.git/internal/models"
)

type Server struct {
	storage Storage
}

func (s *Server) valueHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "content type must be application/json", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "empty request body", http.StatusBadRequest)
		return
	}

	var m models.Metrics
	if err := json.Unmarshal(body, &m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if m.ID == "" || m.MType == "" {
		http.Error(w, "id and type are required", http.StatusBadRequest)
		return
	}

	switch m.MType {
	case models.Gauge:
		value, ok := s.storage.GetGauge(m.ID)
		if !ok {
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		m.Value = &value

	case models.Counter:
		delta, ok := s.storage.GetCounter(m.ID)
		if !ok {
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		m.Delta = &delta

	default:
		http.Error(w, "unknown metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(m)
}

func (s *Server) updateHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "content type must be application/json", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "empty request body", http.StatusBadRequest)
		return
	}

	var m models.Metrics
	if err := json.Unmarshal(body, &m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if m.ID == "" || m.MType == "" {
		http.Error(w, "id and type are required", http.StatusBadRequest)
		return
	}

	switch m.MType {
	case models.Gauge:
		if m.Value == nil {
			http.Error(w, "value is required", http.StatusBadRequest)
			return
		}
		s.storage.UpdateGauge(m.ID, *m.Value)

	case models.Counter:
		if m.Delta == nil {
			http.Error(w, "delta is required", http.StatusBadRequest)
			return
		}
		s.storage.UpdateCounter(m.ID, *m.Delta)

	default:
		http.Error(w, "unknown metric type", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) updateHandler(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	valueStr := chi.URLParam(r, "value")
	if name == "" {
		http.Error(w, "Metric name not found", http.StatusNotFound)
		return
	}

	switch mType {
	case models.Gauge:
		v, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			http.Error(w, "invalid value", http.StatusBadRequest)
			return
		}
		s.storage.UpdateGauge(name, v)
	case models.Counter:
		delta, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid value", http.StatusBadRequest)
			return
		}
		s.storage.UpdateCounter(name, delta)
	default:
		http.Error(w, "unknown metric type", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) valueHandler(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	switch mType {
	case models.Gauge:
		if v, ok := s.storage.GetGauge(name); ok {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "%g", v)
			return
		}
	case models.Counter:
		if v, ok := s.storage.GetCounter(name); ok {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "%d", v)
			return
		}
	}

	http.Error(w, "metric not found", http.StatusNotFound)
}

func (s *Server) pageHandler(w http.ResponseWriter, r *http.Request) {
	type MetricRow struct {
		Name  string
		Type  string
		Value string
	}

	gauges, counters := s.storage.GetAll()

	var rows []MetricRow
	for name, v := range gauges {
		rows = append(rows, MetricRow{name, models.Gauge, fmt.Sprintf("%f", v)})
	}
	for name, v := range counters {
		rows = append(rows, MetricRow{name, models.Counter, fmt.Sprintf("%d", v)})
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
