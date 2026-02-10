package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

func (a *Agent) collectRuntimeMetrics() {
	fmt.Printf("%s \n", "collect runtime metrics "+time.Now().String())

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
