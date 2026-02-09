package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zheki1/yaprmtrc.git/internal/models"
)

// вспомогательная функция для разархивации gzip
func gunzipBody(body []byte) ([]byte, error) {
	buf := bytes.NewReader(body)
	r, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// внутренняя функция, которая возвращает ошибку
func (a *Agent) sendMetricForTest(metric models.Metrics, compress bool) error {
	var metrics []models.Metrics
	metrics = append(metrics, metric)
	bodyJSON, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("http://%s/updates", a.cfg.Addr)
	return sendWithResty(a.client, url, bodyJSON, compress, a.cfg.Key)
}

// тест отправки одной метрики через Resty
func TestSendMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/updates", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var data []byte
		var err error
		if r.Header.Get("Content-Encoding") == "gzip" {
			body, _ := io.ReadAll(r.Body)
			data, err = gunzipBody(body)
			assert.NoError(t, err)
		} else {
			data, err = io.ReadAll(r.Body)
			assert.NoError(t, err)
		}

		var metrics []models.Metrics
		err = json.Unmarshal(data, &metrics)
		assert.NoError(t, err)
		assert.Len(t, metrics, 1)
		assert.Equal(t, "RandomValue", metrics[0].ID)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &Config{
		Addr: server.Listener.Addr().String(),
		Key:  "",
	}
	agent := NewAgent(cfg)

	metric := models.Metrics{
		ID:    "RandomValue",
		MType: models.Gauge,
		Value: new(float64),
	}

	err := agent.sendMetricForTest(metric, true)
	assert.NoError(t, err)
}

// тест отправки пачки метрик
// тест отправки пачки метрик
func TestSendBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/updates", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var data []byte
		if r.Header.Get("Content-Encoding") == "gzip" {
			data, err = gunzipBody(body)
			assert.NoError(t, err)
		} else {
			data = body
		}

		var metrics []models.Metrics
		err = json.Unmarshal(data, &metrics)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(metrics), 2) // теперь действительно отправляем 2 метрики
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &Config{
		Addr: server.Listener.Addr().String(),
		Key:  "",
	}
	agent := NewAgent(cfg)

	// отправляем сразу всю пачку
	metrics := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: new(float64)},
		{ID: "PollCount", MType: models.Counter, Delta: new(int64)},
	}

	bodyJSON, _ := json.Marshal(metrics)
	err := sendWithResty(agent.client, fmt.Sprintf("http://%s/updates", cfg.Addr), bodyJSON, true, "")
	assert.NoError(t, err)
}

// тест HMAC заголовка
func TestSendMetricWithHMAC(t *testing.T) {
	key := "secret123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header.Get("HashSHA256"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &Config{
		Addr: server.Listener.Addr().String(),
		Key:  key,
	}
	agent := NewAgent(cfg)

	metric := models.Metrics{
		ID:    "RandomValue",
		MType: models.Gauge,
		Value: new(float64),
	}

	err := agent.sendMetricForTest(metric, false)
	assert.NoError(t, err)
}
