package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
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
	storage := NewMemStorage()
	fileStorage := NewFileStorage(cfg.FileStoragePath)

	if cfg.Restore {
		if metrics, err := fileStorage.Load(); err == nil {
			storage.Import(metrics)
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
				metrics := storage.Export()
				if err := fileStorage.Save(metrics); err != nil {
					log.Printf("cannot save metrics into file %s", err.Error())
				}
			}
		}()
	}

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

	if err := fileStorage.Save(storage.Export()); err != nil {
		log.Fatalf("Metrics save failed %s", err.Error())
	} else {
		log.Print("Metrics saved successfully")
	}

	if dbConn != nil {
		dbConn.Close(context.Background())
	}
}
