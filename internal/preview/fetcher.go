// Package preview fetches HTML metadata (title, description) for URLs.
package preview

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
	"golang.org/x/net/html"
)

const (
	previewTimeout = 3 * time.Second
	previewMaxBody = 512 * 1024 // 512 KB
)

var _ shortener.PreviewFetcher = (*HTTPFetcher)(nil)

// HTTPFetcher fetches real pages and extracts title and description metadata
// using the x/net/html tokenizer. Parses og:title/og:description with HTML fallback.
type HTTPFetcher struct {
	client *http.Client
}

// NewHTTPFetcher returns an HTTPFetcher with a safe dialer that rejects private IPs.
func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{
		client: &http.Client{
			Timeout:   previewTimeout,
			Transport: &http.Transport{DialContext: safeDialer()},
		},
	}
}

// NewHTTPFetcherForTest returns an HTTPFetcher using http.DefaultTransport,
// allowing loopback httptest servers in tests.
func NewHTTPFetcherForTest(client *http.Client) *HTTPFetcher {
	return &HTTPFetcher{client: client}
}

// safeDialer returns a custom dialer that resolves DNS and rejects connections
// to private/loopback addresses, preventing SSRF via DNS rebinding.
func safeDialer() func(ctx context.Context, network, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: previewTimeout}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("split host:port: %w", err)
		}

		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve %s: %w", host, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no IPs for %s", host)
		}

		for _, ip := range ips {
			if shortener.IsPrivateHost(ip.IP.String()) {
				return nil, fmt.Errorf("resolved IP %s is private", ip.IP)
			}
		}

		return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}

// Fetch GETs rawURL, reads up to 512 KB, and parses HTML metadata.
// All errors are swallowed — callers receive empty strings on failure.
func (f *HTTPFetcher) Fetch(ctx context.Context, rawURL string) (title, description string, err error) {
	fetchCtx, cancel := context.WithTimeout(ctx, previewTimeout)
	defer cancel()

	req, reqErr := http.NewRequestWithContext(fetchCtx, http.MethodGet, rawURL, http.NoBody)
	if reqErr != nil {
		return "", "", nil
	}
	req.Header.Set("User-Agent", "goshort-preview/1.0")

	resp, doErr := f.client.Do(req)
	if doErr != nil {
		return "", "", nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", nil
	}

	body := io.LimitReader(resp.Body, previewMaxBody)
	title, description = parseHTMLMetadata(body)
	return title, description, nil
}

// htmlMeta holds parsed values from a <meta> tag.
type htmlMeta struct {
	name     string // <meta name="...">
	property string // <meta property="...">
	content  string
}

func parseMetaAttrs(z *html.Tokenizer) htmlMeta {
	var m htmlMeta
	for {
		k, v, more := z.TagAttr()
		switch strings.ToLower(string(k)) {
		case "name":
			m.name = strings.ToLower(string(v))
		case "property":
			m.property = strings.ToLower(string(v))
		case "content":
			m.content = string(v)
		}
		if !more {
			break
		}
	}
	return m
}

func applyMeta(m htmlMeta, htmlDesc, ogTitle, ogDesc *string) {
	if m.property == "og:title" && *ogTitle == "" {
		*ogTitle = strings.TrimSpace(m.content)
	}
	if m.property == "og:description" && *ogDesc == "" {
		*ogDesc = strings.TrimSpace(m.content)
	}
	if m.name == "description" && *htmlDesc == "" {
		*htmlDesc = strings.TrimSpace(m.content)
	}
}

//nolint:gocognit // HTML token loop requires several branches by nature
func parseHTMLMetadata(r io.Reader) (title, description string) {
	z := html.NewTokenizer(r)
	inTitle := false
	var htmlTitle, ogTitle, htmlDesc, ogDesc string

	for {
		switch z.Next() {
		case html.ErrorToken:
			goto done

		case html.StartTagToken:
			name, hasAttr := z.TagName()
			switch string(name) {
			case "title":
				inTitle = true
			case "meta":
				if hasAttr {
					applyMeta(parseMetaAttrs(z), &htmlDesc, &ogTitle, &ogDesc)
				}
			case "body":
				goto done
			}

		case html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			if string(name) == "meta" && hasAttr {
				applyMeta(parseMetaAttrs(z), &htmlDesc, &ogTitle, &ogDesc)
			}

		case html.EndTagToken:
			name, _ := z.TagName()
			if string(name) == "title" {
				inTitle = false
			}

		case html.TextToken:
			if inTitle && htmlTitle == "" {
				htmlTitle = strings.TrimSpace(string(z.Text()))
			}
		}
	}

done:
	title = ogTitle
	if title == "" {
		title = htmlTitle
	}
	description = ogDesc
	if description == "" {
		description = htmlDesc
	}
	return title, description
}
