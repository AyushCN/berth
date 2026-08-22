package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool

// Init initializes the PostgreSQL connection pool.
func Init(dsn string) error {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("failed to parse dsn: %w", err)
	}
	config.MaxConns = 100
	config.MinConns = 25
	config.MaxConnLifetime = 5 * time.Minute

	var poolErr error
	for i := 1; i <= 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		tempPool, err := pgxpool.NewWithConfig(ctx, config)
		cancel()
		if err == nil {
			pool = tempPool
			break
		}
		if tempPool != nil {
			tempPool.Close()
		}
		poolErr = err
		slog.Warn("database not ready", "attempt", i, "error", err)
		time.Sleep(2 * time.Second)
	}
	if poolErr != nil {
		return fmt.Errorf("failed to connect to database after 10 attempts: %w", poolErr)
	}

	slog.Info("database connection established")
	return nil
}

// Pool returns the connection pool.
func Pool() *pgxpool.Pool {
	return pool
}

// Close closes the connection pool.
func Close() {
	if pool != nil {
		pool.Close()
	}
}
