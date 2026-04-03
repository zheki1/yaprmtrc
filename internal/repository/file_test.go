package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zheki1/yaprmtrc/internal/models"
)

func tempFilePath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "metrics.json")
}

func TestFileRepository_UpdateGauge(t *testing.T) {
	path := tempFilePath(t)
	repo := NewFileRepository(path)

	ctx := context.Background()

	if err := repo.UpdateGauge(ctx, "Alloc", 1.5); err != nil {
		t.Fatalf("UpdateGauge: %v", err)
	}

	val, ok, err := repo.GetGauge(ctx, "Alloc")
	if err != nil {
		t.Fatalf("GetGauge: %v", err)
	}
	if !ok {
		t.Fatal("expected gauge to exist")
	}
	if val != 1.5 {
		t.Fatalf("expected 1.5, got %v", val)
	}

	// update existing
	if err := repo.UpdateGauge(ctx, "Alloc", 2.5); err != nil {
		t.Fatalf("UpdateGauge: %v", err)
	}
	val, ok, err = repo.GetGauge(ctx, "Alloc")
	if err != nil {
		t.Fatalf("GetGauge: %v", err)
	}
	if !ok || val != 2.5 {
		t.Fatalf("expected 2.5, got %v (ok=%v)", val, ok)
	}
}

func TestFileRepository_UpdateCounter(t *testing.T) {
	path := tempFilePath(t)
	repo := NewFileRepository(path)
	ctx := context.Background()

	if err := repo.UpdateCounter(ctx, "PollCount", 5); err != nil {
		t.Fatalf("UpdateCounter: %v", err)
	}

	val, ok, err := repo.GetCounter(ctx, "PollCount")
	if err != nil {
		t.Fatalf("GetCounter: %v", err)
	}
	if !ok {
		t.Fatal("expected counter to exist")
	}
	if val != 5 {
		t.Fatalf("expected 5, got %v", val)
	}

	// accumulate
	if err := repo.UpdateCounter(ctx, "PollCount", 3); err != nil {
		t.Fatalf("UpdateCounter: %v", err)
	}
	val, _, _ = repo.GetCounter(ctx, "PollCount")
	if val != 8 {
		t.Fatalf("expected 8, got %v", val)
	}
}

func TestFileRepository_GetGauge_NotFound(t *testing.T) {
	path := tempFilePath(t)
	repo := NewFileRepository(path)
	ctx := context.Background()

	// write something first so file exists
	_ = repo.UpdateGauge(ctx, "X", 1.0)

	_, ok, err := repo.GetGauge(ctx, "NonExistent")
	if err != nil {
		t.Fatalf("GetGauge: %v", err)
	}
	if ok {
		t.Fatal("expected not found")
	}
}

func TestFileRepository_GetCounter_NotFound(t *testing.T) {
	path := tempFilePath(t)
	repo := NewFileRepository(path)
	ctx := context.Background()

	_ = repo.UpdateCounter(ctx, "X", 1)

	_, ok, err := repo.GetCounter(ctx, "NonExistent")
	if err != nil {
		t.Fatalf("GetCounter: %v", err)
	}
	if ok {
		t.Fatal("expected not found")
	}
}

func TestFileRepository_GetAll(t *testing.T) {
	path := tempFilePath(t)
	repo := NewFileRepository(path)
	ctx := context.Background()

	_ = repo.UpdateGauge(ctx, "A", 1.1)
	_ = repo.UpdateCounter(ctx, "B", 2)

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2, got %d", len(all))
	}
}

func TestFileRepository_UpdateBatch(t *testing.T) {
	path := tempFilePath(t)
	repo := NewFileRepository(path)
	ctx := context.Background()

	v1 := 1.1
	d1 := int64(5)
	batch := []models.Metrics{
		{ID: "G1", MType: models.Gauge, Value: &v1},
		{ID: "C1", MType: models.Counter, Delta: &d1},
	}

	if err := repo.UpdateBatch(ctx, batch); err != nil {
		t.Fatalf("UpdateBatch: %v", err)
	}

	gVal, ok, _ := repo.GetGauge(ctx, "G1")
	if !ok || gVal != 1.1 {
		t.Fatalf("expected gauge 1.1, got %v", gVal)
	}

	cVal, ok, _ := repo.GetCounter(ctx, "C1")
	if !ok || cVal != 5 {
		t.Fatalf("expected counter 5, got %v", cVal)
	}

	// batch update existing
	v2 := 2.2
	d2 := int64(3)
	batch2 := []models.Metrics{
		{ID: "G1", MType: models.Gauge, Value: &v2},
		{ID: "C1", MType: models.Counter, Delta: &d2},
	}
	if err := repo.UpdateBatch(ctx, batch2); err != nil {
		t.Fatalf("UpdateBatch: %v", err)
	}

	gVal, _, _ = repo.GetGauge(ctx, "G1")
	if gVal != 2.2 {
		t.Fatalf("expected gauge 2.2, got %v", gVal)
	}
	cVal, _, _ = repo.GetCounter(ctx, "C1")
	if cVal != 8 {
		t.Fatalf("expected counter 8, got %v", cVal)
	}
}

func TestFileRepository_Close(t *testing.T) {
	path := tempFilePath(t)
	repo := NewFileRepository(path)
	if err := repo.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestFileRepository_RestoreNoFile(t *testing.T) {
	path := tempFilePath(t)
	// remove file to ensure it doesn't exist
	os.Remove(path)
	repo := NewFileRepository(path)
	ctx := context.Background()

	// GetAll on missing file returns error
	_, err := repo.GetAll(ctx)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
