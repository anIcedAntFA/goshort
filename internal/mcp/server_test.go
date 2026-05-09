package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/encoder"
	mcpserver "github.com/anIcedAntFA/goshort/internal/mcp"
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTestMCPServer creates a GoShort MCP server for transport-level tests.
func newTestMCPServer(t *testing.T) *mcpserver.Server {
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

	return mcpserver.NewServer(shortener.NewService(store, enc), testBaseURL)
}

func TestNewServer_ToolsRegistered(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res, err := cs.ListTools(context.Background(), &sdkmcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	want := map[string]bool{
		"shorten_url":        false,
		"list_urls":          false,
		"get_url_stats":      false,
		"delete_url":         false,
		"lookup_url":         false,
		"batch_shorten_urls": false,
	}
	for _, tool := range res.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("tool %q not registered", name)
		}
	}
	if len(res.Tools) != 6 {
		t.Errorf("expected 6 tools, got %d", len(res.Tools))
	}
}

func TestResource_StatsSummary_Empty(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res, err := cs.ReadResource(context.Background(), &sdkmcp.ReadResourceParams{
		URI: "goshort://stats/summary",
	})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(res.Contents) == 0 {
		t.Fatal("expected non-empty contents")
	}

	var summary struct {
		TotalURLs int   `json:"total_urls"`
		TopURLs   []any `json:"top_urls"`
	}
	if err := json.Unmarshal([]byte(res.Contents[0].Text), &summary); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if summary.TotalURLs != 0 {
		t.Errorf("total_urls = %d, want 0", summary.TotalURLs)
	}
	if summary.TopURLs == nil {
		t.Error("top_urls should be empty slice, not nil")
	}
}

func TestResource_StatsSummary_WithData(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	for i := range 3 {
		callTool(t, cs, "shorten_url", map[string]any{
			"url": fmt.Sprintf("https://example.com/%d", i),
		})
	}

	res, err := cs.ReadResource(context.Background(), &sdkmcp.ReadResourceParams{
		URI: "goshort://stats/summary",
	})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}

	var summary struct {
		TotalURLs int `json:"total_urls"`
	}
	if err := json.Unmarshal([]byte(res.Contents[0].Text), &summary); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if summary.TotalURLs != 3 {
		t.Errorf("total_urls = %d, want 3", summary.TotalURLs)
	}
}

func TestResource_URLByCode(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	r := callTool(t, cs, "shorten_url", map[string]any{
		"url":   "https://example.com",
		"alias": "res-test",
	})
	if r.IsError {
		t.Fatalf("create failed: %s", textOf(t, r))
	}

	res, err := cs.ReadResource(context.Background(), &sdkmcp.ReadResourceParams{
		URI: "goshort://urls/res-test",
	})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(res.Contents) == 0 {
		t.Fatal("expected contents")
	}

	var detail struct {
		ShortCode   string `json:"short_code"`
		OriginalURL string `json:"original_url"`
	}
	if err := json.Unmarshal([]byte(res.Contents[0].Text), &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if detail.ShortCode != "res-test" {
		t.Errorf("short_code = %q, want res-test", detail.ShortCode)
	}
	if detail.OriginalURL != "https://example.com" {
		t.Errorf("original_url = %q, want https://example.com", detail.OriginalURL)
	}
}

func TestResource_URLByCode_NotFound(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	_, err := cs.ReadResource(context.Background(), &sdkmcp.ReadResourceParams{
		URI: "goshort://urls/nonexistent",
	})
	if err == nil {
		t.Error("expected error for nonexistent resource")
	}
}

func TestPrompt_ShortenAndShare(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res, err := cs.GetPrompt(context.Background(), &sdkmcp.GetPromptParams{
		Name: "shorten_and_share",
		Arguments: map[string]string{
			"url":      "https://example.com",
			"platform": "slack",
		},
	})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(res.Messages) == 0 {
		t.Fatal("expected messages")
	}

	tc, ok := res.Messages[0].Content.(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Messages[0].Content)
	}
	if !strings.Contains(tc.Text, "slack") {
		t.Errorf("prompt text missing platform, got: %s", tc.Text)
	}
	if !strings.Contains(tc.Text, "https://example.com") {
		t.Errorf("prompt text missing URL, got: %s", tc.Text)
	}
}

func TestPrompt_ShortenAndShare_DefaultPlatform(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res, err := cs.GetPrompt(context.Background(), &sdkmcp.GetPromptParams{
		Name:      "shorten_and_share",
		Arguments: map[string]string{"url": "https://example.com"},
	})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}

	tc, ok := res.Messages[0].Content.(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Messages[0].Content)
	}
	if !strings.Contains(tc.Text, "general") {
		t.Errorf("expected default platform 'general', got: %s", tc.Text)
	}
}

func TestPrompt_BatchShorten(t *testing.T) {
	t.Parallel()
	cs := newTestClient(t)

	res, err := cs.GetPrompt(context.Background(), &sdkmcp.GetPromptParams{
		Name: "batch_shorten",
		Arguments: map[string]string{
			"urls": "https://a.com\nhttps://b.com",
		},
	})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	if len(res.Messages) == 0 {
		t.Fatal("expected messages")
	}

	tc, ok := res.Messages[0].Content.(*sdkmcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Messages[0].Content)
	}
	if !strings.Contains(tc.Text, "https://a.com") {
		t.Errorf("prompt text missing first URL, got: %s", tc.Text)
	}
}

// TestServer_HTTPHandler verifies that HTTPHandler returns a working http.Handler
// and that the internal `return s.server` closure is invoked on a request.
func TestServer_HTTPHandler(t *testing.T) {
	t.Parallel()

	srv := newTestMCPServer(t)
	h := srv.HTTPHandler("")
	if h == nil {
		t.Fatal("HTTPHandler returned nil")
	}

	// POST with the MCP-required Accept header so the request passes the
	// streamable handler's validation and reaches the `return s.server` closure.
	// A short context timeout ensures ServeHTTP returns even if the session blocks.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	h.ServeHTTP(httptest.NewRecorder(), req)
}

// TestServer_RunHTTP_GracefulShutdown starts the MCP HTTP server, sends one
// request to invoke the internal `return s.server` closure, then cancels the
// context and waits for a clean nil return.
func TestServer_RunHTTP_GracefulShutdown(t *testing.T) {
	t.Parallel()
	// Reserve a random free port, release it, then hand the address to RunHTTP.
	// Port reuse within tests on loopback is safe enough; the window is tiny.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pre-listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	srv := newTestMCPServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.RunHTTP(ctx, addr, "")
	}()

	// Retry until the server accepts a connection — this also invokes the closure.
	// Both application/json and text/event-stream are required by the MCP
	// streamable HTTP handler to pass Accept validation and reach getServer.
	mcpURL := "http://" + addr + "/mcp"
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		httpReq, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, mcpURL, strings.NewReader("{}"))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json, text/event-stream")
		resp, reqErr := http.DefaultClient.Do(httpReq)
		if reqErr == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("RunHTTP after cancel: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunHTTP did not stop after context cancellation")
	}
}

// TestServer_RunHTTP_ListenError verifies the error return when the port is occupied.
func TestServer_RunHTTP_ListenError(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pre-listen: %v", err)
	}
	addr := ln.Addr().String()
	defer ln.Close()

	srv := newTestMCPServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = srv.RunHTTP(ctx, addr, "")
	if err == nil {
		t.Fatal("expected error for occupied port")
	}
	if !strings.Contains(err.Error(), "mcp http server:") {
		t.Errorf("error = %q, want prefix 'mcp http server:'", err.Error())
	}
}
