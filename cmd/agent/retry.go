package main

import (
	"context"
	"errors"
	"net"
	"net/url"
	"time"
)

var retryDelays = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

func DoRetry(ctx context.Context, op func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var err error
	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err = op()
		if err == nil || !isRetryableNetErr(err) {
			return err
		}

		if attempt == len(retryDelays) {
			break
		}

		if err := wait(ctx, retryDelays[attempt]); err != nil {
			return err
		}
	}

	return err
}

func wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isRetryableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var urlErr *url.Error
	return errors.As(err, &urlErr)
}
