package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

// AuditEvent represents an audit log entry emitted after successful metric processing.
type AuditEvent struct {
	Ts        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

// AuditObserver is the Observer interface for receiving audit events.
type AuditObserver interface {
	Notify(event AuditEvent)
}

// AuditPublisher manages a list of observers and publishes events to all of them.
type AuditPublisher struct {
	mu        sync.RWMutex
	observers []AuditObserver
	logger    Logger
}

// NewAuditPublisher creates a new AuditPublisher.
func NewAuditPublisher(logger Logger) *AuditPublisher {
	return &AuditPublisher{logger: logger}
}

// Register adds an observer to the publisher.
func (p *AuditPublisher) Register(o AuditObserver) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.observers = append(p.observers, o)
}

// Publish sends an audit event to all registered observers.
func (p *AuditPublisher) Publish(event AuditEvent) {
	p.mu.RLock()
	observers := make([]AuditObserver, len(p.observers))
	copy(observers, p.observers)
	p.mu.RUnlock()

	for _, o := range observers {
		o.Notify(event)
	}
}

// FileAuditObserver writes audit events as JSON lines to a file.
type FileAuditObserver struct {
	mu     sync.Mutex
	file   *os.File
	logger Logger
}

// NewFileAuditObserver creates a new FileAuditObserver with an already-opened file.
func NewFileAuditObserver(filePath string, logger Logger) (*FileAuditObserver, error) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("audit file: open error: %w", err)
	}
	return &FileAuditObserver{file: f, logger: logger}, nil
}

// Notify appends the audit event as a JSON line to the configured file.
func (o *FileAuditObserver) Notify(event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		o.logger.Errorf("audit file: marshal error: %v", err)
		return
	}

	data = append(data, '\n')

	o.mu.Lock()
	defer o.mu.Unlock()

	if _, err := o.file.Write(data); err != nil {
		o.logger.Errorf("audit file: write error: %v", err)
	}
}

// Close closes the underlying file.
func (o *FileAuditObserver) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.file.Close()
}

// HTTPAuditObserver sends audit events as JSON via HTTP POST to a remote URL.
type HTTPAuditObserver struct {
	url    string
	client *retryablehttp.Client
	logger Logger
}

// NewHTTPAuditObserver creates a new HTTPAuditObserver.
func NewHTTPAuditObserver(url string, logger Logger) *HTTPAuditObserver {
	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.RetryWaitMin = 100 * time.Millisecond
	client.RetryWaitMax = 1 * time.Second

	// логирование через твой logger
	client.Logger = nil // отключаем стандартный лог, если не нужен

	return &HTTPAuditObserver{
		url:    url,
		client: client,
		logger: logger,
	}
}

// Notify sends the audit event as a JSON POST request to the configured URL.
func (o *HTTPAuditObserver) Notify(event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		o.logger.Errorf("audit http: marshal error: %v", err)
		return
	}

	req, err := retryablehttp.NewRequest("POST", o.url, bytes.NewReader(data))
	if err != nil {
		o.logger.Errorf("audit http: request error: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		o.logger.Errorf("audit http: post error: %v", err)
		return
	}
	defer resp.Body.Close()
}

// notifyAudit builds an AuditEvent from the request and metric names, then publishes it.
func (s *Server) notifyAudit(r *http.Request, metricNames []string) {
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}

	event := AuditEvent{
		Ts:        time.Now().Unix(),
		Metrics:   metricNames,
		IPAddress: ip,
	}

	s.audit.Publish(event)
}
