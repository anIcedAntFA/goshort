package api_test

import (
	"fmt"
	"net/http"
	"testing"
)

func BenchmarkHandler_Redirect_CacheMiss(b *testing.B) {
	deps := setupTestDeps(b)

	body := mustMarshal(b, map[string]string{"url": "https://example.com/bench"})
	rec := serve(b, deps.router, http.MethodPost, "/api/v1/urls", body)
	if rec.Code != http.StatusCreated {
		b.Fatalf("create: %d", rec.Code)
	}
	created := decodeJSON[createURLResp](b, rec)

	b.ResetTimer()
	for range b.N {
		serve(b, deps.router, http.MethodGet, "/"+created.ShortCode, nil)
	}
}

func BenchmarkHandler_CreateURL(b *testing.B) {
	deps := setupTestDeps(b)
	b.ResetTimer()
	for i := range b.N {
		body := mustMarshal(b, map[string]string{"url": fmt.Sprintf("https://example.com/%d", i)})
		serve(b, deps.router, http.MethodPost, "/api/v1/urls", body)
	}
}
