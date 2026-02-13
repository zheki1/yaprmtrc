package main

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestCollectRuntimeMetrics(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	// Создаем экземпляр Agent
	agent := &Agent{
		logger: logger.Sugar(),

		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}

	// Вызываем метод collectRuntimeMetrics
	agent.collectRuntimeMetrics()

	// Проверяем, что метрики были установлены
	if _, exists := agent.Gauge["Alloc"]; !exists {
		t.Errorf("Gauge metric 'Alloc' was not collected")
	}
	if _, exists := agent.Gauge["HeapAlloc"]; !exists {
		t.Errorf("Gauge metric 'HeapAlloc' was not collected")
	}
	if _, exists := agent.Gauge["HeapSys"]; !exists {
		t.Errorf("Gauge metric 'HeapSys' was not collected")
	}
	if _, exists := agent.Gauge["NumGC"]; !exists {
		t.Errorf("Gauge metric 'NumGC' was not collected")
	}
	if _, exists := agent.Gauge["TotalAlloc"]; !exists {
		t.Errorf("Gauge metric 'TotalAlloc' was not collected")
	}
	if _, exists := agent.Gauge["RandomValue"]; !exists {
		t.Errorf("Gauge metric 'RandomValue' was not collected")
	}

	// Проверяем, что значение RandomValue в пределах ожидаемого диапазона [0, 1)
	if value := agent.Gauge["RandomValue"]; value < 0 || value >= 1 {
		t.Errorf("RandomValue should be in range [0,1), got %f", value)
	}

	// Проверяем счетчик PollCount
	if agent.Counter["PollCount"] != 1 {
		t.Errorf("Counter 'PollCount' should be 1, got %d", agent.Counter["PollCount"])
	}
}

func TestCollectRuntimeMetricsMultipleCalls(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	// Создаем экземпляр Agent
	agent := &Agent{
		logger:  logger.Sugar(),
		Gauge:   make(map[string]float64),
		Counter: make(map[string]int64),
	}

	// Вызываем метод collectRuntimeMetrics несколько раз
	for i := 0; i < 3; i++ {
		agent.collectRuntimeMetrics()
		time.Sleep(1 * time.Millisecond) // Даем время для изменения метрик
	}

	// Проверяем, что счетчик увеличился
	if agent.Counter["PollCount"] != 3 {
		t.Errorf("Counter 'PollCount' should be 3 after 3 calls, got %d", agent.Counter["PollCount"])
	}
}
