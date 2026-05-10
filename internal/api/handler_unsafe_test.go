package api_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/api"
	"github.com/anIcedAntFA/goshort/internal/cache"
	"github.com/anIcedAntFA/goshort/internal/encoder"
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
)

// mockURLChecker is a controllable URLChecker for handler-level tests.
type mockURLChecker struct{ err error }

func (m *mockURLChecker) Check(_ context.Context, _ string) error { return m.err }

func TestCreateURL_UnsafeURL(t *testing.T) {
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

	checker := &mockURLChecker{err: shortener.ErrUnsafeURL}
	svc := shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, checker)
	h := api.NewHandler(svc, cache.NewNoopCache(), slog.Default(), "http://localhost:8080")
	router := api.NewRouter(h, api.RouterConfig{})

	body := mustMarshal(t, map[string]string{"url": "https://example.com"})
	rec := serve(t, router, http.MethodPost, "/api/v1/urls", body)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422; body: %s", rec.Code, rec.Body.String())
	}
	resp := decodeJSON[errResp](t, rec)
	if resp.Error.Code != "unsafe_url" {
		t.Errorf("error.code = %q, want %q", resp.Error.Code, "unsafe_url")
	}
}
