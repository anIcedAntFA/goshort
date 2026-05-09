// Package shortener exposes internal helpers for testing via this file.
// It is compiled only during `go test`.
package shortener

import "net/http"

// NewHTTPPreviewFetcherForTest returns an HTTPPreviewFetcher that skips the
// private-host guard, allowing tests to point at httptest servers on loopback.
func NewHTTPPreviewFetcherForTest(client *http.Client) *HTTPPreviewFetcher {
	return &HTTPPreviewFetcher{
		client:      client,
		skipPrivate: true,
	}
}
