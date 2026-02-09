package repository

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/zheki1/yaprmtrc.git/internal/models"
)

func openTestDB(t *testing.T) *pgx.Conn {

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set")
	}

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("cannot connect db: %v", err)
	}

	_, err = conn.Exec(context.Background(), `
	CREATE TABLE IF NOT EXISTS metrics (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		delta BIGINT,
		value DOUBLE PRECISION
	)
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = conn.Exec(context.Background(), `DELETE FROM metrics`)

	return conn
}

func TestPostgresGauge(t *testing.T) {

	conn := openTestDB(t)
	defer conn.Close(context.Background())

	repo := NewPostgresRepository(conn)

	ctx := context.Background()

	err := repo.UpdateGauge(ctx, "Alloc", 11.1)
	if err != nil {
		t.Fatal(err)
	}

	v, ok, err := repo.GetGauge(ctx, "Alloc")
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("not found")
	}

	if v != 11.1 {
		t.Fatalf("bad value %v", v)
	}
}

func TestPostgresCounter(t *testing.T) {

	conn := openTestDB(t)
	defer conn.Close(context.Background())

	repo := NewPostgresRepository(conn)

	ctx := context.Background()

	err := repo.UpdateCounter(ctx, "Poll", 2)
	if err != nil {
		t.Fatal(err)
	}

	err = repo.UpdateCounter(ctx, "Poll", 3)
	if err != nil {
		t.Fatal(err)
	}

	v, ok, err := repo.GetCounter(ctx, "Poll")
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("not found")
	}

	if v != 5 {
		t.Fatalf("bad counter %v", v)
	}
}

func TestPostgresGetAll(t *testing.T) {

	conn := openTestDB(t)
	defer conn.Close(context.Background())

	repo := NewPostgresRepository(conn)

	ctx := context.Background()

	_ = repo.UpdateGauge(ctx, "A", 1.1)
	_ = repo.UpdateCounter(ctx, "B", 2)

	list, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 got %d", len(list))
	}
}

func TestPostgresBatch(t *testing.T) {

	conn := openTestDB(t)
	defer conn.Close(context.Background())

	repo := NewPostgresRepository(conn)

	ctx := context.Background()

	data := []models.Metrics{
		{
			ID:    "A",
			MType: models.Gauge,
			Value: ptrFloat(2.2),
		},
		{
			ID:    "B",
			MType: models.Counter,
			Delta: ptrInt(3),
		},
	}

	err := repo.UpdateBatch(ctx, data)
	if err != nil {
		t.Fatal(err)
	}

	v, ok, _ := repo.GetGauge(ctx, "A")
	if !ok {
		t.Fatal("A not found")
	}

	c, ok, _ := repo.GetCounter(ctx, "B")
	if !ok {
		t.Fatal("B not found")
	}

	if v != 2.2 || c != 3 {
		t.Fatal("batch failed")
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}

func ptrInt(v int64) *int64 {
	return &v
}
