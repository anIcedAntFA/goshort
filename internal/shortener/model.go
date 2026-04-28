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
