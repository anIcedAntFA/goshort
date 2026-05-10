package shortener_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

// alphanumericRe matches codes produced by SqidsEncoder — no hyphens allowed.
var alphanumericRe = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

func TestService_Create_GeneratedCode(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	got, err := svc.Create(ctx, shortener.CreateRequest{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got.ShortCode == "" {
		t.Error("ShortCode is empty")
	}
	if !alphanumericRe.MatchString(got.ShortCode) {
		t.Errorf("ShortCode %q contains non-alphanumeric characters", got.ShortCode)
	}
	if got.IsCustom {
		t.Error("IsCustom = true, want false for a generated code")
	}
}

func TestService_Create_CustomAlias(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	got, err := svc.Create(ctx, shortener.CreateRequest{
		URL:         "https://example.com",
		CustomAlias: "my-link",
	})
	if err != nil {
		t.Fatalf("Create(custom alias): %v", err)
	}
	if got.ShortCode != "my-link" {
		t.Errorf("ShortCode = %q, want %q", got.ShortCode, "my-link")
	}
	if !got.IsCustom {
		t.Error("IsCustom = false, want true")
	}
}

func TestService_Create_ReservedPath(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, shortener.CreateRequest{
		URL:         "https://example.com",
		CustomAlias: "api",
	})
	if !errors.Is(err, shortener.ErrReservedPath) {
		t.Errorf("Create(reserved alias) = %v, want wrapping ErrReservedPath", err)
	}
}

func TestService_Create_InvalidURL(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, shortener.CreateRequest{URL: "not-a-url"})
	if err == nil {
		t.Fatal("Create(invalid URL) returned nil error, want non-nil")
	}
}

func TestService_Create_WithExpiry(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	before := time.Now()
	got, err := svc.Create(ctx, shortener.CreateRequest{
		URL:       "https://example.com",
		ExpiresIn: "7d",
	})
	if err != nil {
		t.Fatalf("Create(expires 7d): %v", err)
	}
	if got.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil, want non-nil")
	}

	want := before.Add(7 * 24 * time.Hour)
	diff := got.ExpiresAt.Sub(want)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("ExpiresAt = %v, want ~%v (±5s)", got.ExpiresAt, want)
	}
}

func TestService_Create_DuplicateAlias(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, shortener.CreateRequest{
		URL:         "https://example.com",
		CustomAlias: "taken",
	}); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err := svc.Create(ctx, shortener.CreateRequest{
		URL:         "https://other.com",
		CustomAlias: "taken",
	})
	if !errors.Is(err, shortener.ErrAliasTaken) {
		t.Errorf("second Create(duplicate alias) = %v, want wrapping ErrAliasTaken", err)
	}
}

func TestService_GetByCode(t *testing.T) {
	t.Parallel()

	t.Run("existing_returns_url", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestService(t)
		ctx := context.Background()

		created, err := svc.Create(ctx, shortener.CreateRequest{URL: "https://example.com"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := svc.GetByCode(ctx, created.ShortCode)
		if err != nil {
			t.Fatalf("GetByCode: %v", err)
		}
		if got.OriginalURL != "https://example.com" {
			t.Errorf("OriginalURL = %q, want %q", got.OriginalURL, "https://example.com")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()

		svc, _ := newTestService(t)
		ctx := context.Background()

		_, err := svc.GetByCode(ctx, "nonexistent")
		if !errors.Is(err, shortener.ErrNotFound) {
			t.Errorf("GetByCode(nonexistent) = %v, want wrapping ErrNotFound", err)
		}
	})

	t.Run("expired_returns_error", func(t *testing.T) {
		t.Parallel()

		svc, store := newTestService(t)
		ctx := context.Background()

		// Inject an already-expired URL directly — cannot go through Create
		// because ValidateExpiresIn rejects past durations.
		past := time.Now().Add(-1 * time.Hour)
		store.insertDirect(&shortener.URL{
			ID:          99,
			ShortCode:   "expired-abc",
			OriginalURL: "https://example.com",
			CreatedAt:   time.Now().Add(-2 * time.Hour),
			ExpiresAt:   &past,
		})

		_, err := svc.GetByCode(ctx, "expired-abc")
		if !errors.Is(err, shortener.ErrExpired) {
			t.Errorf("GetByCode(expired) = %v, want wrapping ErrExpired", err)
		}
	})
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
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
		t.Errorf("GetByCode after Delete = %v, want wrapping ErrNotFound", err)
	}
}

func TestService_List(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	for i := range 5 {
		u := fmt.Sprintf("https://example.com/%d", i)
		if _, err := svc.Create(ctx, shortener.CreateRequest{URL: u}); err != nil {
			t.Fatalf("Create URL %d: %v", i, err)
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

	page2, total2, err := svc.List(ctx, shortener.ListOptions{Page: 2, PerPage: 3})
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("page2 len = %d, want 2", len(page2))
	}
	if total2 != 5 {
		t.Errorf("total2 = %d, want 5", total2)
	}
}

func TestService_List_DefaultOptions(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	for i := range 3 {
		u := fmt.Sprintf("https://example.com/%d", i)
		if _, err := svc.Create(ctx, shortener.CreateRequest{URL: u}); err != nil {
			t.Fatalf("Create URL %d: %v", i, err)
		}
	}

	page, total, err := svc.List(ctx, shortener.ListOptions{})
	if err != nil {
		t.Fatalf("List(default options): %v", err)
	}
	if len(page) != 3 {
		t.Errorf("len = %d, want 3", len(page))
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
}

func TestService_IncrementClicks(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, shortener.CreateRequest{URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.IncrementClicks(ctx, created.ShortCode); err != nil {
		t.Fatalf("IncrementClicks: %v", err)
	}

	got, err := svc.GetByCode(ctx, created.ShortCode)
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got.ClickCount != 1 {
		t.Errorf("ClickCount = %d, want 1", got.ClickCount)
	}
}
