package storage

import (
	"context"
	"log/slog"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

// StartCleanupJob runs a background goroutine that deletes expired URLs
// every hour. It stops when ctx is canceled.
func StartCleanupJob(ctx context.Context, store shortener.Storage) {
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
