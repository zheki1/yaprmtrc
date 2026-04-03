package main

import (
	"path/filepath"
	"testing"

	"github.com/zheki1/yaprmtrc/internal/models"
)

func TestFileStorage_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metrics.json")

	fs := NewFileStorage(path)

	v := 1.5
	d := int64(10)
	metrics := []models.Metrics{
		{ID: "Alloc", MType: models.Gauge, Value: &v},
		{ID: "PollCount", MType: models.Counter, Delta: &d},
	}

	if err := fs.Save(metrics); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := fs.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(loaded))
	}

	found := false
	for _, m := range loaded {
		if m.ID == "Alloc" && m.MType == models.Gauge && *m.Value == 1.5 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected Alloc gauge metric")
	}
}

func TestFileStorage_Load_MissingFile(t *testing.T) {
	fs := NewFileStorage("/nonexistent/path/metrics.json")

	_, err := fs.Load()
	if err == nil {
		t.Fatal("expected error when loading from nonexistent path")
	}
}

func TestFileStorage_Save_InvalidPath(t *testing.T) {
	fs := NewFileStorage("/nonexistent/dir/metrics.json")

	err := fs.Save([]models.Metrics{})
	if err == nil {
		t.Fatal("expected error when saving to invalid path")
	}
}
