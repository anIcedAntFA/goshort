package encoder

import (
	"testing"

	sqids "github.com/sqids/sqids-go"
)

func TestNewSqidsEncoder_InvalidAlphabetReturnsError(t *testing.T) {
	t.Parallel()

	// alphabet shorter than 3 chars causes sqids.New to return an error.
	_, err := newSqidsEncoder(sqids.Options{Alphabet: "ab"})
	if err == nil {
		t.Fatal("newSqidsEncoder must return an error for an invalid alphabet")
	}
}

func TestSqidsEncoder_Encode_AllPermutationsBlockedReturnsError(t *testing.T) {
	t.Parallel()

	// alphabet "abc" has 3 prefix positions → 3 possible codes for [0].
	// Blocklist covers all three ("cab", "abc", "bca"), so sqids exhausts every
	// attempt and returns its max-regeneration-attempts error.
	// Sourced from sqids-go's own TestMaxEncodingAttempts.
	s, err := sqids.New(sqids.Options{
		Alphabet:  "abc",
		MinLength: 3,
		Blocklist: []string{"cab", "abc", "bca"},
	})
	if err != nil {
		t.Fatalf("sqids.New: %v", err)
	}

	enc := &SqidsEncoder{s: s}

	_, err = enc.Encode(0)
	if err == nil {
		t.Fatal("Encode must return an error when all permutations are blocked")
	}
}
