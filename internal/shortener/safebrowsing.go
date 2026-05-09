package shortener

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	safeBrowsingEndpoint = "https://safebrowsing.googleapis.com/v4/threatMatches:find"
	safeBrowsingTimeout  = 2 * time.Second
)

// NoopChecker always returns nil. Used when no API key is configured.
type NoopChecker struct{}

// Check implements URLChecker. Always returns nil (disabled).
func (NoopChecker) Check(_ context.Context, _ string) error { return nil }

// SafeBrowsingChecker calls the Google Safe Browsing Lookup API v4.
// Fails open: any transport or API error returns nil so URL creation is not blocked.
type SafeBrowsingChecker struct {
	apiKey   string
	client   *http.Client
	endpoint string
}

// NewSafeBrowsingChecker creates a checker backed by the Google Safe Browsing API.
func NewSafeBrowsingChecker(apiKey string) *SafeBrowsingChecker {
	return &SafeBrowsingChecker{
		apiKey:   apiKey,
		client:   &http.Client{Timeout: safeBrowsingTimeout},
		endpoint: safeBrowsingEndpoint,
	}
}

type sbRequest struct {
	Client     sbClient     `json:"client"`
	ThreatInfo sbThreatInfo `json:"threatInfo"`
}

type sbClient struct {
	ClientID      string `json:"clientId"`
	ClientVersion string `json:"clientVersion"`
}

type sbThreatInfo struct {
	ThreatTypes      []string  `json:"threatTypes"`
	PlatformTypes    []string  `json:"platformTypes"`
	ThreatEntryTypes []string  `json:"threatEntryTypes"`
	ThreatEntries    []sbEntry `json:"threatEntries"`
}

type sbEntry struct {
	URL string `json:"url"`
}

type sbResponse struct {
	Matches []json.RawMessage `json:"matches"`
}

// Check calls the Safe Browsing API. Returns ErrUnsafeURL if the URL is flagged,
// nil for safe URLs, and nil (fail-open) for any API or network error.
func (c *SafeBrowsingChecker) Check(ctx context.Context, rawURL string) error {
	payload := sbRequest{
		Client: sbClient{ClientID: "goshort", ClientVersion: "0.5.0"},
		ThreatInfo: sbThreatInfo{
			ThreatTypes:      []string{"MALWARE", "SOCIAL_ENGINEERING", "UNWANTED_SOFTWARE"},
			PlatformTypes:    []string{"ANY_PLATFORM"},
			ThreatEntryTypes: []string{"URL"},
			ThreatEntries:    []sbEntry{{URL: rawURL}},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.WarnContext(ctx, "safe browsing: marshal request", "error", err)
		return nil // fail-open
	}

	endpoint := fmt.Sprintf("%s?key=%s", c.endpoint, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		slog.WarnContext(ctx, "safe browsing: build request", "error", err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "safe browsing: request failed", "error", err)
		return nil // fail-open on timeout or network error
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "safe browsing: non-200 response", "status", resp.StatusCode)
		return nil // fail-open on API errors
	}

	var sbResp sbResponse
	if err := json.NewDecoder(resp.Body).Decode(&sbResp); err != nil {
		slog.WarnContext(ctx, "safe browsing: decode response", "error", err)
		return nil
	}

	if len(sbResp.Matches) > 0 {
		return fmt.Errorf("safe browsing check %q: %w", rawURL, ErrUnsafeURL)
	}

	return nil
}
