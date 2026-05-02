package cache

import (
	"context"
	"time"
)

// NoopCache is a cache implementation that never stores anything.
// It is used when caching is disabled (cache.driver = "none").
type NoopCache struct{}

var _ Cache = (*NoopCache)(nil)

// NewNoopCache returns a new NoopCache.
func NewNoopCache() *NoopCache {
	return &NoopCache{}
}

// Get always returns ("", false) — the noop cache never has entries.
func (NoopCache) Get(_ context.Context, _ string) (string, bool) {
	return "", false
}

// Set is a no-op.
func (NoopCache) Set(_ context.Context, _, _ string, _ time.Duration) error {
	return nil
}

// Delete is a no-op.
func (NoopCache) Delete(_ context.Context, _ string) error {
	return nil
}
