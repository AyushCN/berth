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
	var err error
	for i := 1; i <= 10; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pool, err = pgxpool.New(ctx, dsn)
		cancel()
		if err == nil {
			break
		}
		slog.Warn("database not ready", "attempt", i, "error", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to database after 10 attempts: %w", err)
	}

	pool.Config().MaxConns = 100
	pool.Config().MinConns = 25
	pool.Config().MaxConnLifetime = 5 * time.Minute

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
