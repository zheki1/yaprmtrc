package main

import (
	"flag"
	"log"
)

type Config struct {
	Addr           string
	ReportInterval int
	PollInterval   int
}

func main() {
	cfg := &Config{}

	flag.StringVar(&cfg.Addr, "a", "localhost:8080", "Address of metrics server")
	flag.IntVar(&cfg.ReportInterval, "r", 10, "Report interval in seconds")
	flag.IntVar(&cfg.PollInterval, "p", 2, "Poll interval in seconds")

	flag.Parse()

	if len(flag.Args()) != 0 {
		log.Fatalf("unknown flags: %v", flag.Args())
	}

	agent := NewAgent(cfg)
	agent.Start()
}
