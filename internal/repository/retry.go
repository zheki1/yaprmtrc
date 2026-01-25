package repository

import (
	"errors"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
)

var retryDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

func withPgRetry(fn func() error) error {

	var lastErr error

	for i := 0; i <= len(retryDelays); i++ {

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		var pgErr *pgconn.PgError

		if !errors.As(err, &pgErr) {
			return err
		}

		// 08 — Connection Exception
		if !pgerrcode.IsConnectionException(pgErr.Code) {
			return err
		}

		if i < len(retryDelays) {
			time.Sleep(retryDelays[i])
		}
	}

	return lastErr
}
