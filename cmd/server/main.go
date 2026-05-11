package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zheki1/yaprmtrc/internal/buildinfo"
	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/repository"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var buildVersion string
var buildDate string
var buildCommit string

func main() {
	buildinfo.Version = buildVersion
	buildinfo.Date = buildDate
	buildinfo.Commit = buildCommit
	buildinfo.Print()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server: %v\n", err)
	}
}

func run() error {
	logger, err := NewLogger()
	if err != nil {
		return fmt.Errorf("cannot init logger: %w", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("logger sync failed: %v", err)
		}
	}()

	cfg := LoadConfig(logger)

	var dbConn *pgxpool.Pool
	if cfg.DatabaseDSN != "" {
		conn, err := pgxpool.New(context.Background(), cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		dbConn = conn
		if dbConn != nil {
			if err := runMigrations(cfg.DatabaseDSN); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}
		}
	}

	//storage := NewMemStorage()
	var storage repository.Repository
	switch {
	case cfg.DatabaseDSN != "":
		logger.Info("using postgres storage")
		storage = repository.NewPostgresRepository(dbConn)
	// case cfg.FileStoragePath != "":
	// 	logger.Info("using file storage")
	// 	storage = repository.NewFileRepository(cfg.FileStoragePath)
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
					log.Printf("cannot get metrics for save: %s", err.Error())
					continue
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
		key:         cfg.Key,
		audit:       NewAuditPublisher(logger),
		cryptoKey:   cfg.CryptoKey,
	}

	if cfg.AuditFile != "" {
		fileObs, err := NewFileAuditObserver(cfg.AuditFile, logger)
		if err != nil {
			return fmt.Errorf("audit file observer: %w", err)
		}
		defer fileObs.Close()
		server.audit.Register(fileObs)
	}
	if cfg.AuditURL != "" {
		server.audit.Register(NewHTTPAuditObserver(cfg.AuditURL, logger))
	}

	httpServer := &http.Server{
		Addr:    cfg.Address,
		Handler: router(server),
	}

	go func() {
		log.Printf("Starting server on %s\n", cfg.Address)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Listen failed %s", err.Error())
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
		return fmt.Errorf("http server shutdown failed: %w", err)
	}

	metrics, err := storage.GetAll(context.Background())
	if err != nil {
		return fmt.Errorf("cannot get metrics on shutdown: %w", err)
	}
	if err := fileStorage.Save(metrics); err != nil {
		return fmt.Errorf("metrics save failed: %w", err)
	} else {
		log.Print("Metrics saved successfully")
	}

	if storage != nil {
		if err := storage.Close(); err != nil {
			logger.Errorw("storage close failed", "error", err)
		}
	}
	return nil
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
