package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

func setupTestDeps(t *testing.T) testDeps {
	t.Helper()
	store, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		t.Fatalf("create encoder: %v", err)
	}

	svc := shortener.NewService(store, cache.NewNoopCache(), enc)
	h := api.NewHandler(svc, cache.NewNoopCache(), slog.Default(), "http://localhost:8080")
	return testDeps{router: api.NewRouter(h), store: store}
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
func serve(t *testing.T, router http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
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

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decode response body: %v", err)
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
	svc := shortener.NewService(store, mc, enc)
	h := api.NewHandler(svc, mc, slog.Default(), "http://localhost:8080")
	router := api.NewRouter(h)

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
