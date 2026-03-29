package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"go.uber.org/zap"

	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/repository"

	"runtime/pprof"
)

func newProfileServer() *Server {
	logger, _ := zap.NewDevelopment()
	st := repository.NewMemRepository()
	return &Server{
		storage: st,
		logger:  logger.Sugar(),
		audit:   NewAuditPublisher(logger.Sugar()),
	}
}

// TestProfileMemory generates a heap profile after simulating realistic load.
// Run with: go test -run TestProfileMemory -count=1 ./cmd/server/
// The profile file path is controlled by PPROF_OUTPUT env var (default: profiles/base.pprof).
func TestProfileMemory(t *testing.T) {
	outPath := os.Getenv("PPROF_OUTPUT")
	if outPath == "" {
		outPath = "../../profiles/base.pprof"
	}

	s := newProfileServer()
	ctx := context.Background()

	// Pre-populate storage with realistic data
	for i := 0; i < 500; i++ {
		_ = s.storage.UpdateGauge(ctx, fmt.Sprintf("gauge_%d", i), float64(i)*1.1)
		_ = s.storage.UpdateCounter(ctx, fmt.Sprintf("counter_%d", i), int64(i))
	}

	// Simulate batch updates
	batchMetrics := make([]models.Metrics, 0, 200)
	for i := 0; i < 100; i++ {
		v := float64(i) * 2.5
		batchMetrics = append(batchMetrics, models.Metrics{
			ID:    fmt.Sprintf("batch_gauge_%d", i),
			MType: models.Gauge,
			Value: &v,
		})
		d := int64(i) * 3
		batchMetrics = append(batchMetrics, models.Metrics{
			ID:    fmt.Sprintf("batch_counter_%d", i),
			MType: models.Counter,
			Delta: &d,
		})
	}

	batchPayload, _ := json.Marshal(batchMetrics)

	// Simulate heavy load: update, value, batch, page requests
	for round := 0; round < 500; round++ {
		// Single gauge update
		m := models.Metrics{
			ID:    fmt.Sprintf("gauge_%d", round%500),
			MType: models.Gauge,
			Value: func() *float64 { v := float64(round); return &v }(),
		}
		payload, _ := json.Marshal(m)

		req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.updateHandlerJSON(w, req)

		// Single counter update
		mc := models.Metrics{
			ID:    fmt.Sprintf("counter_%d", round%500),
			MType: models.Counter,
			Delta: func() *int64 { d := int64(1); return &d }(),
		}
		payloadC, _ := json.Marshal(mc)
		req = httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(payloadC))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		s.updateHandlerJSON(w, req)

		// Value request
		vReq := models.Metrics{ID: fmt.Sprintf("gauge_%d", round%500), MType: models.Gauge}
		vPayload, _ := json.Marshal(vReq)
		req = httptest.NewRequest(http.MethodPost, "/value", bytes.NewReader(vPayload))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		s.valueHandlerJSON(w, req)

		// Batch update
		req = httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(batchPayload))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		s.batchUpdateHandler(w, req)

		// Page render
		req = httptest.NewRequest(http.MethodGet, "/", nil)
		w = httptest.NewRecorder()
		s.pageHandler(w, req)
	}

	// Force GC to get accurate retained memory picture
	runtime.GC()

	f, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("cannot create profile file: %v", err)
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		t.Fatalf("cannot write heap profile: %v", err)
	}

	t.Logf("Heap profile written to %s", outPath)
}
