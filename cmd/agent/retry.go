package main

import (
	"context"
	"errors"
	"log"
	"net"
	"strings"
	"time"
)

var retryDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

func runWithRetries(fn func() error) {
	var lastErr error

	for i := 0; i <= len(retryDelays); i++ {
		err := fn()
		if err == nil {
			return
		}
		lastErr = err

		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Return 1: %v ", err)
			return
		}

		var netErr net.Error

		if errors.As(err, &netErr) && netErr.Timeout() {
		} else if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "connection reset") ||
			strings.Contains(err.Error(), "EOF") {
		} else {
			log.Printf("Return 2: %v ", err)
			return
		}

		if i < len(retryDelays) {
			log.Printf("Retry number end: %v %v %v", i, len(retryDelays), err)
			time.Sleep(retryDelays[i])
		}
	}

	log.Printf("All retry attempts failed: %v", lastErr)
}
