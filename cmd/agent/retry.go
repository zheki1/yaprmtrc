package main

import (
	"net"
	"net/http"
	"time"
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

	for i := 0; i <= len(retryDelays); i++ {

		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if ne, ok := err.(net.Error); !ok || !ne.Temporary() {
			break
		}

		if i < len(retryDelays) {
			time.Sleep(retryDelays[i])
		}
	}

	return nil, lastErr
}
