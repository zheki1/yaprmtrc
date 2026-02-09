package main

import (
	"time"

	"github.com/go-resty/resty/v2"
)

func newRestyClient(retries int, timeout time.Duration) *resty.Client {
	client := resty.New()
	client.SetRetryCount(retries)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)
	client.SetTimeout(timeout)

	client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			if err != nil {
				return true
			}
			if r.StatusCode() >= 500 {
				return true
			}
			return false
		},
	)

	return client
}
