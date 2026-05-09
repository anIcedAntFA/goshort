package shortener

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// ServiceImpl implements the Service interface.
type ServiceImpl struct {
	store   Storage
	encoder Encoder
	preview PreviewFetcher
}

// compile-time interface check.
var _ Service = (*ServiceImpl)(nil)

// NewService creates a new Service backed by the provided storage, encoder, and preview fetcher.
func NewService(store Storage, enc Encoder, preview PreviewFetcher) *ServiceImpl {
	return &ServiceImpl{store: store, encoder: enc, preview: preview}
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
		dur, err := parseExpiresIn(req.ExpiresIn)
		if err != nil {
			return nil, err
		}
		t := time.Now().Add(dur)
		expiresAt = &t
	}

	shortCode, isCustom, err := s.resolveCode(ctx, req.CustomAlias)
	if err != nil {
		return nil, err
	}

	title, desc, _ := s.preview.Fetch(ctx, req.URL)

	created, err := s.store.CreateURL(ctx, &CreateParams{
		ShortCode:   shortCode,
		OriginalURL: req.URL,
		IsCustom:    isCustom,
		ExpiresAt:   expiresAt,
		Title:       title,
		Description: desc,
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

// Delete removes a URL by its short code.
func (s *ServiceImpl) Delete(ctx context.Context, code string) error {
	if err := s.store.DeleteByCode(ctx, code); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
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

const batchMaxSize = 50

// CreateBatch creates up to 50 shortened URLs in one call.
// Per-item errors are collected into BatchResult.Error; the batch always
// processes every item regardless of individual failures.
func (s *ServiceImpl) CreateBatch(ctx context.Context, reqs []CreateRequest) ([]BatchResult, error) {
	if len(reqs) == 0 {
		return nil, ErrBatchEmpty
	}
	if len(reqs) > batchMaxSize {
		return nil, ErrBatchTooLarge
	}

	results := make([]BatchResult, len(reqs))
	for i, req := range reqs {
		u, err := s.Create(ctx, req)
		results[i] = BatchResult{URL: u, Error: err}
	}

	return results, nil
}

func parseExpiresIn(s string) (time.Duration, error) {
	unit := s[len(s)-1]
	n, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("parse expires_in %q: %w", s, ErrInvalidExpires)
	}

	switch unit {
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("parse expires_in %q: unsupported unit %q: %w", s, string(unit), ErrInvalidExpires)
	}
}
