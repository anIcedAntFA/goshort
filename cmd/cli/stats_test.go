package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
)

func newStatsTestCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	return cmd
}

func TestRunStats_Success(t *testing.T) { //nolint:paralleltest // mutates global serverURL

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/v1/urls/abc123" {
			t.Errorf("path = %s, want /api/v1/urls/abc123", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(URLDetail{
			ShortCode:   "abc123",
			ShortURL:    "http://localhost:8080/abc123",
			OriginalURL: "https://example.com",
			ClickCount:  42,
			CreatedAt:   "2025-06-01T12:00:00Z",
		})
	}))
	defer srv.Close()

	origServerURL := serverURL
	serverURL = srv.URL
	defer func() { serverURL = origServerURL }()

	if err := runStats(newStatsTestCmd(context.Background()), []string{"abc123"}); err != nil {
		t.Fatalf("runStats: %v", err)
	}
}

func TestRunStats_NotFound(t *testing.T) { //nolint:paralleltest // mutates global serverURL

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "not_found", "message": "not found"},
		})
	}))
	defer srv.Close()

	origServerURL := serverURL
	serverURL = srv.URL
	defer func() { serverURL = origServerURL }()

	if err := runStats(newStatsTestCmd(context.Background()), []string{"ghost"}); err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}
