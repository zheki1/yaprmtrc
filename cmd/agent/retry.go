package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
)

var retryDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

func doWithRetry(
	client *http.Client,
	req *http.Request,
) (*http.Response, error) {

	var lastErr error

	for i := 0; i < len(retryDelays); i++ {

		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}

		var netErr net.Error

		if errors.As(err, &netErr) && netErr.Timeout() {
		} else if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "connection reset") ||
			strings.Contains(err.Error(), "EOF") {
		} else {
			if pgErr, ok := err.(*pgconn.PgError); ok {
				if pgerrcode.IsConnectionException(pgErr.Code) {
					// Connection exception from PostgreSQL
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}

		time.Sleep(retryDelays[i])
	}

	return nil, lastErr
}
