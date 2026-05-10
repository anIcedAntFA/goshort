package shortener

import "context"

// PreviewFetcher fetches Open Graph / HTML metadata for a URL.
// On any error the implementation returns empty strings so URL creation
// always succeeds regardless of the target page's availability.
type PreviewFetcher interface {
	Fetch(ctx context.Context, rawURL string) (title, description string, err error)
}

// NoopPreviewFetcher always returns empty metadata. Used in tests and when
// preview fetching is disabled.
type NoopPreviewFetcher struct{}

// Fetch implements PreviewFetcher and always returns empty strings.
func (NoopPreviewFetcher) Fetch(_ context.Context, _ string) (title, description string, err error) {
	return "", "", nil
}
