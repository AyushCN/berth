package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client
var pubSub *PubSub

// PubSub wraps Redis client for pub/sub operations
type PubSub struct {
	client *redis.Client
}

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

// NewPubSub creates a new PubSub client for pub/sub operations
func NewPubSub(url string) (*PubSub, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &PubSub{client: client}, nil
}

// InitPubSub initializes the Redis pub/sub client.
func InitPubSub(url string) (*PubSub, error) {
	var err error
	pubSub, err = NewPubSub(url)
	if err != nil {
		return nil, err
	}
	return pubSub, nil
}

// Client returns the Redis client.
func Client() *redis.Client {
	return client
}

// PubSub returns the Redis pub/sub client.
func GetPubSub() *PubSub {
	return pubSub
}

// Close closes the Redis client.
func Close() {
	if client != nil {
		client.Close()
	}
	if pubSub != nil {
		pubSub.Close()
	}
}

// Close closes the PubSub connection
func (p *PubSub) Close() error {
	return p.client.Close()
}

// Publish publishes a message to a channel
func (p *PubSub) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	return p.client.Publish(ctx, channel, data).Err()
}

// Subscribe subscribes to a channel
func (p *PubSub) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return p.client.Subscribe(ctx, channel)
}

// Set sets a key with expiration
func (p *PubSub) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return p.client.Set(ctx, key, data, expiration).Err()
}

// Get gets a key
func (p *PubSub) Get(ctx context.Context, key string) (string, error) {
	return p.client.Get(ctx, key).Result()
}

// Delete deletes a key
func (p *PubSub) Delete(ctx context.Context, keys ...string) error {
	return p.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (p *PubSub) Exists(ctx context.Context, keys ...string) (int64, error) {
	return p.client.Exists(ctx, keys...).Result()
}

// SAdd adds members to a set
func (p *PubSub) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return p.client.SAdd(ctx, key, members...).Err()
}

// SRem removes members from a set
func (p *PubSub) SRem(ctx context.Context, key string, members ...interface{}) error {
	return p.client.SRem(ctx, key, members...).Err()
}

// SMembers gets all members of a set
func (p *PubSub) SMembers(ctx context.Context, key string) ([]string, error) {
	return p.client.SMembers(ctx, key).Result()
}

// Expire sets expiration on a key
func (p *PubSub) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return p.client.Expire(ctx, key, expiration).Err()
}
