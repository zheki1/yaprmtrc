package main

import (
	"context"
	"errors"
	"strings"
	"time"

	resty "github.com/go-resty/resty/v2"
)

var retryDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

func doWithRetry(req *resty.Request) (*resty.Response, error) {
	var lastErr error

	for i := 0; i <= len(retryDelays); i++ {
		resp, err := req.Execute(req.Method, req.URL)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}

		e := &resty.ResponseError{}
		if errors.As(err, e) && e.Err != nil && strings.Contains(e.Err.Error(), "timeout") {
		} else if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "connection reset") ||
			strings.Contains(err.Error(), "EOF") {
		} else {
			return nil, err
		}

		if i < len(retryDelays) {
			time.Sleep(retryDelays[i])
		}
	}

	return nil, lastErr
}
