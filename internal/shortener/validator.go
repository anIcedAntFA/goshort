package shortener

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	maxURLLength    = 2048
	minAliasLength  = 3
	maxAliasLength  = 30
	minExpiresHours = 1        // 1 hour minimum
	maxExpiresHours = 365 * 24 // 365 days maximum
)

// aliasRe allows alphanumeric characters and interior hyphens only.
// First and last character must be alphanumeric.
var aliasRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// reservedPaths are the lowercase names that conflict with built-in routes.
var reservedPaths = map[string]struct{}{
	"api":     {},
	"health":  {},
	"metrics": {},
	"docs":    {},
}

// privateNets covers RFC-1918 private ranges, loopback, and link-local blocks.
var privateNets = func() []*net.IPNet {
	cidrs := []string{
		"127.0.0.0/8",    // IPv4 loopback (RFC 1122)
		"10.0.0.0/8",     // RFC 1918 class A private
		"172.16.0.0/12",  // RFC 1918 class B private
		"192.168.0.0/16", // RFC 1918 class C private
		"169.254.0.0/16", // IPv4 link-local (RFC 3927)
		"::1/128",        // IPv6 loopback
		"fc00::/7",       // IPv6 unique local (RFC 4193)
		"fe80::/10",      // IPv6 link-local (RFC 4291)
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, block, _ := net.ParseCIDR(cidr)
		nets = append(nets, block)
	}
	return nets
}()

// ValidateURL checks that rawURL is an absolute HTTP(S) URL within the length
// limit and not targeting a private-network host.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("validate url: empty: %w", ErrInvalidURL)
	}
	if len(rawURL) > maxURLLength {
		return fmt.Errorf("validate url: exceeds %d characters: %w", maxURLLength, ErrInvalidURL)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("validate url: malformed: %w", ErrInvalidURL)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("validate url: scheme %q not allowed (use http or https): %w", u.Scheme, ErrInvalidURL)
	}

	if u.Host == "" {
		return fmt.Errorf("validate url: missing host: %w", ErrInvalidURL)
	}

	if isPrivateHost(u.Host) {
		return fmt.Errorf("validate url: host targets a private or reserved address: %w", ErrInvalidURL)
	}

	return nil
}

// isPrivateHost reports whether host (as returned by url.URL.Host) resolves to
// a private, loopback, or link-local address.
func isPrivateHost(host string) bool {
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}
	// Strip IPv6 brackets that SplitHostPort leaves when there is no port.
	hostname = strings.Trim(hostname, "[]")

	if strings.EqualFold(hostname, "localhost") {
		return true
	}

	ip := net.ParseIP(hostname)
	if ip == nil {
		return false
	}

	for _, block := range privateNets {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

// ValidateAlias checks that alias is 3–30 characters, uses only alphanumeric
// characters and interior hyphens, and does not collide with a reserved path.
func ValidateAlias(alias string) error {
	n := len(alias)
	if n < minAliasLength || n > maxAliasLength {
		return fmt.Errorf(
			"validate alias: length %d out of range [%d, %d]: %w",
			n, minAliasLength, maxAliasLength, ErrInvalidAlias,
		)
	}

	if !aliasRe.MatchString(alias) {
		return fmt.Errorf(
			"validate alias: %q must match ^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$: %w",
			alias, ErrInvalidAlias,
		)
	}

	if _, reserved := reservedPaths[strings.ToLower(alias)]; reserved {
		return fmt.Errorf("validate alias: %q is a reserved path: %w", alias, ErrReservedPath)
	}

	return nil
}

// ValidateExpiresIn checks that expiresIn is either empty (meaning no expiry)
// or a duration string of the form "<N>h" or "<N>d" within [1h, 365d].
func ValidateExpiresIn(expiresIn string) error {
	if expiresIn == "" {
		return nil
	}
	if len(expiresIn) < 2 {
		return fmt.Errorf("validate expires_in: %q too short to be a valid duration: %w", expiresIn, ErrInvalidExpires)
	}

	unit := expiresIn[len(expiresIn)-1]
	numStr := expiresIn[:len(expiresIn)-1]

	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil || n <= 0 {
		return fmt.Errorf("validate expires_in: %q: number must be a positive integer: %w", expiresIn, ErrInvalidExpires)
	}

	var hours int64
	switch unit {
	case 'h':
		hours = n
	case 'd':
		hours = n * 24
	default:
		return fmt.Errorf(
			"validate expires_in: %q: unsupported unit %q (use h for hours, d for days): %w",
			expiresIn, string(unit), ErrInvalidExpires,
		)
	}

	if hours < minExpiresHours {
		return fmt.Errorf(
			"validate expires_in: %q: must be at least %dh: %w",
			expiresIn, minExpiresHours, ErrInvalidExpires,
		)
	}
	if hours > maxExpiresHours {
		return fmt.Errorf(
			"validate expires_in: %q: must be at most %dd (%dh): %w",
			expiresIn, maxExpiresHours/24, maxExpiresHours, ErrInvalidExpires,
		)
	}

	return nil
}
