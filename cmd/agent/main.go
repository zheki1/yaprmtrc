package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/zheki1/yaprmtrc/internal/buildinfo"
)

var buildVersion string
var buildDate string
var buildCommit string

// Config хранит конфигурацию агента: адрес сервера, интервалы опроса и отправки,
// ключ HMAC и лимит одновременных запросов.
type Config struct {
	Addr           string
	ReportInterval int
	PollInterval   int
	Key            string
	RateLimit      int
	CryptoKey      string
}

// fileConfig описывает структуру JSON-файла конфигурации агента.
// Все поля — указатели, чтобы отличать явно заданные значения от отсутствующих.
type fileConfig struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	CryptoKey      *string `json:"crypto_key"`
}

func main() {
	buildinfo.Version = buildVersion
	buildinfo.Date = buildDate
	buildinfo.Commit = buildCommit
	buildinfo.Print()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "agent: %v\n", err)
	}
}

func run() error {
	// default values
	cfg := &Config{
		Addr:           "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		Key:            "",
		RateLimit:      1,
		CryptoKey:      "",
	}

	// flags
	var flagConfigFile string
	flag.StringVar(&cfg.Addr, "a", cfg.Addr, "Address of metrics server")
	flag.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval in seconds")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Hash key")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limit")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "Path to public key file")
	flag.StringVar(&flagConfigFile, "c", "", "Path to JSON config file")
	flag.StringVar(&flagConfigFile, "config", "", "Path to JSON config file (alias for -c)")
	flag.Parse()

	if len(flag.Args()) != 0 {
		return fmt.Errorf("unknown flags: %v", flag.Args())
	}

	// --- определяем путь к файлу конфигурации: флаг → переменная окружения ---
	configPath := flagConfigFile
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	// --- применяем JSON-файл (наименьший приоритет после дефолтов) ---
	if configPath != "" {
		fc, err := loadFileConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config file %s: %w", configPath, err)
		}
		if fc.Address != nil {
			cfg.Addr = *fc.Address
		}
		if fc.ReportInterval != nil {
			sec, err := parseDurationSeconds(*fc.ReportInterval)
			if err != nil || sec <= 0 {
				return fmt.Errorf("invalid report_interval in config file: %s", *fc.ReportInterval)
			}
			cfg.ReportInterval = sec
		}
		if fc.PollInterval != nil {
			sec, err := parseDurationSeconds(*fc.PollInterval)
			if err != nil || sec <= 0 {
				return fmt.Errorf("invalid poll_interval in config file: %s", *fc.PollInterval)
			}
			cfg.PollInterval = sec
		}
		if fc.CryptoKey != nil {
			cfg.CryptoKey = *fc.CryptoKey
		}
	}

	// --- применяем флаги (перезаписывают файл, но только если переданы явно) ---
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.Addr = f.Value.String()
		case "r":
			if v, err := strconv.Atoi(f.Value.String()); err == nil {
				cfg.ReportInterval = v
			}
		case "p":
			if v, err := strconv.Atoi(f.Value.String()); err == nil {
				cfg.PollInterval = v
			}
		case "k":
			cfg.Key = f.Value.String()
		case "l":
			if v, err := strconv.Atoi(f.Value.String()); err == nil {
				cfg.RateLimit = v
			}
		case "crypto-key":
			cfg.CryptoKey = f.Value.String()
		}
	})

	// --- переменные окружения (наивысший приоритет) ---
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Addr = v
	}
	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		i, err := strconv.Atoi(v)
		if err != nil || i <= 0 {
			return fmt.Errorf("invalid REPORT_INTERVAL: %s", v)
		}
		cfg.ReportInterval = i
	}
	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		i, err := strconv.Atoi(v)
		if err != nil || i <= 0 {
			return fmt.Errorf("invalid POLL_INTERVAL: %s", v)
		}
		cfg.PollInterval = i
	}
	if v, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = v
	}
	if v, ok := os.LookupEnv("RATE_LIMIT"); ok {
		i, err := strconv.Atoi(v)
		if err != nil || i <= 0 {
			return fmt.Errorf("invalid RATE_LIMIT: %s", v)
		}
		cfg.RateLimit = i
	}
	if v, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = v
	}

	agent, err := NewAgent(cfg)
	if err != nil {
		return err
	}
	agent.Start()
	return nil
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

// parseDurationSeconds разбирает строку как Go duration ("1s", "2m") и возвращает
// целое количество секунд. Для обратной совместимости также принимает голое число ("10").
func parseDurationSeconds(s string) (int, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return int(d.Seconds()), nil
	}
	return strconv.Atoi(s)
}
