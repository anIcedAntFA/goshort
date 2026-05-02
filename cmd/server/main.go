// Package main is the entry point of the GoShort HTTP server.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/anIcedAntFA/goshort/internal/api"
	"github.com/anIcedAntFA/goshort/internal/cache"
	"github.com/anIcedAntFA/goshort/internal/encoder"
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
)

func main() {
	port := envOr("GOSHORT_PORT", "8080")
	dataDir := envOr("GOSHORT_DATA_DIR", "./data")
	baseURL := envOr("GOSHORT_BASE_URL", "http://localhost:8080")
	cacheDriver := envOr("GOSHORT_CACHE_DRIVER", "none")

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		slog.Error("create data dir", "error", err)
		os.Exit(1)
	}

	// Initialize fallible dependencies before registering deferred cleanup so
	// os.Exit calls on startup errors do not skip deferred functions.
	dbPath := dataDir + "/goshort.db" //nolint:gosec // path from trusted env var
	store, err := storage.NewSQLiteStorage(context.Background(), dbPath)
	if err != nil {
		slog.Error("open storage", "error", err)
		os.Exit(1)
	}

	enc, err := encoder.NewSqidsEncoder(6)
	if err != nil {
		slog.Error("create encoder", "error", err)
		os.Exit(1)
	}

	// All fallible setup complete. Register signal handling + deferred cleanup.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var c shortener.Cache
	switch cacheDriver {
	case "memory":
		c = cache.NewMemoryCache()
	case "redis":
		redisAddr := envOr("GOSHORT_REDIS_URL", "localhost:6379")
		rc, rcErr := cache.NewRedisCache(redisAddr)
		if rcErr != nil {
			slog.Warn("redis unavailable, falling back to noop cache", "error", rcErr)
			c = cache.NewNoopCache()
		} else {
			c = rc
		}
	default:
		c = cache.NewNoopCache()
	}

	svc := shortener.NewService(store, c, enc)
	h := api.NewHandler(svc, c, logger, baseURL)
	router := api.NewRouter(h)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go startCleanupJob(ctx, store, logger)

	go func() {
		slog.Info("server starting", "addr", srv.Addr, "base_url", baseURL)
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
	if err := store.Close(); err != nil {
		slog.Error("close storage", "error", err)
	}
	slog.Info("server stopped")
}

func startCleanupJob(ctx context.Context, store shortener.Storage, logger *slog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("cleanup job stopped")
			return
		case <-ticker.C:
			deleted, err := store.DeleteExpired(ctx, 1000)
			if err != nil {
				logger.Error("cleanup failed", "error", err)
				continue
			}
			if deleted > 0 {
				logger.Info("cleanup completed", "deleted", deleted)
			}
		}
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
