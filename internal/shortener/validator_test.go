package shortener_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

// urlMaxLen must stay in sync with the maxURLLength constant in validator.go.
const urlMaxLen = 2048

func TestValidateURL(t *testing.T) {
	t.Parallel()

	// Build boundary cases once so table entries stay readable.
	validAtLimit := "https://example.com/" + strings.Repeat("a", urlMaxLen-len("https://example.com/"))
	tooLong := validAtLimit + "x"

	cases := []struct {
		name    string
		rawURL  string
		wantErr error
	}{
		// --- valid ---
		{name: "https_bare", rawURL: "https://example.com", wantErr: nil},
		{name: "http_bare", rawURL: "http://example.com", wantErr: nil},
		{name: "https_with_path", rawURL: "https://example.com/a/b/c", wantErr: nil},
		{name: "https_with_query", rawURL: "https://example.com/search?q=golang&lang=en", wantErr: nil},
		{name: "https_with_port", rawURL: "https://example.com:8443/path", wantErr: nil},
		{name: "https_with_fragment", rawURL: "https://example.com/page#section", wantErr: nil},
		{name: "public_ip", rawURL: "https://8.8.8.8/dns", wantErr: nil},
		{name: "exact_max_length", rawURL: validAtLimit, wantErr: nil},

		// --- empty / no-scheme ---
		{name: "empty", rawURL: "", wantErr: shortener.ErrInvalidURL},
		{name: "no_scheme", rawURL: "example.com/path", wantErr: shortener.ErrInvalidURL},
		{name: "relative_path", rawURL: "/just/a/path", wantErr: shortener.ErrInvalidURL},
		{name: "scheme_only", rawURL: "https://", wantErr: shortener.ErrInvalidURL},

		// --- disallowed schemes ---
		{name: "ftp", rawURL: "ftp://example.com/file.txt", wantErr: shortener.ErrInvalidURL},
		{name: "file", rawURL: "file:///etc/passwd", wantErr: shortener.ErrInvalidURL},
		{name: "javascript", rawURL: "javascript:alert(1)", wantErr: shortener.ErrInvalidURL},
		{name: "data", rawURL: "data:text/html,<h1>hi</h1>", wantErr: shortener.ErrInvalidURL},

		// --- too long ---
		{name: "too_long", rawURL: tooLong, wantErr: shortener.ErrInvalidURL},

		// --- private / reserved hosts ---
		{name: "localhost", rawURL: "http://localhost/path", wantErr: shortener.ErrInvalidURL},
		{name: "localhost_upper", rawURL: "http://LOCALHOST/path", wantErr: shortener.ErrInvalidURL},
		{name: "loopback_127_0_0_1", rawURL: "http://127.0.0.1/path", wantErr: shortener.ErrInvalidURL},
		{name: "loopback_127_x_y_z", rawURL: "http://127.100.200.1/path", wantErr: shortener.ErrInvalidURL},
		{name: "private_10_x", rawURL: "http://10.0.0.1/api", wantErr: shortener.ErrInvalidURL},
		{name: "private_172_16", rawURL: "http://172.16.0.1/api", wantErr: shortener.ErrInvalidURL},
		{name: "private_172_31", rawURL: "http://172.31.255.254/api", wantErr: shortener.ErrInvalidURL},
		{name: "private_192_168", rawURL: "http://192.168.1.1/api", wantErr: shortener.ErrInvalidURL},
		{name: "link_local_169_254", rawURL: "http://169.254.1.1/api", wantErr: shortener.ErrInvalidURL},
		{name: "loopback_ipv6", rawURL: "http://[::1]/path", wantErr: shortener.ErrInvalidURL},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := shortener.ValidateURL(tc.rawURL)

			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateURL(%q) = %v, want nil", tc.rawURL, err)
				}
				return
			}

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("ValidateURL(%q) = %v, want wrapping %v", tc.rawURL, err, tc.wantErr)
			}
		})
	}
}

