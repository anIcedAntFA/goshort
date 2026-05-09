package shortener

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	previewTimeout = 3 * time.Second
	previewMaxBody = 512 * 1024 // 512 KB
)

// PreviewFetcher fetches Open Graph / HTML metadata for a URL.
// On any error the implementation returns empty strings so URL creation
// always succeeds regardless of the target page's availability.
type PreviewFetcher interface {
	Fetch(ctx context.Context, rawURL string) (title, description string, err error)
}

// NoopPreviewFetcher always returns empty metadata. Used in tests and when
// preview fetching is disabled.
type NoopPreviewFetcher struct{}

// Fetch implements PreviewFetcher and always returns empty strings.
func (NoopPreviewFetcher) Fetch(_ context.Context, _ string) (title, description string, err error) {
	return "", "", nil
}

// HTTPPreviewFetcher fetches real pages and extracts <title> and
// <meta name="description"> using the x/net/html tokenizer.
type HTTPPreviewFetcher struct {
	client      *http.Client
	skipPrivate bool // set to true in tests to allow loopback addresses
}

// NewHTTPPreviewFetcher returns an HTTPPreviewFetcher with sensible defaults.
func NewHTTPPreviewFetcher() *HTTPPreviewFetcher {
	return &HTTPPreviewFetcher{
		client: &http.Client{Timeout: previewTimeout},
	}
}

// Fetch GETs rawURL, reads up to 512 KB, and parses HTML metadata.
// Private/loopback hosts are silently skipped.
// All errors are swallowed — callers receive empty strings on failure.
func (f *HTTPPreviewFetcher) Fetch(ctx context.Context, rawURL string) (title, description string, err error) {
	u, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", "", nil
	}

	if !f.skipPrivate && isPrivateHost(u.Host) {
		return "", "", nil
	}

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
	name    string
	content string
}

// parseMetaAttrs reads all attributes from the current tokenizer position
// and returns the meta name/content pair.
func parseMetaAttrs(z *html.Tokenizer) htmlMeta {
	var m htmlMeta
	for {
		k, v, more := z.TagAttr()
		switch strings.ToLower(string(k)) {
		case "name":
			m.name = strings.ToLower(string(v))
		case "content":
			m.content = string(v)
		}
		if !more {
			break
		}
	}
	return m
}

// parseHTMLMetadata extracts the first <title> text and
// <meta name="description" content="..."> from an HTML stream.
func parseHTMLMetadata(r io.Reader) (title, description string) {
	z := html.NewTokenizer(r)
	inTitle := false

	for {
		switch z.Next() {
		case html.ErrorToken:
			return

		case html.StartTagToken:
			if done := handleHTMLStartTag(z, &inTitle, &description); done {
				return
			}

		case html.SelfClosingTagToken:
			handleHTMLSelfClose(z, &description)

		case html.EndTagToken:
			name, _ := z.TagName()
			if string(name) == "title" {
				inTitle = false
			}

		case html.TextToken:
			if inTitle && title == "" {
				title = strings.TrimSpace(string(z.Text()))
			}
		}
	}
}

func handleHTMLStartTag(z *html.Tokenizer, inTitle *bool, description *string) (stop bool) {
	name, hasAttr := z.TagName()
	switch string(name) {
	case "title":
		*inTitle = true
	case "meta":
		if hasAttr {
			m := parseMetaAttrs(z)
			if m.name == "description" && *description == "" {
				*description = strings.TrimSpace(m.content)
			}
		}
	case "body":
		return true
	}
	return false
}

func handleHTMLSelfClose(z *html.Tokenizer, description *string) {
	name, hasAttr := z.TagName()
	if string(name) != "meta" || !hasAttr {
		return
	}
	m := parseMetaAttrs(z)
	if m.name == "description" && *description == "" {
		*description = strings.TrimSpace(m.content)
	}
}
