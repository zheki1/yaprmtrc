package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupRouter() http.Handler {
	store := NewMemStorage()
	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", updateHandler(store))
	r.Get("/value/{type}/{name}", valueHandler(store))
	r.Get("/", pageHandler(store))
	return r
}

func TestUpdateGaugeOK(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("POST", "/update/gauge/Alloc/123.45", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestUpdateCounterOK(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("POST", "/update/counter/Requests/10", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestUpdateNoName(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("POST", "/update/gauge//123", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestUpdateBadType(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("POST", "/update/badtype/MyMetric/5", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateBadValue(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("POST", "/update/gauge/MyMetric/abc", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestValueGaugeOK(t *testing.T) {
	r := setupRouter()

	// сначала запишем gauge
	req1 := httptest.NewRequest("POST", "/update/gauge/Alloc/555.0", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	// читаем gauge
	req2 := httptest.NewRequest("GET", "/value/gauge/Alloc", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	body, _ := io.ReadAll(w2.Body)
	if !strings.Contains(string(body), "555") {
		t.Fatalf("expected value 555, got %s", body)
	}
}

func TestValueCounterOK(t *testing.T) {
	r := setupRouter()

	req1 := httptest.NewRequest("POST", "/update/counter/Requests/10", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("GET", "/value/counter/Requests", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	body, _ := io.ReadAll(w2.Body)
	if !strings.Contains(string(body), "10") {
		t.Fatalf("expected value 10, got %s", body)
	}
}

func TestMetricNotFound(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("GET", "/value/gauge/UnknownMetric", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMetricTypeNotFound(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("GET", "/value/badtype/Alloc", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestIndexHTML(t *testing.T) {
	r := setupRouter()

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body, _ := io.ReadAll(w.Body)
	if !strings.Contains(string(body), "<html>") {
		t.Fatalf("expected HTML page, got %s", body)
	}
}
