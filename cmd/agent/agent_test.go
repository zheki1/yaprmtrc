package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zheki1/yaprmtrc/internal/models"
)

func readGzipBody(t *testing.T, r *http.Request) []byte {
	t.Helper()
	var reader io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("gzip reader error: %v", err)
		}
		defer gz.Close()
		reader = gz
	}
	b, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read body error: %v", err)
	}
	return b
}

func TestAgent_SendBatch(t *testing.T) {
	var called int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called, 1)

		if r.URL.Path != "/updates" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected content-type")
		}

		body := readGzipBody(t, r)

		var metrics []models.Metrics
		if err := json.Unmarshal(body, &metrics); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if len(metrics) == 0 {
			t.Fatalf("empty batch received")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	agent := NewAgent(&Config{
		Addr:           srv.Listener.Addr().String(),
		PollInterval:   10,
		ReportInterval: 50,
		RateLimit:      1,
	})

	metrics := []models.Metrics{{
		ID:    "HeapInuse",
		MType: models.Gauge,
		Value: ptrFloat(123),
	}}

	agent.sendBatch(metrics)

	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected 1 request, got %d", called)
	}
}

func TestAgent_WorkerPoolRateLimit(t *testing.T) {
	var maxParallel int32
	var current int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := atomic.AddInt32(&current, 1)
		for {
			mp := atomic.LoadInt32(&maxParallel)
			if c > mp && atomic.CompareAndSwapInt32(&maxParallel, mp, c) {
				break
			}
			if c <= mp {
				break
			}
		}

		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&current, -1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	agent := NewAgent(&Config{
		Addr:           srv.Listener.Addr().String(),
		PollInterval:   10,
		ReportInterval: 20,
		RateLimit:      2,
	})

	agent.startWorkers()

	for i := 0; i < 10; i++ {
		agent.metricsCh <- models.Metrics{
			ID:    "Alloc",
			MType: models.Gauge,
			Value: ptrFloat(float64(i)),
		}
	}

	time.Sleep(300 * time.Millisecond)

	if maxParallel > 2 {
		t.Fatalf("rate limit violated: %d", maxParallel)
	}
}

func TestAgent_NoEmptyBatch(t *testing.T) {
	called := int32(0)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	agent := NewAgent(&Config{
		Addr:           srv.Listener.Addr().String(),
		PollInterval:   10,
		ReportInterval: 30,
		RateLimit:      1,
	})

	agent.startWorkers()

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&called) != 0 {
		t.Fatalf("empty batch should not be sent")
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}
