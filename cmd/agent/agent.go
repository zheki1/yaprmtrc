package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/url"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/retry"
	"github.com/zheki1/yaprmtrc/internal/security"
)

type Agent struct {
	cfg       *Config
	client    *resty.Client
	metricsCh chan models.Metrics
	stopCh    chan struct{}

	Gauge   map[string]float64
	Counter map[string]int64
}

func NewAgent(cfg *Config) *Agent {
	return &Agent{
		cfg:    cfg,
		client: resty.New().SetBaseURL("http://" + cfg.Addr).SetTimeout(5 * time.Second),

		metricsCh: make(chan models.Metrics, 64),
		stopCh:    make(chan struct{}),

		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}
}

func (a *Agent) Start() {
	fmt.Printf("Agent started. Server=%s, poll=%ds, report=%ds\n",
		a.cfg.Addr, a.cfg.PollInterval, a.cfg.ReportInterval)

	go a.collectRuntimeMetrics()
	go a.collectGopsutilMetrics()

	a.startWorkers()

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

func (a *Agent) collectRuntimeMetrics() {
	var r runtime.MemStats
	runtime.ReadMemStats(&r)
	fmt.Printf("%s \n", "collect runtime metrics "+time.Now().String())

	a.pushGauge("Alloc", float64(r.Alloc))
	a.pushGauge("BuckHashSys", float64(r.BuckHashSys))
	a.pushGauge("Frees", float64(r.Frees))
	a.pushGauge("GCCPUFraction", r.GCCPUFraction)
	a.pushGauge("GCSys", float64(r.GCSys))
	a.pushGauge("HeapAlloc", float64(r.HeapAlloc))
	a.pushGauge("HeapIdle", float64(r.HeapIdle))
	a.pushGauge("HeapInuse", float64(r.HeapInuse))
	a.pushGauge("HeapObjects", float64(r.HeapObjects))
	a.pushGauge("HeapReleased", float64(r.HeapReleased))
	a.pushGauge("HeapSys", float64(r.HeapSys))
	a.pushGauge("LastGC", float64(r.LastGC))
	a.pushGauge("Lookups", float64(r.Lookups))
	a.pushGauge("MCacheInuse", float64(r.MCacheInuse))
	a.pushGauge("MCacheSys", float64(r.MCacheSys))
	a.pushGauge("MSpanInuse", float64(r.MSpanInuse))
	a.pushGauge("MSpanSys", float64(r.MSpanSys))
	a.pushGauge("Mallocs", float64(r.Mallocs))
	a.pushGauge("NextGC", float64(r.NextGC))
	a.pushGauge("NumForcedGC", float64(r.NumForcedGC))
	a.pushGauge("NumGC", float64(r.NumGC))
	a.pushGauge("OtherSys", float64(r.OtherSys))
	a.pushGauge("PauseTotalNs", float64(r.PauseTotalNs))
	a.pushGauge("StackInuse", float64(r.StackInuse))
	a.pushGauge("StackSys", float64(r.StackSys))
	a.pushGauge("Sys", float64(r.Sys))
	a.pushGauge("TotalAlloc", float64(r.TotalAlloc))

	a.pushGauge("RandomValue", rand.Float64())

	a.pushCounter("PollCount", 1)
}

func (a *Agent) collectGopsutilMetrics() {
	ticker := time.NewTicker(time.Duration(a.cfg.PollInterval))
	defer ticker.Stop()
	fmt.Printf("%s \n", "collect gopsutil metrics "+time.Now().String())

	for range ticker.C {
		vm, err := mem.VirtualMemory()
		if err == nil {
			a.pushGauge("TotalMemory", float64(vm.Total))
			a.pushGauge("FreeMemory", float64(vm.Free))
		}

		cpuPercents, err := cpu.Percent(0, true)
		if err == nil {
			for i, p := range cpuPercents {
				a.pushGauge(
					fmt.Sprintf("CPUutilization%d", i+1),
					p,
				)
			}
		}
	}
}

func (a *Agent) pushGauge(name string, v float64) {
	val := v
	a.metricsCh <- models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &val,
	}
}

func (a *Agent) pushCounter(name string, d int64) {
	delta := d
	a.metricsCh <- models.Metrics{
		ID:    name,
		MType: models.Counter,
		Delta: &delta,
	}
}

func (a *Agent) sendAllMetrics() {
	for name, value := range a.Gauge {
		metric := models.Metrics{
			ID:    name,
			MType: models.Gauge,
			Value: &value,
		}
		a.sendMetric(metric)
	}

	for name, value := range a.Counter {
		metric := models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &value,
		}
		a.sendMetric(metric)
	}
}

func (a *Agent) startWorkers() {
	for i := 0; i < a.cfg.RateLimit; i++ {
		go a.worker()
	}
}

func (a *Agent) worker() {
	batch := make([]models.Metrics, 0)

	timer := time.NewTicker(time.Duration(a.cfg.ReportInterval))
	defer timer.Stop()

	for {
		select {
		case m := <-a.metricsCh:
			batch = append(batch, m)

		case <-timer.C:
			if len(batch) == 0 {
				continue
			}

			a.sendBatch(batch)
			batch = nil
		}
	}
}

func (a *Agent) sendMetric(metric models.Metrics) {
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
	}
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
