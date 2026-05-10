package safebrowsing_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/safebrowsing"
	"github.com/anIcedAntFA/goshort/internal/shortener"
)

func TestChecker_SafeURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{}) // empty matches = safe
	}))
	defer srv.Close()

	checker := safebrowsing.NewCheckerForTest(srv.URL, &http.Client{})
	if err := checker.Check(context.Background(), "https://safe.example.com"); err != nil {
		t.Errorf("Check(safe URL) = %v, want nil", err)
	}
}

func TestChecker_UnsafeURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"matches": []map[string]any{
				{"threatType": "MALWARE", "platformType": "ANY_PLATFORM"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	checker := safebrowsing.NewCheckerForTest(srv.URL, &http.Client{})
	err := checker.Check(context.Background(), "https://evil.example.com")
	if !errors.Is(err, shortener.ErrUnsafeURL) {
		t.Errorf("Check(unsafe URL) = %v, want ErrUnsafeURL", err)
	}
}

func TestChecker_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	checker := safebrowsing.NewCheckerForTest(srv.URL, &http.Client{})
	// Fail-open: API unavailability must not block URL creation.
	if err := checker.Check(context.Background(), "https://example.com"); err != nil {
		t.Errorf("Check(API 500) = %v, want nil (fail-open)", err)
	}
}

func TestChecker_Timeout(t *testing.T) {
	t.Parallel()

	unblock := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case <-unblock:
		case <-time.After(5 * time.Second):
		}
		http.Error(w, "canceled", http.StatusServiceUnavailable)
	}))
	t.Cleanup(func() { close(unblock); srv.Close() })

	client := &http.Client{Timeout: 50 * time.Millisecond}
	checker := safebrowsing.NewCheckerForTest(srv.URL, client)
	// Fail-open: timeout must not block URL creation.
	if err := checker.Check(context.Background(), "https://example.com"); err != nil {
		t.Errorf("Check(timeout) = %v, want nil (fail-open)", err)
	}
}
