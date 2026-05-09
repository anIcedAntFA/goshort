// Package shortener implements the core URL shortening business logic.
package shortener

import "time"

// URL represents a shortened URL entry in the system.
type URL struct {
	ID          int64
	ShortCode   string
	OriginalURL string
	IsCustom    bool
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	ClickCount  int64
	Title       string
	Description string
}

// CreateRequest holds the input data for creating a new shortened URL.
type CreateRequest struct {
	URL         string
	CustomAlias string
	ExpiresIn   string
}

// ListOptions configures pagination for listing URLs.
type ListOptions struct {
	Page    int
	PerPage int
}

// BatchResult holds the outcome of a single item in a CreateBatch call.
// Exactly one of URL or Error is non-nil.
type BatchResult struct {
	URL   *URL
	Error error
}
