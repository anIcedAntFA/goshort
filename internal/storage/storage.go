// Package storage defines the persistence interface for the shortener service.
package storage

import (
	"context"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

// CreateParams holds the settable fields for a new URL record, excluding DB-assigned ID and CreatedAt.
type CreateParams struct {
	ShortCode   string
	OriginalURL string
	IsCustom    bool
	ExpiresAt   *time.Time
}

// Storage defines the data persistence operations required by the shortener service.
type Storage interface {
	// CreateURL inserts a new URL record and returns the created row.
	CreateURL(ctx context.Context, params CreateParams) (shortener.URL, error)
	// GetByCode retrieves a URL record by its short code.
	GetByCode(ctx context.Context, code string) (*shortener.URL, error)
	// DeleteByCode removes a URL record by its short code.
	DeleteByCode(ctx context.Context, code string) error
	// ListURLs returns a page of URL records ordered by creation time descending.
	ListURLs(ctx context.Context, limit, offset int) ([]shortener.URL, error)
	// CountURLs returns the total number of URL records.
	CountURLs(ctx context.Context) (int, error)
	// IncrementClicks atomically increments the click counter for a URL.
	IncrementClicks(ctx context.Context, code string) error
	// DeleteExpired removes expired URL records in batches and returns the count deleted.
	DeleteExpired(ctx context.Context, batchSize int) (int64, error)
	// GetCounter returns the current value of the global URL counter.
	GetCounter(ctx context.Context) (int64, error)
	// IncrementCounter atomically increments the global URL counter and returns the new value.
	IncrementCounter(ctx context.Context) (int64, error)
}
