package storage_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/anIcedAntFA/goshort/internal/storage"
)

// newTestStorage creates a fresh in-memory SQLiteStorage for each test.
// The database is closed automatically when the test ends.
func newTestStorage(t *testing.T) *storage.SQLiteStorage {
	t.Helper()

	s, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage: %v", err)
	}

	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("close storage: %v", err)
		}
	})

	return s
}

// sampleParams returns minimal valid CreateParams for test use.
func sampleParams(code, url string) shortener.CreateParams {
	return shortener.CreateParams{
		ShortCode:   code,
		OriginalURL: url,
		IsCustom:    false,
		ExpiresAt:   nil,
	}
}

// TestSQLiteStorage_CreateURL_GetByCode verifies the basic create → retrieve roundtrip.
func TestSQLiteStorage_CreateURL_GetByCode(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	created, err := s.CreateURL(ctx, sampleParams("abc123", "https://example.com"))
	if err != nil {
		t.Fatalf("CreateURL: %v", err)
	}

	if created.ShortCode != "abc123" {
		t.Errorf("ShortCode = %q, want %q", created.ShortCode, "abc123")
	}
	if created.OriginalURL != "https://example.com" {
		t.Errorf("OriginalURL = %q, want %q", created.OriginalURL, "https://example.com")
	}
	if created.IsCustom {
		t.Error("IsCustom = true, want false")
	}
	if created.ID == 0 {
		t.Error("ID must be non-zero after creation")
	}
	if created.CreatedAt.IsZero() {
		t.Error("CreatedAt must be set")
	}

	got, err := s.GetByCode(ctx, "abc123")
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}

	if got.ShortCode != created.ShortCode {
		t.Errorf("retrieved ShortCode = %q, want %q", got.ShortCode, created.ShortCode)
	}
	if got.OriginalURL != created.OriginalURL {
		t.Errorf("retrieved OriginalURL = %q, want %q", got.OriginalURL, created.OriginalURL)
	}
}

// TestSQLiteStorage_CreateURL_CustomAlias verifies custom aliases are stored correctly.
func TestSQLiteStorage_CreateURL_CustomAlias(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	params := shortener.CreateParams{
		ShortCode:   "my-link",
		OriginalURL: "https://example.com/long/path",
		IsCustom:    true,
		ExpiresAt:   nil,
	}

	created, err := s.CreateURL(ctx, params)
	if err != nil {
		t.Fatalf("CreateURL: %v", err)
	}

	if !created.IsCustom {
		t.Error("IsCustom = false, want true")
	}
	if created.ShortCode != "my-link" {
		t.Errorf("ShortCode = %q, want %q", created.ShortCode, "my-link")
	}

	got, err := s.GetByCode(ctx, "my-link")
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if !got.IsCustom {
		t.Error("retrieved IsCustom = false, want true")
	}
}

// TestSQLiteStorage_CreateURL_WithExpiry verifies expiry timestamps are stored and retrieved.
func TestSQLiteStorage_CreateURL_WithExpiry(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	expiry := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	params := shortener.CreateParams{
		ShortCode:   "exp-url",
		OriginalURL: "https://example.com",
		IsCustom:    false,
		ExpiresAt:   &expiry,
	}

	created, err := s.CreateURL(ctx, params)
	if err != nil {
		t.Fatalf("CreateURL: %v", err)
	}

	if created.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil, want non-nil")
	}
	if !created.ExpiresAt.Equal(expiry) {
		t.Errorf("ExpiresAt = %v, want %v", created.ExpiresAt, expiry)
	}
}

// TestSQLiteStorage_GetByCode_NotFound verifies ErrNotFound for missing short codes.
func TestSQLiteStorage_GetByCode_NotFound(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	_, err := s.GetByCode(ctx, "nonexistent")
	if !errors.Is(err, shortener.ErrNotFound) {
		t.Errorf("GetByCode(nonexistent) = %v, want wrapping ErrNotFound", err)
	}
}

