package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGzipMiddleware_WithAcceptEncoding(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Fatal("expected gzip content encoding")
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gr.Close()

	body, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if string(body) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(body))
	}
}

func TestGzipMiddleware_WithoutAcceptEncoding(t *testing.T) {
	handler := GzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.Header.Get("Content-Encoding") == "gzip" {
		t.Fatal("should not be gzip encoded")
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello world" {
		t.Fatalf("expected 'hello world', got %q", string(body))
	}
}

func TestHashMiddleware_NoKey(t *testing.T) {
	handler := HashMiddleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHashMiddleware_WithKey(t *testing.T) {
	handler := HashMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("body"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("HashSHA256") == "" {
		t.Fatal("expected HashSHA256 header")
	}
}

func TestHashMiddleware_WithKeyAndHash(t *testing.T) {
	handler := HashMiddleware("secret")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("body"))
	req.Header.Set("HashSHA256", "somehash")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	logger := &testLogger{}

	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if len(logger.calls) == 0 {
		t.Fatal("expected logger to be called")
	}
}

func TestLoggingMiddleware_Error(t *testing.T) {
	logger := &testLogger{}

	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if len(logger.calls) == 0 {
		t.Fatal("expected logger to be called")
	}
}

func TestResponseRecorder(t *testing.T) {
	rec := NewRecorder(nil)

	rec.Header().Set("X-Test", "value")
	if rec.Header().Get("X-Test") != "value" {
		t.Fatal("expected header")
	}

	rec.WriteHeader(http.StatusCreated)
	// double write should be ignored
	rec.WriteHeader(http.StatusOK)

	n, err := rec.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != 5 {
		t.Fatalf("expected 5, got %d", n)
	}

	if string(rec.Body()) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(rec.Body()))
	}

	w := httptest.NewRecorder()
	rec.FlushTo(w)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

func TestResponseRecorder_DefaultStatus(t *testing.T) {
	rec := NewRecorder(nil)
	rec.Write([]byte("data"))

	w := httptest.NewRecorder()
	rec.FlushTo(w)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestResponseRecorder_FlushTo_NoBody(t *testing.T) {
	rec := NewRecorder(nil)

	w := httptest.NewRecorder()
	rec.FlushTo(w)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

type testLogger struct {
	calls []string
}

func (m *testLogger) Infow(msg string, fields ...any) {
	m.calls = append(m.calls, msg)
}

func (m *testLogger) Fatalf(template string, args ...interface{}) {
	m.calls = append(m.calls, template)
}

func (m *testLogger) Error(args ...interface{}) {
	m.calls = append(m.calls, "error")
}

func (m *testLogger) Errorf(template string, args ...interface{}) {
	m.calls = append(m.calls, template)
}

func TestLoggingResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	lrw := &loggingResponseWriter{ResponseWriter: w}

	n, err := lrw.Write([]byte("test"))
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4, got %d", n)
	}
	if lrw.status != http.StatusOK {
		t.Fatalf("expected default 200, got %d", lrw.status)
	}
	if lrw.size != 4 {
		t.Fatalf("expected size 4, got %d", lrw.size)
	}
}

func TestLoggingResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	lrw := &loggingResponseWriter{ResponseWriter: w}

	lrw.WriteHeader(http.StatusNotFound)

	if lrw.status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", lrw.status)
	}
}
