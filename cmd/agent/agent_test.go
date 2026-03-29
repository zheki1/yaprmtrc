package main

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"
)

func TestGzipPayload(t *testing.T) {
	input := []byte("hello world test payload")
	result, err := gzipPayload(input)
	if err != nil {
		t.Fatalf("gzipPayload: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
}

func TestGzipPayload_Empty(t *testing.T) {
	result, err := gzipPayload([]byte{})
	if err != nil {
		t.Fatalf("gzipPayload: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty result even for empty input")
	}
}

func TestIsRetryableNetErr_Nil(t *testing.T) {
	if isRetryableNetErr(nil) {
		t.Fatal("nil error should not be retryable")
	}
}

func TestIsRetryableNetErr_Canceled(t *testing.T) {
	if isRetryableNetErr(context.Canceled) {
		t.Fatal("context.Canceled should not be retryable")
	}
}

func TestIsRetryableNetErr_DeadlineExceeded(t *testing.T) {
	if !isRetryableNetErr(context.DeadlineExceeded) {
		t.Fatal("DeadlineExceeded should be retryable")
	}
}

func TestIsRetryableNetErr_NetError(t *testing.T) {
	err := &net.OpError{Op: "dial", Err: errors.New("connection refused")}
	if !isRetryableNetErr(err) {
		t.Fatal("net.OpError should be retryable")
	}
}

func TestIsRetryableNetErr_URLError(t *testing.T) {
	err := &url.Error{Op: "Get", URL: "http://localhost", Err: errors.New("fail")}
	if !isRetryableNetErr(err) {
		t.Fatal("url.Error should be retryable")
	}
}

func TestIsRetryableNetErr_GenericError(t *testing.T) {
	err := errors.New("some random error")
	if isRetryableNetErr(err) {
		t.Fatal("generic error should not be retryable")
	}
}

func TestCollectGopsutilMetrics(t *testing.T) {
	a := &Agent{
		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}

	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	a.logger = logger

	a.collectGopsutilMetrics()

	if _, ok := a.Gauge["TotalMemory"]; !ok {
		t.Error("expected TotalMemory gauge metric")
	}
	if _, ok := a.Gauge["FreeMemory"]; !ok {
		t.Error("expected FreeMemory gauge metric")
	}
}

func TestNewAgent(t *testing.T) {
	cfg := &Config{
		Addr:           "localhost:8080",
		PollInterval:   2,
		ReportInterval: 10,
		RateLimit:      1,
	}

	a := NewAgent(cfg)
	if a == nil {
		t.Fatal("expected non-nil agent")
	}
	if a.cfg != cfg {
		t.Fatal("expected config to match")
	}
	if a.Gauge == nil {
		t.Fatal("expected non-nil Gauge map")
	}
	if a.Counter == nil {
		t.Fatal("expected non-nil Counter map")
	}
}
