package storage_test

import (
	"context"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/storage"
)

func newBenchStorage(b *testing.B) *storage.SQLiteStorage {
	b.Helper()
	s, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
	if err != nil {
		b.Fatalf("NewSQLiteStorage: %v", err)
	}
	b.Cleanup(func() { _ = s.Close() })
	return s
}

func BenchmarkSQLiteStorage_GetByCode(b *testing.B) {
	s := newBenchStorage(b)
	ctx := context.Background()

	if _, err := s.CreateURL(ctx, sampleParams("bench-code", "https://example.com")); err != nil {
		b.Fatalf("CreateURL: %v", err)
	}
	b.ResetTimer()
	for range b.N {
		_, _ = s.GetByCode(ctx, "bench-code")
	}
}

func BenchmarkSQLiteStorage_CreateURL(b *testing.B) {
	s := newBenchStorage(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := range b.N {
		code := "bench-create-" + string(rune('a'+i%26))
		_, _ = s.CreateURL(ctx, sampleParams(code, "https://example.com"))
	}
}
