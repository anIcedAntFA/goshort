package preview_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/preview"
)

func TestHTTPFetcher(t *testing.T) {
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
		{
			name: "og tags take priority",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`<html><head>
					<title>HTML Title</title>
					<meta name="description" content="HTML desc">
					<meta property="og:title" content="Clean OG Title">
					<meta property="og:description" content="OG description">
				</head><body></body></html>`))
			},
			wantTitle: "Clean OG Title",
			wantDesc:  "OG description",
		},
		{
			name: "fallback to html when og missing",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`<html><head>
					<title>HTML Title</title>
					<meta name="description" content="HTML desc">
				</head><body></body></html>`))
			},
			wantTitle: "HTML Title",
			wantDesc:  "HTML desc",
		},
		{
			name: "og title only with html description fallback",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`<html><head>
					<title>HTML Title</title>
					<meta name="description" content="HTML Desc">
					<meta property="og:title" content="OG Only">
				</head><body></body></html>`))
			},
			wantTitle: "OG Only",
			wantDesc:  "HTML Desc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(tc.handler)
			defer srv.Close()

			f := preview.NewHTTPFetcherForTest(http.DefaultClient)
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

func TestHTTPFetcher_Timeout(t *testing.T) {
	t.Parallel()

	slowSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
		_, _ = w.Write([]byte("<html><head><title>Never</title></head></html>"))
	}))
	defer slowSrv.Close()

	shortClient := &http.Client{Timeout: 50 * time.Millisecond}
	f := preview.NewHTTPFetcherForTest(shortClient)

	title, desc, err := f.Fetch(context.Background(), slowSrv.URL)
	if err != nil {
		t.Fatalf("want nil error (timeout swallowed), got %v", err)
	}
	if title != "" || desc != "" {
		t.Errorf("want empty on timeout, got title=%q desc=%q", title, desc)
	}
}

func TestSafeDialer_RejectsPrivateIPs(t *testing.T) {
	t.Parallel()

	f := preview.NewHTTPFetcher()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Attempt to fetch a URL resolving to localhost — should be blocked by safe dialer.
	title, desc, err := f.Fetch(ctx, "http://127.0.0.1:9/")
	if err != nil {
		t.Fatalf("Fetch must swallow errors, got %v", err)
	}
	if title != "" || desc != "" {
		t.Errorf("want empty for private IP, got title=%q desc=%q", title, desc)
	}
}
