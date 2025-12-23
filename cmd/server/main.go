package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	cfg := LoadConfig()

	logger, _ := NewLogger()
	defer logger.Sync()

	storage := NewMemStorage()
	fileStorage := NewFileStorage(cfg.FileStoragePath)

	if cfg.Restore {
		if metrics, err := fileStorage.Load(); err == nil {
			storage.Import(metrics)
			logger.Infow("metrics restored", len(metrics))
		}
	}

	if cfg.StoreInterval > 0 {
		go func() {
			ticker := time.NewTicker(cfg.StoreInterval)
			defer ticker.Stop()

			for range ticker.C {
				metrics := storage.Export()
				_ = fileStorage.Save(metrics)
			}
		}()
	}

	server := &Server{
		storage:     storage,
		fileStorage: fileStorage,
		syncSave:    cfg.StoreInterval == 0,
	}

	log.Printf("Starting server on %s\n", cfg.Address)
	log.Fatal(http.ListenAndServe(cfg.Address, router(server, logger)))
}
