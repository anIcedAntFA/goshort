package shortener

import "context"

// Service defines the operations for managing shortened URLs.
type Service interface {
	// Create creates a new shortened URL from the given request.
	Create(ctx context.Context, req CreateRequest) (*URL, error)
	// GetByCode retrieves a URL by its short code.
	GetByCode(ctx context.Context, code string) (*URL, error)
	// Delete removes a shortened URL by its short code.
	Delete(ctx context.Context, code string) error
	// List returns a paginated slice of URLs and the total count.
	List(ctx context.Context, opts ListOptions) ([]URL, int, error)
	// IncrementClicks atomically increments the click counter for a URL.
	IncrementClicks(ctx context.Context, code string) error
	// CreateBatch creates multiple shortened URLs in a single call.
	// Per-item failures are returned inside BatchResult.Error; the top-level error
	// is reserved for batch-wide validation (empty slice, exceeds cap).
	CreateBatch(ctx context.Context, reqs []CreateRequest) ([]BatchResult, error)
}
