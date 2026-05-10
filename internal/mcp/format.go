package mcp

import "time"

func formatUTC(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func formatUTCPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := formatUTC(*t)
	return &s
}
