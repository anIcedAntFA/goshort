package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/api"
	"github.com/anIcedAntFA/goshort/internal/cache"
	"github.com/anIcedAntFA/goshort/internal/encoder"
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
)

type testDeps struct {
	router http.Handler
	store  *storage.SQLiteStorage
}

func setupTestDeps(tb testing.TB) testDeps {
	tb.Helper()
	return setupTestDepsWithConfig(tb, api.RouterConfig{})
}

func setupTestDepsWithAuth(t *testing.T, apiKey string) testDeps {
	t.Helper()
	return setupTestDepsWithConfig(t, api.RouterConfig{APIKey: apiKey})
}

func setupTestDepsWithConfig(tb testing.TB, rcfg api.RouterConfig) testDeps {
	tb.Helper()
	store, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
	if err != nil {
		tb.Fatalf("create storage: %v", err)
	}
	tb.Cleanup(func() { _ = store.Close() })

	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		tb.Fatalf("create encoder: %v", err)
	}

	svc := shortener.NewService(store, enc)
	h := api.NewHandler(svc, cache.NewNoopCache(), slog.Default(), "http://localhost:8080")
	return testDeps{router: api.NewRouter(h, rcfg), store: store}
}

// Response structs for decoding test responses.

type createURLResp struct {
	ShortCode   string  `json:"short_code"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	ExpiresAt   *string `json:"expires_at"`
	CreatedAt   string  `json:"created_at"`
}

type urlResp struct {
	ShortCode   string  `json:"short_code"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	IsCustom    bool    `json:"is_custom"`
	ExpiresAt   *string `json:"expires_at"`
	CreatedAt   string  `json:"created_at"`
	ClickCount  int64   `json:"click_count"`
}

type listResp struct {
	Data       []urlResp    `json:"data"`
	Pagination paginateMeta `json:"pagination"`
}

type paginateMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type errResp struct {
	Error errDetail `json:"error"`
}

type errDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// serve executes a single request against the router and returns the recorder.
func serve(tb testing.TB, router http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	tb.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, http.NoBody)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func mustMarshal(tb testing.TB, v any) []byte {
	tb.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		tb.Fatalf("marshal: %v", err)
	}
	return b
}

func decodeJSON[T any](tb testing.TB, rec *httptest.ResponseRecorder) T {
	tb.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		tb.Fatalf("decode response body: %v", err)
	}
	return v
}

func TestCreateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        any
		wantStatus  int
		wantErrCode string
		checkResp   func(t *testing.T, resp createURLResp)
	}{
		{
			name:       "valid URL returns 201",
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
				if resp.CreatedAt == "" {
					t.Error("expected non-empty created_at")
				}
				if resp.ExpiresAt != nil {
					t.Errorf("expected nil expires_at, got %q", *resp.ExpiresAt)
				}
			},
		},
		{
			name: "custom alias becomes short code",
			body: map[string]string{
				"url":          "https://example.com",
				"custom_alias": "my-link",
			},
			wantStatus: http.StatusCreated,
			checkResp: func(t *testing.T, resp createURLResp) {
				t.Helper()
				if resp.ShortCode != "my-link" {
					t.Errorf("short_code = %q, want %q", resp.ShortCode, "my-link")
				}
				wantURL := "http://localhost:8080/my-link"
				if resp.ShortURL != wantURL {
					t.Errorf("short_url = %q, want %q", resp.ShortURL, wantURL)
				}
			},
		},
		{
			name: "expires_in 7d sets expires_at to approximately 7 days from now",
			body: map[string]string{
				"url":        "https://example.com",
				"expires_in": "7d",
			},
			wantStatus: http.StatusCreated,
			checkResp: func(t *testing.T, resp createURLResp) {
				t.Helper()
				if resp.ExpiresAt == nil {
					t.Fatal("expected non-nil expires_at")
				}
				exp, err := time.Parse(time.RFC3339, *resp.ExpiresAt)
				if err != nil {
					t.Fatalf("parse expires_at %q: %v", *resp.ExpiresAt, err)
				}
				want := time.Now().Add(7 * 24 * time.Hour)
				diff := exp.Sub(want)
				if diff < 0 {
					diff = -diff
				}
				if diff > time.Minute {
					t.Errorf("expires_at off by %v, want within 1 minute of 7 days", diff)
				}
			},
		},
		{
			name:        "empty body returns 400 invalid_body",
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "invalid_body",
		},
		{
			name:        "non-URL string returns 400 invalid_url",
			body:        map[string]string{"url": "not-a-url"},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "invalid_url",
		},
		{
			name:        "invalid expires_in returns 400 invalid_expires",
			body:        map[string]string{"url": "https://example.com", "expires_in": "abc"},
			wantStatus:  http.StatusBadRequest,
			wantErrCode: "invalid_expires",
		},
		{
			name:        "alias too short returns 422 invalid_alias",
			body:        map[string]string{"url": "https://example.com", "custom_alias": "a"},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "invalid_alias",
		},
		{
			name:        "reserved alias returns 422 reserved_path",
			body:        map[string]string{"url": "https://example.com", "custom_alias": "api"},
			wantStatus:  http.StatusUnprocessableEntity,
			wantErrCode: "reserved_path",
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
			rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls", bodyBytes)

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

func TestCreateURL_DuplicateAlias(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	body := mustMarshal(t, map[string]string{
		"url":          "https://example.com",
		"custom_alias": "my-link",
	})

	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first create: status = %d; body: %s", rec.Code, rec.Body.String())
	}

	rec = serve(t, deps.router, http.MethodPost, "/api/v1/urls", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate: status = %d, want 409; body: %s", rec.Code, rec.Body.String())
	}
	resp := decodeJSON[errResp](t, rec)
	if resp.Error.Code != "alias_taken" {
		t.Errorf("error.code = %q, want %q", resp.Error.Code, "alias_taken")
	}
}

