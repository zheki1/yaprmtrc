package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/retry"
	"github.com/zheki1/yaprmtrc/internal/security"
	"go.uber.org/zap"
)

// Agent — агент сбора метрик. Периодически собирает runtime- и gopsutil-метрики
// и отправляет их на сервер пакетно (через /updates).
type Agent struct {
	cfg     *Config
	client  *resty.Client
	logger  *zap.SugaredLogger
	agentIP string

	Gauge   map[string]float64
	Counter map[string]int64
}

// NewAgent создаёт новый агент с указанной конфигурацией.
func NewAgent(cfg *Config) (*Agent, error) {
	logger, err := NewLogger()
	if err != nil {
		return nil, fmt.Errorf("cannot init logger: %w", err)
	}

	agentIP := getAgentIP()

	return &Agent{
		cfg:     cfg,
		client:  resty.New().SetBaseURL("http://" + cfg.Addr).SetTimeout(5 * time.Second),
		logger:  logger,
		agentIP: agentIP,

		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}, nil
}

// getAgentIP получает IP адрес хоста агента
func getAgentIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP.To4()
		if ip != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
			return ip.String()
		}
	}

	return "127.0.0.1"
}

// Start запускает циклы сбора и отправки метрик. Блокирует вызывающую горутину.
func (a *Agent) Start() {
	a.logger.Infoln(fmt.Sprintf("Agent started. Server=%s, poll=%ds, report=%ds\n",
		a.cfg.Addr, a.cfg.PollInterval, a.cfg.ReportInterval))

	jobs := make(chan Job, a.cfg.RateLimit)
	StartWorkers(a.cfg.RateLimit, jobs)

	go func() {
		ticker := time.NewTicker(time.Duration(a.cfg.PollInterval) * time.Second)
		defer ticker.Stop()

		a.collectRuntimeMetrics()
		for range ticker.C {
			a.collectRuntimeMetrics()
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Duration(a.cfg.PollInterval) * time.Second)
		defer ticker.Stop()

		a.collectGopsutilMetrics()
		for range ticker.C {
			a.collectGopsutilMetrics()
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Duration(a.cfg.ReportInterval) * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			a.sendAllMetrics(jobs)
		}
	}()

	select {}

	// TODO - use sync.WaitGroup to wait for workers to finish (on graceful shutdown implementing)
}

func (a *Agent) sendAllMetrics(jobs chan<- Job) {
	a.logger.Infoln("send all metrics " + time.Now().String())

	metrics := make([]models.Metrics, 0, len(a.Gauge)+len(a.Counter))
	for name, value := range a.Gauge {
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		})
	}
	for name, delta := range a.Counter {
		metrics = append(metrics, models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &delta,
		})
	}

	jobs <- func() error {
		return a.sendBatch(metrics)
	}
}

func (a *Agent) sendMetric(metric models.Metrics) error {
	if err := retry.DoRetry(context.Background(), isRetryableNetErr, func() error {
		payload, err := json.Marshal(metric)
		if err != nil {
			return err
		}

		body := payload
		if a.cfg.CryptoKey != "" {
			pubKey, err := security.LoadPublicKey(a.cfg.CryptoKey)
			if err != nil {
				return fmt.Errorf("failed to load public key: %w", err)
			}
			body, err = security.EncryptHybrid(payload, pubKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt payload: %w", err)
			}
		}

		body, err = gzipPayload(body)
		if err != nil {
			return err
		}

		req := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("X-Real-IP", a.agentIP).
			SetBody(body)

		if a.cfg.CryptoKey != "" {
			req.SetHeader("Encrypted", "true")
		}

		if a.cfg.Key != "" {
			req.SetHeader("HashSHA256", security.CalcHash(payload, a.cfg.Key))
		}

		resp, err := req.Post("/update")
		if err != nil {
			return err
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("bad status: %s", resp.Status())
		}
		return nil
	}); err != nil {
		a.logger.Info("failed sending metric")
		return err
	}

	return nil
}

func (a *Agent) sendBatch(metrics []models.Metrics) error {
	if err := retry.DoRetry(context.Background(), isRetryableNetErr, func() error {
		payload, err := json.Marshal(metrics)
		if err != nil {
			return err
		}

		body := payload
		if a.cfg.CryptoKey != "" {
			pubKey, err := security.LoadPublicKey(a.cfg.CryptoKey)
			if err != nil {
				return fmt.Errorf("failed to load public key: %w", err)
			}
			body, err = security.EncryptHybrid(payload, pubKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt payload: %w", err)
			}
		}

		body, err = gzipPayload(body)
		if err != nil {
			return err
		}

		req := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetHeader("X-Real-IP", a.agentIP).
			SetBody(body)

		if a.cfg.CryptoKey != "" {
			req.SetHeader("Encrypted", "true")
		}

		if a.cfg.Key != "" {
			req.SetHeader("HashSHA256", security.CalcHash(payload, a.cfg.Key))
		}

		resp, err := req.Post("/updates")
		if err != nil {
			return err
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("bad status: %s", resp.Status())
		}
		return nil
	}); err != nil {
		a.logger.Info("failed sending metric")
		return err
	}

	return nil
}

func gzipPayload(payload []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(payload); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var urlErr *url.Error
	return errors.As(err, &urlErr)
}
