package api_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

// pngMagic is the 8-byte PNG signature.
var pngMagic = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

// jpegMagic is the 3-byte JPEG signature.
var jpegMagic = []byte{0xFF, 0xD8, 0xFF}

func TestGetQRCode(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	body := mustMarshal(t, map[string]string{"url": "https://example.com"})
	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create URL: %d; body: %s", rec.Code, rec.Body.String())
	}
	created := decodeJSON[createURLResp](t, rec)
	code := created.ShortCode

	tests := []struct {
		name       string
		code       string
		query      string
		wantStatus int
		wantCT     string
		checkBody  func(t *testing.T, b []byte)
	}{
		{
			name: "default format is PNG", code: code, query: "", wantStatus: http.StatusOK,
			wantCT: "image/png",
			checkBody: func(t *testing.T, b []byte) {
				t.Helper()
				if len(b) < 8 || !bytes.Equal(b[:8], pngMagic) {
					t.Errorf("missing PNG magic bytes, got %x", b[:min(8, len(b))])
				}
			},
		},
		{
			name: "explicit png format", code: code, query: "?format=png", wantStatus: http.StatusOK,
			wantCT: "image/png",
			checkBody: func(t *testing.T, b []byte) {
				t.Helper()
				if len(b) < 8 || !bytes.Equal(b[:8], pngMagic) {
					t.Errorf("missing PNG magic bytes, got %x", b[:min(8, len(b))])
				}
			},
		},
		{
			name: "jpeg format", code: code, query: "?format=jpeg", wantStatus: http.StatusOK,
			wantCT: "image/jpeg",
			checkBody: func(t *testing.T, b []byte) {
				t.Helper()
				if len(b) < 3 || !bytes.Equal(b[:3], jpegMagic) {
					t.Errorf("missing JPEG magic bytes, got %x", b[:min(3, len(b))])
				}
			},
		},
		{
			name: "svg format", code: code, query: "?format=svg", wantStatus: http.StatusOK,
			wantCT: "image/svg+xml",
			checkBody: func(t *testing.T, b []byte) {
				t.Helper()
				if !strings.HasPrefix(string(b), "<svg") {
					t.Errorf("SVG body does not start with <svg, got: %.40s", b)
				}
			},
		},
		{name: "invalid format", code: code, query: "?format=gif", wantStatus: http.StatusBadRequest},
		{name: "explicit size 128", code: code, query: "?size=128", wantStatus: http.StatusOK, wantCT: "image/png"},
		{name: "size 2000 clamped to 1024", code: code, query: "?size=2000", wantStatus: http.StatusOK, wantCT: "image/png"},
		{name: "size 50 clamped to 128", code: code, query: "?size=50", wantStatus: http.StatusOK, wantCT: "image/png"},
		{name: "nonexistent code", code: "doesnotexist", query: "", wantStatus: http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := serve(t, deps.router, http.MethodGet, "/api/v1/urls/"+tc.code+"/qr"+tc.query, nil)
			if r.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", r.Code, tc.wantStatus, r.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			if tc.wantCT != "" {
				if ct := r.Header().Get("Content-Type"); ct != tc.wantCT {
					t.Errorf("Content-Type = %q, want %q", ct, tc.wantCT)
				}
			}
			if tc.checkBody != nil {
				tc.checkBody(t, r.Body.Bytes())
			}
		})
	}
}
