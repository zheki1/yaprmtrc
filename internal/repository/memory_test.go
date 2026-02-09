package repository

import (
	"context"
	"testing"
)

func TestMemStorage_Gauge(t *testing.T) {
	s := NewMemRepository()

	s.UpdateGauge(context.Background(), "Alloc", 123.45)

	val, ok, err := s.GetGauge(context.Background(), "Alloc")
	if !ok || err != nil {
		t.Fatal("metric not found")
	}

	if val != 123.45 {
		t.Fatalf("expected 123.45, got %v", val)
	}
}

func TestMemStorage_Counter(t *testing.T) {
	s := NewMemRepository()

	s.UpdateCounter(context.Background(), "PollCount", 5)
	s.UpdateCounter(context.Background(), "PollCount", 3)

	val, ok, err := s.GetCounter(context.Background(), "PollCount")
	if !ok || err != nil {
		t.Fatal("metric not found")
	}

	if val != 8 {
		t.Fatalf("expected 8, got %v", val)
	}
}
