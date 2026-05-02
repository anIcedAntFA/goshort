package shortener_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

func TestErrWrappedIsStillDetected(t *testing.T) {
	t.Parallel()

	wrapped := fmt.Errorf("get url: %w", shortener.ErrNotFound)
	if !errors.Is(wrapped, shortener.ErrNotFound) {
		t.Fatal("wrapped ErrNotFound must be detectable via errors.Is")
	}
}
