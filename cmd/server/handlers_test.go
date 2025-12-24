package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func testRouter(storage Storage) http.Handler {
	r := chi.NewRouter()
	s := &Server{storage: storage}

	r.Post("/update/{type}/{name}/{value}", s.updateHandler)
	r.Get("/value/{type}/{name}", s.valueHandler)
	r.Get("/", s.pageHandler)

	return r
}

func TestHandleUpdateGaugeOK(t *testing.T) {
	st := NewMemStorage()
	router := testRouter(st)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	val, ok := st.GetGauge("Alloc")
	if !ok || val != 123.45 {
		t.Fatalf("metric not stored correctly")
	}
}

func TestHandleUpdateCounterOK(t *testing.T) {
	st := NewMemStorage()
	router := testRouter(st)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/5", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	val, ok := st.GetCounter("PollCount")
	if !ok || val != 5 {
		t.Fatalf("counter not stored correctly")
	}
}

func TestHandleUpdateInvalidType(t *testing.T) {
	st := NewMemStorage()
	router := testRouter(st)

	req := httptest.NewRequest(http.MethodPost, "/update/unknown/Alloc/1", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleUpdateInvalidValue(t *testing.T) {
	st := NewMemStorage()
	router := testRouter(st)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/abc", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleGetValueGaugeOK(t *testing.T) {
	st := NewMemStorage()
	st.UpdateGauge("Alloc", 10.5)

	router := testRouter(st)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if strings.TrimSpace(string(body)) != "10.5" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestHandleGetValueCounterOK(t *testing.T) {
	st := NewMemStorage()
	st.UpdateCounter("PollCount", 7)

	router := testRouter(st)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if strings.TrimSpace(string(body)) != "7" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestHandleGetValueNotFound(t *testing.T) {
	st := NewMemStorage()
	router := testRouter(st)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Unknown", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGetValueInvalidType(t *testing.T) {
	st := NewMemStorage()
	router := testRouter(st)

	req := httptest.NewRequest(http.MethodGet, "/value/invalid/Alloc", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGetAll(t *testing.T) {
	st := NewMemStorage()
	st.UpdateGauge("Alloc", 1.23)
	st.UpdateCounter("PollCount", 2)

	router := testRouter(st)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if !strings.Contains(string(body), "Alloc") ||
		!strings.Contains(string(body), "PollCount") {
		t.Fatalf("response does not contain metrics")
	}
}
