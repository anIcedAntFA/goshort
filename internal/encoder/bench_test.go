package encoder_test

import (
	"testing"

	"github.com/anIcedAntFA/goshort/internal/encoder"
)

func BenchmarkSqidsEncoder_Encode(b *testing.B) {
	enc, err := encoder.NewSqidsEncoder(6)
	if err != nil {
		b.Fatalf("NewSqidsEncoder: %v", err)
	}
	b.ResetTimer()
	for i := range b.N {
		_, _ = enc.Encode(int64(i))
	}
}

func BenchmarkSqidsEncoder_Decode(b *testing.B) {
	enc, err := encoder.NewSqidsEncoder(6)
	if err != nil {
		b.Fatalf("NewSqidsEncoder: %v", err)
	}
	code, err := enc.Encode(42)
	if err != nil {
		b.Fatalf("Encode seed: %v", err)
	}
	b.ResetTimer()
	for range b.N {
		_, _ = enc.Decode(code)
	}
}
