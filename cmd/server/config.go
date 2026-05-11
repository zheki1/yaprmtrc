package main

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"time"
)

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
	TrustedSubnet   string
	GRPCAddr        string
}

// fileConfig описывает структуру JSON-файла конфигурации.
// Все поля — указатели, чтобы отличать явно заданные значения от отсутствующих.
type fileConfig struct {
	Address       *string `json:"address"`
	GRPCAddr      *string `json:"grpc_address"`
	Restore       *bool   `json:"restore"`
	StoreInterval *string `json:"store_interval"`
	StoreFile     *string `json:"store_file"`
	DatabaseDSN   *string `json:"database_dsn"`
	CryptoKey     *string `json:"crypto_key"`
	TrustedSubnet *string `json:"trusted_subnet"`
}

// loadFileConfig читает и парсит JSON-файл конфигурации по заданному пути.
func loadFileConfig(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, err
	}
	return &fc, nil
}

// LoadConfig читает конфигурацию из флагов командной строки, переменных окружения
// и (опционально) JSON-файла конфигурации.
//
// Приоритет (от высшего к низшему):
//  1. Переменные окружения
//  2. Флаги командной строки
//  3. JSON-файл конфигурации
//  4. Значения по умолчанию
func LoadConfig(logger Logger) *Config {
	defaults := &Config{
		Address:         "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "./metrics-recovery.json",
		Restore:         true,
		DatabaseDSN:     "",
		Key:             "",
		AuditFile:       "",
		AuditURL:        "",
		CryptoKey:       "",
		TrustedSubnet:   "",
		GRPCAddr:        "localhost:50051",
	}

	var (
		flagAddress         = flag.String("a", defaults.Address, "server address")
		flagStoreInterval   = flag.Duration("i", defaults.StoreInterval, "store interval")
		flagFileStoragePath = flag.String("f", defaults.FileStoragePath, "file storage path")
		flagRestore         = flag.Bool("r", defaults.Restore, "restore from file")
		flagDatabaseDSN     = flag.String("d", defaults.DatabaseDSN, "database dsn")
		flagKey             = flag.String("k", defaults.Key, "hash key")
		flagAuditFile       = flag.String("audit-file", defaults.AuditFile, "audit log file path")
		flagAuditURL        = flag.String("audit-url", defaults.AuditURL, "audit log remote URL")
		flagCryptoKey       = flag.String("crypto-key", defaults.CryptoKey, "path to private key file")
		flagGRPCAddr        = flag.String("grpc-addr", defaults.GRPCAddr, "gRPC server address")
		flagConfigFile      = flag.String("c", "", "path to JSON config file")
	)
	flag.StringVar(flagConfigFile, "config", "", "path to JSON config file (alias for -c)")
	flag.Parse()

	// определяем путь к файлу конфигурации
	// Приоритет: флаг → переменная окружения CONFIG
	configPath := *flagConfigFile
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	// начинаем со значений по умолчанию
	cfg := *defaults

	// применяем JSON-файл (наименьший приоритет после дефолтов)
	if configPath != "" {
		fc, err := loadFileConfig(configPath)
		if err != nil {
			logger.Fatalf("failed to load config file %s: %v", configPath, err)
		}
		if fc.Address != nil {
			cfg.Address = *fc.Address
		}
		if fc.GRPCAddr != nil {
			cfg.GRPCAddr = *fc.GRPCAddr
		}
		if fc.Restore != nil {
			cfg.Restore = *fc.Restore
		}
		if fc.StoreInterval != nil {
			d, err := time.ParseDuration(*fc.StoreInterval)
			if err != nil {
				logger.Fatalf("invalid store_interval in config file: %s", *fc.StoreInterval)
			}
			cfg.StoreInterval = d
		}
		if fc.StoreFile != nil {
			cfg.FileStoragePath = *fc.StoreFile
		}
		if fc.DatabaseDSN != nil {
			cfg.DatabaseDSN = *fc.DatabaseDSN
		}
		if fc.CryptoKey != nil {
			cfg.CryptoKey = *fc.CryptoKey
		}
		if fc.TrustedSubnet != nil {
			cfg.TrustedSubnet = *fc.TrustedSubnet
		}
	}

	// применяем флаги (перезаписывают значения из файла)
	// Используем Visit, чтобы применять только явно переданные флаги,
	// а не все (в том числе со значениями по умолчанию).
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.Address = *flagAddress
		case "i":
			cfg.StoreInterval = *flagStoreInterval
		case "f":
			cfg.FileStoragePath = *flagFileStoragePath
		case "r":
			cfg.Restore = *flagRestore
		case "d":
			cfg.DatabaseDSN = *flagDatabaseDSN
		case "k":
			cfg.Key = *flagKey
		case "audit-file":
			cfg.AuditFile = *flagAuditFile
		case "audit-url":
			cfg.AuditURL = *flagAuditURL
		case "crypto-key":
			cfg.CryptoKey = *flagCryptoKey
		case "grpc-addr":
			cfg.GRPCAddr = *flagGRPCAddr
		}
	})

	// переменные окружения (наивысший приоритет)
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Address = v
	}
	if v, ok := os.LookupEnv("GRPC_ADDR"); ok {
		cfg.GRPCAddr = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.StoreInterval = d
		} else if sec, err := strconv.Atoi(v); err == nil {
			cfg.StoreInterval = time.Duration(sec) * time.Second
		} else {
			logger.Fatalf("invalid STORE_INTERVAL: %s", v)
		}
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			logger.Fatalf("invalid RESTORE: %s", v)
		}
		cfg.Restore = b
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
	if v, ok := os.LookupEnv("TRUSTED_SUBNET"); ok {
		cfg.TrustedSubnet = v
	}

	return &cfg
}
