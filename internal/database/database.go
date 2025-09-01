package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	client *sqlx.DB
}

func (d *DB) Close() error {
	return d.client.Close()
}

func NewDB(connString string) (*DB, error) {
	ctx := context.Background()

	pgxCfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, pgxCfg.ConnString())
	if err != nil {
		return nil, fmt.Errorf("failed to create db connection pool: %w", err)
	}

	client := sqlx.NewDb(stdlib.OpenDBFromPool(pool), "pgx")

	err = client.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		client: client,
	}, nil
}

func (d *DB) HealthCheck(ctx context.Context) error {
	return d.client.PingContext(ctx)
}
