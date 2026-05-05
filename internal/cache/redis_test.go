//go:build redis

package cache_test

import (
	"context"
	"os"
	"strings"
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

	key := "test:redis:setget:" + t.Name()
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

	key := "test:redis:delete:" + t.Name()
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

	if _, ok := c.Get(ctx, "test:redis:nonexistent:"+t.Name()); ok {
		t.Error("expected miss for nonexistent key")
	}
}

func TestRedisCache_TTLExpiry(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	key := "test:redis:ttl:" + t.Name()
	const ttl = 50 * time.Millisecond
	if err := c.Set(ctx, key, "value", ttl); err != nil {
		t.Fatalf("Set: %v", err)
	}
	time.Sleep(3 * ttl) // 150ms — 3× margin for CI reliability
	if _, ok := c.Get(ctx, key); ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestRedisCache_ZeroTTLIsNoop(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	key := "test:redis:zerott:" + t.Name()
	if err := c.Set(ctx, key, "value", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, ok := c.Get(ctx, key); ok {
		t.Error("expected miss for zero-TTL set")
	}
}

func TestRedisCache_SetGet_Overwrite(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	key := "test:redis:overwrite:" + t.Name()
	if err := c.Set(ctx, key, "first", time.Minute); err != nil {
		t.Fatalf("Set first: %v", err)
	}
	if err := c.Set(ctx, key, "second", time.Minute); err != nil {
		t.Fatalf("Set second: %v", err)
	}

	val, ok := c.Get(ctx, key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if val != "second" {
		t.Errorf("val = %q, want %q", val, "second")
	}
}

func TestRedisCache_Delete_Idempotent(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	// Delete a key that was never set — should not error.
	key := "test:redis:delete-idempotent:" + t.Name()
	if err := c.Delete(ctx, key); err != nil {
		t.Errorf("Delete(nonexistent) = %v, want nil", err)
	}
}

func TestRedisCache_Close(t *testing.T) {
	// Not parallel — we close the client directly without cleanup registration.
	c, err := cache.NewRedisCache(redisAddr(t))
	if err != nil {
		t.Skip("Redis not available:", err)
	}
	ctx := context.Background()

	key := "test:redis:close:" + t.Name()
	if err := c.Set(ctx, key, "value", time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Operations after close should not panic.
	_, _ = c.Get(ctx, key)
}

func TestRedisCache_LargeValue(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)
	ctx := context.Background()

	key := "test:redis:large:" + t.Name()
	large := strings.Repeat("x", 1024*1024) // 1 MB
	if err := c.Set(ctx, key, large, time.Minute); err != nil {
		t.Fatalf("Set large value: %v", err)
	}

	val, ok := c.Get(ctx, key)
	if !ok {
		t.Fatal("expected cache hit for large value")
	}
	if len(val) != len(large) {
		t.Errorf("retrieved value length = %d, want %d", len(val), len(large))
	}
}

func TestRedisCache_ContextCancellation(t *testing.T) {
	t.Parallel()
	c := newTestRedis(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Operations with cancelled context should not panic.
	_, _ = c.Get(ctx, "test:redis:cancelled:"+t.Name())
	_ = c.Set(ctx, "test:redis:cancelled:"+t.Name(), "v", time.Minute)
	_ = c.Delete(ctx, "test:redis:cancelled:"+t.Name())
}

func TestNewRedisCache_URLFormat(t *testing.T) {
	t.Parallel()
	c, err := cache.NewRedisCache("redis://localhost:6379")
	if err != nil {
		t.Skip("Redis not available:", err)
	}
	defer func() { _ = c.Close() }()

	ctx := context.Background()
	key := "test:redis:urlformat:" + t.Name()
	if err := c.Set(ctx, key, "works", time.Minute); err != nil {
		t.Fatalf("Set via URL format: %v", err)
	}
	val, ok := c.Get(ctx, key)
	if !ok || val != "works" {
		t.Errorf("Get = (%q, %v), want (\"works\", true)", val, ok)
	}
}