// TestSQLiteStorage_DeleteByCode verifies delete makes the record unreachable.
func TestSQLiteStorage_DeleteByCode(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	if _, err := s.CreateURL(ctx, sampleParams("del-me", "https://example.com")); err != nil {
		t.Fatalf("CreateURL: %v", err)
	}

	if err := s.DeleteByCode(ctx, "del-me"); err != nil {
		t.Fatalf("DeleteByCode: %v", err)
	}

	_, err := s.GetByCode(ctx, "del-me")
	if !errors.Is(err, shortener.ErrNotFound) {
		t.Errorf("GetByCode after delete = %v, want wrapping ErrNotFound", err)
	}
}

// TestSQLiteStorage_DeleteByCode_NotFound verifies ErrNotFound when deleting a missing code.
func TestSQLiteStorage_DeleteByCode_NotFound(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	err := s.DeleteByCode(ctx, "does-not-exist")
	if !errors.Is(err, shortener.ErrNotFound) {
		t.Errorf("DeleteByCode(nonexistent) = %v, want wrapping ErrNotFound", err)
	}
}

// TestSQLiteStorage_ListURLs_Pagination verifies page-based listing with correct ordering.
func TestSQLiteStorage_ListURLs_Pagination(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	// Insert 5 records. CreatedAt uses datetime('now') which has 1-second resolution,
	// so we sleep briefly between inserts to guarantee ordering.
	codes := []string{"code1", "code2", "code3", "code4", "code5"}
	for _, code := range codes {
		if _, err := s.CreateURL(ctx, sampleParams(code, "https://example.com/"+code)); err != nil {
			t.Fatalf("CreateURL(%s): %v", code, err)
		}
		time.Sleep(1100 * time.Millisecond)
	}

	// Page 1: first 3 (most recent).
	page1, err := s.ListURLs(ctx, 3, 0)
	if err != nil {
		t.Fatalf("ListURLs page1: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("page1 len = %d, want 3", len(page1))
	}
	if page1[0].ShortCode != "code5" {
		t.Errorf("page1[0].ShortCode = %q, want %q", page1[0].ShortCode, "code5")
	}

	// Page 2: remaining 2.
	page2, err := s.ListURLs(ctx, 3, 3)
	if err != nil {
		t.Fatalf("ListURLs page2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page2 len = %d, want 2", len(page2))
	}
	if page2[0].ShortCode != "code2" {
		t.Errorf("page2[0].ShortCode = %q, want %q", page2[0].ShortCode, "code2")
	}

	// Empty page beyond range.
	page3, err := s.ListURLs(ctx, 10, 10)
	if err != nil {
		t.Fatalf("ListURLs page3: %v", err)
	}
	if len(page3) != 0 {
		t.Errorf("page3 len = %d, want 0", len(page3))
	}
}

// TestSQLiteStorage_CountURLs verifies the total count tracks inserts and deletes.
func TestSQLiteStorage_CountURLs(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	n, err := s.CountURLs(ctx)
	if err != nil {
		t.Fatalf("CountURLs (empty): %v", err)
	}
	if n != 0 {
		t.Errorf("initial count = %d, want 0", n)
	}

	for _, code := range []string{"a", "b", "c"} {
		if _, err := s.CreateURL(ctx, sampleParams(code, "https://example.com")); err != nil {
			t.Fatalf("CreateURL(%s): %v", code, err)
		}
	}

	n, err = s.CountURLs(ctx)
	if err != nil {
		t.Fatalf("CountURLs after 3 inserts: %v", err)
	}
	if n != 3 {
		t.Errorf("count after 3 inserts = %d, want 3", n)
	}
}

// TestSQLiteStorage_IncrementClicks verifies click counts increment correctly.
func TestSQLiteStorage_IncrementClicks(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	if _, err := s.CreateURL(ctx, sampleParams("click-me", "https://example.com")); err != nil {
		t.Fatalf("CreateURL: %v", err)
	}

	for i := range 3 {
		if err := s.IncrementClicks(ctx, "click-me"); err != nil {
			t.Fatalf("IncrementClicks #%d: %v", i+1, err)
		}
	}

	got, err := s.GetByCode(ctx, "click-me")
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if got.ClickCount != 3 {
		t.Errorf("ClickCount = %d, want 3", got.ClickCount)
	}
}

// TestSQLiteStorage_DeleteExpired verifies only expired records are removed.
func TestSQLiteStorage_DeleteExpired(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Hour).UTC()
	future := time.Now().Add(24 * time.Hour).UTC()

	// Two expired, one not expired, one with no expiry.
	expired1 := shortener.CreateParams{ShortCode: "exp1", OriginalURL: "https://example.com", ExpiresAt: &past}
	expired2 := shortener.CreateParams{ShortCode: "exp2", OriginalURL: "https://example.com", ExpiresAt: &past}
	live := shortener.CreateParams{ShortCode: "live", OriginalURL: "https://example.com", ExpiresAt: &future}
	noexp := shortener.CreateParams{ShortCode: "noexp", OriginalURL: "https://example.com", ExpiresAt: nil}

	for _, p := range []shortener.CreateParams{expired1, expired2, live, noexp} {
		if _, err := s.CreateURL(ctx, p); err != nil {
			t.Fatalf("CreateURL(%s): %v", p.ShortCode, err)
		}
	}

	deleted, err := s.DeleteExpired(ctx, 100)
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2", deleted)
	}

	// Expired records should be gone.
	for _, code := range []string{"exp1", "exp2"} {
		_, err := s.GetByCode(ctx, code)
		if !errors.Is(err, shortener.ErrNotFound) {
			t.Errorf("GetByCode(%s) after DeleteExpired = %v, want ErrNotFound", code, err)
		}
	}

	// Non-expired records must survive.
	for _, code := range []string{"live", "noexp"} {
		if _, err := s.GetByCode(ctx, code); err != nil {
			t.Errorf("GetByCode(%s) after DeleteExpired = %v, want nil", code, err)
		}
	}
}

