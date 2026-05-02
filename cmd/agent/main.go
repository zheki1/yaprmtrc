package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

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
	flag.StringVar(&cfg.Addr, "a", cfg.Addr, "Address of metrics server")
	flag.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval in seconds")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Hash key")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limit")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", cfg.CryptoKey, "Path to public key file")
	flag.Parse()

	if len(flag.Args()) != 0 {
		return fmt.Errorf("unknown flags: %v", flag.Args())
	}

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
