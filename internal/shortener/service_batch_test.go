package shortener_test

import (
	"context"
	"errors"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

// TestService_CreateBatch_AllValid verifies all results have short codes on success.
func TestService_CreateBatch_AllValid(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	reqs := []shortener.CreateRequest{
		{URL: "https://a.com"},
		{URL: "https://b.com"},
		{URL: "https://c.com"},
	}

	results, err := svc.CreateBatch(ctx, reqs)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	for i, r := range results {
		if r.Error != nil {
			t.Errorf("results[%d].Error = %v, want nil", i, r.Error)
		}
		if r.URL == nil {
			t.Errorf("results[%d].URL is nil, want non-nil", i)
		} else if r.URL.ShortCode == "" {
			t.Errorf("results[%d].URL.ShortCode is empty", i)
		}
	}
}

// TestService_CreateBatch_PartialFailure verifies per-item errors don't short-circuit the batch.
func TestService_CreateBatch_PartialFailure(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	reqs := []shortener.CreateRequest{
		{URL: "https://good.com"},
		{URL: "not-a-url"},
		{URL: "https://also-good.com"},
	}

	results, err := svc.CreateBatch(ctx, reqs)
	if err != nil {
		t.Fatalf("CreateBatch returned unexpected top-level error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	if results[0].Error != nil || results[0].URL == nil {
		t.Errorf("results[0]: expected success, got error=%v url=%v", results[0].Error, results[0].URL)
	}
	if !errors.Is(results[1].Error, shortener.ErrInvalidURL) {
		t.Errorf("results[1].Error = %v, want wrapping ErrInvalidURL", results[1].Error)
	}
	if results[2].Error != nil || results[2].URL == nil {
		t.Errorf("results[2]: expected success, got error=%v url=%v", results[2].Error, results[2].URL)
	}
}

// TestService_CreateBatch_ExceedsCap verifies batches over 50 are rejected.
func TestService_CreateBatch_ExceedsCap(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	reqs := make([]shortener.CreateRequest, 51)
	for i := range reqs {
		reqs[i] = shortener.CreateRequest{URL: "https://example.com"}
	}

	_, err := svc.CreateBatch(ctx, reqs)
	if err == nil {
		t.Fatal("expected error for batch exceeding 50, got nil")
	}
}

// TestService_CreateBatch_Empty verifies an empty batch is rejected.
func TestService_CreateBatch_Empty(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.CreateBatch(ctx, []shortener.CreateRequest{})
	if err == nil {
		t.Fatal("expected error for empty batch, got nil")
	}
}

// TestService_CreateBatch_DuplicateAlias verifies alias conflicts are per-item errors, not fatal.
func TestService_CreateBatch_DuplicateAlias(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	// First request establishes the alias.
	reqs := []shortener.CreateRequest{
		{URL: "https://first.com", CustomAlias: "clash"},
		{URL: "https://second.com", CustomAlias: "clash"},
	}

	results, err := svc.CreateBatch(ctx, reqs)
	if err != nil {
		t.Fatalf("CreateBatch returned unexpected top-level error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	if results[0].Error != nil {
		t.Errorf("results[0]: expected success, got %v", results[0].Error)
	}
	if !errors.Is(results[1].Error, shortener.ErrAliasTaken) {
		t.Errorf("results[1].Error = %v, want wrapping ErrAliasTaken", results[1].Error)
	}
}
