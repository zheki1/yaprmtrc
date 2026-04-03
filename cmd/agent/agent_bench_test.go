package main

import (
	"testing"

	"go.uber.org/zap"
)

func BenchmarkCollectRuntimeMetrics(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	agent := &Agent{
		logger:  logger.Sugar(),
		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}

	for b.Loop() {
		agent.collectRuntimeMetrics()
	}
}
