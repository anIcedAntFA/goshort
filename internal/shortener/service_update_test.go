package shortener_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

func TestService_Update_SetExpiry(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := context.Background()

	store.insertDirect(&shortener.URL{ShortCode: "abc", OriginalURL: "https://example.com"})

	got, err := svc.Update(ctx, "abc", shortener.UpdateRequest{ExpiresIn: "7d"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ExpiresAt == nil {
		t.Fatal("ExpiresAt should be set, got nil")
	}
	if got.ShortCode != "abc" {
		t.Errorf("ShortCode = %q, want %q", got.ShortCode, "abc")
	}
}

func TestService_Update_ChangeExpiry(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := context.Background()

	old := time.Now().Add(30 * 24 * time.Hour)
	store.insertDirect(&shortener.URL{ShortCode: "abc2", OriginalURL: "https://example.com", ExpiresAt: &old})

	got, err := svc.Update(ctx, "abc2", shortener.UpdateRequest{ExpiresIn: "7d"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ExpiresAt == nil {
		t.Fatal("ExpiresAt should be set")
	}
	if !got.ExpiresAt.Before(old) {
		t.Errorf("new ExpiresAt %v should be before old %v", got.ExpiresAt, old)
	}
}

func TestService_Update_RemoveExpiry(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := context.Background()

	exp := time.Now().Add(24 * time.Hour)
	store.insertDirect(&shortener.URL{ShortCode: "abc3", OriginalURL: "https://example.com", ExpiresAt: &exp})

	got, err := svc.Update(ctx, "abc3", shortener.UpdateRequest{ExpiresIn: "0"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ExpiresAt != nil {
		t.Errorf("ExpiresAt should be nil after removal, got %v", got.ExpiresAt)
	}
}

func TestService_Update_NotFound(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	_, err := svc.Update(context.Background(), "no-such-code", shortener.UpdateRequest{ExpiresIn: "7d"})
	if !errors.Is(err, shortener.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestService_Update_InvalidExpiresIn(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	_, err := svc.Update(context.Background(), "abc", shortener.UpdateRequest{ExpiresIn: "bad"})
	if !errors.Is(err, shortener.ErrInvalidExpires) {
		t.Errorf("want ErrInvalidExpires, got %v", err)
	}
}
