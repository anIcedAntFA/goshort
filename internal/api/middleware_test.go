package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/api"
)

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		apiKey      string
		headerKey   string
		method      string
		path        string
		wantStatus  int
		wantErrCode string
	}{
		{
			name:   "no header on protected route returns 401",
			apiKey: "test-key", headerKey: "",
			method: http.MethodGet, path: "/api/v1/urls",
			wantStatus: http.StatusUnauthorized, wantErrCode: "unauthorized",
		},
		{
			name:   "wrong key on protected route returns 401",
			apiKey: "test-key", headerKey: "wrong-key",
			method: http.MethodGet, path: "/api/v1/urls",
			wantStatus: http.StatusUnauthorized, wantErrCode: "unauthorized",
		},
		{
			name:   "correct key on protected route returns 200",
			apiKey: "test-key", headerKey: "test-key",
			method: http.MethodGet, path: "/api/v1/urls",
			wantStatus: http.StatusOK,
		},
		{
			name:   "correct key on POST returns 400 not 401",
			apiKey: "test-key", headerKey: "test-key",
			method: http.MethodPost, path: "/api/v1/urls",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "redirect without key returns 404 not 401",
			apiKey: "test-key", headerKey: "",
			method: http.MethodGet, path: "/some-code",
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "health without key returns 200",
			apiKey: "test-key", headerKey: "",
			method: http.MethodGet, path: "/health",
			wantStatus: http.StatusOK,
		},
		{
			name:   "auth disabled allows request without key",
			apiKey: "", headerKey: "",
			method: http.MethodGet, path: "/api/v1/urls",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := setupTestDepsWithAuth(t, tc.apiKey)

			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			if tc.headerKey != "" {
				req.Header.Set("X-API-Key", tc.headerKey)
			}
			rec := httptest.NewRecorder()
			deps.router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantErrCode != "" {
				resp := decodeJSON[errResp](t, rec)
				if resp.Error.Code != tc.wantErrCode {
					t.Errorf("error.code = %q, want %q", resp.Error.Code, tc.wantErrCode)
				}
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("disabled allows unlimited requests", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDepsWithConfig(t, api.RouterConfig{
			RateLimitEnabled: false,
			RateLimitRPM:     1,
		})

		for range 10 {
			rec := serve(t, deps.router, http.MethodGet, "/api/v1/urls", nil)
			if rec.Code == http.StatusTooManyRequests {
				t.Fatal("rate limiter should be disabled")
			}
		}
	})

	t.Run("enabled returns 429 after exceeding burst", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDepsWithConfig(t, api.RouterConfig{
			RateLimitEnabled: true,
			RateLimitRPM:     3,
		})

		for i := range 3 {
			rec := serve(t, deps.router, http.MethodGet, "/api/v1/urls", nil)
			if rec.Code == http.StatusTooManyRequests {
				t.Fatalf("request %d should not be rate limited", i+1)
			}
		}

		rec := serve(t, deps.router, http.MethodGet, "/api/v1/urls", nil)
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429", rec.Code)
		}
		resp := decodeJSON[errResp](t, rec)
		if resp.Error.Code != "rate_limited" {
			t.Errorf("error.code = %q, want %q", resp.Error.Code, "rate_limited")
		}
		if ra := rec.Header().Get("Retry-After"); ra == "" {
			t.Error("missing Retry-After header")
		}
	})

	t.Run("public routes are not rate limited", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDepsWithConfig(t, api.RouterConfig{
			RateLimitEnabled: true,
			RateLimitRPM:     1,
		})

		serve(t, deps.router, http.MethodGet, "/api/v1/urls", nil)

		rec := serve(t, deps.router, http.MethodGet, "/health", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("health status = %d, want 200", rec.Code)
		}
	})

	t.Run("different IPs have independent limits", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDepsWithConfig(t, api.RouterConfig{
			RateLimitEnabled: true,
			RateLimitRPM:     1,
		})

		reqA := httptest.NewRequest(http.MethodGet, "/api/v1/urls", http.NoBody)
		reqA.RemoteAddr = "1.1.1.1:12345"
		recA := httptest.NewRecorder()
		deps.router.ServeHTTP(recA, reqA)

		reqB := httptest.NewRequest(http.MethodGet, "/api/v1/urls", http.NoBody)
		reqB.RemoteAddr = "2.2.2.2:12345"
		recB := httptest.NewRecorder()
		deps.router.ServeHTTP(recB, reqB)

		if recB.Code == http.StatusTooManyRequests {
			t.Fatal("IP B should have its own rate limit independent of IP A")
		}
	})
}
