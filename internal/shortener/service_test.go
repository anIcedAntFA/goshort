package shortener_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/encoder"
	"github.com/anIcedAntFA/goshort/internal/shortener"
)

// errDuplicateCode is returned by mockStorage when a short code already exists.
var errDuplicateCode = errors.New("short code already exists")

// mockStorage is an in-memory implementation of shortener.Storage for service tests.
type mockStorage struct {
	mu      sync.Mutex
	urls    map[string]*shortener.URL
	counter int64
	nextID  int64

	// Per-method error injection: when set, the matching method returns this error
	// instead of executing its normal logic.
	errListURLs         error
	errCountURLs        error
	errCreateURL        error
	errIncrementCounter error
	errGetByCode        error
}

// mockEncoder is a controllable Encoder for error-path testing.
type mockEncoder struct {
	encodeErr error
}

func (m *mockEncoder) Encode(_ int64) (string, error) {
	if m.encodeErr != nil {
		return "", m.encodeErr
	}
	return "mocked-code", nil
}

func newMockStorage() *mockStorage {
	return &mockStorage{urls: make(map[string]*shortener.URL), nextID: 1}
}

// insertDirect bypasses the service and injects a URL directly into the mock.
// Used to set up test fixtures that cannot go through normal service validation.
func (m *mockStorage) insertDirect(u *shortener.URL) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *u
	m.urls[u.ShortCode] = &cp
}

func (m *mockStorage) CreateURL(_ context.Context, p *shortener.CreateParams) (shortener.URL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errCreateURL != nil {
		return shortener.URL{}, m.errCreateURL
	}

	if _, exists := m.urls[p.ShortCode]; exists {
		return shortener.URL{}, errDuplicateCode
	}

	u := shortener.URL{
		ID:          m.nextID,
		ShortCode:   p.ShortCode,
		OriginalURL: p.OriginalURL,
		IsCustom:    p.IsCustom,
		CreatedAt:   time.Now(),
		ExpiresAt:   p.ExpiresAt,
		Title:       p.Title,
		Description: p.Description,
	}
	m.nextID++

	cp := u
	m.urls[p.ShortCode] = &cp

	return u, nil
}

func (m *mockStorage) GetByCode(_ context.Context, code string) (*shortener.URL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errGetByCode != nil {
		return nil, m.errGetByCode
	}

	u, ok := m.urls[code]
	if !ok {
		return nil, fmt.Errorf("get by code %q: %w", code, shortener.ErrNotFound)
	}

	cp := *u
	return &cp, nil
}

func (m *mockStorage) DeleteByCode(_ context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.urls[code]; !ok {
		return fmt.Errorf("delete by code %q: %w", code, shortener.ErrNotFound)
	}

	delete(m.urls, code)
	return nil
}

func (m *mockStorage) ListURLs(_ context.Context, limit, offset int) ([]shortener.URL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errListURLs != nil {
		return nil, m.errListURLs
	}

	all := make([]shortener.URL, 0, len(m.urls))
	for _, u := range m.urls {
		all = append(all, *u)
	}

	sort.Slice(all, func(i, j int) bool {
		if all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].ID > all[j].ID
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	if offset >= len(all) {
		return []shortener.URL{}, nil
	}

	end := offset + limit
	if end > len(all) {
		end = len(all)
	}

	return all[offset:end], nil
}

func (m *mockStorage) CountURLs(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errCountURLs != nil {
		return 0, m.errCountURLs
	}

	return len(m.urls), nil
}

func (m *mockStorage) IncrementClicks(_ context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.urls[code]
	if !ok {
		return fmt.Errorf("increment clicks %q: %w", code, shortener.ErrNotFound)
	}

	u.ClickCount++
	return nil
}

func (m *mockStorage) DeleteExpired(_ context.Context, batchSize int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deleted int64
	now := time.Now()

	for code, u := range m.urls {
		if int(deleted) >= batchSize {
			break
		}
		if u.ExpiresAt != nil && now.After(*u.ExpiresAt) {
			delete(m.urls, code)
			deleted++
		}
	}

	return deleted, nil
}

func (m *mockStorage) GetCounter(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.counter, nil
}

func (m *mockStorage) IncrementCounter(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.errIncrementCounter != nil {
		return 0, m.errIncrementCounter
	}

	m.counter++
	return m.counter, nil
}

// newTestService creates a Service with mock storage, noop cache, and real SqidsEncoder.
func newTestService(t *testing.T) (shortener.Service, *mockStorage) {
	t.Helper()

	store := newMockStorage()

	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		t.Fatalf("NewSqidsEncoder: %v", err)
	}

	return shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}), store
}

