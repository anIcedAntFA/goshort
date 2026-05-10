package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

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

	svc := shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, shortener.NoopChecker{})
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
