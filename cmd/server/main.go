// Package main is the entry point of the GoShort HTTP server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/anIcedAntFA/goshort/internal/api"
	"github.com/anIcedAntFA/goshort/internal/cache"
	"github.com/anIcedAntFA/goshort/internal/config"
	"github.com/anIcedAntFA/goshort/internal/encoder"
	mcpserver "github.com/anIcedAntFA/goshort/internal/mcp"
	"github.com/anIcedAntFA/goshort/internal/preview"
	"github.com/anIcedAntFA/goshort/internal/safebrowsing"
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "", "path to TOML config file (default: env vars + built-in defaults)")
	mcpMode := flag.Bool("mcp", false, "run as MCP server over stdio")
	mcpHTTP := flag.String("mcp-http", "", "run MCP server over Streamable HTTP on this address (e.g. :9090)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	config.SetupLogger(cfg.Logging)

	if err := os.MkdirAll(filepath.Dir(cfg.Storage.SQLitePath), 0o750); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	store, err := storage.NewSQLiteStorage(context.Background(), cfg.Storage.SQLitePath)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer func() { _ = store.Close() }()

	enc, err := encoder.NewSqidsEncoder(uint8(cfg.Shortener.CodeLength)) //nolint:gosec // Validate() ensures [1,255]
	if err != nil {
		return fmt.Errorf("create encoder: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.Auth.APIKey != "" {
		slog.Info("API key auth enabled")
	} else {
		slog.Warn("API key auth disabled — all endpoints are public")
	}

	var checker shortener.URLChecker
	if cfg.Security.SafeBrowsingAPIKey != "" {
		checker = safebrowsing.NewChecker(cfg.Security.SafeBrowsingAPIKey)
		slog.Info("safe browsing enabled")
	} else {
		checker = shortener.NoopChecker{}
	}

	svc := shortener.NewService(store, enc, preview.NewHTTPFetcher(), checker)

	if *mcpMode || *mcpHTTP != "" {
		if err := runMCPMode(ctx, cfg, svc, *mcpHTTP); err != nil {
			slog.Error("mcp server error", "error", err)
		}
		return nil
	}

	go storage.StartCleanupJob(ctx, store)
	runHTTPServer(ctx, cfg, svc)

	slog.Info("server stopped")
	return nil
}

func runMCPMode(ctx context.Context, cfg *config.Config, svc shortener.Service, httpAddr string) error {
	baseURL := cfg.MCP.BaseURL
	if baseURL == "" {
		baseURL = cfg.Server.BaseURL
	}
	srv := mcpserver.NewServer(svc, baseURL)
	if httpAddr != "" {
		slog.Info("mcp server starting (http)", "addr", httpAddr)
		return srv.RunHTTP(ctx, httpAddr, cfg.Auth.APIKey)
	}
	return srv.RunStdio(ctx)
}

func runHTTPServer(ctx context.Context, cfg *config.Config, svc shortener.Service) {
	c := cache.Build(cfg.Cache.Driver, cfg.Cache.RedisURL)
	h := api.NewHandler(svc, c, slog.Default(), cfg.Server.BaseURL)
	router := api.NewRouter(h, api.RouterConfig{
		APIKey:           cfg.Auth.APIKey,
		RateLimitEnabled: cfg.RateLimit.Enabled,
		RateLimitRPM:     cfg.RateLimit.RequestsPerMinute,
	})

	mcpBaseURL := cfg.MCP.BaseURL
	if mcpBaseURL == "" {
		mcpBaseURL = cfg.Server.BaseURL
	}
	mcpSrv := mcpserver.NewServer(svc, mcpBaseURL)
	router.Handle("/mcp", mcpSrv.HTTPHandler(cfg.Auth.APIKey))

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", srv.Addr, "base_url", cfg.Server.BaseURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
