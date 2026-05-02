package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIClient_CreateURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/v1/urls" {
			t.Errorf("path = %s, want /api/v1/urls", r.URL.Path)
		}
		if r.Header.Get("X-API-Key") != "test-key" {
			t.Errorf("X-API-Key = %q, want test-key", r.Header.Get("X-API-Key"))
		}
		var req CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.URL != "https://example.com" {
			t.Errorf("url = %q, want https://example.com", req.URL)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateResponse{
			ShortCode:   "abc123",
			ShortURL:    "http://localhost:8080/abc123",
			OriginalURL: "https://example.com",
			CreatedAt:   "2025-06-01T12:00:00Z",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL, "test-key")
	resp, err := client.CreateURL(context.Background(), CreateRequest{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("CreateURL: %v", err)
	}
	if resp.ShortCode != "abc123" {
		t.Errorf("ShortCode = %q, want abc123", resp.ShortCode)
	}
	if resp.ShortURL != "http://localhost:8080/abc123" {
		t.Errorf("ShortURL = %q, want http://localhost:8080/abc123", resp.ShortURL)
	}
}

func TestAPIClient_CreateURL_ErrorResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(apiError{Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{Code: "alias_taken", Message: "The alias is taken"}})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL, "test-key")
	_, err := client.CreateURL(context.Background(), CreateRequest{URL: "https://example.com", CustomAlias: "taken"})
	if err == nil {
		t.Fatal("expected error for 409 response")
	}
	if !strings.Contains(err.Error(), "alias_taken") {
		t.Errorf("error = %q, want containing 'alias_taken'", err.Error())
	}
}

func TestAPIClient_ListURLs(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/urls" {
			t.Errorf("path = %s, want /api/v1/urls", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("page = %q, want 2", r.URL.Query().Get("page"))
		}
		_ = json.NewEncoder(w).Encode(ListResponse{
			Data:       []URLDetail{{ShortCode: "abc", OriginalURL: "https://example.com"}},
			Pagination: PaginationMeta{Page: 2, PerPage: 10, Total: 15, TotalPages: 2},
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL, "")
	resp, err := client.ListURLs(context.Background(), 2, 10)
	if err != nil {
		t.Fatalf("ListURLs: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("len(data) = %d, want 1", len(resp.Data))
	}
	if resp.Pagination.Total != 15 {
		t.Errorf("total = %d, want 15", resp.Pagination.Total)
	}
}

func TestAPIClient_GetURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/urls/abc123" {
			t.Errorf("path = %s, want /api/v1/urls/abc123", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(URLDetail{
			ShortCode:   "abc123",
			OriginalURL: "https://example.com",
			ClickCount:  7,
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL, "")
	u, err := client.GetURL(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetURL: %v", err)
	}
	if u.ClickCount != 7 {
		t.Errorf("ClickCount = %d, want 7", u.ClickCount)
	}
}

func TestAPIClient_DeleteURL(t *testing.T) {
	t.Parallel()

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/api/v1/urls/abc123" {
			t.Errorf("path = %s, want /api/v1/urls/abc123", r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL, "")
	if err := client.DeleteURL(context.Background(), "abc123"); err != nil {
		t.Fatalf("DeleteURL: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestAPIClient_ConnectionError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // closed immediately — all connections will be refused

	client := NewAPIClient(srv.URL, "")
	_, err := client.CreateURL(context.Background(), CreateRequest{URL: "https://example.com"})
	if err == nil {
		t.Fatal("expected error for closed server")
	}
	if !strings.Contains(err.Error(), "request failed") {
		t.Errorf("error = %q, want containing 'request failed'", err.Error())
	}
}

func TestAPIClient_NonJSONError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "internal server error")
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL, "")
	_, err := client.CreateURL(context.Background(), CreateRequest{URL: "https://example.com"})
	if err == nil {
		t.Fatal("expected error for 500 response with non-JSON body")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want containing '500'", err.Error())
	}
}

func TestAPIClient_APIKeyHeader(t *testing.T) {
	t.Parallel()

	t.Run("sends key when set", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("X-API-Key"); got != "my-key" {
				t.Errorf("X-API-Key = %q, want my-key", got)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()
		_ = NewAPIClient(srv.URL, "my-key").DeleteURL(context.Background(), "x")
	})

	t.Run("omits key when empty", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("X-API-Key"); got != "" {
				t.Errorf("X-API-Key = %q, want empty", got)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer srv.Close()
		_ = NewAPIClient(srv.URL, "").DeleteURL(context.Background(), "x")
	})
}
