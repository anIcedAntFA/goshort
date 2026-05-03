package mcp

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func handleShortenAndShare(_ context.Context, req *sdkmcp.GetPromptRequest) (*sdkmcp.GetPromptResult, error) {
	url := req.Params.Arguments["url"]
	platform := req.Params.Arguments["platform"]
	if platform == "" {
		platform = "general"
	}
	text := fmt.Sprintf(
		"Shorten the following URL and format the result for sharing on %s.\nURL: %s",
		platform, url,
	)
	return &sdkmcp.GetPromptResult{
		Description: "Shorten a URL and format it for sharing on a platform",
		Messages: []*sdkmcp.PromptMessage{
			{Role: "user", Content: &sdkmcp.TextContent{Text: text}},
		},
	}, nil
}

func handleBatchShorten(_ context.Context, req *sdkmcp.GetPromptRequest) (*sdkmcp.GetPromptResult, error) {
	urls := req.Params.Arguments["urls"]
	text := fmt.Sprintf(
		"Shorten each of these URLs and return a formatted table with the original and shortened versions:\n%s",
		urls,
	)
	return &sdkmcp.GetPromptResult{
		Description: "Batch shorten multiple URLs and return a formatted table",
		Messages: []*sdkmcp.PromptMessage{
			{Role: "user", Content: &sdkmcp.TextContent{Text: text}},
		},
	}, nil
}
