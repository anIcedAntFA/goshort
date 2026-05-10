package cache

import (
	"log/slog"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

// Build creates a Cache implementation based on the driver name.
// Supported drivers: "memory", "redis", anything else returns NoopCache.
// Redis connection failures fall back to NoopCache with a warning.
func Build(driver, redisURL string) shortener.Cache {
	switch driver {
	case "memory":
		return NewMemoryCache()
	case "redis":
		rc, err := NewRedisCache(redisURL)
		if err != nil {
			slog.Warn("redis unavailable, falling back to noop cache", "error", err)
			return NewNoopCache()
		}
		return rc
	default:
		return NewNoopCache()
	}
}
