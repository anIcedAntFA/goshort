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
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
)

func main() {
	configPath := flag.String("config", "", "path to TOML config file (default: env vars + built-in defaults)")
	mcpMode := flag.Bool("mcp", false, "run as MCP server over stdio")
	mcpHTTP := flag.String("mcp-http", "", "run MCP server over Streamable HTTP on this address (e.g. :9090)")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}

	setupLogger(cfg.Logging)

	if err := os.MkdirAll(filepath.Dir(cfg.Storage.SQLitePath), 0o750); err != nil {
		slog.Error("create data dir", "error", err)
		os.Exit(1)
	}

	store, err := storage.NewSQLiteStorage(context.Background(), cfg.Storage.SQLitePath)
	if err != nil {
		slog.Error("open storage", "error", err)
		os.Exit(1)
	}

	enc, err := encoder.NewSqidsEncoder(uint8(cfg.Shortener.CodeLength)) //nolint:gosec // Validate() ensures [1,255]
	if err != nil {
		slog.Error("create encoder", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.Auth.APIKey != "" {
		slog.Info("API key auth enabled")
	} else {
		slog.Warn("API key auth disabled — all endpoints are public")
	}

	svc := shortener.NewService(store, enc, shortener.NewHTTPPreviewFetcher())

	if *mcpMode || *mcpHTTP != "" {
		if err := runMCPMode(ctx, cfg, svc, *mcpHTTP); err != nil {
			slog.Error("mcp server error", "error", err)
		}
		return
	}

	go startCleanupJob(ctx, store)
	runHTTPServer(ctx, cfg, svc)

	if err := store.Close(); err != nil {
		slog.Error("close storage", "error", err)
	}
	slog.Info("server stopped")
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
	c := buildCache(cfg.Cache)
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

func setupLogger(cfg config.LoggingConfig) {
	logLevel := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: logLevel}
	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

func buildCache(cfg config.CacheConfig) shortener.Cache {
	switch cfg.Driver {
	case "memory":
		return cache.NewMemoryCache()
	case "redis":
		rc, err := cache.NewRedisCache(cfg.RedisURL)
		if err != nil {
			slog.Warn("redis unavailable, falling back to noop cache", "error", err)
			return cache.NewNoopCache()
		}
		return rc
	default:
		return cache.NewNoopCache()
	}
}

func startCleanupJob(ctx context.Context, store shortener.Storage) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("cleanup job stopped")
			return
		case <-ticker.C:
			deleted, err := store.DeleteExpired(ctx, 1000)
			if err != nil {
				slog.Error("cleanup failed", "error", err)
				continue
			}
			if deleted > 0 {
				slog.Info("cleanup completed", "deleted", deleted)
			}
		}
	}
}
