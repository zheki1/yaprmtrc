package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zheki1/yaprmtrc/internal/models"
	"github.com/zheki1/yaprmtrc/internal/retry"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (p *PostgresRepository) UpdateGauge(
	ctx context.Context,
	name string,
	value float64,
) error {
	return retry.DoRetry(ctx, isRetryablePGErr, func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, err := p.pool.Exec(ctx, `
		INSERT INTO metrics (id, type, value)
		VALUES ($1, 'gauge', $2)
		ON CONFLICT (id) DO UPDATE
		SET value = EXCLUDED.value
	`, name, value)

		return err
	})
}

func (p *PostgresRepository) UpdateCounter(
	ctx context.Context,
	name string,
	delta int64,
) error {
	return retry.DoRetry(ctx, isRetryablePGErr, func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, err := p.pool.Exec(ctx, `
		INSERT INTO metrics (id, type, delta)
		VALUES ($1, 'counter', $2)
		ON CONFLICT (id) DO UPDATE
		SET delta = metrics.delta + EXCLUDED.delta
	`, name, delta)
		return err
	})
}

func (p *PostgresRepository) GetGauge(
	ctx context.Context,
	name string,
) (float64, bool, error) {
	var (
		v  float64
		ok bool
	)
	err := retry.DoRetry(ctx, isRetryablePGErr, func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := p.pool.QueryRow(ctx,
			`SELECT value FROM metrics WHERE id=$1 AND type='gauge'`,
			name,
		).Scan(&v)

		if err == pgx.ErrNoRows {
			v = 0
			ok = false
			return nil
		}
		if err != nil {
			return err
		}

		ok = true
		return nil
	})

	return v, ok, err
}

func (p *PostgresRepository) GetCounter(
	ctx context.Context,
	name string,
) (int64, bool, error) {
	var (
		v  int64
		ok bool
	)
	err := retry.DoRetry(ctx, isRetryablePGErr, func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := p.pool.QueryRow(ctx,
			`SELECT delta FROM metrics WHERE id=$1 AND type='counter'`,
			name,
		).Scan(&v)

		if err == pgx.ErrNoRows {
			v = 0
			ok = false
			return nil
		}
		if err != nil {
			return err
		}

		ok = true
		return nil
	})

	return v, ok, err
}

func (p *PostgresRepository) GetAll(
	ctx context.Context,
) ([]models.Metrics, error) {
	var res []models.Metrics

	err := retry.DoRetry(ctx, isRetryablePGErr, func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		rows, err := p.pool.Query(ctx,
			`SELECT id, type, delta, value FROM metrics`,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		var tmp []models.Metrics
		for rows.Next() {
			var m models.Metrics

			if err := rows.Scan(
				&m.ID,
				&m.MType,
				&m.Delta,
				&m.Value,
			); err != nil {
				return err
			}

			tmp = append(tmp, m)
		}

		if err := rows.Err(); err != nil {
			return err
		}

		res = tmp
		return nil
	})

	return res, err
}

func (p *PostgresRepository) UpdateBatch(
	ctx context.Context,
	metrics []models.Metrics,
) error {
	return retry.DoRetry(ctx, isRetryablePGErr, func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		tx, err := p.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		for _, m := range metrics {
			switch m.MType {

			case models.Gauge:
				_, err = tx.Exec(ctx, `
				INSERT INTO metrics (id, type, value)
				VALUES ($1, 'gauge', $2)
				ON CONFLICT (id) DO UPDATE
				SET value = EXCLUDED.value
			`, m.ID, *m.Value)

			case models.Counter:
				_, err = tx.Exec(ctx, `
				INSERT INTO metrics (id, type, delta)
				VALUES ($1, 'counter', $2)
				ON CONFLICT (id) DO UPDATE
				SET delta = metrics.delta + EXCLUDED.delta
			`, m.ID, *m.Delta)
			}

			if err != nil {
				return err
			}
		}

		return tx.Commit(ctx)
	})
}

func (p *PostgresRepository) Close() error {
	p.pool.Close()
	return nil
}

func isRetryablePGErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return strings.HasPrefix(pgErr.Code, "08")
	}

	return false
}
