package shortener

import (
	"context"
	"time"
)

// Cache defines the caching operations for URL lookups.
type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
