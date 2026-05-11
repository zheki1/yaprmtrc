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

// AgentConfig представляет конфигурацию агента из JSON файла.
type AgentConfig struct {
	Address        string `json:"address,omitempty"`
	ReportInterval string `json:"report_interval,omitempty"`
	PollInterval   string `json:"poll_interval,omitempty"`
	CryptoKey      string `json:"crypto_key,omitempty"`
	Key            string `json:"key,omitempty"`
	RateLimit      *int   `json:"rate_limit,omitempty"`
}

// Config хранит конфигурацию агента: адрес сервера, интервалы опроса и отправки,
// ключ HMAC и лимит одновременных запросов.
type Config struct {
	Addr           string
	ReportInterval time.Duration
	PollInterval   time.Duration
	Key            string
	RateLimit      int
	CryptoKey      string
	ConfigFile     string
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
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
		Key:            "",
		RateLimit:      1,
		CryptoKey:      "",
		ConfigFile:     "",
	}

	// flags
	flag.StringVar(&cfg.Addr, "a", cfg.Addr, "Address of metrics server")
	flag.DurationVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval")
	flag.DurationVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Hash key")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limit")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "Path to public key file")
	flag.StringVar(&cfg.ConfigFile, "c", cfg.ConfigFile, "config file path")
	flag.StringVar(&cfg.ConfigFile, "config", cfg.ConfigFile, "config file path")
	flag.Parse()

	if len(flag.Args()) != 0 {
		return fmt.Errorf("unknown flags: %v", flag.Args())
	}

	// load from config file if specified
	if cfg.ConfigFile != "" {
		if err := loadAgentConfigFromFile(cfg.ConfigFile, cfg); err != nil {
			return fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Addr = v
	}

	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if d, err := time.ParseDuration(v); err != nil {
			return fmt.Errorf("invalid REPORT_INTERVAL: %s", v)
		} else {
			cfg.ReportInterval = d
		}
	}

	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if d, err := time.ParseDuration(v); err != nil {
			return fmt.Errorf("invalid POLL_INTERVAL: %s", v)
		} else {
			cfg.PollInterval = d
		}
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

	if v, ok := os.LookupEnv("CONFIG"); ok {
		cfg.ConfigFile = v
	}

	agent, err := NewAgent(cfg)
	if err != nil {
		return err
	}
	agent.Start()
	return nil
}

// loadAgentConfigFromFile загружает конфигурацию из JSON файла.
func loadAgentConfigFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var ac AgentConfig
	if err := json.Unmarshal(data, &ac); err != nil {
		return err
	}
	// apply to cfg if not empty
	if ac.Address != "" {
		cfg.Addr = ac.Address
	}
	if ac.ReportInterval != "" {
		if d, err := time.ParseDuration(ac.ReportInterval); err == nil {
			cfg.ReportInterval = d
		}
	}
	if ac.PollInterval != "" {
		if d, err := time.ParseDuration(ac.PollInterval); err == nil {
			cfg.PollInterval = d
		}
	}
	if ac.CryptoKey != "" {
		cfg.CryptoKey = ac.CryptoKey
	}
	if ac.Key != "" {
		cfg.Key = ac.Key
	}
	if ac.RateLimit != nil {
		cfg.RateLimit = *ac.RateLimit
	}
	return nil
}