// alphanumericRe matches codes produced by SqidsEncoder — no hyphens allowed.
var alphanumericRe = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// TestService_Create_GeneratedCode verifies that Create returns an alphanumeric short code.
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

// TestService_Create_CustomAlias verifies that Create uses the provided alias as the short code.
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

// TestService_Create_ReservedPath verifies that reserved paths are rejected.
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

// TestService_Create_InvalidURL verifies that malformed URLs are rejected.
func TestService_Create_InvalidURL(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, shortener.CreateRequest{URL: "not-a-url"})
	if err == nil {
		t.Fatal("Create(invalid URL) returned nil error, want non-nil")
	}
}

// TestService_Create_WithExpiry verifies that ExpiresIn is parsed and stored as ExpiresAt.
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

// TestService_Create_DuplicateAlias verifies that a second create with the same alias fails.
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

// TestService_GetByCode covers retrieval, missing codes, and expired URLs.
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

// TestService_Delete verifies a deleted URL becomes unreachable.
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

// TestService_List verifies paginated listing returns correct counts and slices.
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

// TestService_List_DefaultOptions verifies zero-value ListOptions defaults to page 1 / 20 per page.
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

	// Zero-value options: Page=0 and PerPage=0 should default to page 1 with 20 per page.
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

// TestService_List_StoreError verifies that a storage list failure is propagated.
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

// TestService_List_CountError verifies that a storage count failure is propagated.
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

// TestService_Delete_NotFound verifies that deleting a missing code returns ErrNotFound.
func TestService_Delete_NotFound(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)
	ctx := context.Background()

	err := svc.Delete(ctx, "ghost")
	if !errors.Is(err, shortener.ErrNotFound) {
		t.Errorf("Delete(nonexistent) = %v, want wrapping ErrNotFound", err)
	}
}

// TestService_IncrementClicks verifies that click counts are forwarded to storage.
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

// TestService_Create_InvalidExpiresIn verifies that a malformed ExpiresIn is rejected.
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

// TestService_Create_WithHourExpiry verifies that ExpiresIn in hours is parsed correctly.
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

// TestService_Create_IncrementCounterError verifies that a counter storage failure
// is propagated when generating a short code.
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

// TestService_Create_EncodeCounterError verifies that an encoder failure is propagated.
func TestService_Create_EncodeCounterError(t *testing.T) {
	t.Parallel()

	store := newMockStorage()
	enc := &mockEncoder{encodeErr: errors.New("encode failed")}
	svc := shortener.NewService(store, enc, shortener.NoopPreviewFetcher{})

	_, err := svc.Create(context.Background(), shortener.CreateRequest{
		URL: "https://example.com",
	})
	if err == nil {
		t.Fatal("Create must return an error when Encode fails")
	}
}

// TestService_Create_StoreCreateError verifies that a CreateURL storage failure is propagated.
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

// TestService_Create_AliasCheckUnexpectedError verifies that a non-ErrNotFound error
// from GetByCode during alias availability check is propagated.
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

// TestService_IncrementClicks_Error verifies that an IncrementClicks storage failure is propagated.
func TestService_IncrementClicks_Error(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t)

	// Mock returns ErrNotFound for any unknown code, which is a non-nil error.
	err := svc.IncrementClicks(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("IncrementClicks must return an error for a non-existent code")
	}
}
