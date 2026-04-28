// Package cache defines the caching interface for the shortener service.
package cache

import (
	"context"
	"time"
)

// Cache defines the operations for caching URL lookups.
type Cache interface {
	// Get retrieves a cached value by key. The second return value is false on a cache miss.
	Get(ctx context.Context, key string) (string, bool)
	// Set stores a value under key with the given TTL.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	// Delete removes a cached value by key.
	Delete(ctx context.Context, key string) error
}
