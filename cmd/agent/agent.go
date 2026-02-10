package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/retry"
	"github.com/zheki1/yaprmtrc/internal/security"
)

type Agent struct {
	cfg    *Config
	client *resty.Client

	Gauge   map[string]float64
	Counter map[string]int64
}

func NewAgent(cfg *Config) *Agent {
	return &Agent{
		cfg:    cfg,
		client: resty.New().SetBaseURL("http://" + cfg.Addr).SetTimeout(5 * time.Second),

		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}
}

func (a *Agent) Start() {
	fmt.Printf("Agent started. Server=%s, poll=%ds, report=%ds\n",
		a.cfg.Addr, a.cfg.PollInterval, a.cfg.ReportInterval)

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

	// tickerPoll := time.NewTicker(time.Duration(a.cfg.PollInterval) * time.Second)
	// defer tickerPoll.Stop()

	// tickerReport := time.NewTicker(time.Duration(a.cfg.ReportInterval) * time.Second)
	// defer tickerReport.Stop()

	// for {
	// 	select {
	// 	case <-tickerPoll.C:
	// 		fmt.Printf("%s \n", "collect "+time.Now().String())
	// 		a.collectMetrics()

	// 	case <-tickerReport.C:
	// 		fmt.Printf("%s \n", "send all "+time.Now().String())
	// 		a.sendAllMetrics()
	// 	}
	// }
}

func (a *Agent) sendAllMetrics(jobs chan<- Job) {
	fmt.Printf("%s \n", "send all metrics "+time.Now().String())
	for name, value := range a.Gauge {
		jobs <- func() error {
			return a.sendMetric(models.Metrics{
				ID:    name,
				MType: models.Gauge,
				Value: &value,
			})
		}
	}

	for name, value := range a.Counter {
		jobs <- func() error {
			return a.sendMetric(models.Metrics{
				ID:    name,
				MType: models.Counter,
				Delta: &value,
			})
		}
	}
}

func (a *Agent) sendMetric(metric models.Metrics) error {
	if err := retry.DoRetry(context.Background(), isRetryableNetErr, func() error {
		payload, err := json.Marshal(metric)
		if err != nil {
			return err
		}

		body, err := gzipPayload(payload)
		if err != nil {
			return err
		}

		req := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(body)

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
		log.Print("failed sending metric")
		return err
	}

	return nil
}

func (a *Agent) sendBatch(metrics []models.Metrics) {
	if err := retry.DoRetry(context.Background(), isRetryableNetErr, func() error {
		payload, err := json.Marshal(metrics)
		if err != nil {
			return err
		}

		body, err := gzipPayload(payload)
		if err != nil {
			return err
		}

		req := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(body)

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
		log.Print("failed sending metric")
	}
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
