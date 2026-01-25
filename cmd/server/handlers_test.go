package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/zheki1/yaprmtrc.git/internal/models"
	"github.com/zheki1/yaprmtrc.git/internal/repository"
)

func ptrFloat(v float64) *float64 {
	return &v
}

func ptrInt(v int64) *int64 {
	return &v
}

func newTestServer() *Server {

	logger, _ := zap.NewDevelopment()

	st := repository.NewMemRepository()

	return &Server{
		storage: st,
		logger:  logger.Sugar(),
	}
}

func gzipBody(t *testing.T, b []byte) io.Reader {

	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)

	_, err := gz.Write(b)
	if err != nil {
		t.Fatal(err)
	}

	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	return &buf
}

func TestUpdateHandlerJSON_Gauge(t *testing.T) {

	s := newTestServer()

	m := models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
		Value: ptrFloat(12.3),
	}

	b, _ := json.Marshal(m)

	req := httptest.NewRequest(
		http.MethodPost,
		"/update",
		bytes.NewBuffer(b),
	)

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	s.updateHandlerJSON(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
}

func TestUpdateHandlerJSON_Counter(t *testing.T) {

	s := newTestServer()

	m := models.Metrics{
		ID:    "PollCount",
		MType: models.Counter,
		Delta: ptrInt(5),
	}

	b, _ := json.Marshal(m)

	req := httptest.NewRequest(
		http.MethodPost,
		"/update",
		bytes.NewBuffer(b),
	)

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	s.updateHandlerJSON(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("bad status")
	}
}

func TestUpdateHandlerJSON_Gzip(t *testing.T) {

	s := newTestServer()

	m := models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
		Value: ptrFloat(55.5),
	}

	b, _ := json.Marshal(m)

	req := httptest.NewRequest(
		http.MethodPost,
		"/update",
		gzipBody(t, b),
	)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	w := httptest.NewRecorder()

	s.updateHandlerJSON(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("gzip failed")
	}
}

func TestValueHandlerJSON(t *testing.T) {

	s := newTestServer()

	_ = s.storage.UpdateGauge(context.Background(), "Alloc", 99.9)

	reqBody := models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
	}

	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(
		http.MethodPost,
		"/value",
		bytes.NewBuffer(b),
	)

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	s.valueHandlerJSON(w, req)

	var res models.Metrics

	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}

	if *res.Value != 99.9 {
		t.Fatalf("wrong value %v", *res.Value)
	}
}

func TestBatchUpdateHandler(t *testing.T) {

	s := newTestServer()

	data := []models.Metrics{
		{
			ID:    "A",
			MType: models.Gauge,
			Value: ptrFloat(1.1),
		},
		{
			ID:    "B",
			MType: models.Counter,
			Delta: ptrInt(2),
		},
	}

	b, _ := json.Marshal(data)

	req := httptest.NewRequest(
		http.MethodPost,
		"/updates",
		bytes.NewBuffer(b),
	)

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	s.batchUpdateHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("batch failed")
	}
}

func TestPageHandler(t *testing.T) {

	s := newTestServer()

	_ = s.storage.UpdateGauge(context.Background(), "A", 1.1)

	req := httptest.NewRequest(
		http.MethodGet,
		"/",
		nil,
	)

	w := httptest.NewRecorder()

	s.pageHandler(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("bad status")
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("Metrics")) {
		t.Fatal("no html output")
	}
}

func TestPingHandler_NoDB(t *testing.T) {

	s := newTestServer()

	req := httptest.NewRequest(
		http.MethodGet,
		"/ping",
		nil,
	)

	w := httptest.NewRecorder()

	s.pingHandler(w, req)

	if w.Result().StatusCode != http.StatusInternalServerError {
		t.Fatal("expected 500")
	}
}
