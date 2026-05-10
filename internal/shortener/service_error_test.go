package shortener_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

func TestService_List_StoreError(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := context.Background()

	store.errListURLs = errors.New("db unavailable")

	_, _, err := svc.List(ctx, shortener.ListOptions{Page: 1, PerPage: 10})
	if err == nil {
		t.Fatal("List(store error) = nil, want non-nil")
	}
}

func TestService_List_CountError(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	ctx := context.Background()

	if _, err := svc.Create(ctx, shortener.CreateRequest{URL: "https://example.com"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	store.errCountURLs = errors.New("db unavailable")

	_, _, err := svc.List(ctx, shortener.ListOptions{Page: 1, PerPage: 10})
	if err == nil {
		t.Fatal("List(count error) = nil, want non-nil")
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	err := svc.Delete(ctx, "ghost")
	if !errors.Is(err, shortener.ErrNotFound) {
		t.Errorf("Delete(nonexistent) = %v, want wrapping ErrNotFound", err)
	}
}

func TestService_IncrementClicks_Error(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	err := svc.IncrementClicks(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("IncrementClicks must return an error for a non-existent code")
	}
}

func TestService_Create_InvalidExpiresIn(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, shortener.CreateRequest{
		URL:       "https://example.com",
		ExpiresIn: "0h",
	})
	if err == nil {
		t.Fatal("Create(invalid ExpiresIn) = nil, want non-nil")
	}
}

func TestService_Create_WithHourExpiry(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	before := time.Now()
	got, err := svc.Create(ctx, shortener.CreateRequest{
		URL:       "https://example.com",
		ExpiresIn: "24h",
	})
	if err != nil {
		t.Fatalf("Create(expires 24h): %v", err)
	}
	if got.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil, want non-nil")
	}

	want := before.Add(24 * time.Hour)
	diff := got.ExpiresAt.Sub(want)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("ExpiresAt = %v, want ~%v (±5s)", got.ExpiresAt, want)
	}
}

func TestService_Create_IncrementCounterError(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	store.errIncrementCounter = errors.New("counter db failed")

	_, err := svc.Create(context.Background(), shortener.CreateRequest{
		URL: "https://example.com",
	})
	if err == nil {
		t.Fatal("Create must return an error when IncrementCounter fails")
	}
}

func TestService_Create_EncodeCounterError(t *testing.T) {
	t.Parallel()

	store := newMockStorage()
	enc := &mockEncoder{encodeErr: errors.New("encode failed")}
	svc := shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, shortener.NoopChecker{})

	_, err := svc.Create(context.Background(), shortener.CreateRequest{
		URL: "https://example.com",
	})
	if err == nil {
		t.Fatal("Create must return an error when Encode fails")
	}
}

func TestService_Create_StoreCreateError(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	store.errCreateURL = errors.New("db write failed")

	_, err := svc.Create(context.Background(), shortener.CreateRequest{
		URL: "https://example.com",
	})
	if err == nil {
		t.Fatal("Create must return an error when CreateURL fails")
	}
}

func TestService_Create_AliasCheckUnexpectedError(t *testing.T) {
	t.Parallel()

	svc, store := newTestService(t)
	store.errGetByCode = errors.New("db connection lost")

	_, err := svc.Create(context.Background(), shortener.CreateRequest{
		URL:         "https://example.com",
		CustomAlias: "my-alias",
	})
	if err == nil {
		t.Fatal("Create must return an error when GetByCode returns an unexpected error")
	}
}
