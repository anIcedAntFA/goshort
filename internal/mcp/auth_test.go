package mcp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	mcpserver "github.com/anIcedAntFA/goshort/internal/mcp"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAPIKeyMiddleware_NoKeyRequired(t *testing.T) {
	handler := mcpserver.APIKeyMiddleware("", okHandler())
	req := httptest.NewRequest(http.MethodPost, "/mcp", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (no auth required)", rec.Code)
	}
}

func TestAPIKeyMiddleware_MissingKey(t *testing.T) {
	handler := mcpserver.APIKeyMiddleware("secret", okHandler())
	req := httptest.NewRequest(http.MethodPost, "/mcp", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAPIKeyMiddleware_WrongKey(t *testing.T) {
	handler := mcpserver.APIKeyMiddleware("secret", okHandler())
	req := httptest.NewRequest(http.MethodPost, "/mcp", http.NoBody)
	req.Header.Set("X-API-Key", "wrong")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAPIKeyMiddleware_CorrectKey(t *testing.T) {
	handler := mcpserver.APIKeyMiddleware("secret", okHandler())
	req := httptest.NewRequest(http.MethodPost, "/mcp", http.NoBody)
	req.Header.Set("X-API-Key", "secret")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
