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

	svc := shortener.NewService(store, enc)
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
	cs := newTestClient(t)

	res := callTool(t, cs, "shorten_url", map[string]any{"url": "not-a-url"})
	if !res.IsError {
		t.Error("expected error for invalid URL, got success")
	}
}

func TestShortenURL_InvalidExpiry(t *testing.T) {
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
	cs := newTestClient(t)

	res := callTool(t, cs, "get_url_stats", map[string]any{"code": "nope"})
	if !res.IsError {
		t.Error("expected error for missing code, got success")
	}
}

func TestDeleteURL_Existing(t *testing.T) {
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
	cs := newTestClient(t)

	res := callTool(t, cs, "delete_url", map[string]any{"code": "ghost"})
	if !res.IsError {
		t.Error("expected error for missing code, got success")
	}
}

func TestLookupURL_Existing(t *testing.T) {
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
	cs := newTestClient(t)

	res := callTool(t, cs, "lookup_url", map[string]any{"code": "missing"})
	if !res.IsError {
		t.Error("expected error for missing code, got success")
	}
}
