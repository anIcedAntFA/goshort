package shortener_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

func TestSentinelErrorsIdentity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", shortener.ErrNotFound},
		{"ErrExpired", shortener.ErrExpired},
		{"ErrAliasTaken", shortener.ErrAliasTaken},
		{"ErrReservedPath", shortener.ErrReservedPath},
		{"ErrInvalidURL", shortener.ErrInvalidURL},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if !errors.Is(tc.err, tc.err) {
				t.Errorf("%s must match itself via errors.Is", tc.name)
			}
		})
	}
}

func TestErrWrappedIsStillDetected(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("get url: %w", shortener.ErrNotFound)
	if !errors.Is(wrapped, shortener.ErrNotFound) {
		t.Fatal("wrapped ErrNotFound must be detectable via errors.Is")
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	t.Parallel()

	errs := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", shortener.ErrNotFound},
		{"ErrExpired", shortener.ErrExpired},
		{"ErrAliasTaken", shortener.ErrAliasTaken},
		{"ErrReservedPath", shortener.ErrReservedPath},
		{"ErrInvalidURL", shortener.ErrInvalidURL},
	}

	for i, a := range errs {
		for j, b := range errs {
			if i == j {
				continue
			}

			t.Run(a.name+"_vs_"+b.name, func(t *testing.T) {
				t.Parallel()

				if errors.Is(a.err, b.err) {
					t.Errorf("%s and %s must be distinct errors", a.name, b.name)
				}
			})
		}
	}
}
