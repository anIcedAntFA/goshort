package shortener_test

import (
	"context"
	"errors"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/encoder"
	"github.com/anIcedAntFA/goshort/internal/shortener"
)

func TestService_Create_UnsafeURL(t *testing.T) {
	t.Parallel()

	store := newMockStorage()
	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		t.Fatalf("NewSqidsEncoder: %v", err)
	}

	checker := &mockChecker{checkErr: shortener.ErrUnsafeURL}
	svc := shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, checker)

	_, err = svc.Create(context.Background(), shortener.CreateRequest{URL: "https://evil.example.com"})
	if !errors.Is(err, shortener.ErrUnsafeURL) {
		t.Errorf("Create(unsafe URL) = %v, want ErrUnsafeURL", err)
	}
}

func TestService_CreateBatch_UnsafeURL(t *testing.T) {
	t.Parallel()

	store := newMockStorage()
	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		t.Fatalf("NewSqidsEncoder: %v", err)
	}

	checker := &mockChecker{checkErr: shortener.ErrUnsafeURL}
	svc := shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, checker)

	results, err := svc.CreateBatch(context.Background(), []shortener.CreateRequest{
		{URL: "https://evil.example.com"},
		{URL: "https://also-evil.example.com"},
	})
	if err != nil {
		t.Fatalf("CreateBatch returned top-level error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}
	for i, r := range results {
		if !errors.Is(r.Error, shortener.ErrUnsafeURL) {
			t.Errorf("results[%d].Error = %v, want ErrUnsafeURL", i, r.Error)
		}
	}
}
