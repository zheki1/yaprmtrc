package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"go.uber.org/zap"

	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/repository"
)

func newExampleServer() (*httptest.Server, *Server) {
	logger, _ := zap.NewDevelopment()
	st := repository.NewMemRepository()
	s := &Server{
		storage: st,
		logger:  logger.Sugar(),
		audit:   NewAuditPublisher(logger.Sugar()),
	}
	ts := httptest.NewServer(router(s))
	return ts, s
}

// ExampleServer_updateHandler демонстрирует обновление gauge-метрики через URL-параметры.
func ExampleServer_updateHandler() {
	ts, _ := newExampleServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/update/gauge/Temperature/36.6", "text/plain", nil)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	// Output:
	// 200
}

// ExampleServer_updateHandlerJSON демонстрирует обновление counter-метрики через JSON API.
func ExampleServer_updateHandlerJSON() {
	ts, _ := newExampleServer()
	defer ts.Close()

	delta := int64(5)
	m := models.Metrics{
		ID:    "PollCount",
		MType: models.Counter,
		Delta: &delta,
	}
	body, _ := json.Marshal(m)

	resp, err := http.Post(ts.URL+"/update", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	// Output:
	// 200
}

// ExampleServer_valueHandler демонстрирует получение значения gauge-метрики через URL.
func ExampleServer_valueHandler() {
	ts, s := newExampleServer()
	defer ts.Close()

	// предварительно записываем метрику
	_ = s.storage.UpdateGauge(context.Background(), "Alloc", 123456)

	resp, err := http.Get(ts.URL + "/value/gauge/Alloc")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	fmt.Println(resp.StatusCode)
	fmt.Println(string(b))
	// Output:
	// 200
	// 123456
}

// ExampleServer_valueHandlerJSON демонстрирует получение метрики через JSON API.
func ExampleServer_valueHandlerJSON() {
	ts, s := newExampleServer()
	defer ts.Close()

	value := 3.14
	_ = s.storage.UpdateGauge(context.Background(), "TestGauge", value)

	reqBody, _ := json.Marshal(models.Metrics{
		ID:    "TestGauge",
		MType: models.Gauge,
	})

	resp, err := http.Post(ts.URL+"/value", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()

	var m models.Metrics
	_ = json.NewDecoder(resp.Body).Decode(&m)

	fmt.Println(resp.StatusCode)
	fmt.Printf("%.2f\n", *m.Value)
	// Output:
	// 200
	// 3.14
}

// ExampleServer_batchUpdateHandler демонстрирует пакетное обновление метрик через /updates.
func ExampleServer_batchUpdateHandler() {
	ts, _ := newExampleServer()
	defer ts.Close()

	gaugeVal := 42.5
	counterDelta := int64(10)

	metrics := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &gaugeVal},
		{ID: "PollCount", MType: models.Counter, Delta: &counterDelta},
	}
	body, _ := json.Marshal(metrics)

	resp, err := http.Post(ts.URL+"/updates", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	// Output:
	// 200
}

// ExampleServer_pageHandler демонстрирует получение HTML-страницы со списком всех метрик.
func ExampleServer_pageHandler() {
	ts, s := newExampleServer()
	defer ts.Close()

	_ = s.storage.UpdateGauge(context.Background(), "Temp", 36.6)

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)
	fmt.Println(resp.Header.Get("Content-Type"))
	// Output:
	// 200
	// text/html
}
