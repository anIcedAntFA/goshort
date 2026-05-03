package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewServer_ToolsRegistered(t *testing.T) {
	cs := newTestClient(t)

	res, err := cs.ListTools(context.Background(), &sdkmcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	want := map[string]bool{
		"shorten_url":   false,
		"list_urls":     false,
		"get_url_stats": false,
		"delete_url":    false,
		"lookup_url":    false,
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
	if len(res.Tools) != 5 {
		t.Errorf("expected 5 tools, got %d", len(res.Tools))
	}
}

func TestResource_StatsSummary_Empty(t *testing.T) {
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
	cs := newTestClient(t)

	_, err := cs.ReadResource(context.Background(), &sdkmcp.ReadResourceParams{
		URI: "goshort://urls/nonexistent",
	})
	if err == nil {
		t.Error("expected error for nonexistent resource")
	}
}

func TestPrompt_ShortenAndShare(t *testing.T) {
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
