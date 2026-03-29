package models

import "testing"

func TestMetricConstants(t *testing.T) {
	if Counter != "counter" {
		t.Fatalf("expected counter, got %s", Counter)
	}
	if Gauge != "gauge" {
		t.Fatalf("expected gauge, got %s", Gauge)
	}
}

func TestMetricsStruct(t *testing.T) {
	val := 42.5
	delta := int64(10)

	m := Metrics{
		ID:    "test",
		MType: Gauge,
		Value: &val,
	}

	if m.ID != "test" {
		t.Fatalf("expected test, got %s", m.ID)
	}
	if m.MType != Gauge {
		t.Fatalf("expected gauge, got %s", m.MType)
	}
	if *m.Value != 42.5 {
		t.Fatalf("expected 42.5, got %f", *m.Value)
	}

	m2 := Metrics{
		ID:    "poll",
		MType: Counter,
		Delta: &delta,
	}

	if m2.ID != "poll" {
		t.Fatalf("expected poll, got %s", m2.ID)
	}
	if *m2.Delta != 10 {
		t.Fatalf("expected 10, got %d", *m2.Delta)
	}
}
