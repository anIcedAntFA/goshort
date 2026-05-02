package encoder

import (
	"fmt"
	"math"

	"github.com/anIcedAntFA/goshort/internal/shortener"
	sqids "github.com/sqids/sqids-go"
)

// SqidsEncoder encodes integer IDs into short alphanumeric codes using the Sqids algorithm.
type SqidsEncoder struct {
	s *sqids.Sqids
}

// compile-time interface check.
var _ shortener.Encoder = (*SqidsEncoder)(nil)

// NewSqidsEncoder creates a SqidsEncoder that produces codes of at least minLength characters.
func NewSqidsEncoder(minLength uint8) (*SqidsEncoder, error) {
	s, err := sqids.New(sqids.Options{
		MinLength: minLength,
	})
	if err != nil {
		return nil, fmt.Errorf("create sqids encoder: %w", err)
	}

	return &SqidsEncoder{s: s}, nil
}

// Encode converts a non-negative integer ID to a short alphanumeric code.
func (e *SqidsEncoder) Encode(id int64) (string, error) {
	if id < 0 {
		return "", fmt.Errorf("encode: %w", ErrNegativeID)
	}

	code, err := e.s.Encode([]uint64{uint64(id)})
	if err != nil {
		return "", fmt.Errorf("encode: %w", err)
	}

	return code, nil
}

// Decode converts a short code back to its original integer ID.
func (e *SqidsEncoder) Decode(code string) (int64, error) {
	nums := e.s.Decode(code)
	if len(nums) == 0 || nums[0] > math.MaxInt64 {
		return 0, fmt.Errorf("decode: %w", ErrInvalidCode)
	}

	return int64(nums[0]), nil //nolint:gosec // bounds checked: nums[0] <= math.MaxInt64
}
