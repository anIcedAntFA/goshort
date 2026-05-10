package api_test

import (
	"bytes"
	"net/http"
	"testing"
)

// pngMagic is the 8-byte PNG signature.
var pngMagic = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}

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
	}{
		{"valid code default size", code, "", http.StatusOK},
		{"valid code explicit size 128", code, "?size=128", http.StatusOK},
		{"size 2000 clamped to 1024", code, "?size=2000", http.StatusOK},
		{"size 50 clamped to 128", code, "?size=50", http.StatusOK},
		{"nonexistent code", "doesnotexist", "", http.StatusNotFound},
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
			if ct := r.Header().Get("Content-Type"); ct != "image/png" {
				t.Errorf("Content-Type = %q, want %q", ct, "image/png")
			}
			b := r.Body.Bytes()
			if len(b) < 8 {
				t.Fatalf("body too short (%d bytes) to contain PNG magic", len(b))
			}
			if !bytes.Equal(b[:8], pngMagic) {
				t.Errorf("body does not start with PNG magic bytes, got %x", b[:8])
			}
		})
	}
}
