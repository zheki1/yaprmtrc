package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/repository"
)

func newBenchServer() *Server {
	logger, _ := zap.NewDevelopment()
	st := repository.NewMemRepository()
	return &Server{
		storage: st,
		logger:  logger.Sugar(),
		audit:   NewAuditPublisher(logger.Sugar()),
	}
}

func BenchmarkUpdateHandlerJSON_Gauge(b *testing.B) {
	s := newBenchServer()

	m := models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
		Value: ptrFloat(12.3),
	}
	payload, _ := json.Marshal(m)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.updateHandlerJSON(w, req)
	}
}

func BenchmarkUpdateHandlerJSON_Counter(b *testing.B) {
	s := newBenchServer()

	m := models.Metrics{
		ID:    "PollCount",
		MType: models.Counter,
		Delta: ptrInt(1),
	}
	payload, _ := json.Marshal(m)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.updateHandlerJSON(w, req)
	}
}

func BenchmarkValueHandlerJSON(b *testing.B) {
	s := newBenchServer()
	_ = s.storage.UpdateGauge(context.Background(), "Alloc", 99.9)

	reqBody := models.Metrics{ID: "Alloc", MType: models.Gauge}
	payload, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.valueHandlerJSON(w, req)
	}
}

func BenchmarkBatchUpdateHandler(b *testing.B) {
	s := newBenchServer()

	data := make([]models.Metrics, 0, 200)
	for i := 0; i < 100; i++ {
		v := float64(i)
		data = append(data, models.Metrics{
			ID:    fmt.Sprintf("gauge_%d", i),
			MType: models.Gauge,
			Value: &v,
		})
		d := int64(i)
		data = append(data, models.Metrics{
			ID:    fmt.Sprintf("counter_%d", i),
			MType: models.Counter,
			Delta: &d,
		})
	}
	payload, _ := json.Marshal(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.batchUpdateHandler(w, req)
	}
}

func BenchmarkPageHandler(b *testing.B) {
	s := newBenchServer()
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		_ = s.storage.UpdateGauge(ctx, fmt.Sprintf("gauge_%d", i), float64(i))
		_ = s.storage.UpdateCounter(ctx, fmt.Sprintf("counter_%d", i), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		s.pageHandler(w, req)
	}
}
