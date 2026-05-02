package encoder_test

import (
	"errors"
	"math"
	"regexp"
	"slices"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/encoder"
)

func newTestEncoder(t *testing.T) *encoder.SqidsEncoder {
	t.Helper()

	enc, err := encoder.NewSqidsEncoder(6)
	if err != nil {
		t.Fatalf("NewSqidsEncoder: %v", err)
	}

	return enc
}

func TestSqidsEncoder_Encode_ReturnsNonEmpty(t *testing.T) {
	t.Parallel()

	enc := newTestEncoder(t)

	code, err := enc.Encode(1)
	if err != nil {
		t.Fatalf("Encode(1): %v", err)
	}

	if code == "" {
		t.Error("Encode(1) returned empty string")
	}
}

func TestSqidsEncoder_Roundtrip(t *testing.T) {
	t.Parallel()

	enc := newTestEncoder(t)

	cases := []struct {
		name string
		id   int64
	}{
		{"zero", 0},
		{"one", 1},
		{"large", 1_000_000},
		{"max_int32", math.MaxInt32},
		{"max_int64", math.MaxInt64},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			code, err := enc.Encode(tc.id)
			if err != nil {
				t.Fatalf("Encode(%d): %v", tc.id, err)
			}

			got, err := enc.Decode(code)
			if err != nil {
				t.Fatalf("Decode(%q): %v", code, err)
			}

			if got != tc.id {
				t.Errorf("roundtrip: Encode(%d) -> %q -> Decode -> %d", tc.id, code, got)
			}
		})
	}
}

func TestSqidsEncoder_DifferentIDsDifferentCodes(t *testing.T) {
	t.Parallel()

	enc := newTestEncoder(t)

	ids := []int64{0, 1, 2, 100, 1000, math.MaxInt32, math.MaxInt64}
	seen := make(map[string]struct{}, len(ids))

	for _, id := range ids {
		code, err := enc.Encode(id)
		if err != nil {
			t.Fatalf("Encode(%d): %v", id, err)
		}

		if _, ok := seen[code]; ok {
			t.Errorf("duplicate code %q for id %d", code, id)
		}

		seen[code] = struct{}{}
	}
}

func TestSqidsEncoder_CodesOnlyAlphanumeric(t *testing.T) {
	t.Parallel()

	enc := newTestEncoder(t)
	pattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	for _, id := range []int64{0, 1, 42, 999, math.MaxInt32} {
		code, err := enc.Encode(id)
		if err != nil {
			t.Fatalf("Encode(%d): %v", id, err)
		}

		if !pattern.MatchString(code) {
			t.Errorf("Encode(%d) = %q contains non-alphanumeric characters", id, code)
		}
	}
}

func TestSqidsEncoder_ConsecutiveIDsNonSequential(t *testing.T) {
	t.Parallel()

	enc := newTestEncoder(t)

	const n = 10

	codes := make([]string, n)
	for i := range codes {
		code, err := enc.Encode(int64(i))
		if err != nil {
			t.Fatalf("Encode(%d): %v", i, err)
		}

		codes[i] = code
	}

	if slices.IsSorted(codes) {
		t.Error("codes for consecutive IDs are lexicographically sorted — encoding appears sequential")
	}
}

func TestSqidsEncoder_MinLength(t *testing.T) {
	t.Parallel()

	const minLen = 6

	enc := newTestEncoder(t)

	for _, id := range []int64{0, 1, 999} {
		code, err := enc.Encode(id)
		if err != nil {
			t.Fatalf("Encode(%d): %v", id, err)
		}

		if len(code) < minLen {
			t.Errorf("Encode(%d) = %q: length %d < minimum %d", id, code, len(code), minLen)
		}
	}
}

func TestSqidsEncoder_Encode_NegativeIDReturnsError(t *testing.T) {
	t.Parallel()

	enc := newTestEncoder(t)

	_, err := enc.Encode(-1)
	if err == nil {
		t.Fatal("Encode(-1) must return an error")
	}

	if !errors.Is(err, encoder.ErrNegativeID) {
		t.Errorf("Encode(-1) error = %v, want wrapping ErrNegativeID", err)
	}
}

func TestSqidsEncoder_Decode_InvalidCodeReturnsError(t *testing.T) {
	t.Parallel()

	enc := newTestEncoder(t)

	// sqids Decode returns empty only when the code contains characters
	// outside the alphabet — any valid alphabet string can be decoded.
	cases := []struct {
		name string
		code string
	}{
		{"empty", ""},
		{"invalid_chars", "!!@@##"},
		{"hyphens_not_in_alphabet", "my-link"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := enc.Decode(tc.code)
			if err == nil {
				t.Errorf("Decode(%q) must return an error for invalid code", tc.code)
			}

			if !errors.Is(err, encoder.ErrInvalidCode) {
				t.Errorf("Decode(%q) error = %v, want wrapping ErrInvalidCode", tc.code, err)
			}
		})
	}
}

func FuzzSqidsEncoder_Encode(f *testing.F) {
	enc, err := encoder.NewSqidsEncoder(6)
	if err != nil {
		f.Fatalf("NewSqidsEncoder: %v", err)
	}

	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(1_000_000))
	f.Add(int64(math.MaxInt64))
	f.Add(int64(-1))

	f.Fuzz(func(t *testing.T, id int64) {
		code, err := enc.Encode(id)
		if err != nil {
			return
		}

		if code == "" {
			t.Errorf("Encode(%d) returned empty code without error", id)
		}

		got, err := enc.Decode(code)
		if err != nil {
			t.Errorf("Decode(%q) failed after Encode(%d): %v", code, id, err)
		}

		if got != id {
			t.Errorf("roundtrip: Encode(%d) -> %q -> Decode -> %d", id, code, got)
		}
	})
}
