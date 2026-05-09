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

// NewSafeBrowsingCheckerForTest returns a SafeBrowsingChecker that targets
// endpoint instead of the Google Safe Browsing API, for use in unit tests.
func NewSafeBrowsingCheckerForTest(endpoint string, client *http.Client) *SafeBrowsingChecker {
	return &SafeBrowsingChecker{
		apiKey:   "test-key",
		client:   client,
		endpoint: endpoint,
	}
}
