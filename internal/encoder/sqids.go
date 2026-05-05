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
// sqids.New only errors for invalid alphabets; with the default alphabet and uint8 minLength
// (0–255) the constructor never fails in practice — the branch is kept for safety.
func NewSqidsEncoder(minLength uint8) (*SqidsEncoder, error) {
	return newSqidsEncoder(sqids.Options{MinLength: minLength})
}

func newSqidsEncoder(opts sqids.Options) (*SqidsEncoder, error) {
	s, err := sqids.New(opts)
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

	// sqids.Encode only errors when every permutation of the code is on the blocklist;
	// with the default blocklist and valid uint64 input this never happens in practice.
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
