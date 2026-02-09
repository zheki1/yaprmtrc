package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"
)

var retryDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

func doWithRetry(client *http.Client, originalReq *http.Request) (*http.Response, error) {

	var lastErr error

	for i := 0; i <= len(retryDelays); i++ {

		// Создаем новый запрос на каждой итерации, чтобы избежать повторного использования
		req, err := http.NewRequest(originalReq.Method, originalReq.URL.String(), originalReq.Body)
		if err != nil {
			return nil, err
		}

		// Копируем заголовки
		req.Header = originalReq.Header.Clone()

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
			return nil, err
		}

		if i < len(retryDelays) {
			time.Sleep(retryDelays[i])
		}
	}

	return nil, lastErr
}
