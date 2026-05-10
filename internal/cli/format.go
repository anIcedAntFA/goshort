package cli

import (
	"strings"
	"time"
)

// FormatTime parses an RFC3339 timestamp and returns a human-readable UTC string.
// Falls back to stripping the trailing "Z" if parsing fails.
func FormatTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return strings.TrimSuffix(s, "Z")
	}
	return t.UTC().Format("2006-01-02 15:04")
}
