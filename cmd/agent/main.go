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
}

func main() {
	// default values
	cfg := &Config{
		Addr:           "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
	}

	// flags
	flag.StringVar(&cfg.Addr, "a", cfg.Addr, "Address of metrics server")
	flag.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval in seconds")
	flag.Parse()

	if len(flag.Args()) != 0 {
		log.Fatalf("unknown flags: %v", flag.Args())
	}

	if addr := os.Getenv("ADDRESS"); addr != "" {
		cfg.Addr = addr
	}

	if v := os.Getenv("REPORT_INTERVAL"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil || i <= 0 {
			log.Fatalf("invalid REPORT_INTERVAL: %s", v)
		}
		cfg.ReportInterval = i
	}

	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil || i <= 0 {
			log.Fatalf("invalid POLL_INTERVAL: %s", v)
		}
		cfg.PollInterval = i
	}

	agent := NewAgent(cfg)
	agent.Start()
}
