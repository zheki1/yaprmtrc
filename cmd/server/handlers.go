package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	storage Storage
}

func (s *Server) updateHandler() http.HandlerFunc {
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
			s.storage.UpdateGauge(name, v)
		case "counter":
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
}

func (s *Server) valueHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mType := chi.URLParam(r, "type")
		name := chi.URLParam(r, "name")

		switch mType {
		case "gauge":
			if v, ok := s.storage.GetGauge(name); ok {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "%g", v)
				return
			}
		case "counter":
			if v, ok := s.storage.GetCounter(name); ok {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "%d", v)
				return
			}
		}

		http.Error(w, "metric not found", http.StatusNotFound)
	}
}

func (s *Server) pageHandler() http.HandlerFunc {
	type MetricRow struct {
		Name  string
		Type  string
		Value string
	}

	return func(w http.ResponseWriter, r *http.Request) {
		gauges, counters := s.storage.GetAll()

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
