package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Metrics struct {
	Gauge   map[string]float64
	Counter map[string]int64
}

var serverAddr *string

func main() {
	serverAddr = flag.String("a", "localhost:8080", "Address of metrics server")
	reportInterval := flag.Int("r", 10, "Report interval (seconds)")
	pollInterval := flag.Int("p", 2, "Poll interval (seconds)")
	flag.Parse()
	if len(flag.Args()) != 0 {
		log.Fatalf("unknown flags: %v", flag.Args())
	}

	fmt.Printf("Agent started. Server=%s, poll=%ds, report=%ds\n",
		*serverAddr, *pollInterval, *reportInterval)

	metrics := &Metrics{
		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}

	for {
		collectRuntimeMetrics(metrics)

		if metrics.Counter["PollCount"]%int64(*reportInterval / *pollInterval) == 0 {
			sendAllMetrics(metrics)
		}

		time.Sleep(time.Duration(*pollInterval) * time.Second)
	}
}

func collectRuntimeMetrics(m *Metrics) {
	var r runtime.MemStats
	runtime.ReadMemStats(&r)

	// Gauge metrics
	m.Gauge["Alloc"] = float64(r.Alloc)
	m.Gauge["BuckHashSys"] = float64(r.BuckHashSys)
	m.Gauge["Frees"] = float64(r.Frees)
	m.Gauge["GCCPUFraction"] = r.GCCPUFraction
	m.Gauge["GCSys"] = float64(r.GCSys)
	m.Gauge["HeapAlloc"] = float64(r.HeapAlloc)
	m.Gauge["HeapIdle"] = float64(r.HeapIdle)
	m.Gauge["HeapInuse"] = float64(r.HeapInuse)
	m.Gauge["HeapObjects"] = float64(r.HeapObjects)
	m.Gauge["HeapReleased"] = float64(r.HeapReleased)
	m.Gauge["HeapSys"] = float64(r.HeapSys)
	m.Gauge["LastGC"] = float64(r.LastGC)
	m.Gauge["Lookups"] = float64(r.Lookups)
	m.Gauge["MCacheInuse"] = float64(r.MCacheInuse)
	m.Gauge["MCacheSys"] = float64(r.MCacheSys)
	m.Gauge["MSpanInuse"] = float64(r.MSpanInuse)
	m.Gauge["MSpanSys"] = float64(r.MSpanSys)
	m.Gauge["Mallocs"] = float64(r.Mallocs)
	m.Gauge["NextGC"] = float64(r.NextGC)
	m.Gauge["NumForcedGC"] = float64(r.NumForcedGC)
	m.Gauge["NumGC"] = float64(r.NumGC)
	m.Gauge["OtherSys"] = float64(r.OtherSys)
	m.Gauge["PauseTotalNs"] = float64(r.PauseTotalNs)
	m.Gauge["StackInuse"] = float64(r.StackInuse)
	m.Gauge["StackSys"] = float64(r.StackSys)
	m.Gauge["Sys"] = float64(r.Sys)
	m.Gauge["TotalAlloc"] = float64(r.TotalAlloc)

	// RandomValue gauge
	m.Gauge["RandomValue"] = rand.Float64()

	// Counter
	m.Counter["PollCount"]++
}

func sendAllMetrics(m *Metrics) {
	for name, value := range m.Gauge {
		sendMetric("gauge", name, fmt.Sprintf("%f", value))
	}

	for name, value := range m.Counter {
		sendMetric("counter", name, strconv.FormatInt(value, 10))
	}
}

func sendMetric(metricType, name, value string) {
	url := fmt.Sprintf("%s/update/%s/%s/%s", serverAddr, metricType, name, value)

	req, err := http.NewRequest("POST", url, strings.NewReader(""))
	if err != nil {
		log.Println("POST error:", err)
		return
	}

	req.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed sending metric %s/%s: %v\n", metricType, name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Server returned status %d for metric %s/%s", resp.StatusCode, metricType, name)
	}
}
