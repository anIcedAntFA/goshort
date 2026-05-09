package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/encoder"
	mcpserver "github.com/anIcedAntFA/goshort/internal/mcp"
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

const testBaseURL = "http://localhost:8080"

// newTestClient creates an MCP server backed by a real in-memory SQLite DB and
// connects an in-process client to it. Both sessions are cleaned up via t.Cleanup.
func newTestClient(t *testing.T) *sdkmcp.ClientSession {
	t.Helper()
	ctx := context.Background()

	store, err := storage.NewSQLiteStorage(ctx, ":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		t.Fatalf("NewSqidsEncoder: %v", err)
	}

	svc := shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, shortener.NoopChecker{})
	srv := mcpserver.NewServer(svc, testBaseURL)

	ct, st := sdkmcp.NewInMemoryTransports()

	ss, err := srv.MCPServer().Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server Connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })

	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "v1"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })

	return cs
}

// textOf extracts the text from the first TextContent in a tool result.
func textOf(t *testing.T, res *sdkmcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatal("tool result has no content")
	}
	tc, ok := res.Content[0].(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

// callTool is a helper that calls a named tool and fails the test on RPC error.
func callTool(t *testing.T, cs *sdkmcp.ClientSession, name string, args map[string]any) *sdkmcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool %q: %v", name, err)
	}
	return res
}

func TestShortenURL_Basic(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "shorten_url", map[string]any{"url": "https://example.com"})

	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}

	var out struct {
		ShortCode   string `json:"short_code"`
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if out.ShortCode == "" {
		t.Error("short_code is empty")
	}
	if out.OriginalURL != "https://example.com" {
		t.Errorf("original_url = %q, want %q", out.OriginalURL, "https://example.com")
	}
	if out.ShortURL == "" {
		t.Error("short_url is empty")
	}
}

func TestShortenURL_WithAlias(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "shorten_url", map[string]any{
		"url":   "https://example.com/long",
		"alias": "my-alias",
	})

	if res.IsError {
		t.Fatalf("expected success: %s", textOf(t, res))
	}

	var out struct {
		ShortCode string `json:"short_code"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ShortCode != "my-alias" {
		t.Errorf("short_code = %q, want %q", out.ShortCode, "my-alias")
	}
}

func TestShortenURL_AliasTaken(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	callTool(t, cs, "shorten_url", map[string]any{
		"url":   "https://a.com",
		"alias": "taken",
	})

	res := callTool(t, cs, "shorten_url", map[string]any{
		"url":   "https://b.com",
		"alias": "taken",
	})
	if !res.IsError {
		t.Error("expected error for duplicate alias, got success")
	}
}

func TestShortenURL_InvalidURL(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "shorten_url", map[string]any{"url": "not-a-url"})
	if !res.IsError {
		t.Error("expected error for invalid URL, got success")
	}
}

func TestShortenURL_InvalidExpiry(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "shorten_url", map[string]any{
		"url":        "https://example.com",
		"expires_in": "bad-expiry",
	})
	if !res.IsError {
		t.Error("expected error for invalid expiry, got success")
	}
}

func TestListURLs_Empty(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "list_urls", map[string]any{})
	if res.IsError {
		t.Fatalf("expected success: %s", textOf(t, res))
	}

	var out struct {
		URLs       []any `json:"urls"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.URLs) != 0 {
		t.Errorf("want 0 urls, got %d", len(out.URLs))
	}
	if out.Pagination.Total != 0 {
		t.Errorf("want total=0, got %d", out.Pagination.Total)
	}
}

func TestListURLs_WithData(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	for i := range 3 {
		callTool(t, cs, "shorten_url", map[string]any{
			"url": "https://example.com/" + string(rune('a'+i)),
		})
	}

	res := callTool(t, cs, "list_urls", map[string]any{})
	if res.IsError {
		t.Fatalf("expected success: %s", textOf(t, res))
	}

	var out struct {
		URLs       []any `json:"urls"`
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.URLs) != 3 {
		t.Errorf("want 3 urls, got %d", len(out.URLs))
	}
	if out.Pagination.Total != 3 {
		t.Errorf("want total=3, got %d", out.Pagination.Total)
	}
}

