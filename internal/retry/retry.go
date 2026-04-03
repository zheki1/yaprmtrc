// Package retry реализует политику повторных попыток с возрастающими интервалами (1s, 3s, 5s).
package retry

import (
	"context"
	"time"
)

var retryDelays = []time.Duration{
	1 * time.Second,
	3 * time.Second,
	5 * time.Second,
}

// DoRetry выполняет операцию op с повторными попытками.
// Попытка повторяется, если isRetryable возвращает true для полученной ошибки.
// Максимум 3 повтора с задержками 1, 3 и 5 секунд. Поддерживает отмену через ctx.
func DoRetry(ctx context.Context, isRetryable func(error) bool, op func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var err error
	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err = op()
		if err == nil || !isRetryable(err) {
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
