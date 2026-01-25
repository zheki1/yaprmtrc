package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/zheki1/yaprmtrc.git/internal/models"
)

type Agent struct {
	cfg    *Config
	client *http.Client

	Gauge   map[string]float64
	Counter map[string]int64
}

func NewAgent(cfg *Config) *Agent {
	return &Agent{
		cfg:    cfg,
		client: &http.Client{},

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
		a.sendMetric(metric, true)
	}

	for name, value := range a.Counter {
		metric := models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: &value,
		}
		a.sendMetric(metric, true)
	}
}

func (a *Agent) sendMetric(metric models.Metrics, compressReq bool) {
	bodyJSON, err := json.Marshal(metric)
	if err != nil {
		log.Printf("Failed serializing metric %s/%s: %v\n", metric.MType, metric.ID, err)
		return
	}

	var buf bytes.Buffer
	if compressReq {
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(bodyJSON); err != nil {
			log.Printf("Failed gzip metric %s/%s: %v\n", metric.MType, metric.ID, err)
		}
		gz.Close()
	} else {
		buf.Write(bodyJSON)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://%s/update", a.cfg.Addr),
		&buf,
	)
	if err != nil {
		log.Printf("Cannot prepare request for metric %s/%s: %v\n", metric.MType, metric.ID, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if compressReq {
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("Failed sending metric %s/%s: %v\n", metric.MType, metric.ID, err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Server returned status %d for metric %s/%s", resp.StatusCode, metric.MType, metric.ID)
	}
}

func (a *Agent) pushMetricGZIP(metric models.Metrics) {
	bodyJSON, err := json.Marshal(metric)
	if err != nil {
		log.Printf("Failed serializing metric %s/%s: %v\n", metric.MType, metric.ID, err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(bodyJSON); err != nil {
		log.Printf("Failed gzip metric %s/%s: %v\n", metric.MType, metric.ID, err)
	}
	gz.Close()

	req, err := http.NewRequest(http.MethodPost, "http://"+a.cfg.Addr+"/update", &buf)
	if err != nil {
		log.Printf("Cannot prepare request for metric %s/%s: %v\n", metric.MType, metric.ID, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("Failed sending metric %s/%s: %v\n", metric.MType, metric.ID, err)
	}
	defer resp.Body.Close()
}

func (a *Agent) sendBatch(metrics []models.Metrics, compressReq bool) {

	bodyJSON, err := json.Marshal(metrics)
	if err != nil {
		log.Printf("Failed serializing batch metric: %v\n", err)
		return
	}

	var buf bytes.Buffer
	if compressReq {
		gz := gzip.NewWriter(&buf)
		if _, err := gz.Write(bodyJSON); err != nil {
			log.Printf("Failed gzip metric: %v\n", err)
		}
		gz.Close()
	} else {
		buf.Write(bodyJSON)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("http://%s/updates", a.cfg.Addr),
		&buf,
	)
	if err != nil {
		log.Printf("Cannot prepare request for batch send metrics: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if compressReq {
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("Failed sending batch metrics: %v\n", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Server returned status %d for batch metrics", resp.StatusCode)
	}
}
