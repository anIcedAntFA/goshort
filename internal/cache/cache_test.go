package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/cache"
)

func TestMemoryCache_SetGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		key   string
		value string
		ttl   time.Duration
	}{
		{
			name:  "short ttl",
			key:   "key1",
			value: "https://example.com",
			ttl:   10 * time.Second,
		},
		{
			name:  "long ttl",
			key:   "key2",
			value: "https://other.com/path",
			ttl:   24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := cache.NewMemoryCache()
			ctx := context.Background()

			if err := c.Set(ctx, tt.key, tt.value, tt.ttl); err != nil {
				t.Fatalf("Set: %v", err)
			}

			got, ok := c.Get(ctx, tt.key)
			if !ok {
				t.Fatal("Get = false, want true")
			}
			if got != tt.value {
				t.Errorf("Get = %q, want %q", got, tt.value)
			}
		})
	}
}

func TestMemoryCache_Get_NonExistent(t *testing.T) {
	t.Parallel()

	c := cache.NewMemoryCache()
	ctx := context.Background()

	got, ok := c.Get(ctx, "missing")
	if ok {
		t.Error("Get(missing) = true, want false")
	}
	if got != "" {
		t.Errorf("Get(missing) = %q, want empty string", got)
	}
}

func TestMemoryCache_Get_Expired(t *testing.T) {
	t.Parallel()

	c := cache.NewMemoryCache()
	ctx := context.Background()

	const ttl = 50 * time.Millisecond
	if err := c.Set(ctx, "exp", "value", ttl); err != nil {
		t.Fatalf("Set: %v", err)
	}

	time.Sleep(3 * ttl) // 150ms — 3× margin for CI reliability

	got, ok := c.Get(ctx, "exp")
	if ok {
		t.Error("Get(expired) = true, want false")
	}
	if got != "" {
		t.Errorf("Get(expired) = %q, want empty string", got)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	t.Parallel()

	c := cache.NewMemoryCache()
	ctx := context.Background()

	if err := c.Set(ctx, "del", "value", time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := c.Delete(ctx, "del"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, ok := c.Get(ctx, "del")
	if ok {
		t.Error("Get after Delete = true, want false")
	}
}

func TestMemoryCache_Delete_NonExistent(t *testing.T) {
	t.Parallel()

	c := cache.NewMemoryCache()
	ctx := context.Background()

	if err := c.Delete(ctx, "ghost"); err != nil {
		t.Errorf("Delete(nonexistent) = %v, want nil", err)
	}
}

func TestMemoryCache_Set_ZeroTTL_NoCaching(t *testing.T) {
	t.Parallel()

	c := cache.NewMemoryCache()
	ctx := context.Background()

	if err := c.Set(ctx, "zero", "value", 0); err != nil {
		t.Fatalf("Set(TTL=0): %v", err)
	}

	_, ok := c.Get(ctx, "zero")
	if ok {
		t.Error("Get after Set(TTL=0) = true, want false (TTL=0 means no caching)")
	}
}

func TestNoopCache_Get(t *testing.T) {
	t.Parallel()

	c := cache.NewNoopCache()
	ctx := context.Background()

	if err := c.Set(ctx, "k", "v", time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok := c.Get(ctx, "k")
	if ok {
		t.Error("NoopCache.Get = true, want false (always miss)")
	}
	if got != "" {
		t.Errorf("NoopCache.Get = %q, want empty string", got)
	}
}

func TestMemoryCache_Concurrent(t *testing.T) {
	t.Parallel()

	c := cache.NewMemoryCache()
	ctx := context.Background()

	const goroutines = 50
	done := make(chan struct{}, goroutines)
	for range goroutines {
		go func() {
			_ = c.Set(ctx, "shared-key", "value", time.Minute)
			_, _ = c.Get(ctx, "shared-key")
			_ = c.Delete(ctx, "shared-key")
			done <- struct{}{}
		}()
	}
	for range goroutines {
		<-done
	}
	// Race detector catches any unsynchronized access.
}

func TestNoopCache_Delete(t *testing.T) {
	t.Parallel()

	c := cache.NewNoopCache()
	ctx := context.Background()

	if err := c.Delete(ctx, "k"); err != nil {
		t.Errorf("NoopCache.Delete = %v, want nil", err)
	}
}
