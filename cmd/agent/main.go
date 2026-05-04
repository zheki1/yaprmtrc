package main

import (
	"crypto/rsa"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/zheki1/yaprmtrc/internal/buildinfo"
	"github.com/zheki1/yaprmtrc/internal/security"
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
	Addr            string
	ReportInterval  time.Duration
	PollInterval    time.Duration
	Key             string
	RateLimit       int
	CryptoKey       string
	ConfigFile      string
	CachedPublicKey *rsa.PublicKey
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

	// First, determine config file from flags or environment
	// This is done before full flag parsing to ensure config file can be loaded
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
		if err := loadAgentConfigFromFile(configFile, cfg); err != nil {
			return fmt.Errorf("failed to load config from file: %w", err)
		}
	}

	// Load from environment variables (override config file)
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

	// Parse flags (highest priority - overrides everything)
	flag.StringVar(&cfg.Addr, "a", cfg.Addr, "Address of metrics server")
	flag.DurationVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval")
	flag.DurationVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Hash key")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limit")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "Path to public key file")
	flag.StringVar(&cfg.ConfigFile, "c", "", "config file path")
	flag.StringVar(&cfg.ConfigFile, "config", "", "config file path")
	flag.Parse()

	if len(flag.Args()) != 0 {
		return fmt.Errorf("unknown flags: %v", flag.Args())
	}

	// Load and cache public key if specified
	if cfg.CryptoKey != "" {
		pubKey, err := security.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			return fmt.Errorf("failed to load public key: %w", err)
		}
		cfg.CachedPublicKey = pubKey
	}

	agent, err := NewAgent(cfg)
	if err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	go func() {
		<-c
		fmt.Println("Received shutdown signal, shutting down agent...")
		agent.cancel()
	}()

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
