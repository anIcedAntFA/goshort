package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIClient makes HTTP requests to the GoShort server.
type APIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAPIClient creates an APIClient targeting baseURL, authenticating with apiKey.
func NewAPIClient(baseURL, key string) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		apiKey:  key,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Request/response types — defined here so cmd/cli has no internal/ imports.

type CreateRequest struct {
	URL         string `json:"url"`
	CustomAlias string `json:"custom_alias,omitempty"`
	ExpiresIn   string `json:"expires_in,omitempty"`
}

type CreateResponse struct {
	ShortCode   string  `json:"short_code"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	ExpiresAt   *string `json:"expires_at"`
	CreatedAt   string  `json:"created_at"`
}

type URLDetail struct {
	ShortCode   string  `json:"short_code"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	IsCustom    bool    `json:"is_custom"`
	ExpiresAt   *string `json:"expires_at"`
	CreatedAt   string  `json:"created_at"`
	ClickCount  int64   `json:"click_count"`
}

type ListResponse struct {
	Data       []URLDetail    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// CreateURL creates a new short URL.
func (c *APIClient) CreateURL(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	resp, err := c.do(ctx, http.MethodPost, "/api/v1/urls", req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}
	var out CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// ListURLs returns a paginated list of short URLs.
func (c *APIClient) ListURLs(ctx context.Context, page, perPage int) (*ListResponse, error) {
	path := fmt.Sprintf("/api/v1/urls?page=%d&per_page=%d", page, perPage)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}
	var out ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// GetURL returns details for a single short URL.
func (c *APIClient) GetURL(ctx context.Context, code string) (*URLDetail, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/urls/"+code, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, err
	}
	var out URLDetail
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}

// DeleteURL deletes a short URL by code.
func (c *APIClient) DeleteURL(ctx context.Context, code string) error {
	resp, err := c.do(ctx, http.MethodDelete, "/api/v1/urls/"+code, nil)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	return c.checkResponse(resp)
}

// do executes an HTTP request. body is marshaled to JSON when non-nil.
func (c *APIClient) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &buf)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

// checkResponse returns an error for non-2xx responses, parsing the API error body.
func (c *APIClient) checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	var ae apiError
	if err := json.NewDecoder(resp.Body).Decode(&ae); err != nil {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return fmt.Errorf("%s: %s", ae.Error.Code, ae.Error.Message)
}
