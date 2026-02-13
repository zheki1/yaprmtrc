package main

import (
	"flag"
	"log"
	"os"
	"strconv"
)

type Config struct {
	Addr           string
	ReportInterval int
	PollInterval   int
	Key            string
	RateLimit      int
}

func main() {
	// default values
	cfg := &Config{
		Addr:           "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		Key:            "",
		RateLimit:      1,
	}

	// flags
	flag.StringVar(&cfg.Addr, "a", cfg.Addr, "Address of metrics server")
	flag.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval in seconds")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Hash key")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Poll interval in seconds")
	flag.Parse()

	if len(flag.Args()) != 0 {
		log.Fatalf("unknown flags: %v", flag.Args())
	}

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Addr = v
	}

	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		i, err := strconv.Atoi(v)
		if err != nil || i <= 0 {
			log.Fatalf("invalid REPORT_INTERVAL: %s", v)
		}
		cfg.ReportInterval = i
	}

	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		i, err := strconv.Atoi(v)
		if err != nil || i <= 0 {
			log.Fatalf("invalid POLL_INTERVAL: %s", v)
		}
		cfg.PollInterval = i
	}

	if v, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = v
	}

	if v, ok := os.LookupEnv("RATE_LIMIT"); ok {
		i, err := strconv.Atoi(v)
		if err != nil || i <= 0 {
			log.Fatalf("invalid RATE_LIMIT: %s", v)
		}
		cfg.RateLimit = i
	}

	agent := NewAgent(cfg)
	agent.Start()
}
