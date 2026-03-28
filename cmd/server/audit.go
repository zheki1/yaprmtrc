package main

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"time"
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
	observers []AuditObserver
	logger    Logger
}

// NewAuditPublisher creates a new AuditPublisher.
func NewAuditPublisher(logger Logger) *AuditPublisher {
	return &AuditPublisher{logger: logger}
}

// Register adds an observer to the publisher.
func (p *AuditPublisher) Register(o AuditObserver) {
	p.observers = append(p.observers, o)
}

// Publish sends an audit event to all registered observers.
func (p *AuditPublisher) Publish(event AuditEvent) {
	for _, o := range p.observers {
		o.Notify(event)
	}
}

// FileAuditObserver writes audit events as JSON lines to a file.
type FileAuditObserver struct {
	filePath string
	logger   Logger
}

// NewFileAuditObserver creates a new FileAuditObserver.
func NewFileAuditObserver(filePath string, logger Logger) *FileAuditObserver {
	return &FileAuditObserver{filePath: filePath, logger: logger}
}

// Notify appends the audit event as a JSON line to the configured file.
func (o *FileAuditObserver) Notify(event AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		o.logger.Errorf("audit file: marshal error: %v", err)
		return
	}

	f, err := os.OpenFile(o.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		o.logger.Errorf("audit file: open error: %v", err)
		return
	}
	defer f.Close()

	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		o.logger.Errorf("audit file: write error: %v", err)
	}
}

// HTTPAuditObserver sends audit events as JSON via HTTP POST to a remote URL.
type HTTPAuditObserver struct {
	url    string
	client *http.Client
	logger Logger
}

// NewHTTPAuditObserver creates a new HTTPAuditObserver.
func NewHTTPAuditObserver(url string, logger Logger) *HTTPAuditObserver {
	return &HTTPAuditObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
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

	resp, err := o.client.Post(o.url, "application/json", bytes.NewReader(data))
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
