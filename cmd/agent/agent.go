package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"
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
			fmt.Printf("%s", "collect "+time.Now().String())
			a.collectMetrics()

		case <-tickerReport.C:
			fmt.Printf("%s", "send all "+time.Now().String())
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
		a.sendMetric("gauge", name, fmt.Sprintf("%f", value))
	}

	for name, value := range a.Counter {
		a.sendMetric("counter", name, strconv.FormatInt(value, 10))
	}
}

func (a *Agent) sendMetric(metricType, name, value string) {
	url := fmt.Sprintf("http://%s/update/%s/%s/%s",
		a.cfg.Addr, metricType, name, value)

	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("Failed sending metric %s/%s: %v\n", metricType, name, err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Server returned status %d for metric %s/%s", resp.StatusCode, metricType, name)
	}
}
