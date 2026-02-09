package main

import (
	"io"
	"net/http"
	"time"
)

var retryDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

func doWithRetry(client *http.Client, newReq func() (*http.Request, error)) (*http.Response, error) {
	var resp *http.Response
	var err error
	retries := 3

	for i := 0; i < retries; i++ {
		req, reqErr := newReq()
		if reqErr != nil {
			return nil, reqErr
		}

		resp, err = client.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return resp, nil
		}

		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		time.Sleep(time.Second * time.Duration(i+1))
	}
	return resp, err
}
