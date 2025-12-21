package main

import (
	"flag"
	"log"
	"net/http"
	"os"
)

func main() {
	// default values
	serverAddr := "localhost:8080"

	// flags
	flag.StringVar(&serverAddr, "a", serverAddr, "HTTP server address")
	flag.Parse()

	if len(flag.Args()) != 0 {
		log.Fatalf("unknown flags: %v", flag.Args())
	}

	// env
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		serverAddr = envAddr
	}

	log.Printf("Starting server on %s\n", serverAddr)

	if err := http.ListenAndServe(serverAddr, router()); err != nil {
		log.Fatal(err)
	}
}
