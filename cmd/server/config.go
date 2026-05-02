package main

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"time"
)

// ServerConfig представляет конфигурацию сервера из JSON файла.
type ServerConfig struct {
	Address       string `json:"address,omitempty"`
	Restore       *bool  `json:"restore,omitempty"`
	StoreInterval string `json:"store_interval,omitempty"`
	StoreFile     string `json:"store_file,omitempty"`
	DatabaseDSN   string `json:"database_dsn,omitempty"`
	CryptoKey     string `json:"crypto_key,omitempty"`
	Key           string `json:"key,omitempty"`
	AuditFile     string `json:"audit_file,omitempty"`
	AuditURL      string `json:"audit_url,omitempty"`
}

// Config хранит конфигурацию сервера: адрес, интервалы сохранения,
// путь к хранилищу, DSN базы данных и параметры аудита.
type Config struct {
	Address         string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	Key             string
	AuditFile       string
	AuditURL        string
	CryptoKey       string
	ConfigFile      string
}

// LoadConfig читает конфигурацию из файла, флагов командной строки и переменных окружения.
// Переменные окружения и флаги имеют приоритет над файлом.
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
		CryptoKey:       "",
		ConfigFile:      "",
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
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "Path to private key file")
	flag.StringVar(&cfg.ConfigFile, "c", cfg.ConfigFile, "config file path")
	flag.StringVar(&cfg.ConfigFile, "config", cfg.ConfigFile, "config file path")
	flag.Parse()

	// load from config file if specified
	if cfg.ConfigFile != "" {
		if err := loadServerConfigFromFile(cfg.ConfigFile, cfg); err != nil {
			logger.Fatalf("failed to load config from file: %v", err)
		}
	}

	// env priority
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Address = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.StoreInterval = d
		} else {
			logger.Fatalf("invalid STORE_INTERVAL: %s", v)
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
	if v, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = v
	}
	if v, ok := os.LookupEnv("CONFIG"); ok {
		cfg.ConfigFile = v
	}

	return cfg
}

// loadServerConfigFromFile загружает конфигурацию из JSON файла.
func loadServerConfigFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var sc ServerConfig
	if err := json.Unmarshal(data, &sc); err != nil {
		return err
	}
	// apply to cfg if not empty
	if sc.Address != "" {
		cfg.Address = sc.Address
	}
	if sc.Restore != nil {
		cfg.Restore = *sc.Restore
	}
	if sc.StoreInterval != "" {
		if d, err := time.ParseDuration(sc.StoreInterval); err == nil {
			cfg.StoreInterval = d
		}
	}
	if sc.StoreFile != "" {
		cfg.FileStoragePath = sc.StoreFile
	}
	if sc.DatabaseDSN != "" {
		cfg.DatabaseDSN = sc.DatabaseDSN
	}
	if sc.CryptoKey != "" {
		cfg.CryptoKey = sc.CryptoKey
	}
	if sc.Key != "" {
		cfg.Key = sc.Key
	}
	if sc.AuditFile != "" {
		cfg.AuditFile = sc.AuditFile
	}
	if sc.AuditURL != "" {
		cfg.AuditURL = sc.AuditURL
	}
	return nil
}
