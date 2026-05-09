package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newShortenTestCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("alias", "", "")
	cmd.Flags().String("expires", "", "")
	cmd.SetContext(ctx)
	return cmd
}

func TestRunShorten_Stdin(t *testing.T) { //nolint:paralleltest // mutates os.Stdin and serverURL globals

	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotURL = req.URL
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(CreateResponse{
			ShortCode:   "xyz999",
			ShortURL:    "http://localhost/xyz999",
			OriginalURL: req.URL,
		})
	}))
	defer srv.Close()

	origServerURL := serverURL
	serverURL = srv.URL
	defer func() { serverURL = origServerURL }()

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_, _ = pw.WriteString("https://example.com/stdin-test\n")
	_ = pw.Close()

	origStdin := os.Stdin
	os.Stdin = pr
	defer func() { os.Stdin = origStdin; _ = pr.Close() }()

	if err := runShorten(newShortenTestCmd(context.Background()), nil); err != nil {
		t.Fatalf("runShorten: %v", err)
	}
	if gotURL != "https://example.com/stdin-test" {
		t.Errorf("server received URL = %q, want https://example.com/stdin-test", gotURL)
	}
}

func TestRunShorten_StdinEmpty(t *testing.T) { //nolint:paralleltest // mutates os.Stdin global

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	_ = pw.Close() // EOF immediately → empty stdin

	origStdin := os.Stdin
	os.Stdin = pr
	defer func() { os.Stdin = origStdin; _ = pr.Close() }()

	err = runShorten(newShortenTestCmd(context.Background()), nil)
	if err == nil {
		t.Fatal("expected error for empty stdin")
	}
	if !strings.Contains(err.Error(), "url is required") {
		t.Errorf("error = %q, want containing 'url is required'", err.Error())
	}
}
