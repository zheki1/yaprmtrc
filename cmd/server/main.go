package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
	"github.com/zheki1/yaprmtrc.git/internal/models"
	"github.com/zheki1/yaprmtrc.git/internal/repository"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	logger, err := NewLogger()
	if err != nil {
		log.Fatalf("cannot init logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("logger sync failed: %v", err)
		}
	}()

	cfg := LoadConfig(logger)

	var dbConn *pgx.Conn
	if cfg.DatabaseDSN != "" {
		conn, err := pgx.Connect(context.Background(), cfg.DatabaseDSN)
		if err != nil {
			logger.Fatalw(
				"failed to connect to database",
				"error", err,
			)
		}
		dbConn = conn
		if dbConn != nil {
			if err := runMigrations(cfg.DatabaseDSN); err != nil {
				logger.Fatalw("migration failed", "error", err)
			}
		}
	}

	//storage := NewMemStorage()
	var storage repository.Repository
	switch {
	case cfg.DatabaseDSN != "":
		logger.Info("using postgres storage")
		storage = repository.NewPostgresRepository(dbConn)
	case cfg.FileStoragePath != "":
		logger.Info("using file storage")
		storage = repository.NewFileRepository(cfg.FileStoragePath)
	default:
		logger.Info("using memory storage")
		storage = repository.NewMemRepository()
	}

	fileStorage := NewFileStorage(cfg.FileStoragePath)

	if cfg.Restore {
		if metrics, err := fileStorage.Load(); err == nil {
			for _, ms := range metrics {
				switch ms.MType {
				case models.Gauge:
					if ms.Value != nil {
						storage.UpdateGauge(context.Background(), ms.ID, *ms.Value)
					}
				case models.Counter:
					if ms.Delta != nil {
						storage.UpdateCounter(context.Background(), ms.ID, *ms.Delta)
					}
				}
			}
			//storage.Import(metrics)
			log.Printf("metrics restored %v", len(metrics))
		} else {
			log.Printf("cannot restore metrics %s", err.Error())
		}
	}

	if cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(cfg.StoreInterval)
			defer ticker.Stop()

			for range ticker.C {
				metrics, err := storage.GetAll(context.Background())
				if err != nil {
					log.Fatal(err.Error())
				}
				if err := fileStorage.Save(metrics); err != nil {
					log.Printf("cannot save metrics into file %s", err.Error())
				}
			}
		}()
	}

	server := &Server{
		logger:      logger,
		storage:     storage,
		fileStorage: fileStorage,
		syncSave:    cfg.StoreInterval == 0,
		db:          dbConn,
	}

	httpServer := &http.Server{
		Addr:    cfg.Address,
		Handler: router(server),
	}

	go func() {
		log.Printf("Starting server on %s\n", cfg.Address)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen failed %s", err.Error())
		}
	}()

	// graceful shutdown
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	<-ctx.Done()
	log.Print("Shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("http server shutdown failed %s", err.Error())
	}

	metrics, err := storage.GetAll(context.Background())
	if err != nil {
		log.Fatal(err.Error())
	}
	if err := fileStorage.Save(metrics); err != nil {
		log.Fatalf("Metrics save failed %s", err.Error())
	} else {
		log.Print("Metrics saved successfully")
	}

	if storage != nil {
		if err := storage.Close(); err != nil {
			logger.Errorw("storage close failed", "error", err)
		}
	}
}

func runMigrations(dsn string) error {
	m, err := migrate.New(
		"file://migrations",
		dsn,
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
