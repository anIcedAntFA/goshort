package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T, deps testDeps) string
		body       string
		wantStatus int
		check      func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "set expiry from none",
			setup: func(t *testing.T, deps testDeps) string {
				t.Helper()
				rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls",
					mustMarshal(t, map[string]string{"url": "https://example.com"}))
				if rec.Code != http.StatusCreated {
					t.Fatalf("setup: create failed %d", rec.Code)
				}
				r := decodeJSON[createURLResp](t, rec)
				return r.ShortCode
			},
			body:       `{"expires_in":"7d"}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Helper()
				r := decodeJSON[urlResp](t, rec)
				if r.ExpiresAt == nil {
					t.Error("ExpiresAt should be set, got nil")
				}
			},
		},
		{
			name: "remove expiry",
			setup: func(t *testing.T, deps testDeps) string {
				t.Helper()
				rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls",
					mustMarshal(t, map[string]string{"url": "https://example.com", "expires_in": "7d"}))
				if rec.Code != http.StatusCreated {
					t.Fatalf("setup: create failed %d", rec.Code)
				}
				r := decodeJSON[createURLResp](t, rec)
				return r.ShortCode
			},
			body:       `{"expires_in":"0"}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Helper()
				r := decodeJSON[urlResp](t, rec)
				if r.ExpiresAt != nil {
					t.Errorf("ExpiresAt should be nil after removal, got %v", *r.ExpiresAt)
				}
			},
		},
		{
			name:       "not found",
			setup:      func(_ *testing.T, _ testDeps) string { return "no-such-code" },
			body:       `{"expires_in":"7d"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid expires_in",
			setup:      func(_ *testing.T, _ testDeps) string { return "irrelevant" },
			body:       `{"expires_in":"bad"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON body",
			setup:      func(_ *testing.T, _ testDeps) string { return "irrelevant" },
			body:       `not-json`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := setupTestDeps(t)
			code := tc.setup(t, deps)

			rec := serve(t, deps.router, http.MethodPatch,
				"/api/v1/urls/"+code, []byte(tc.body))

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", rec.Code, tc.wantStatus, rec.Body)
			}
			if tc.check != nil && rec.Code == tc.wantStatus {
				tc.check(t, rec)
			}
		})
	}
}
