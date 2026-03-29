package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/zheki1/yaprmtrc/internal/models"
)

func BenchmarkMemRepository_UpdateGauge(b *testing.B) {
	repo := NewMemRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.UpdateGauge(ctx, "Alloc", float64(i))
	}
}

func BenchmarkMemRepository_UpdateCounter(b *testing.B) {
	repo := NewMemRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.UpdateCounter(ctx, "PollCount", 1)
	}
}

func BenchmarkMemRepository_GetGauge(b *testing.B) {
	repo := NewMemRepository()
	ctx := context.Background()
	_ = repo.UpdateGauge(ctx, "Alloc", 123.45)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = repo.GetGauge(ctx, "Alloc")
	}
}

func BenchmarkMemRepository_GetCounter(b *testing.B) {
	repo := NewMemRepository()
	ctx := context.Background()
	_ = repo.UpdateCounter(ctx, "PollCount", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = repo.GetCounter(ctx, "PollCount")
	}
}

func BenchmarkMemRepository_GetAll(b *testing.B) {
	repo := NewMemRepository()
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		_ = repo.UpdateGauge(ctx, fmt.Sprintf("gauge_%d", i), float64(i))
		_ = repo.UpdateCounter(ctx, fmt.Sprintf("counter_%d", i), int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = repo.GetAll(ctx)
	}
}

func BenchmarkMemRepository_UpdateBatch(b *testing.B) {
	repo := NewMemRepository()
	ctx := context.Background()

	metrics := make([]models.Metrics, 0, 200)
	for i := 0; i < 100; i++ {
		v := float64(i)
		metrics = append(metrics, models.Metrics{
			ID:    fmt.Sprintf("gauge_%d", i),
			MType: models.Gauge,
			Value: &v,
		})
		d := int64(i)
		metrics = append(metrics, models.Metrics{
			ID:    fmt.Sprintf("counter_%d", i),
			MType: models.Counter,
			Delta: &d,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = repo.UpdateBatch(ctx, metrics)
	}
}
