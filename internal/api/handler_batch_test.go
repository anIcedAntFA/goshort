package api_test

import (
	"net/http"
	"testing"
)

type batchCreateReq struct {
	URLs []batchItemReq `json:"urls"`
}

type batchItemReq struct {
	URL         string `json:"url"`
	CustomAlias string `json:"custom_alias,omitempty"`
	ExpiresIn   string `json:"expires_in,omitempty"`
}

type batchCreateResp struct {
	Results []batchItemResp `json:"results"`
	Summary batchSummary    `json:"summary"`
}

type batchItemResp struct {
	ShortCode   string      `json:"short_code,omitempty"`
	ShortURL    string      `json:"short_url,omitempty"`
	OriginalURL string      `json:"original_url,omitempty"`
	ExpiresAt   *string     `json:"expires_at,omitempty"`
	CreatedAt   string      `json:"created_at,omitempty"`
	Error       *batchError `json:"error,omitempty"`
}

type batchError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type batchSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

// TestBatchCreateURL_AllValid verifies a batch where all URLs succeed.
func TestBatchCreateURL_AllValid(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	body := mustMarshal(t, batchCreateReq{
		URLs: []batchItemReq{
			{URL: "https://a.com"},
			{URL: "https://b.com"},
			{URL: "https://c.com"},
		},
	})
	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/batch", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	resp := decodeJSON[batchCreateResp](t, rec)
	if resp.Summary.Total != 3 {
		t.Errorf("summary.total = %d, want 3", resp.Summary.Total)
	}
	if resp.Summary.Success != 3 {
		t.Errorf("summary.success = %d, want 3", resp.Summary.Success)
	}
	if resp.Summary.Failed != 0 {
		t.Errorf("summary.failed = %d, want 0", resp.Summary.Failed)
	}
	for i, r := range resp.Results {
		if r.Error != nil {
			t.Errorf("results[%d].error = %v, want nil", i, r.Error)
		}
		if r.ShortCode == "" {
			t.Errorf("results[%d].short_code is empty", i)
		}
	}
}

// TestBatchCreateURL_PartialFailure verifies mixed results include both successes and per-item errors.
func TestBatchCreateURL_PartialFailure(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	body := mustMarshal(t, batchCreateReq{
		URLs: []batchItemReq{
			{URL: "https://good.com"},
			{URL: "not-a-url"},
			{URL: "https://also-good.com"},
		},
	})
	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/batch", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	resp := decodeJSON[batchCreateResp](t, rec)
	if resp.Summary.Total != 3 {
		t.Errorf("summary.total = %d, want 3", resp.Summary.Total)
	}
	if resp.Summary.Success != 2 {
		t.Errorf("summary.success = %d, want 2", resp.Summary.Success)
	}
	if resp.Summary.Failed != 1 {
		t.Errorf("summary.failed = %d, want 1", resp.Summary.Failed)
	}
	if resp.Results[0].Error != nil {
		t.Errorf("results[0] expected success, got error %v", resp.Results[0].Error)
	}
	if resp.Results[1].Error == nil {
		t.Error("results[1] expected error, got nil")
	}
	if resp.Results[2].Error != nil {
		t.Errorf("results[2] expected success, got error %v", resp.Results[2].Error)
	}
}

// TestBatchCreateURL_ExceedsCap verifies batches over 50 return 400.
func TestBatchCreateURL_ExceedsCap(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	items := make([]batchItemReq, 51)
	for i := range items {
		items[i] = batchItemReq{URL: "https://example.com"}
	}
	body := mustMarshal(t, batchCreateReq{URLs: items})
	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/batch", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestBatchCreateURL_EmptyArray verifies an empty urls array returns 400.
func TestBatchCreateURL_EmptyArray(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	body := mustMarshal(t, batchCreateReq{URLs: []batchItemReq{}})
	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/batch", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestBatchCreateURL_EmptyBody verifies a missing request body returns 400.
func TestBatchCreateURL_EmptyBody(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/batch", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestBatchCreateURL_WithAlias verifies custom aliases work in batch requests.
func TestBatchCreateURL_WithAlias(t *testing.T) {
	t.Parallel()
	deps := setupTestDeps(t)

	body := mustMarshal(t, batchCreateReq{
		URLs: []batchItemReq{
			{URL: "https://example.com", CustomAlias: "my-alias"},
		},
	})
	rec := serve(t, deps.router, http.MethodPost, "/api/v1/urls/batch", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	resp := decodeJSON[batchCreateResp](t, rec)
	if resp.Results[0].ShortCode != "my-alias" {
		t.Errorf("short_code = %q, want my-alias", resp.Results[0].ShortCode)
	}
}
