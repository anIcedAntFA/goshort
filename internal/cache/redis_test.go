//go:build redis

package cache_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/cache"
)

func redisAddr(t *testing.T) string {
	t.Helper()
	if addr := os.Getenv("TEST_REDIS_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

func newTestRedis(t *testing.T) *cache.RedisCache {
	t.Helper()
	c, err := cache.NewRedisCache(redisAddr(t))
	if err != nil {
		t.Skip("Redis not available:", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestRedisCache_SetGet(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	key := "test:redis:setget"
	if err := c.Set(ctx, key, "hello", time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, ok := c.Get(ctx, key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if val != "hello" {
		t.Errorf("val = %q, want %q", val, "hello")
	}
}

func TestRedisCache_Delete(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	key := "test:redis:delete"
	if err := c.Set(ctx, key, "value", time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := c.Get(ctx, key); ok {
		t.Error("expected cache miss after delete")
	}
}

func TestRedisCache_MissOnNonExistentKey(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	if _, ok := c.Get(ctx, "test:redis:nonexistent"); ok {
		t.Error("expected miss for nonexistent key")
	}
}

func TestRedisCache_TTLExpiry(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	key := "test:redis:ttl"
	if err := c.Set(ctx, key, "value", 50*time.Millisecond); err != nil {
		t.Fatalf("Set: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, ok := c.Get(ctx, key); ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestRedisCache_ZeroTTLIsNoop(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	key := "test:redis:zerott"
	if err := c.Set(ctx, key, "value", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, ok := c.Get(ctx, key); ok {
		t.Error("expected miss for zero-TTL set")
	}
}
