package shortener

import (
	"context"
	"time"
)

// CreateParams holds the data for persisting a new URL record.
// ID and CreatedAt are assigned by the storage layer.
type CreateParams struct {
	ShortCode   string
	OriginalURL string
	IsCustom    bool
	ExpiresAt   *time.Time
	Title       string
	Description string
}

// Storage defines the persistence interface consumed by the shortener service.
// It is implemented by the storage package (SQLite) and by test mocks.
type Storage interface {
	// CreateURL inserts a new URL record and returns the created row.
	CreateURL(ctx context.Context, params *CreateParams) (URL, error)
	// GetByCode retrieves a URL record by its short code.
	GetByCode(ctx context.Context, code string) (*URL, error)
	// DeleteByCode removes a URL record by its short code.
	DeleteByCode(ctx context.Context, code string) error
	// ListURLs returns a page of URL records ordered by creation time descending.
	ListURLs(ctx context.Context, limit, offset int) ([]URL, error)
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
