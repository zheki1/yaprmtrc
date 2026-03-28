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
	Key             string
	AuditFile       string
	AuditURL        string
}

func LoadConfig(logger Logger) *Config {
	cfg := &Config{
		Address:         "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "./metrics-recovery.json",
		Restore:         true,
		DatabaseDSN:     "",
		Key:             "",
		AuditFile:       "",
		AuditURL:        "",
	}

	// flags
	flag.StringVar(&cfg.Address, "a", cfg.Address, "server address")
	flag.DurationVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "store interval")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "file storage path")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "restore from file")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "database dsn")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Hash key")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "audit log file path")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "audit log remote URL")
	flag.Parse()

	// env priority
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Address = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if sec, err := strconv.Atoi(v); err == nil {
			cfg.StoreInterval = time.Duration(sec) * time.Second
		} else {
			logger.Fatalf("invalid REPORT_INTERVAL: %s", v)
		}
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Restore = b
		} else {
			logger.Fatalf("invalid RESTORE: %s", v)
		}
	}
	if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = v
	}
	if v, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = v
	}
	if v, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = v
	}
	if v, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = v
	}

	return cfg
}
