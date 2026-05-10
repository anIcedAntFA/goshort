package shortener_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/encoder"
	"github.com/anIcedAntFA/goshort/internal/shortener"
)

// errDuplicateCode is returned by mockStorage when a short code already exists.
var errDuplicateCode = errors.New("short code already exists")

// mockChecker is a controllable URLChecker for error-path testing.
type mockChecker struct {
	checkErr error
}

func (m *mockChecker) Check(_ context.Context, _ string) error { return m.checkErr }

// mockStorage is an in-memory implementation of shortener.Storage for service tests.
type mockStorage struct {
	mu      sync.Mutex
	urls    map[string]*shortener.URL
	counter int64
	nextID  int64

	// Per-method error injection.
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

func (m *mockStorage) UpdateExpiry(_ context.Context, code string, expiresAt *time.Time) (*shortener.URL, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.urls[code]
	if !ok {
		return nil, fmt.Errorf("update expiry %q: %w", code, shortener.ErrNotFound)
	}

	u.ExpiresAt = expiresAt
	return u, nil
}

// newTestService creates a Service with mock storage and real SqidsEncoder.
func newTestService(t *testing.T) (shortener.Service, *mockStorage) {
	t.Helper()

	store := newMockStorage()

	enc, err := encoder.NewSqidsEncoder(4)
	if err != nil {
		t.Fatalf("NewSqidsEncoder: %v", err)
	}

	return shortener.NewService(store, enc, shortener.NoopPreviewFetcher{}, shortener.NoopChecker{}), store
}
