package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anIcedAntFA/goshort/internal/shortener"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	qrcode "github.com/skip2/go-qrcode"
)

func (s *Server) handleStatsSummary(
	ctx context.Context, req *sdkmcp.ReadResourceRequest,
) (*sdkmcp.ReadResourceResult, error) {
	urls, total, err := s.svc.List(ctx, shortener.ListOptions{Page: 1, PerPage: 5})
	if err != nil {
		return nil, fmt.Errorf("list urls: %w", err)
	}

	type topURL struct {
		ShortCode   string `json:"short_code"`
		OriginalURL string `json:"original_url"`
		ClickCount  int64  `json:"click_count"`
	}
	top := make([]topURL, len(urls))
	for i, u := range urls {
		top[i] = topURL{
			ShortCode:   u.ShortCode,
			OriginalURL: u.OriginalURL,
			ClickCount:  u.ClickCount,
		}
	}

	summary := struct {
		TotalURLs int      `json:"total_urls"`
		TopURLs   []topURL `json:"top_urls"`
	}{
		TotalURLs: total,
		TopURLs:   top,
	}

	b, err := json.Marshal(summary)
	if err != nil {
		return nil, fmt.Errorf("marshal summary: %w", err)
	}

	return &sdkmcp.ReadResourceResult{
		Contents: []*sdkmcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(b),
		}},
	}, nil
}

func (s *Server) handleURLByCode(
	ctx context.Context, req *sdkmcp.ReadResourceRequest,
) (*sdkmcp.ReadResourceResult, error) {
	code := extractCodeFromURI(req.Params.URI)
	if code == "" {
		return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
	}

	url, err := s.svc.GetByCode(ctx, code)
	if err != nil {
		return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
	}

	type urlDetail struct {
		ShortCode   string  `json:"short_code"`
		ShortURL    string  `json:"short_url"`
		OriginalURL string  `json:"original_url"`
		IsCustom    bool    `json:"is_custom"`
		ClickCount  int64   `json:"click_count"`
		CreatedAt   string  `json:"created_at"`
		ExpiresAt   *string `json:"expires_at,omitempty"`
	}

	detail := urlDetail{
		ShortCode:   url.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, url.ShortCode),
		OriginalURL: url.OriginalURL,
		IsCustom:    url.IsCustom,
		ClickCount:  url.ClickCount,
		CreatedAt:   formatUTC(url.CreatedAt),
		ExpiresAt:   formatUTCPtr(url.ExpiresAt),
	}

	b, err := json.Marshal(detail)
	if err != nil {
		return nil, fmt.Errorf("marshal url detail: %w", err)
	}

	return &sdkmcp.ReadResourceResult{
		Contents: []*sdkmcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(b),
		}},
	}, nil
}

// extractCodeFromURI parses the short code from "goshort://urls/{code}".
func extractCodeFromURI(uri string) string {
	const prefix = "goshort://urls/"
	if len(uri) <= len(prefix) {
		return ""
	}
	return uri[len(prefix):]
}

func (s *Server) handleQRCode(
	ctx context.Context, req *sdkmcp.ReadResourceRequest,
) (*sdkmcp.ReadResourceResult, error) {
	code := extractCodeFromQRURI(req.Params.URI)
	if code == "" {
		return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
	}

	if _, err := s.svc.GetByCode(ctx, code); err != nil {
		return nil, sdkmcp.ResourceNotFoundError(req.Params.URI)
	}

	shortURL := fmt.Sprintf("%s/%s", s.baseURL, code)
	png, err := qrcode.Encode(shortURL, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("generate qr code: %w", err)
	}

	return &sdkmcp.ReadResourceResult{
		Contents: []*sdkmcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "image/png",
			Blob:     png,
		}},
	}, nil
}

// extractCodeFromQRURI parses the short code from "goshort://urls/{code}/qr".
func extractCodeFromQRURI(uri string) string {
	const prefix = "goshort://urls/"
	const suffix = "/qr"
	if !strings.HasPrefix(uri, prefix) || !strings.HasSuffix(uri, suffix) {
		return ""
	}
	start, end := len(prefix), len(uri)-len(suffix)
	if start >= end {
		return ""
	}
	return uri[start:end]
}