func TestRedirect(t *testing.T) {
	t.Parallel()

	t.Run("existing code returns 302 with Location and Cache-Control", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)

		body := mustMarshal(t, map[string]string{"url": "https://example.com/target"})
		createRec := serve(t, deps.router, http.MethodPost, "/api/v1/urls", body)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("create: status = %d; body: %s", createRec.Code, createRec.Body.String())
		}
		created := decodeJSON[createURLResp](t, createRec)

		rec := serve(t, deps.router, http.MethodGet, "/"+created.ShortCode, nil)
		if rec.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302; body: %s", rec.Code, rec.Body.String())
		}
		if loc := rec.Header().Get("Location"); loc != "https://example.com/target" {
			t.Errorf("Location = %q, want %q", loc, "https://example.com/target")
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "private, max-age=0, no-cache" {
			t.Errorf("Cache-Control = %q, want %q", cc, "private, max-age=0, no-cache")
		}
	})

	t.Run("nonexistent code returns 404", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)

		rec := serve(t, deps.router, http.MethodGet, "/doesnotexist", nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
		resp := decodeJSON[errResp](t, rec)
		if resp.Error.Code != "not_found" {
			t.Errorf("error.code = %q, want %q", resp.Error.Code, "not_found")
		}
	})

	t.Run("expired URL returns 410", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)

		past := time.Now().Add(-time.Hour)
		_, err := deps.store.CreateURL(context.Background(), shortener.CreateParams{
			ShortCode:   "expired",
			OriginalURL: "https://example.com",
			ExpiresAt:   &past,
		})
		if err != nil {
			t.Fatalf("insert expired URL: %v", err)
		}

		rec := serve(t, deps.router, http.MethodGet, "/expired", nil)
		if rec.Code != http.StatusGone {
			t.Fatalf("status = %d, want 410; body: %s", rec.Code, rec.Body.String())
		}
		resp := decodeJSON[errResp](t, rec)
		if resp.Error.Code != "expired" {
			t.Errorf("error.code = %q, want %q", resp.Error.Code, "expired")
		}
	})
}

