package shortener_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

func TestNoopPreviewFetcher(t *testing.T) {
	t.Parallel()
	f := shortener.NoopPreviewFetcher{}
	title, desc, err := f.Fetch(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "" || desc != "" {
		t.Errorf("want empty strings, got title=%q desc=%q", title, desc)
	}
}

func TestHTTPPreviewFetcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		handler   http.HandlerFunc
		wantTitle string
		wantDesc  string
	}{
		{
			name: "title and description",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte(`<html><head>
					<title>Hello World</title>
					<meta name="description" content="A great page">
				</head><body></body></html>`))
			},
			wantTitle: "Hello World",
			wantDesc:  "A great page",
		},
		{
			name: "only title",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`<html><head><title>Only Title</title></head><body></body></html>`))
			},
			wantTitle: "Only Title",
			wantDesc:  "",
		},
		{
			name: "only description",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`<html><head>
					<meta name="description" content="Only Desc">
				</head><body></body></html>`))
			},
			wantTitle: "",
			wantDesc:  "Only Desc",
		},
		{
			name: "empty body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(""))
			},
			wantTitle: "",
			wantDesc:  "",
		},
		{
			name: "non-2xx response",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantTitle: "",
			wantDesc:  "",
		},
		{
			name: "description case-insensitive meta name",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`<html><head>
					<meta name="Description" content="Case Test">
					<title>CaseTitle</title>
				</head><body></body></html>`))
			},
			wantTitle: "CaseTitle",
			wantDesc:  "Case Test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			// Use ForTest variant so loopback httptest server is not blocked.
			f := shortener.NewHTTPPreviewFetcherForTest(http.DefaultClient)
			title, desc, err := f.Fetch(context.Background(), srv.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if title != tc.wantTitle {
				t.Errorf("title: want %q, got %q", tc.wantTitle, title)
			}
			if desc != tc.wantDesc {
				t.Errorf("description: want %q, got %q", tc.wantDesc, desc)
			}
		})
	}
}

func TestHTTPPreviewFetcher_PrivateHost(t *testing.T) {
	t.Parallel()

	privateURLs := []string{
		"http://localhost/page",
		"http://127.0.0.1/page",
		"http://192.168.1.1/page",
		"http://10.0.0.1/page",
	}

	// Production fetcher — private check enabled.
	f := shortener.NewHTTPPreviewFetcher()
	for _, u := range privateURLs {
		t.Run(u, func(t *testing.T) {
			t.Parallel()
			title, desc, err := f.Fetch(context.Background(), u)
			if err != nil {
				t.Fatalf("want nil error, got %v", err)
			}
			if title != "" || desc != "" {
				t.Errorf("want empty for private host, got title=%q desc=%q", title, desc)
			}
		})
	}
}

func TestHTTPPreviewFetcher_Timeout(t *testing.T) {
	t.Parallel()

	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
		_, _ = w.Write([]byte("<html><head><title>Never</title></head></html>"))
	}))
	defer slowSrv.Close()

	// Use ForTest variant (loopback) but with a very short timeout to trigger timeout path.
	shortClient := &http.Client{Timeout: 50 * time.Millisecond}
	f := shortener.NewHTTPPreviewFetcherForTest(shortClient)

	title, desc, err := f.Fetch(context.Background(), slowSrv.URL)
	if err != nil {
		t.Fatalf("want nil error (timeout swallowed), got %v", err)
	}
	if title != "" || desc != "" {
		t.Errorf("want empty on timeout, got title=%q desc=%q", title, desc)
	}
}