func TestListURLs_Pagination(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	for i := range 5 {
		callTool(t, cs, "shorten_url", map[string]any{
			"url": "https://example.com/" + string(rune('a'+i)),
		})
	}

	res := callTool(t, cs, "list_urls", map[string]any{"page": 1, "per_page": 2})
	if res.IsError {
		t.Fatalf("expected success: %s", textOf(t, res))
	}

	var out struct {
		URLs       []any `json:"urls"`
		Pagination struct {
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.URLs) != 2 {
		t.Errorf("want 2 urls per page, got %d", len(out.URLs))
	}
	if out.Pagination.TotalPages != 3 {
		t.Errorf("want 3 total pages, got %d", out.Pagination.TotalPages)
	}
}

func TestGetURLStats_Existing(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	r := callTool(t, cs, "shorten_url", map[string]any{"url": "https://example.com"})
	var created struct {
		ShortCode string `json:"short_code"`
	}
	if err := json.Unmarshal([]byte(textOf(t, r)), &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}

	res := callTool(t, cs, "get_url_stats", map[string]any{"code": created.ShortCode})
	if res.IsError {
		t.Fatalf("expected success: %s", textOf(t, res))
	}

	var out struct {
		ShortCode   string `json:"short_code"`
		ClickCount  int64  `json:"click_count"`
		OriginalURL string `json:"original_url"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal stats: %v", err)
	}
	if out.ShortCode != created.ShortCode {
		t.Errorf("short_code = %q, want %q", out.ShortCode, created.ShortCode)
	}
}

func TestGetURLStats_NotFound(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "get_url_stats", map[string]any{"code": "nope"})
	if !res.IsError {
		t.Error("expected error for missing code, got success")
	}
}

func TestDeleteURL_Existing(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	r := callTool(t, cs, "shorten_url", map[string]any{"url": "https://example.com"})
	var created struct {
		ShortCode string `json:"short_code"`
	}
	if err := json.Unmarshal([]byte(textOf(t, r)), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	res := callTool(t, cs, "delete_url", map[string]any{"code": created.ShortCode})
	if res.IsError {
		t.Fatalf("expected success: %s", textOf(t, res))
	}

	stats := callTool(t, cs, "get_url_stats", map[string]any{"code": created.ShortCode})
	if !stats.IsError {
		t.Error("expected not-found after delete")
	}
}

func TestDeleteURL_NotFound(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "delete_url", map[string]any{"code": "ghost"})
	if !res.IsError {
		t.Error("expected error for missing code, got success")
	}
}

func TestLookupURL_Existing(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	r := callTool(t, cs, "shorten_url", map[string]any{"url": "https://example.com"})
	var created struct {
		ShortCode string `json:"short_code"`
	}
	if err := json.Unmarshal([]byte(textOf(t, r)), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	res := callTool(t, cs, "lookup_url", map[string]any{"code": created.ShortCode})
	if res.IsError {
		t.Fatalf("expected success: %s", textOf(t, res))
	}

	var out struct {
		OriginalURL string `json:"original_url"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.OriginalURL != "https://example.com" {
		t.Errorf("original_url = %q, want %q", out.OriginalURL, "https://example.com")
	}
}

func TestLookupURL_NotFound(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "lookup_url", map[string]any{"code": "missing"})
	if !res.IsError {
		t.Error("expected error for missing code, got success")
	}
}

func TestBatchShortenURLs_AllValid(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "batch_shorten_urls", map[string]any{
		"urls": []map[string]any{
			{"url": "https://a.com"},
			{"url": "https://b.com"},
			{"url": "https://c.com"},
		},
	})
	if res.IsError {
		t.Fatalf("expected success, got error: %s", textOf(t, res))
	}

	var out struct {
		Results []struct {
			ShortCode   string `json:"short_code"`
			OriginalURL string `json:"original_url"`
			Error       any    `json:"error"`
		} `json:"results"`
		Summary struct {
			Total   int `json:"total"`
			Success int `json:"success"`
			Failed  int `json:"failed"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Results) != 3 {
		t.Errorf("want 3 results, got %d", len(out.Results))
	}
	if out.Summary.Total != 3 {
		t.Errorf("summary.total = %d, want 3", out.Summary.Total)
	}
	if out.Summary.Success != 3 {
		t.Errorf("summary.success = %d, want 3", out.Summary.Success)
	}
	if out.Summary.Failed != 0 {
		t.Errorf("summary.failed = %d, want 0", out.Summary.Failed)
	}
	for i, r := range out.Results {
		if r.ShortCode == "" {
			t.Errorf("result[%d].short_code is empty", i)
		}
		if r.Error != nil {
			t.Errorf("result[%d].error is non-nil: %v", i, r.Error)
		}
	}
}

func TestBatchShortenURLs_PartialSuccess(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "batch_shorten_urls", map[string]any{
		"urls": []map[string]any{
			{"url": "https://valid.com"},
			{"url": "not-a-url"},
			{"url": "https://also-valid.com"},
		},
	})
	if res.IsError {
		t.Fatalf("expected success for partial batch, got error: %s", textOf(t, res))
	}

	var out struct {
		Results []struct {
			ShortCode string `json:"short_code"`
			Error     *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		} `json:"results"`
		Summary struct {
			Total   int `json:"total"`
			Success int `json:"success"`
			Failed  int `json:"failed"`
		} `json:"summary"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Results) != 3 {
		t.Errorf("want 3 results, got %d", len(out.Results))
	}
	if out.Summary.Success != 2 {
		t.Errorf("summary.success = %d, want 2", out.Summary.Success)
	}
	if out.Summary.Failed != 1 {
		t.Errorf("summary.failed = %d, want 1", out.Summary.Failed)
	}
	// result[0]: success
	if out.Results[0].ShortCode == "" {
		t.Error("result[0] should have short_code")
	}
	if out.Results[0].Error != nil {
		t.Errorf("result[0] should not have error, got %+v", out.Results[0].Error)
	}
	// result[1]: error
	if out.Results[1].Error == nil {
		t.Error("result[1] should have error for invalid URL")
	} else if out.Results[1].Error.Code != "invalid_url" {
		t.Errorf("result[1].error.code = %q, want %q", out.Results[1].Error.Code, "invalid_url")
	}
	// result[2]: success
	if out.Results[2].ShortCode == "" {
		t.Error("result[2] should have short_code")
	}
}

func TestBatchShortenURLs_Empty(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "batch_shorten_urls", map[string]any{
		"urls": []map[string]any{},
	})
	if !res.IsError {
		t.Error("expected error for empty batch, got success")
	}
}