func TestValidateAlias(t *testing.T) {
	t.Parallel()

	tooLong := strings.Repeat("a", 31)

	cases := []struct {
		name    string
		alias   string
		wantErr error
	}{
		// --- valid ---
		{name: "min_length_3", alias: "abc", wantErr: nil},
		{name: "alphanumeric", alias: "a1b2c3", wantErr: nil},
		{name: "with_interior_hyphen", alias: "my-link", wantErr: nil},
		{name: "multiple_hyphens", alias: "hello-world-123", wantErr: nil},
		{name: "max_length_30", alias: strings.Repeat("a", 30), wantErr: nil},
		{name: "uppercase_letters", alias: "MyLink", wantErr: nil},
		{name: "mixed_case_hyphen", alias: "Go-Short", wantErr: nil},

		// --- too short ---
		{name: "length_1", alias: "a", wantErr: shortener.ErrInvalidAlias},
		{name: "length_2", alias: "ab", wantErr: shortener.ErrInvalidAlias},

		// --- too long ---
		{name: "length_31", alias: tooLong, wantErr: shortener.ErrInvalidAlias},

		// --- invalid characters / format ---
		{name: "starts_with_hyphen", alias: "-abc", wantErr: shortener.ErrInvalidAlias},
		{name: "ends_with_hyphen", alias: "abc-", wantErr: shortener.ErrInvalidAlias},
		{name: "contains_space", alias: "my link", wantErr: shortener.ErrInvalidAlias},
		{name: "contains_underscore", alias: "my_link", wantErr: shortener.ErrInvalidAlias},
		{name: "contains_dot", alias: "my.link", wantErr: shortener.ErrInvalidAlias},
		{name: "contains_slash", alias: "my/link", wantErr: shortener.ErrInvalidAlias},

		// --- reserved paths → ErrReservedPath, not ErrInvalidAlias ---
		{name: "reserved_api", alias: "api", wantErr: shortener.ErrReservedPath},
		{name: "reserved_health", alias: "health", wantErr: shortener.ErrReservedPath},
		{name: "reserved_metrics", alias: "metrics", wantErr: shortener.ErrReservedPath},
		{name: "reserved_docs", alias: "docs", wantErr: shortener.ErrReservedPath},
		{name: "reserved_api_upper", alias: "API", wantErr: shortener.ErrReservedPath},
		{name: "reserved_health_mixed", alias: "Health", wantErr: shortener.ErrReservedPath},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := shortener.ValidateAlias(tc.alias)

			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateAlias(%q) = %v, want nil", tc.alias, err)
				}
				return
			}

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("ValidateAlias(%q) = %v, want wrapping %v", tc.alias, err, tc.wantErr)
			}
		})
	}
}

func TestValidateExpiresIn(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		expiresIn string
		wantErr   error
	}{
		// --- valid ---
		{name: "empty_no_expiry", expiresIn: "", wantErr: nil},
		{name: "one_hour_min", expiresIn: "1h", wantErr: nil},
		{name: "24h", expiresIn: "24h", wantErr: nil},
		{name: "one_day", expiresIn: "1d", wantErr: nil},
		{name: "seven_days", expiresIn: "7d", wantErr: nil},
		{name: "thirty_days", expiresIn: "30d", wantErr: nil},
		{name: "max_days_365", expiresIn: "365d", wantErr: nil},
		{name: "max_hours_8760", expiresIn: "8760h", wantErr: nil},

		// --- invalid: zero or negative ---
		{name: "zero_hours", expiresIn: "0h", wantErr: shortener.ErrInvalidExpires},
		{name: "zero_days", expiresIn: "0d", wantErr: shortener.ErrInvalidExpires},
		{name: "negative_hours", expiresIn: "-1h", wantErr: shortener.ErrInvalidExpires},

		// --- invalid: exceeds maximum ---
		{name: "366_days", expiresIn: "366d", wantErr: shortener.ErrInvalidExpires},
		{name: "8761_hours", expiresIn: "8761h", wantErr: shortener.ErrInvalidExpires},

		// --- invalid: unsupported unit ---
		{name: "minutes_unit", expiresIn: "60m", wantErr: shortener.ErrInvalidExpires},
		{name: "weeks_unit", expiresIn: "1w", wantErr: shortener.ErrInvalidExpires},
		{name: "seconds_unit", expiresIn: "3600s", wantErr: shortener.ErrInvalidExpires},

		// --- invalid: bad format ---
		{name: "no_unit", expiresIn: "30", wantErr: shortener.ErrInvalidExpires},
		{name: "unit_only_h", expiresIn: "h", wantErr: shortener.ErrInvalidExpires},
		{name: "unit_only_d", expiresIn: "d", wantErr: shortener.ErrInvalidExpires},
		{name: "non_numeric", expiresIn: "abcd", wantErr: shortener.ErrInvalidExpires},
		{name: "non_numeric_with_unit", expiresIn: "abch", wantErr: shortener.ErrInvalidExpires},
		{name: "fractional", expiresIn: "1.5h", wantErr: shortener.ErrInvalidExpires},
		{name: "overflow_integer", expiresIn: "99999999999999999999d", wantErr: shortener.ErrInvalidExpires},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := shortener.ValidateExpiresIn(tc.expiresIn)

			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateExpiresIn(%q) = %v, want nil", tc.expiresIn, err)
				}
				return
			}

			if !errors.Is(err, tc.wantErr) {
				t.Errorf("ValidateExpiresIn(%q) = %v, want wrapping %v", tc.expiresIn, err, tc.wantErr)
			}
		})
	}
}

func FuzzValidateURL(f *testing.F) {
	// Seed corpus covers key categories: valid, invalid, edge cases.
	f.Add("https://example.com")
	f.Add("http://example.com/path?q=1")
	f.Add("")
	f.Add("ftp://example.com")
	f.Add("javascript:alert(1)")
	f.Add("http://127.0.0.1/path")
	f.Add("http://192.168.1.1/path")
	f.Add("http://[::1]/path")
	f.Add("http://localhost/path")
	f.Add(strings.Repeat("a", urlMaxLen+100))

	f.Fuzz(func(_ *testing.T, rawURL string) {
		// Must never panic regardless of input.
		_ = shortener.ValidateURL(rawURL)
	})
}
