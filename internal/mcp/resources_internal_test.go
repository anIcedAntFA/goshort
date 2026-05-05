package mcp

import "testing"

func TestExtractCodeFromURI(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		uri  string
		want string
	}{
		{"normal", "goshort://urls/abc123", "abc123"},
		{"custom alias with hyphen", "goshort://urls/my-link", "my-link"},
		{"empty code trailing slash", "goshort://urls/", ""},
		{"prefix only no slash", "goshort://urls", ""},
		{"multi-segment", "goshort://urls/a/b", "a/b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := extractCodeFromURI(tc.uri)
			if got != tc.want {
				t.Errorf("extractCodeFromURI(%q) = %q, want %q", tc.uri, got, tc.want)
			}
		})
	}
}