func TestListURLs(t *testing.T) {
	t.Parallel()

	createURLs := func(t *testing.T, deps testDeps, n int) {
		t.Helper()
		for i := range n {
			body := mustMarshal(t, map[string]string{
				"url": fmt.Sprintf("https://example.com/%d", i),
			})
			rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls", body)
			if rec.Code != http.StatusCreated {
				t.Fatalf("create url %d: status = %d; body: %s", i, rec.Code, rec.Body.String())
			}
		}
	}

	t.Run("empty DB returns empty data with zero total", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)

		rec := serve(t, deps.router, http.MethodGet, "/api/v1/urls", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		resp := decodeJSON[listResp](t, rec)
		if len(resp.Data) != 0 {
			t.Errorf("data len = %d, want 0", len(resp.Data))
		}
		if resp.Pagination.Total != 0 {
			t.Errorf("total = %d, want 0", resp.Pagination.Total)
		}
	})

	t.Run("page 1 per_page 3 returns first 3 of 5 with correct pagination", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)
		createURLs(t, deps, 5)

		rec := serve(t, deps.router, http.MethodGet, "/api/v1/urls?page=1&per_page=3", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		resp := decodeJSON[listResp](t, rec)
		if len(resp.Data) != 3 {
			t.Errorf("data len = %d, want 3", len(resp.Data))
		}
		if resp.Pagination.Total != 5 {
			t.Errorf("total = %d, want 5", resp.Pagination.Total)
		}
		if resp.Pagination.Page != 1 {
			t.Errorf("page = %d, want 1", resp.Pagination.Page)
		}
		if resp.Pagination.PerPage != 3 {
			t.Errorf("per_page = %d, want 3", resp.Pagination.PerPage)
		}
		if resp.Pagination.TotalPages != 2 {
			t.Errorf("total_pages = %d, want 2", resp.Pagination.TotalPages)
		}
	})

	t.Run("page 2 per_page 3 returns remaining 2 of 5", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)
		createURLs(t, deps, 5)

		rec := serve(t, deps.router, http.MethodGet, "/api/v1/urls?page=2&per_page=3", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		resp := decodeJSON[listResp](t, rec)
		if len(resp.Data) != 2 {
			t.Errorf("data len = %d, want 2", len(resp.Data))
		}
		if resp.Pagination.Total != 5 {
			t.Errorf("total = %d, want 5", resp.Pagination.Total)
		}
	})

	t.Run("default page and per_page applied when absent", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)
		createURLs(t, deps, 3)

		rec := serve(t, deps.router, http.MethodGet, "/api/v1/urls", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		resp := decodeJSON[listResp](t, rec)
		if resp.Pagination.Page != 1 {
			t.Errorf("page = %d, want 1 (default)", resp.Pagination.Page)
		}
		if resp.Pagination.PerPage != 20 {
			t.Errorf("per_page = %d, want 20 (default)", resp.Pagination.PerPage)
		}
	})
}

func TestGetURL(t *testing.T) {
	t.Parallel()

	t.Run("existing code returns 200 with full URL detail", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)

		body := mustMarshal(t, map[string]string{
			"url":          "https://example.com",
			"custom_alias": "detail-link",
		})
		createRec := serve(t, deps.router, http.MethodPost, "/api/v1/urls", body)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("create: status = %d; body: %s", createRec.Code, createRec.Body.String())
		}

		rec := serve(t, deps.router, http.MethodGet, "/api/v1/urls/detail-link", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
		}
		resp := decodeJSON[urlResp](t, rec)
		if resp.ShortCode != "detail-link" {
			t.Errorf("short_code = %q, want %q", resp.ShortCode, "detail-link")
		}
		if resp.OriginalURL != "https://example.com" {
			t.Errorf("original_url = %q, want %q", resp.OriginalURL, "https://example.com")
		}
		if !resp.IsCustom {
			t.Error("is_custom = false, want true for custom alias")
		}
		wantShortURL := "http://localhost:8080/detail-link"
		if resp.ShortURL != wantShortURL {
			t.Errorf("short_url = %q, want %q", resp.ShortURL, wantShortURL)
		}
	})

	t.Run("nonexistent code returns 404", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)

		rec := serve(t, deps.router, http.MethodGet, "/api/v1/urls/doesnotexist", nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
		resp := decodeJSON[errResp](t, rec)
		if resp.Error.Code != "not_found" {
			t.Errorf("error.code = %q, want %q", resp.Error.Code, "not_found")
		}
	})
}

