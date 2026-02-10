package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/zheki1/yaprmtrc.git/internal/models"
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

	tickerPoll := time.NewTicker(time.Duration(a.cfg.PollInterval) * time.Second)
	defer tickerPoll.Stop()

	tickerReport := time.NewTicker(time.Duration(a.cfg.ReportInterval) * time.Second)
	defer tickerReport.Stop()

	for {
		select {
		case <-tickerPoll.C:
			fmt.Printf("%s \n", "collect "+time.Now().String())
			a.collectMetrics()

		case <-tickerReport.C:
			fmt.Printf("%s \n", "send all "+time.Now().String())
			a.sendAllMetrics()
		}
	}
}

func (a *Agent) collectMetrics() {
	var r runtime.MemStats
	runtime.ReadMemStats(&r)

	// Gauge metrics
	a.Gauge["Alloc"] = float64(r.Alloc)
	a.Gauge["BuckHashSys"] = float64(r.BuckHashSys)
	a.Gauge["Frees"] = float64(r.Frees)
	a.Gauge["GCCPUFraction"] = r.GCCPUFraction
	a.Gauge["GCSys"] = float64(r.GCSys)
	a.Gauge["HeapAlloc"] = float64(r.HeapAlloc)
	a.Gauge["HeapIdle"] = float64(r.HeapIdle)
	a.Gauge["HeapInuse"] = float64(r.HeapInuse)
	a.Gauge["HeapObjects"] = float64(r.HeapObjects)
	a.Gauge["HeapReleased"] = float64(r.HeapReleased)
	a.Gauge["HeapSys"] = float64(r.HeapSys)
	a.Gauge["LastGC"] = float64(r.LastGC)
	a.Gauge["Lookups"] = float64(r.Lookups)
	a.Gauge["MCacheInuse"] = float64(r.MCacheInuse)
	a.Gauge["MCacheSys"] = float64(r.MCacheSys)
	a.Gauge["MSpanInuse"] = float64(r.MSpanInuse)
	a.Gauge["MSpanSys"] = float64(r.MSpanSys)
	a.Gauge["Mallocs"] = float64(r.Mallocs)
	a.Gauge["NextGC"] = float64(r.NextGC)
	a.Gauge["NumForcedGC"] = float64(r.NumForcedGC)
	a.Gauge["NumGC"] = float64(r.NumGC)
	a.Gauge["OtherSys"] = float64(r.OtherSys)
	a.Gauge["PauseTotalNs"] = float64(r.PauseTotalNs)
	a.Gauge["StackInuse"] = float64(r.StackInuse)
	a.Gauge["StackSys"] = float64(r.StackSys)
	a.Gauge["Sys"] = float64(r.Sys)
	a.Gauge["TotalAlloc"] = float64(r.TotalAlloc)

	// RandomValue gauge
	a.Gauge["RandomValue"] = rand.Float64()

	// Counter
	a.Counter["PollCount"]++
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

func (a *Agent) sendMetric(metric models.Metrics) {
	if err := DoRetry(context.Background(), func() error {
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

		resp, err := req.Post("/update")
		if err != nil {
			return err
		}

		if !resp.IsSuccess() {
			return fmt.Errorf("bad status: %s", resp.Status())
		}

		// req, err := http.NewRequest(
		// 	http.MethodPost,
		// 	fmt.Sprintf("http://%s/update", a.cfg.Addr),
		// 	&buf,
		// )
		// if err != nil {
		// 	return fmt.Errorf("cannot prepare request for metric %s/%s: %w", metric.MType, metric.ID, err)
		// }

		// req.Header.Set("Content-Type", "application/json")
		// if compressReq {
		// 	req.Header.Set("Content-Encoding", "gzip")
		// 	req.Header.Set("Accept-Encoding", "gzip")
		// }

		// resp, err := a.client.Do(req)
		// if err != nil {
		// 	return fmt.Errorf("failed sending metric %s/%s: %w", metric.MType, metric.ID, err)
		// }
		// defer resp.Body.Close()

		// if resp.StatusCode != http.StatusOK {
		// 	return fmt.Errorf("server returned status %d for metric %s/%s", resp.StatusCode, metric.MType, metric.ID)
		// }
		return nil
	}); err != nil {
		log.Print("failed sending metric")
	}
}

func (a *Agent) sendBatch(metrics []models.Metrics, compressReq bool) {
	// runWithRetries(func() error {
	// 	bodyJSON, err := json.Marshal(metrics)
	// 	if err != nil {
	// 		return fmt.Errorf("failed serializing batch metric: %w", err)
	// 	}

	// 	var buf bytes.Buffer
	// 	if compressReq {
	// 		gz := gzip.NewWriter(&buf)
	// 		if _, err := gz.Write(bodyJSON); err != nil {
	// 			return fmt.Errorf("failed gzip metric: %w", err)
	// 		}
	// 		gz.Close()
	// 	} else {
	// 		buf.Write(bodyJSON)
	// 	}

	// 	req, err := http.NewRequest(
	// 		http.MethodPost,
	// 		fmt.Sprintf("http://%s/updates", a.cfg.Addr),
	// 		&buf,
	// 	)
	// 	if err != nil {
	// 		return fmt.Errorf("cannot prepare request for batch send metrics: %w", err)
	// 	}

	// 	req.Header.Set("Content-Type", "application/json")
	// 	if compressReq {
	// 		req.Header.Set("Content-Encoding", "gzip")
	// 		req.Header.Set("Accept-Encoding", "gzip")
	// 	}

	// 	resp, err := a.client.Do(req)
	// 	if err != nil {
	// 		return fmt.Errorf("failed sending batch metrics: %w", err)
	// 	}
	// 	defer resp.Body.Close()

	// 	if resp.StatusCode != http.StatusOK {
	// 		return fmt.Errorf("server returned status %d for batch metrics", resp.StatusCode)
	// 	}
	// 	return nil
	// })
}

// func runWithRetries(fn func() error) {
// 	var lastErr error

// 	for i := 0; i <= len(retryDelays); i++ {
// 		err := fn()
// 		log.Printf("Retry attempt number start: %v %v %v", i, len(retryDelays), err)
// 		if err == nil {
// 			log.Printf("Retry attempt number successful: %v %v", i, len(retryDelays))
// 			return
// 		}

// 		var netErr net.Error

// 		if errors.As(err, &netErr) && netErr.Timeout() {
// 			log.Printf("Go to retry %v", err)
// 		} else if strings.Contains(err.Error(), "connection refused") ||
// 			strings.Contains(err.Error(), "connection reset") ||
// 			strings.Contains(err.Error(), "EOF") {
// 			log.Printf("Go to retry %v", err)
// 		} else {
// 			log.Printf("Retry err %v", err)
// 			return
// 		}

// 		if i < len(retryDelays) {
// 			log.Printf("Retry attempt number end: %v %v %v", i, len(retryDelays), err)
// 			time.Sleep(retryDelays[i])
// 		}
// 	}

// 	log.Printf("retry attempts failed: %v", lastErr)
// }

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
