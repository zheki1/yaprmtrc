package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zheki1/yaprmtrc.git/internal/models"
)

func newTestAgent(addr string) *Agent {

	cfg := &Config{
		Addr:           addr,
		PollInterval:   1,
		ReportInterval: 1,
	}

	return NewAgent(cfg)
}

func readGzipBody(t *testing.T, r io.Reader) []byte {

	gr, err := gzip.NewReader(r)
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()

	b, err := io.ReadAll(gr)
	if err != nil {
		t.Fatal(err)
	}

	return b
}

func TestCollectMetrics(t *testing.T) {

	a := newTestAgent("localhost:1234")

	a.collectMetrics()

	if len(a.Gauge) == 0 {
		t.Fatal("gauge metrics empty")
	}

	if len(a.Counter) == 0 {
		t.Fatal("counter metrics empty")
	}

	if _, ok := a.Gauge["Alloc"]; !ok {
		t.Fatal("Alloc not collected")
	}

	if a.Counter["PollCount"] != 1 {
		t.Fatal("PollCount not incremented")
	}
}

func TestSendMetric(t *testing.T) {

	var received models.Metrics

	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			if r.URL.Path != "/update" {
				t.Fatalf("wrong path %s", r.URL.Path)
			}

			var body []byte

			if r.Header.Get("Content-Encoding") == "gzip" {
				body = readGzipBody(t, r.Body)
			} else {
				body, _ = io.ReadAll(r.Body)
			}

			json.Unmarshal(body, &received)

			w.WriteHeader(http.StatusOK)
		},
	))
	defer ts.Close()

	a := newTestAgent(ts.Listener.Addr().String())

	m := models.Metrics{
		ID:    "Alloc",
		MType: models.Gauge,
		Value: ptrFloat(10.5),
	}

	a.sendMetric(m, true)

	if received.ID != "Alloc" {
		t.Fatal("metric not received")
	}
}

func TestSendAllMetrics(t *testing.T) {

	count := 0

	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			count++
			w.WriteHeader(http.StatusOK)
		},
	))
	defer ts.Close()

	a := newTestAgent(ts.Listener.Addr().String())

	a.Gauge["A"] = 1.1
	a.Gauge["B"] = 2.2
	a.Counter["C"] = 3

	a.sendAllMetrics()

	if count != 3 {
		t.Fatalf("expected 3 metrics, got %d", count)
	}
}

func TestSendBatch(t *testing.T) {

	var received []models.Metrics

	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			if r.URL.Path != "/updates" {
				t.Fatal("wrong path")
			}

			body, _ := io.ReadAll(r.Body)

			json.Unmarshal(body, &received)

			w.WriteHeader(http.StatusOK)
		},
	))
	defer ts.Close()

	a := newTestAgent(ts.Listener.Addr().String())

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

	a.sendBatch(data, false)

	if len(received) != 2 {
		t.Fatal("batch not received")
	}
}

func TestSendBatchGzip(t *testing.T) {

	var received []models.Metrics

	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			if r.Header.Get("Content-Encoding") != "gzip" {
				t.Fatal("gzip missing")
			}

			body := readGzipBody(t, r.Body)

			json.Unmarshal(body, &received)

			w.WriteHeader(http.StatusOK)
		},
	))
	defer ts.Close()

	a := newTestAgent(ts.Listener.Addr().String())

	data := []models.Metrics{
		{
			ID:    "A",
			MType: models.Gauge,
			Value: ptrFloat(1.1),
		},
	}

	a.sendBatch(data, true)

	if len(received) != 1 {
		t.Fatal("gzip batch failed")
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}

func ptrInt(v int64) *int64 {
	return &v
}
