package api_test

import (
	"net/http"
	"testing"
	"time"
)

func TestPublicCreateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        any
		wantStatus  int
		wantErrCode string
		checkResp   func(t *testing.T, resp createURLResp)
	}{
		{
			name:       "valid URL returns 201 with short_url and 30d expiry",
			body:       map[string]string{"url": "https://example.com/path"},
			wantStatus: http.StatusCreated,
			checkResp: func(t *testing.T, resp createURLResp) {
				t.Helper()
				if resp.ShortCode == "" {
					t.Error("expected non-empty short_code")
				}
				if resp.ShortURL == "" {
					t.Error("expected non-empty short_url")
				}
				if resp.OriginalURL != "https://example.com/path" {
					t.Errorf("original_url = %q, want %q", resp.OriginalURL, "https://example.com/path")
				}
				if resp.ExpiresAt == nil {
					t.Fatal("expected non-nil expires_at: public endpoint forces 30d expiry")
				}
				exp, err := time.Parse(time.RFC3339, *resp.ExpiresAt)
				if err != nil {
					t.Fatalf("parse expires_at %q: %v", *resp.ExpiresAt, err)
				}
				want := time.Now().Add(30 * 24 * time.Hour)
				diff := exp.Sub(want)
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Minute {
					t.Errorf("expires_at off by %v, want within 1 minute of 30 days from now", diff)
				}
			},
		},
		{
			name:        "invalid URL returns 400 invalid_url",
			body:        map[string]string{"url": "not-a-url"},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "invalid_url",
		},
		{
			name:        "empty body returns 400 invalid_body",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "invalid_body",
		},
		{
			name:       "honeypot filled returns fake 201 with decoy short_code",
			body:       map[string]string{"url": "https://example.com", "website": "spam"},
			wantStatus: http.StatusCreated,
			checkResp: func(t *testing.T, resp createURLResp) {
				t.Helper()
				if resp.ShortCode != "decoy" {
					t.Errorf("short_code = %q, want %q (honeypot should return decoy)", resp.ShortCode, "decoy")
				}
				if resp.ExpiresAt == nil {
					t.Error("expected non-nil expires_at in decoy response")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := setupTestDeps(t)

			var bodyBytes []byte
			if tc.body != nil {
				bodyBytes = mustMarshal(t, tc.body)
			}
			rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/public", bodyBytes)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantErrCode != "" {
				resp := decodeJSON[errResp](t, rec)
				if resp.Error.Code != tc.wantErrCode {
					t.Errorf("error.code = %q, want %q", resp.Error.Code, tc.wantErrCode)
				}
				return
			}
			if tc.checkResp != nil {
				resp := decodeJSON[createURLResp](t, rec)
				tc.checkResp(t, resp)
			}
		})
	}
}

func TestPublicCreateURL_NoAuthRequired(t *testing.T) {
	t.Parallel()
	// The public endpoint must be accessible without an API key, even when auth is configured.
	deps := setupTestDepsWithAuth(t, "secret-key")

	body := mustMarshal(t, map[string]string{"url": "https://example.com"})
	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/public", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (no auth required on public endpoint); body: %s", rec.Code, rec.Body.String())
	}
}

func TestPublicCreateURL_RateLimit(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	body := mustMarshal(t, map[string]string{"url": "https://example.com"})

	// Public endpoint has burst=5; first 5 requests from the same IP must succeed.
	for i := range 5 {
		rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/public", body)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d should not be rate limited (burst=5)", i+1)
		}
	}

	// 6th request must be rate limited.
	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/public", body)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429 after burst exhausted; body: %s", rec.Code, rec.Body.String())
	}
	resp := decodeJSON[errResp](t, rec)
	if resp.Error.Code != "rate_limited" {
		t.Errorf("error.code = %q, want %q", resp.Error.Code, "rate_limited")
	}
	if ra := rec.Header().Get("Retry-After"); ra == "" {
		t.Error("missing Retry-After header on 429")
	}
}
