package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AyushCN/berth/migrations"
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

// Migrate applies any pending schema migrations using dsn.
//
// The schema is embedded in the binary and applied on every boot. This used to
// be a manual `psql` step, which meant a fresh deployment started against an
// empty database and failed every query.
func Migrate(dsn string) (uint, error) {
	version, err := migrations.Migrate(dsn)
	if err != nil {
		return version, err
	}
	slog.Info("schema migrations up to date", "version", version)
	return version, nil
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