// TestSQLiteStorage_DeleteExpired_Batch verifies the batchSize limit is respected.
func TestSQLiteStorage_DeleteExpired_Batch(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Hour).UTC()
	for i := range 5 {
		code := fmt.Sprintf("exp%d", i)
		p := shortener.CreateParams{ShortCode: code, OriginalURL: "https://example.com", ExpiresAt: &past}
		if _, err := s.CreateURL(ctx, p); err != nil {
			t.Fatalf("CreateURL(%s): %v", code, err)
		}
	}

	deleted, err := s.DeleteExpired(ctx, 3) // batch of 3
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if deleted != 3 {
		t.Errorf("deleted = %d, want 3", deleted)
	}

	// 2 expired records should remain.
	n, err := s.CountURLs(ctx)
	if err != nil {
		t.Fatalf("CountURLs: %v", err)
	}
	if n != 2 {
		t.Errorf("remaining count = %d, want 2", n)
	}
}

// TestSQLiteStorage_Counter verifies the counter initializes at 0 and increments sequentially.
func TestSQLiteStorage_Counter(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	initial, err := s.GetCounter(ctx)
	if err != nil {
		t.Fatalf("GetCounter: %v", err)
	}
	if initial != 0 {
		t.Errorf("initial counter = %d, want 0", initial)
	}

	for want := int64(1); want <= 5; want++ {
		got, err := s.IncrementCounter(ctx)
		if err != nil {
			t.Fatalf("IncrementCounter: %v", err)
		}
		if got != want {
			t.Errorf("IncrementCounter() = %d, want %d", got, want)
		}
	}

	final, err := s.GetCounter(ctx)
	if err != nil {
		t.Fatalf("GetCounter: %v", err)
	}
	if final != 5 {
		t.Errorf("final counter = %d, want 5", final)
	}
}

// TestSQLiteStorage_Counter_Concurrent verifies IncrementCounter is safe under concurrent access.
func TestSQLiteStorage_Counter_Concurrent(t *testing.T) {
	t.Parallel()

	s := newTestStorage(t)
	ctx := context.Background()

	const goroutines = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			if _, err := s.IncrementCounter(ctx); err != nil {
				t.Errorf("IncrementCounter: %v", err)
			}
		}()
	}

	wg.Wait()

	got, err := s.GetCounter(ctx)
	if err != nil {
		t.Fatalf("GetCounter: %v", err)
	}
	if got != goroutines {
		t.Errorf("counter after %d concurrent increments = %d, want %d", goroutines, got, goroutines)
	}
}
