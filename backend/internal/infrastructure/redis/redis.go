package redis

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client

// Init initializes the Redis client.
func Init(url string) error {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return fmt.Errorf("failed to parse redis url: %w", err)
	}
	client = redis.NewClient(opt)

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}

	slog.Info("redis connection established")
	return nil
}

// Client returns the Redis client.
func Client() *redis.Client {
	return client
}

// Close closes the Redis client.
func Close() {
	if client != nil {
		client.Close()
	}
}
