package shortener

import "context"

// URLChecker verifies a URL before it is persisted.
// Implementations must be safe for concurrent use.
type URLChecker interface {
	// Check returns ErrUnsafeURL when the URL is flagged as dangerous,
	// or nil when the URL is safe or the check is disabled.
	// Implementations must fail-open: transient errors should be logged
	// and nil returned so URL creation is not blocked.
	Check(ctx context.Context, rawURL string) error
}
