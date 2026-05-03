package cache

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/redis/go-redis/v9"
)

// RedisCache implements shortener.Cache backed by Redis.
type RedisCache struct {
	client *redis.Client
}

// compile-time interface check.
var _ shortener.Cache = (*RedisCache)(nil)

// NewRedisCache connects to Redis and verifies connectivity via Ping.
// addr accepts either a redis:// URL (e.g. "redis://localhost:6379") or
// a plain host:port string (e.g. "localhost:6379").
func NewRedisCache(addr string) (*RedisCache, error) {
	opts, err := redis.ParseURL(addr)
	if err != nil {
		// Fall back to plain host:port format.
		opts = &redis.Options{Addr: addr}
	}
	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return &RedisCache{client: client}, nil
}

// Get returns the cached string for key. Returns ("", false) on a miss or error.
// Redis errors (other than key-not-found) are logged as warnings and treated as cache misses.
func (c *RedisCache) Get(ctx context.Context, key string) (string, bool) {
	val, err := c.client.Get(ctx, key).Result()
	if err == nil {
		return val, true
	}
	if !errors.Is(err, redis.Nil) {
		slog.Warn("cache get error", "key", key, "error", err)
	}
	return "", false
}

// Set stores value under key with the given TTL. A zero TTL is a no-op.
// Redis errors are logged as warnings; they never fail the calling request.
func (c *RedisCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if ttl == 0 {
		return nil
	}
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		slog.Warn("cache set error", "key", key, "error", err)
	}
	return nil
}

// Delete removes the key from Redis.
// Redis errors are logged as warnings; they never fail the calling request.
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		slog.Warn("cache delete error", "key", key, "error", err)
	}
	return nil
}

// Close shuts down the Redis client connection pool.
func (c *RedisCache) Close() error {
	return c.client.Close()
}
