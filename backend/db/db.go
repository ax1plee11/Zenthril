package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Open открывает пул соединений с PostgreSQL.
func Open(dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db parse config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	return pool, nil
}
