package cache

import (
	"context"
	"sync"
	"time"
)

type entry struct {
	value     string
	expiresAt time.Time
}

// MemoryCache is an in-process cache backed by sync.Map with lazy TTL expiry.
// A TTL of 0 means the entry is not stored (no-op for that call).
type MemoryCache struct {
	m sync.Map
}

var _ Cache = (*MemoryCache)(nil)

// NewMemoryCache returns a new in-process MemoryCache.
func NewMemoryCache() *MemoryCache {
	return &MemoryCache{}
}

// Get returns the cached value for key. Returns ("", false) on a miss or if the entry has expired.
func (c *MemoryCache) Get(_ context.Context, key string) (string, bool) {
	v, ok := c.m.Load(key)
	if !ok {
		return "", false
	}

	e, ok := v.(entry)
	if !ok {
		return "", false
	}

	if time.Now().After(e.expiresAt) {
		c.m.Delete(key)
		return "", false
	}

	return e.value, true
}

// Set stores value under key for the given TTL. A zero TTL is a no-op.
func (c *MemoryCache) Set(_ context.Context, key, value string, ttl time.Duration) error {
	if ttl == 0 {
		return nil
	}

	c.m.Store(key, entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	})

	return nil
}

// Delete removes the cached value for key.
func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.m.Delete(key)
	return nil
}
