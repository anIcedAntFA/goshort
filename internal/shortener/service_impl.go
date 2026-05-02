package shortener

import (
	"context"

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

// Create creates a new shortened URL from the given request.
func (s *ServiceImpl) Create(_ context.Context, _ CreateRequest) (*URL, error) {
	panic("not implemented")
}

// GetByCode retrieves a URL by its short code.
func (s *ServiceImpl) GetByCode(_ context.Context, _ string) (*URL, error) {
	panic("not implemented")
}

// Delete removes a shortened URL by its short code.
func (s *ServiceImpl) Delete(_ context.Context, _ string) error {
	panic("not implemented")
}

// List returns a paginated slice of URLs and the total count.
func (s *ServiceImpl) List(_ context.Context, _ ListOptions) ([]URL, int, error) {
	panic("not implemented")
}

// IncrementClicks atomically increments the click counter for a URL.
func (s *ServiceImpl) IncrementClicks(_ context.Context, _ string) error {
	panic("not implemented")
}
