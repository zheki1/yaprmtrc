package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/zheki1/yaprmtrc/internal/repository"
	"go.uber.org/zap"
)

func newTestServerWithRouter() (*Server, http.Handler) {
	logger, _ := zap.NewDevelopment()
	st := repository.NewMemRepository()
	s := &Server{
		storage: st,
		logger:  logger.Sugar(),
		audit:   NewAuditPublisher(logger.Sugar()),
	}
	return s, router(s)
}

func TestRouter_UpdateAndValue(t *testing.T) {
	_, r := newTestServerWithRouter()

	// Update gauge via URL
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/42.5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update gauge: expected 200, got %d", w.Code)
	}

	// Get gauge via URL
	req = httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get gauge: expected 200, got %d", w.Code)
	}

	if w.Body.String() != "42.5" {
		t.Fatalf("expected '42.5', got %q", w.Body.String())
	}
}

func TestRouter_UpdateCounter(t *testing.T) {
	_, r := newTestServerWithRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("update counter: expected 200, got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("get counter: expected 200, got %d", w.Code)
	}

	if w.Body.String() != "5" {
		t.Fatalf("expected '5', got %q", w.Body.String())
	}
}

func TestRouter_UpdateInvalidType(t *testing.T) {
	_, r := newTestServerWithRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/unknown/X/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRouter_UpdateInvalidGaugeValue(t *testing.T) {
	_, r := newTestServerWithRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/X/notanumber", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRouter_UpdateInvalidCounterValue(t *testing.T) {
	_, r := newTestServerWithRouter()

	req := httptest.NewRequest(http.MethodPost, "/update/counter/X/notanumber", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRouter_ValueNotFound(t *testing.T) {
	_, r := newTestServerWithRouter()

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/NonExistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRouter_Page(t *testing.T) {
	s, r := newTestServerWithRouter()

	_ = s.storage.UpdateGauge(context.Background(), "Alloc", 1.0)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUpdateHandler_WithChiContext(t *testing.T) {
	s := newTestServer()

	// Test updateHandler directly with chi context
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/TestMetric/99.9", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "gauge")
	rctx.URLParams.Add("name", "TestMetric")
	rctx.URLParams.Add("value", "99.9")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	s.updateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestValueHandler_WithChiContext(t *testing.T) {
	s := newTestServer()
	_ = s.storage.UpdateGauge(context.Background(), "Alloc", 55.5)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "gauge")
	rctx.URLParams.Add("name", "Alloc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	s.valueHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if w.Body.String() != "55.5" {
		t.Fatalf("expected '55.5', got %q", w.Body.String())
	}
}

func TestValueHandler_Counter(t *testing.T) {
	s := newTestServer()
	_ = s.storage.UpdateCounter(context.Background(), "PollCount", 42)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "counter")
	rctx.URLParams.Add("name", "PollCount")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()
	s.valueHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if w.Body.String() != "42" {
		t.Fatalf("expected '42', got %q", w.Body.String())
	}
}
