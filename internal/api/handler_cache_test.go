package api_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/api"
	"github.com/anIcedAntFA/goshort/internal/cache"
	"github.com/anIcedAntFA/goshort/internal/encoder"
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
)

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
	svc := shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, shortener.NoopChecker{})
	h := api.NewHandler(svc, mc, slog.Default(), "http://localhost:8080")
	router := api.NewRouter(h, api.RouterConfig{})

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
