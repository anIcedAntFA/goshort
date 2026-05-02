package main

import "testing"

func TestFormatTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid RFC3339 UTC", "2025-07-04T14:30:00Z", "2025-07-04 14:30"},
		{"valid RFC3339 offset", "2025-07-04T14:30:00+07:00", "2025-07-04 07:30"},
		{"invalid no Z", "not-a-timestamp", "not-a-timestamp"},
		{"empty string", "", ""},
		{"invalid trailing Z", "bad-valueZ", "bad-value"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := formatTime(tc.input)
			if got != tc.want {
				t.Errorf("formatTime(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
