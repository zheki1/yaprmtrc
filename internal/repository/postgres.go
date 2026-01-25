package repository

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/zheki1/yaprmtrc.git/internal/models"
)

type PostgresRepository struct {
	conn *pgx.Conn
}

func NewPostgresRepository(conn *pgx.Conn) *PostgresRepository {
	return &PostgresRepository{conn: conn}
}

func (p *PostgresRepository) UpdateGauge(
	ctx context.Context,
	name string,
	value float64,
) error {
	_, err := p.conn.Exec(ctx, `
		INSERT INTO metrics (id, type, value)
		VALUES ($1, 'gauge', $2)
		ON CONFLICT (id) DO UPDATE
		SET value = EXCLUDED.value
	`, name, value)

	return err
}

func (p *PostgresRepository) UpdateCounter(
	ctx context.Context,
	name string,
	delta int64,
) error {
	_, err := p.conn.Exec(ctx, `
		INSERT INTO metrics (id, type, delta)
		VALUES ($1, 'counter', $2)
		ON CONFLICT (id) DO UPDATE
		SET delta = metrics.delta + EXCLUDED.delta
	`, name, delta)

	return err
}

func (p *PostgresRepository) GetGauge(
	ctx context.Context,
	name string,
) (float64, bool, error) {
	var v float64

	err := p.conn.QueryRow(ctx,
		`SELECT value FROM metrics WHERE id=$1 AND type='gauge'`,
		name,
	).Scan(&v)

	if err == pgx.ErrNoRows {
		return 0, false, nil
	}

	return v, true, err
}

func (p *PostgresRepository) GetCounter(
	ctx context.Context,
	name string,
) (int64, bool, error) {
	var v int64

	err := p.conn.QueryRow(ctx,
		`SELECT delta FROM metrics WHERE id=$1 AND type='counter'`,
		name,
	).Scan(&v)

	if err == pgx.ErrNoRows {
		return 0, false, nil
	}

	return v, true, err
}

func (p *PostgresRepository) GetAll(
	ctx context.Context,
) ([]models.Metrics, error) {
	rows, err := p.conn.Query(ctx,
		`SELECT id, type, delta, value FROM metrics`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.Metrics

	for rows.Next() {
		var m models.Metrics

		if err := rows.Scan(
			&m.ID,
			&m.MType,
			&m.Delta,
			&m.Value,
		); err != nil {
			return nil, err
		}

		res = append(res, m)
	}

	return res, rows.Err()
}

func (p *PostgresRepository) Close() error {
	return p.conn.Close(context.Background())
}
