package mcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP server and wires GoShort tools, resources, and prompts.
type Server struct {
	svc     shortener.Service
	baseURL string
	server  *sdkmcp.Server
}

// NewServer creates a GoShort MCP server and registers all tools, resources, and prompts.
func NewServer(svc shortener.Service, baseURL string) *Server {
	s := &Server{
		svc:     svc,
		baseURL: baseURL,
		server:  sdkmcp.NewServer(&sdkmcp.Implementation{Name: "goshort", Version: "0.4.0"}, nil),
	}
	s.register()
	return s
}

func (s *Server) register() {
	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "shorten_url",
		Description: "Create a short URL from a long URL",
	}, s.handleShortenURL)

	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "list_urls",
		Description: "List shortened URLs with pagination",
	}, s.handleListURLs)

	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "get_url_stats",
		Description: "Get full stats for a short URL (click count, expiry, etc.)",
	}, s.handleGetURLStats)

	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "delete_url",
		Description: "Delete a shortened URL by its short code",
	}, s.handleDeleteURL)

	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "lookup_url",
		Description: "Quickly resolve a short code to its original URL",
	}, s.handleLookupURL)

	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "batch_shorten_urls",
		Description: "Create multiple short URLs in a single call (max 50)",
	}, s.handleBatchShortenURLs)

	sdkmcp.AddTool(s.server, &sdkmcp.Tool{
		Name:        "update_url",
		Description: "Update the expiry of a short URL. Use expires_in='0' to remove expiry.",
	}, s.handleUpdateURL)

	s.server.AddResource(&sdkmcp.Resource{
		URI:         "goshort://stats/summary",
		Name:        "GoShort Stats Summary",
		Description: "Total URL count and top URLs by click count",
		MIMEType:    "application/json",
	}, s.handleStatsSummary)

	s.server.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		URITemplate: "goshort://urls/{code}",
		Name:        "URL Details",
		Description: "Full details for a shortened URL by short code",
	}, s.handleURLByCode)

	s.server.AddResourceTemplate(&sdkmcp.ResourceTemplate{
		URITemplate: "goshort://urls/{code}/qr",
		Name:        "URL QR Code",
		Description: "QR code PNG (256×256) for a shortened URL",
		MIMEType:    "image/png",
	}, s.handleQRCode)

	s.server.AddPrompt(&sdkmcp.Prompt{
		Name:        "shorten_and_share",
		Description: "Shorten a URL and format it for sharing on a platform",
		Arguments: []*sdkmcp.PromptArgument{
			{Name: "url", Description: "URL to shorten", Required: true},
			{Name: "platform", Description: "Target platform: slack, email, twitter", Required: false},
		},
	}, handleShortenAndShare)

	s.server.AddPrompt(&sdkmcp.Prompt{
		Name:        "batch_shorten",
		Description: "Shorten multiple URLs and return a formatted table",
		Arguments: []*sdkmcp.PromptArgument{
			{Name: "urls", Description: "Newline-separated list of URLs to shorten", Required: true},
		},
	}, handleBatchShorten)
}

// RunStdio runs the MCP server over stdin/stdout (for local Claude Code / Cursor use).
func (s *Server) RunStdio(ctx context.Context) error {
	return s.server.Run(ctx, &sdkmcp.StdioTransport{})
}

// RunHTTP starts the MCP server as a Streamable HTTP handler on addr.
// When apiKey is non-empty, requests must include X-API-Key.
func (s *Server) RunHTTP(ctx context.Context, addr, apiKey string) error {
	handler := sdkmcp.NewStreamableHTTPHandler(func(_ *http.Request) *sdkmcp.Server {
		return s.server
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/mcp", APIKeyMiddleware(apiKey, handler))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() { //nolint:gosec // shutdown uses context.Background intentionally — parent ctx is already canceled
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) //nolint:contextcheck
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("mcp http server: %w", err)
	}
	return nil
}

// HTTPHandler returns an http.Handler serving the MCP Streamable HTTP protocol.
// Mount it on any router (e.g. r.Handle("/mcp", srv.HTTPHandler(apiKey))).
// When apiKey is non-empty, requests must include X-API-Key.
func (s *Server) HTTPHandler(apiKey string) http.Handler {
	handler := sdkmcp.NewStreamableHTTPHandler(func(_ *http.Request) *sdkmcp.Server {
		return s.server
	}, nil)
	return APIKeyMiddleware(apiKey, handler)
}

// MCPServer returns the underlying sdkmcp.Server (for in-process testing or router mounting).
func (s *Server) MCPServer() *sdkmcp.Server {
	return s.server
}
