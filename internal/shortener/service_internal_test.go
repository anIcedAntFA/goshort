package shortener

import (
	"errors"
	"testing"
	"time"
)

func TestParseExpiresIn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		wantDur time.Duration
		wantErr bool
	}{
		{"valid hours", "24h", 24 * time.Hour, false},
		{"valid days", "7d", 7 * 24 * time.Hour, false},
		{"non-numeric prefix", "xyzh", 0, true},
		{"zero value", "0d", 0, true},
		{"negative value", "-1h", 0, true},
		{"unsupported unit minutes", "10m", 0, true},
		{"unsupported unit weeks", "2w", 0, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseExpiresIn(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseExpiresIn(%q) err = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidExpires) {
					t.Errorf("parseExpiresIn(%q) error = %v, want wrapping ErrInvalidExpires", tc.input, err)
				}
				return
			}
			if got != tc.wantDur {
				t.Errorf("parseExpiresIn(%q) = %v, want %v", tc.input, got, tc.wantDur)
			}
		})
	}
}
