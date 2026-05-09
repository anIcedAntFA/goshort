package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
)

func newDeleteTestCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	return cmd
}

func TestRunDelete_Success(t *testing.T) { //nolint:paralleltest // mutates global serverURL

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

	origServerURL := serverURL
	serverURL = srv.URL
	defer func() { serverURL = origServerURL }()

	if err := runDelete(newDeleteTestCmd(context.Background()), []string{"abc123"}); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !called {
		t.Error("server handler was not called")
	}
}

func TestRunDelete_NotFound(t *testing.T) { //nolint:paralleltest // mutates global serverURL

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

	if err := runDelete(newDeleteTestCmd(context.Background()), []string{"ghost"}); err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}
