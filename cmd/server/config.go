package main

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Address         string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
}

func LoadConfig(logger Logger) *Config {
	cfg := &Config{
		Address:         "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "./metrics-recovery.json",
		Restore:         true,
		DatabaseDSN:     "",
	}

	// flags
	flag.StringVar(&cfg.Address, "a", cfg.Address, "server address")
	flag.DurationVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "store interval")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore from file")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database dsn")
	flag.Parse()

	// env priority
	if v := os.Getenv("ADDRESS"); v != "" {
		cfg.Address = v
	}
	if v := os.Getenv("STORE_INTERVAL"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil {
			cfg.StoreInterval = time.Duration(sec) * time.Second
		} else {
			logger.Fatalf("invalid REPORT_INTERVAL: %s", v)
		}
	}
	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		cfg.FileStoragePath = v
	}
	if v := os.Getenv("RESTORE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Restore = b
		} else {
			logger.Fatalf("invalid RESTORE: %s", v)
		}
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		cfg.DatabaseDSN = v
	}

	return cfg
}
