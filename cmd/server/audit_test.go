package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// mockObserver records audit events for testing.
type mockObserver struct {
	events []AuditEvent
}

func (m *mockObserver) Notify(event AuditEvent) {
	m.events = append(m.events, event)
}

// mockLogger satisfies Logger interface for tests.
type mockLogger struct{}

func (mockLogger) Infow(string, ...any)          {}
func (mockLogger) Fatalf(string, ...interface{}) {}
func (mockLogger) Error(...interface{})          {}
func (mockLogger) Errorf(string, ...interface{}) {}

func TestAuditPublisher_Publish(t *testing.T) {
	pub := NewAuditPublisher(mockLogger{})
	obs1 := &mockObserver{}
	obs2 := &mockObserver{}
	pub.Register(obs1)
	pub.Register(obs2)

	event := AuditEvent{
		Ts:        1000,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "192.168.0.42",
	}

	pub.Publish(event)

	if len(obs1.events) != 1 {
		t.Fatalf("observer1 expected 1 event, got %d", len(obs1.events))
	}
	if len(obs2.events) != 1 {
		t.Fatalf("observer2 expected 1 event, got %d", len(obs2.events))
	}
	if obs1.events[0].IPAddress != "192.168.0.42" {
		t.Errorf("expected IP 192.168.0.42, got %s", obs1.events[0].IPAddress)
	}
	if len(obs1.events[0].Metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(obs1.events[0].Metrics))
	}
}

func TestAuditPublisher_NoObservers(t *testing.T) {
	pub := NewAuditPublisher(mockLogger{})
	// Should not panic with zero observers.
	pub.Publish(AuditEvent{Ts: 1, Metrics: []string{"m1"}, IPAddress: "1.2.3.4"})
}

func TestFileAuditObserver_Notify(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit-test-*.log")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	obs := NewFileAuditObserver(tmpFile.Name(), mockLogger{})

	event := AuditEvent{
		Ts:        1234567890,
		Metrics:   []string{"Alloc", "Frees"},
		IPAddress: "10.0.0.1",
	}
	obs.Notify(event)

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	var got AuditEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal audit line: %v (data: %s)", err, string(data))
	}
	if got.Ts != 1234567890 {
		t.Errorf("expected ts 1234567890, got %d", got.Ts)
	}
	if got.IPAddress != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %s", got.IPAddress)
	}
	if len(got.Metrics) != 2 || got.Metrics[0] != "Alloc" || got.Metrics[1] != "Frees" {
		t.Errorf("unexpected metrics: %v", got.Metrics)
	}
}

func TestFileAuditObserver_AppendsLines(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit-test-*.log")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	obs := NewFileAuditObserver(tmpFile.Name(), mockLogger{})

	obs.Notify(AuditEvent{Ts: 1, Metrics: []string{"m1"}, IPAddress: "1.1.1.1"})
	obs.Notify(AuditEvent{Ts: 2, Metrics: []string{"m2"}, IPAddress: "2.2.2.2"})

	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Should have exactly 2 newline-terminated JSON lines.
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("expected 2 lines, got %d (content: %s)", lines, string(data))
	}
}

func TestHTTPAuditObserver_Notify(t *testing.T) {
	var receivedBody []byte
	var receivedContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	obs := NewHTTPAuditObserver(srv.URL, mockLogger{})

	event := AuditEvent{
		Ts:        9999,
		Metrics:   []string{"TotalAlloc"},
		IPAddress: "172.16.0.5",
	}
	obs.Notify(event)

	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", receivedContentType)
	}

	var got AuditEvent
	if err := json.Unmarshal(receivedBody, &got); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if got.Ts != 9999 {
		t.Errorf("expected ts 9999, got %d", got.Ts)
	}
	if got.IPAddress != "172.16.0.5" {
		t.Errorf("expected IP 172.16.0.5, got %s", got.IPAddress)
	}
	if len(got.Metrics) != 1 || got.Metrics[0] != "TotalAlloc" {
		t.Errorf("unexpected metrics: %v", got.Metrics)
	}
}
