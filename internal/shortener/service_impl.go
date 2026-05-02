package shortener

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/anIcedAntFA/goshort/internal/cache"
)

// ServiceImpl implements the Service interface.
type ServiceImpl struct {
	store   Storage
	cache   cache.Cache
	encoder Encoder
}

// compile-time interface check.
var _ Service = (*ServiceImpl)(nil)

// NewService creates a new Service backed by the provided storage, cache, and encoder.
func NewService(store Storage, c cache.Cache, enc Encoder) Service {
	return &ServiceImpl{store: store, cache: c, encoder: enc}
}

// Create validates the request, resolves the short code, and persists the URL.
func (s *ServiceImpl) Create(ctx context.Context, req CreateRequest) (*URL, error) {
	if err := ValidateURL(req.URL); err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		if err := ValidateExpiresIn(req.ExpiresIn); err != nil {
			return nil, err
		}
		t := time.Now().Add(parseExpiresIn(req.ExpiresIn))
		expiresAt = &t
	}

	shortCode, isCustom, err := s.resolveCode(ctx, req.CustomAlias)
	if err != nil {
		return nil, err
	}

	created, err := s.store.CreateURL(ctx, CreateParams{
		ShortCode:   shortCode,
		OriginalURL: req.URL,
		IsCustom:    isCustom,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("create url: %w", err)
	}

	return &created, nil
}

// resolveCode returns the short code and whether it is custom.
// For a custom alias it validates the alias and checks it is not already taken.
// For a generated code it increments the counter and encodes it.
func (s *ServiceImpl) resolveCode(ctx context.Context, alias string) (code string, isCustom bool, err error) {
	if alias == "" {
		counter, cErr := s.store.IncrementCounter(ctx)
		if cErr != nil {
			return "", false, fmt.Errorf("create url: increment counter: %w", cErr)
		}

		code, cErr = s.encoder.Encode(counter)
		if cErr != nil {
			return "", false, fmt.Errorf("create url: encode counter: %w", cErr)
		}

		return code, false, nil
	}

	if vErr := ValidateAlias(alias); vErr != nil {
		return "", false, vErr
	}

	_, err = s.store.GetByCode(ctx, alias)
	if err == nil {
		return "", false, fmt.Errorf("create url: alias %q: %w", alias, ErrAliasTaken)
	}

	if !errors.Is(err, ErrNotFound) {
		return "", false, fmt.Errorf("create url: check alias: %w", err)
	}

	return alias, true, nil
}

// GetByCode retrieves a URL by its short code.
// Returns ErrExpired if the URL has passed its expiration time.
func (s *ServiceImpl) GetByCode(ctx context.Context, code string) (*URL, error) {
	u, err := s.store.GetByCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("get by code: %w", err)
	}

	if u.ExpiresAt != nil && time.Now().After(*u.ExpiresAt) {
		return nil, fmt.Errorf("get by code %q: %w", code, ErrExpired)
	}

	return u, nil
}

// Delete removes a URL by its short code and invalidates the cache entry.
func (s *ServiceImpl) Delete(ctx context.Context, code string) error {
	if err := s.store.DeleteByCode(ctx, code); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	_ = s.cache.Delete(ctx, code)

	return nil
}

// List returns a paginated slice of URLs and the total count.
func (s *ServiceImpl) List(ctx context.Context, opts ListOptions) ([]URL, int, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}

	urls, err := s.store.ListURLs(ctx, opts.PerPage, (opts.Page-1)*opts.PerPage)
	if err != nil {
		return nil, 0, fmt.Errorf("list: %w", err)
	}

	total, err := s.store.CountURLs(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list count: %w", err)
	}

	return urls, total, nil
}

// IncrementClicks atomically increments the click counter for a URL.
func (s *ServiceImpl) IncrementClicks(ctx context.Context, code string) error {
	if err := s.store.IncrementClicks(ctx, code); err != nil {
		return fmt.Errorf("increment clicks: %w", err)
	}

	return nil
}

// parseExpiresIn converts a validated duration string (e.g. "7d", "24h") to a time.Duration.
// Precondition: the caller must have verified the input with ValidateExpiresIn.
func parseExpiresIn(expiresIn string) time.Duration {
	unit := expiresIn[len(expiresIn)-1]
	n, err := strconv.ParseInt(expiresIn[:len(expiresIn)-1], 10, 64)

	if err != nil || n <= 0 {
		return 0
	}

	switch unit {
	case 'h':
		return time.Duration(n) * time.Hour
	case 'd':
		return time.Duration(n) * 24 * time.Hour
	default:
		return 0
	}
}