func TestDeleteURL(t *testing.T) {
	t.Parallel()

	t.Run("existing code returns 204 with empty body", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)

		body := mustMarshal(t, map[string]string{"url": "https://example.com"})
		createRec := serve(t, deps.router, http.MethodPost, "/api/v1/urls", body)
		if createRec.Code != http.StatusCreated {
			t.Fatalf("create: status = %d; body: %s", createRec.Code, createRec.Body.String())
		}
		created := decodeJSON[createURLResp](t, createRec)

		rec := serve(t, deps.router, http.MethodDelete, "/api/v1/urls/"+created.ShortCode, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204; body: %s", rec.Code, rec.Body.String())
		}
		if rec.Body.Len() != 0 {
			t.Errorf("expected empty body, got %q", rec.Body.String())
		}
	})

	t.Run("nonexistent code returns 404", func(t *testing.T) {
		t.Parallel()
		deps := setupTestDeps(t)

		rec := serve(t, deps.router, http.MethodDelete, "/api/v1/urls/doesnotexist", nil)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
		resp := decodeJSON[errResp](t, rec)
		if resp.Error.Code != "not_found" {
			t.Errorf("error.code = %q, want %q", resp.Error.Code, "not_found")
		}
	})
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	rec := serve(t, deps.router, http.MethodGet, "/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

func TestRedirect_CacheHit(t *testing.T) {
	t.Parallel()

	store, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		t.Fatalf("create encoder: %v", err)
	}

	mc := cache.NewMemoryCache()
	svc := shortener.NewService(store, enc)
	h := api.NewHandler(svc, mc, slog.Default(), "http://localhost:8080")
	router := api.NewRouter(h, api.RouterConfig{})

	// Pre-populate cache — the code does not exist in the DB.
	if err := mc.Set(context.Background(), "short:cached-code", "https://cached.example.com", time.Hour); err != nil {
		t.Fatalf("cache set: %v", err)
	}

	rec := serve(t, router, http.MethodGet, "/cached-code", nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302; body: %s", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "https://cached.example.com" {
		t.Errorf("Location = %q, want %q", loc, "https://cached.example.com")
	}
}

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

		serve(t, deps.router, http.MethodGet, "/api/v1/urls", nil) // exhaust limit

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
		deps.router.ServeHTTP(recA, reqA) // exhausts IP A's limit

		reqB := httptest.NewRequest(http.MethodGet, "/api/v1/urls", http.NoBody)
		reqB.RemoteAddr = "2.2.2.2:12345"
		recB := httptest.NewRecorder()
		deps.router.ServeHTTP(recB, reqB)

		if recB.Code == http.StatusTooManyRequests {
			t.Fatal("IP B should have its own rate limit independent of IP A")
		}
	})
}

func TestServeDocs(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	rec := serve(t, deps.router, http.MethodGet, "/docs", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/html; charset=utf-8")
	}
	body := rec.Body.String()
	if !strings.Contains(body, "GoShort API") {
		t.Error("body does not contain page title")
	}
	if !strings.Contains(body, "/docs/openapi.yaml") {
		t.Error("body does not reference OpenAPI spec URL")
	}
}

func TestRedirect_ClickCountIncrements(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	body := mustMarshal(t, map[string]string{"url": "https://example.com/clicks"})
	createRec := serve(t, deps.router, http.MethodPost, "/api/v1/urls", body)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create: %d; body: %s", createRec.Code, createRec.Body.String())
	}
	created := decodeJSON[createURLResp](t, createRec)

	rec := serve(t, deps.router, http.MethodGet, "/"+created.ShortCode, nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("redirect: %d; body: %s", rec.Code, rec.Body.String())
	}

	// Allow the fire-and-forget IncrementClicks goroutine to complete.
	time.Sleep(50 * time.Millisecond)

	detailRec := serve(t, deps.router, http.MethodGet, "/api/v1/urls/"+created.ShortCode, nil)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("get url: %d; body: %s", detailRec.Code, detailRec.Body.String())
	}
	detail := decodeJSON[urlResp](t, detailRec)
	if detail.ClickCount < 1 {
		t.Errorf("ClickCount = %d, want >= 1 after redirect", detail.ClickCount)
	}
}

func TestCacheTTL(t *testing.T) {
	t.Parallel()

	now := time.Now()
	nearFuture := now.Add(30 * time.Minute)
	farFuture := now.Add(48 * time.Hour)
	past := now.Add(-time.Hour)

	cases := []struct {
		name    string
		url     *shortener.URL
		wantMin time.Duration
		wantMax time.Duration
	}{
		{
			name:    "no expiry returns 24h",
			url:     &shortener.URL{ExpiresAt: nil},
			wantMin: 23*time.Hour + 59*time.Minute,
			wantMax: 24*time.Hour + time.Second,
		},
		{
			name:    "expires in 30 min returns short TTL",
			url:     &shortener.URL{ExpiresAt: &nearFuture},
			wantMin: 29 * time.Minute,
			wantMax: 31 * time.Minute,
		},
		{
			name:    "expires in 48h is capped at 24h",
			url:     &shortener.URL{ExpiresAt: &farFuture},
			wantMin: 23*time.Hour + 59*time.Minute,
			wantMax: 24*time.Hour + time.Second,
		},
		{
			name:    "already expired returns zero TTL",
			url:     &shortener.URL{ExpiresAt: &past},
			wantMin: 0,
			wantMax: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := api.CacheTTL(tc.url)
			if got < tc.wantMin || got > tc.wantMax {
				t.Errorf("CacheTTL = %v, want [%v, %v]", got, tc.wantMin, tc.wantMax)
			}
		})
	}
}

func TestServeOpenAPISpec(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	rec := serve(t, deps.router, http.MethodGet, "/docs/openapi.yaml", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() == 0 {
		t.Error("expected non-empty OpenAPI spec body")
	}
}
