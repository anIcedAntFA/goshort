package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/anIcedAntFA/goshort/internal/shortener"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type shortenInput struct {
	URL       string `json:"url"                  jsonschema:"required,The URL to shorten (http/https, max 2048 chars)"`
	Alias     string `json:"alias,omitempty"      jsonschema:"Custom alias (3-30 chars, letters/digits/hyphens)"`
	ExpiresIn string `json:"expires_in,omitempty" jsonschema:"Expiration duration e.g. 1h 7d 30d 365d"`
}

type shortenOutput struct {
	ShortCode   string  `json:"short_code"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func (s *Server) handleShortenURL(
	ctx context.Context, _ *sdkmcp.CallToolRequest, in shortenInput,
) (*sdkmcp.CallToolResult, shortenOutput, error) {
	url, err := s.svc.Create(ctx, shortener.CreateRequest{
		URL:         in.URL,
		CustomAlias: in.Alias,
		ExpiresIn:   in.ExpiresIn,
	})
	if err != nil {
		return nil, shortenOutput{}, err
	}
	out := shortenOutput{
		ShortCode:   url.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, url.ShortCode),
		OriginalURL: url.OriginalURL,
		CreatedAt:   url.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if url.ExpiresAt != nil {
		t := url.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")
		out.ExpiresAt = &t
	}
	return nil, out, nil
}

type listInput struct {
	Page    int `json:"page,omitempty"     jsonschema:"Page number (default 1)"`
	PerPage int `json:"per_page,omitempty" jsonschema:"Items per page (default 20, max 100)"`
}

type listURLItem struct {
	ShortCode   string  `json:"short_code"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	IsCustom    bool    `json:"is_custom"`
	ClickCount  int64   `json:"click_count"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

type listPagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type listOutput struct {
	URLs       []listURLItem  `json:"urls"`
	Pagination listPagination `json:"pagination"`
}

func (s *Server) handleListURLs(
	ctx context.Context, _ *sdkmcp.CallToolRequest, in listInput,
) (*sdkmcp.CallToolResult, listOutput, error) {
	page := in.Page
	if page < 1 {
		page = 1
	}
	perPage := in.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	urls, total, err := s.svc.List(ctx, shortener.ListOptions{Page: page, PerPage: perPage})
	if err != nil {
		return nil, listOutput{}, err
	}

	items := make([]listURLItem, len(urls))
	for i, u := range urls {
		item := listURLItem{
			ShortCode:   u.ShortCode,
			ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, u.ShortCode),
			OriginalURL: u.OriginalURL,
			IsCustom:    u.IsCustom,
			ClickCount:  u.ClickCount,
			CreatedAt:   u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if u.ExpiresAt != nil {
			t := u.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")
			item.ExpiresAt = &t
		}
		items[i] = item
	}

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	return nil, listOutput{
		URLs: items,
		Pagination: listPagination{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

type codeInput struct {
	Code string `json:"code" jsonschema:"required,Short code or custom alias"`
}

type statsOutput struct {
	ShortCode   string  `json:"short_code"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	IsCustom    bool    `json:"is_custom"`
	ClickCount  int64   `json:"click_count"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
}

func (s *Server) handleGetURLStats(
	ctx context.Context, _ *sdkmcp.CallToolRequest, in codeInput,
) (*sdkmcp.CallToolResult, statsOutput, error) {
	url, err := s.svc.GetByCode(ctx, in.Code)
	if err != nil {
		return nil, statsOutput{}, err
	}
	out := statsOutput{
		ShortCode:   url.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, url.ShortCode),
		OriginalURL: url.OriginalURL,
		IsCustom:    url.IsCustom,
		ClickCount:  url.ClickCount,
		CreatedAt:   url.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if url.ExpiresAt != nil {
		t := url.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")
		out.ExpiresAt = &t
	}
	return nil, out, nil
}

type deleteOutput struct {
	Message string `json:"message"`
}

func (s *Server) handleDeleteURL(
	ctx context.Context, _ *sdkmcp.CallToolRequest, in codeInput,
) (*sdkmcp.CallToolResult, deleteOutput, error) {
	if err := s.svc.Delete(ctx, in.Code); err != nil {
		return nil, deleteOutput{}, err
	}
	return nil, deleteOutput{Message: fmt.Sprintf("Deleted short URL: %s", in.Code)}, nil
}

type lookupOutput struct {
	OriginalURL string `json:"original_url"`
}

func (s *Server) handleLookupURL(
	ctx context.Context, _ *sdkmcp.CallToolRequest, in codeInput,
) (*sdkmcp.CallToolResult, lookupOutput, error) {
	url, err := s.svc.GetByCode(ctx, in.Code)
	if err != nil {
		return nil, lookupOutput{}, err
	}
	return nil, lookupOutput{OriginalURL: url.OriginalURL}, nil
}

type batchShortenInput struct {
	URLs []shortenInput `json:"urls" jsonschema:"required,Array of URLs to shorten (max 50)"`
}

type batchResultItem struct {
	ShortCode   string          `json:"short_code,omitempty"`
	ShortURL    string          `json:"short_url,omitempty"`
	OriginalURL string          `json:"original_url,omitempty"`
	ExpiresAt   *string         `json:"expires_at,omitempty"`
	CreatedAt   string          `json:"created_at,omitempty"`
	Error       *batchItemError `json:"error,omitempty"`
}

type batchItemError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type batchSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

type batchShortenOutput struct {
	Results []batchResultItem `json:"results"`
	Summary batchSummary      `json:"summary"`
}

func (s *Server) handleBatchShortenURLs(
	ctx context.Context, _ *sdkmcp.CallToolRequest, in batchShortenInput,
) (*sdkmcp.CallToolResult, batchShortenOutput, error) {
	reqs := make([]shortener.CreateRequest, len(in.URLs))
	for i, u := range in.URLs {
		reqs[i] = shortener.CreateRequest{
			URL:         u.URL,
			CustomAlias: u.Alias,
			ExpiresIn:   u.ExpiresIn,
		}
	}

	batchResults, err := s.svc.CreateBatch(ctx, reqs)
	if err != nil {
		return nil, batchShortenOutput{}, err
	}

	out := batchShortenOutput{
		Results: make([]batchResultItem, len(batchResults)),
		Summary: batchSummary{Total: len(batchResults)},
	}

	for i, r := range batchResults {
		if r.Error != nil {
			out.Results[i] = batchResultItem{Error: toBatchItemError(r.Error)}
			out.Summary.Failed++
		} else {
			item := batchResultItem{
				ShortCode:   r.URL.ShortCode,
				ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, r.URL.ShortCode),
				OriginalURL: r.URL.OriginalURL,
				CreatedAt:   r.URL.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			}
			if r.URL.ExpiresAt != nil {
				t := r.URL.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z")
				item.ExpiresAt = &t
			}
			out.Results[i] = item
			out.Summary.Success++
		}
	}

	return nil, out, nil
}

func toBatchItemError(err error) *batchItemError {
	switch {
	case errors.Is(err, shortener.ErrInvalidURL):
		return &batchItemError{Code: "invalid_url", Message: "The URL format is invalid"}
	case errors.Is(err, shortener.ErrAliasTaken):
		return &batchItemError{Code: "alias_taken", Message: "The requested alias is already in use"}
	case errors.Is(err, shortener.ErrReservedPath):
		return &batchItemError{Code: "reserved_path", Message: "The alias is a reserved path"}
	case errors.Is(err, shortener.ErrInvalidAlias):
		return &batchItemError{Code: "invalid_alias", Message: "The alias format is invalid"}
	case errors.Is(err, shortener.ErrInvalidExpires):
		return &batchItemError{Code: "invalid_expires", Message: "The expires_in duration is invalid"}
	default:
		return &batchItemError{Code: "internal_error", Message: "An internal error occurred"}
	}
}