func TestBatchShortenURLs_ExceedsCap(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	urls := make([]map[string]any, 51)
	for i := range urls {
		urls[i] = map[string]any{"url": "https://example.com"}
	}

	res := callTool(t, cs, "batch_shorten_urls", map[string]any{"urls": urls})
	if !res.IsError {
		t.Error("expected error for batch exceeding 50, got success")
	}
}

func TestUpdateURL_SetExpiry(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	// Create a URL with no expiry.
	createRes := callTool(t, cs, "shorten_url", map[string]any{"url": "https://example.com"})
	if createRes.IsError {
		t.Fatalf("create: %s", textOf(t, createRes))
	}
	var created struct {
		ShortCode string `json:"short_code"`
	}
	if err := json.Unmarshal([]byte(textOf(t, createRes)), &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}

	res := callTool(t, cs, "update_url", map[string]any{
		"code":       created.ShortCode,
		"expires_in": "7d",
	})
	if res.IsError {
		t.Fatalf("update_url: %s", textOf(t, res))
	}

	var out struct {
		ShortCode string  `json:"short_code"`
		ExpiresAt *string `json:"expires_at"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ExpiresAt == nil {
		t.Error("ExpiresAt should be set, got nil")
	}
	if out.ShortCode != created.ShortCode {
		t.Errorf("ShortCode = %q, want %q", out.ShortCode, created.ShortCode)
	}
}

func TestUpdateURL_RemoveExpiry(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	createRes := callTool(t, cs, "shorten_url", map[string]any{
		"url":        "https://example.com",
		"expires_in": "7d",
	})
	if createRes.IsError {
		t.Fatalf("create: %s", textOf(t, createRes))
	}
	var created struct {
		ShortCode string `json:"short_code"`
	}
	if err := json.Unmarshal([]byte(textOf(t, createRes)), &created); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}

	res := callTool(t, cs, "update_url", map[string]any{
		"code":       created.ShortCode,
		"expires_in": "0",
	})
	if res.IsError {
		t.Fatalf("update_url: %s", textOf(t, res))
	}

	var out struct {
		ExpiresAt *string `json:"expires_at"`
	}
	if err := json.Unmarshal([]byte(textOf(t, res)), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ExpiresAt != nil {
		t.Errorf("ExpiresAt should be nil after removal, got %v", *out.ExpiresAt)
	}
}

func TestUpdateURL_NotFound(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res := callTool(t, cs, "update_url", map[string]any{
		"code":       "no-such-code",
		"expires_in": "7d",
	})
	if !res.IsError {
		t.Error("expected error for non-existent code, got success")
	}
}
