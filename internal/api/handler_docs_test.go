package api_test

import (
	"net/http"
	"strings"
	"testing"
)

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
