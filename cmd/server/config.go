package main

import (
	"crypto/rsa"
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zheki1/yaprmtrc/internal/security"
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
	Address          string
	StoreInterval    time.Duration
	FileStoragePath  string
	Restore          bool
	DatabaseDSN      string
	Key              string
	AuditFile        string
	AuditURL         string
	CryptoKey        string
	ConfigFile       string
	CachedPrivateKey *rsa.PrivateKey
}

// LoadConfig читает конфигурацию из файла, флагов командной строки и переменных окружения.
// Порядок приоритетов: значения по умолчанию < конфиг-файл < переменные окружения < флаги
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

	// First, determine config file from flags or environment
	configFile := ""
	for i, arg := range os.Args[1:] {
		if arg == "-c" || arg == "-config" {
			if i+1 < len(os.Args)-1 {
				configFile = os.Args[i+2]
				break
			}
		} else if len(arg) > 3 && (arg[:2] == "-c=" || arg[:8] == "-config=") {
			configFile = arg[strings.Index(arg, "=")+1:]
			break
		}
	}

	// Check environment variable if config file not found in flags
	if configFile == "" {
		if v, ok := os.LookupEnv("CONFIG"); ok {
			configFile = v
		}
	}

	// Load from config file if specified
	if configFile != "" {
		if err := loadServerConfigFromFile(configFile, cfg); err != nil {
			logger.Fatalf("failed to load config from file: %v", err)
		}
		cfg.ConfigFile = configFile
	}

	// Load from environment variables (override config file)
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

	// If CONFIG environment variable specifies a different config file, reload it
	if v, ok := os.LookupEnv("CONFIG"); ok && v != cfg.ConfigFile {
		if err := loadServerConfigFromFile(v, cfg); err != nil {
			logger.Fatalf("failed to load config from file: %v", err)
		}
		cfg.ConfigFile = v
	}

	// Parse flags (highest priority - overrides everything)
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

	// If config file flag was set and is different from current, reload it
	if cfg.ConfigFile != "" && cfg.ConfigFile != configFile {
		if err := loadServerConfigFromFile(cfg.ConfigFile, cfg); err != nil {
			logger.Fatalf("failed to load config from file: %v", err)
		}
	}

	// Load and cache private key if specified
	if cfg.CryptoKey != "" {
		privKey, err := security.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			logger.Fatalf("failed to load private key: %v", err)
		}
		cfg.CachedPrivateKey = privKey
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
