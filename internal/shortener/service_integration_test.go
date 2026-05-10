package shortener_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/encoder"
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
)

func newIntegrationService(t *testing.T) shortener.Service {
	t.Helper()

	store, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		t.Fatalf("NewSqidsEncoder: %v", err)
	}

	return shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, shortener.NoopChecker{})
}

func TestIntegration_CreateAndGetByCode(t *testing.T) {
	t.Parallel()

	svc := newIntegrationService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, shortener.CreateRequest{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ShortCode == "" {
		t.Fatal("ShortCode is empty")
	}

	got, err := svc.GetByCode(ctx, created.ShortCode)
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got.OriginalURL != "https://example.com" {
		t.Errorf("OriginalURL = %q, want %q", got.OriginalURL, "https://example.com")
	}
}

func TestIntegration_CreateDeleteGetByCode(t *testing.T) {
	t.Parallel()

	svc := newIntegrationService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, shortener.CreateRequest{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Delete(ctx, created.ShortCode); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = svc.GetByCode(ctx, created.ShortCode)
	if !errors.Is(err, shortener.ErrNotFound) {
		t.Errorf("GetByCode after Delete = %v, want ErrNotFound", err)
	}
}

func TestIntegration_ListPagination(t *testing.T) {
	t.Parallel()

	svc := newIntegrationService(t)
	ctx := context.Background()

	for i := range 5 {
		if _, err := svc.Create(ctx, shortener.CreateRequest{
			URL: fmt.Sprintf("https://example.com/%d", i),
		}); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}

	page1, total, err := svc.List(ctx, shortener.ListOptions{Page: 1, PerPage: 3})
	if err != nil {
		t.Fatalf("List page 1: %v", err)
	}
	if len(page1) != 3 {
		t.Errorf("page1 len = %d, want 3", len(page1))
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}

	page2, _, err := svc.List(ctx, shortener.ListOptions{Page: 2, PerPage: 3})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("page2 len = %d, want 2", len(page2))
	}
}

func TestIntegration_CreateBatchPartialFailure(t *testing.T) {
	t.Parallel()

	svc := newIntegrationService(t)
	ctx := context.Background()

	results, err := svc.CreateBatch(ctx, []shortener.CreateRequest{
		{URL: "https://good.com"},
		{URL: "not-a-url"},
		{URL: "https://also-good.com"},
	})
	if err != nil {
		t.Fatalf("CreateBatch top-level error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	if results[0].Error != nil || results[0].URL == nil {
		t.Errorf("results[0]: want success, got error=%v", results[0].Error)
	}
	if !errors.Is(results[1].Error, shortener.ErrInvalidURL) {
		t.Errorf("results[1].Error = %v, want ErrInvalidURL", results[1].Error)
	}
	if results[2].Error != nil || results[2].URL == nil {
		t.Errorf("results[2]: want success, got error=%v", results[2].Error)
	}
}

func TestIntegration_ExpiredURL(t *testing.T) {
	t.Parallel()

	store, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		t.Fatalf("NewSqidsEncoder: %v", err)
	}
	svc := shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, shortener.NoopChecker{})
	ctx := context.Background()

	past := time.Now().Add(-time.Hour)
	_, err = store.CreateURL(ctx, &shortener.CreateParams{
		ShortCode:   "old-link",
		OriginalURL: "https://example.com",
		ExpiresAt:   &past,
	})
	if err != nil {
		t.Fatalf("CreateURL: %v", err)
	}

	_, err = svc.GetByCode(ctx, "old-link")
	if !errors.Is(err, shortener.ErrExpired) {
		t.Errorf("GetByCode(expired) = %v, want ErrExpired", err)
	}
}

func TestIntegration_UpdateExpiry(t *testing.T) {
	t.Parallel()

	svc := newIntegrationService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, shortener.CreateRequest{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := svc.Update(ctx, created.ShortCode, shortener.UpdateRequest{ExpiresIn: "7d"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil after update")
	}

	want := time.Now().Add(7 * 24 * time.Hour)
	diff := got.ExpiresAt.Sub(want)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("ExpiresAt = %v, want ~%v (±5s)", got.ExpiresAt, want)
	}

	// Remove expiry.
	got, err = svc.Update(ctx, created.ShortCode, shortener.UpdateRequest{ExpiresIn: "0"})
	if err != nil {
		t.Fatalf("Update(remove expiry): %v", err)
	}
	if got.ExpiresAt != nil {
		t.Errorf("ExpiresAt should be nil after removal, got %v", got.ExpiresAt)
	}
}
