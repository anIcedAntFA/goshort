package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/go-chi/chi/v5"
	qrcode "github.com/skip2/go-qrcode"
)

// Handler holds the HTTP handler dependencies.
type Handler struct {
	svc     shortener.Service
	cache   shortener.Cache
	logger  *slog.Logger
	baseURL string
}

// NewHandler creates a new Handler.
func NewHandler(svc shortener.Service, c shortener.Cache, logger *slog.Logger, baseURL string) *Handler {
	return &Handler{svc: svc, cache: c, logger: logger, baseURL: baseURL}
}

type createURLRequest struct {
	URL         string `json:"url"`
	CustomAlias string `json:"custom_alias"`
	ExpiresIn   string `json:"expires_in"`
}

type createURLResponse struct {
	ShortCode   string  `json:"short_code"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	ExpiresAt   *string `json:"expires_at"`
	CreatedAt   string  `json:"created_at"`
}

type urlResponse struct {
	ShortCode   string  `json:"short_code"`
	ShortURL    string  `json:"short_url"`
	OriginalURL string  `json:"original_url"`
	IsCustom    bool    `json:"is_custom"`
	ExpiresAt   *string `json:"expires_at"`
	CreatedAt   string  `json:"created_at"`
	ClickCount  int64   `json:"click_count"`
}

type listURLsResponse struct {
	Data       []urlResponse  `json:"data"`
	Pagination paginationMeta `json:"pagination"`
}

type paginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// CreateURL handles POST /api/v1/urls.
func (h *Handler) CreateURL(w http.ResponseWriter, r *http.Request) {
	var req createURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{Error: errorDetail{
			Code:    "invalid_body",
			Message: "Request body is invalid JSON",
		}})
		return
	}

	url, err := h.svc.Create(r.Context(), shortener.CreateRequest{
		URL:         req.URL,
		CustomAlias: req.CustomAlias,
		ExpiresIn:   req.ExpiresIn,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	urlType := "generated"
	if url.IsCustom {
		urlType = "custom"
	}
	urlsCreatedTotal.WithLabelValues(urlType).Inc()

	respondJSON(w, http.StatusCreated, createURLResponse{
		ShortCode:   url.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", h.baseURL, url.ShortCode),
		OriginalURL: url.OriginalURL,
		ExpiresAt:   formatTimePtr(url.ExpiresAt),
		CreatedAt:   url.CreatedAt.Format(time.RFC3339),
	})
}

// Redirect handles GET /{code} using cache-aside pattern.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	// Cache hit — skip DB entirely.
	if cached, ok := h.cache.Get(r.Context(), "short:"+code); ok {
		redirectsTotal.WithLabelValues("302").Inc()
		clickCtx := context.WithoutCancel(r.Context())
		go func() {
			if err := h.svc.IncrementClicks(clickCtx, code); err != nil {
				h.logger.Warn("increment clicks failed", "code", code, "error", err)
			}
		}()
		w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
		http.Redirect(w, r, cached, http.StatusFound) //nolint:gosec // URL validated on creation
		return
	}

	url, err := h.svc.GetByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, shortener.ErrExpired) {
			redirectsTotal.WithLabelValues("410").Inc()
		} else {
			redirectsTotal.WithLabelValues("404").Inc()
		}
		respondError(w, err)
		return
	}

	// Populate cache for subsequent requests.
	_ = h.cache.Set(r.Context(), "short:"+code, url.OriginalURL, cacheTTL(url))

	redirectsTotal.WithLabelValues("302").Inc()

	// Fire-and-forget: detach from request context so cancellation on response send doesn't abort the write.
	clickCtx := context.WithoutCancel(r.Context())
	go func() {
		if err := h.svc.IncrementClicks(clickCtx, code); err != nil {
			h.logger.Warn("increment clicks failed", "code", code, "error", err)
		}
	}()

	w.Header().Set("Cache-Control", "private, max-age=0, no-cache")
	http.Redirect(w, r, url.OriginalURL, http.StatusFound) //nolint:gosec // URL validated on creation
}

// ListURLs handles GET /api/v1/urls.
func (h *Handler) ListURLs(w http.ResponseWriter, r *http.Request) {
	page := parseIntQuery(r, "page", 1)
	perPage := parseIntQuery(r, "per_page", 20)

	urls, total, err := h.svc.List(r.Context(), shortener.ListOptions{
		Page:    page,
		PerPage: perPage,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	data := make([]urlResponse, len(urls))
	for i, u := range urls {
		data[i] = toURLResponse(&u, h.baseURL)
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	respondJSON(w, http.StatusOK, listURLsResponse{
		Data: data,
		Pagination: paginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// GetURL handles GET /api/v1/urls/{code}.
func (h *Handler) GetURL(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	url, err := h.svc.GetByCode(r.Context(), code)
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, toURLResponse(url, h.baseURL))
}

// DeleteURL handles DELETE /api/v1/urls/{code}.
func (h *Handler) DeleteURL(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	if err := h.svc.Delete(r.Context(), code); err != nil {
		respondError(w, err)
		return
	}

	_ = h.cache.Delete(r.Context(), "short:"+code)
	w.WriteHeader(http.StatusNoContent)
}

// HealthCheck handles GET /health.
func (h *Handler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type batchCreateURLRequest struct {
	URLs []createURLRequest `json:"urls"`
}

type batchItemResponse struct {
	ShortCode   string       `json:"short_code,omitempty"`
	ShortURL    string       `json:"short_url,omitempty"`
	OriginalURL string       `json:"original_url,omitempty"`
	ExpiresAt   *string      `json:"expires_at,omitempty"`
	CreatedAt   string       `json:"created_at,omitempty"`
	Error       *errorDetail `json:"error,omitempty"`
}

type batchSummary struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

type batchCreateURLResponse struct {
	Results []batchItemResponse `json:"results"`
	Summary batchSummary        `json:"summary"`
}

// BatchCreateURL handles POST /api/v1/urls/batch.
func (h *Handler) BatchCreateURL(w http.ResponseWriter, r *http.Request) {
	var req batchCreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, errorResponse{Error: errorDetail{
			Code:    "invalid_body",
			Message: "Request body is invalid JSON",
		}})
		return
	}

	svcReqs := make([]shortener.CreateRequest, len(req.URLs))
	for i, u := range req.URLs {
		svcReqs[i] = shortener.CreateRequest{
			URL:         u.URL,
			CustomAlias: u.CustomAlias,
			ExpiresIn:   u.ExpiresIn,
		}
	}

	batchResults, err := h.svc.CreateBatch(r.Context(), svcReqs)
	if err != nil {
		respondError(w, err)
		return
	}

	items := make([]batchItemResponse, len(batchResults))
	var succeeded int
	for i, br := range batchResults {
		if br.Error != nil {
			items[i] = batchItemResponse{Error: toErrorDetail(br.Error)}
		} else {
			succeeded++
			urlType := "generated"
			if br.URL.IsCustom {
				urlType = "custom"
			}
			urlsCreatedTotal.WithLabelValues(urlType).Inc()
			items[i] = batchItemResponse{
				ShortCode:   br.URL.ShortCode,
				ShortURL:    fmt.Sprintf("%s/%s", h.baseURL, br.URL.ShortCode),
				OriginalURL: br.URL.OriginalURL,
				ExpiresAt:   formatTimePtr(br.URL.ExpiresAt),
				CreatedAt:   br.URL.CreatedAt.Format(time.RFC3339),
			}
		}
	}

	respondJSON(w, http.StatusOK, batchCreateURLResponse{
		Results: items,
		Summary: batchSummary{
			Total:   len(batchResults),
			Success: succeeded,
			Failed:  len(batchResults) - succeeded,
		},
	})
}

// GetQRCode handles GET /api/v1/urls/{code}/qr.
// Returns a PNG QR code for the full short URL. Size is clamped to [128, 1024].
func (h *Handler) GetQRCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	size := parseIntQuery(r, "size", 256)
	if size < 128 {
		size = 128
	}
	if size > 1024 {
		size = 1024
	}

	if _, err := h.svc.GetByCode(r.Context(), code); err != nil {
		respondError(w, err)
		return
	}

	shortURL := fmt.Sprintf("%s/%s", h.baseURL, code)
	png, err := qrcode.Encode(shortURL, qrcode.Medium, size)
	if err != nil {
		h.logger.Error("qr code generation failed", "code", code, "error", err)
		respondJSON(w, http.StatusInternalServerError, errorResponse{Error: errorDetail{
			Code:    "internal_error",
			Message: "An internal error occurred",
		}})
		return
	}

	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png) //nolint:gosec // binary PNG data; Content-Type is image/png, not HTML
}

func parseIntQuery(r *http.Request, key string, defaultVal int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}

func toURLResponse(u *shortener.URL, baseURL string) urlResponse {
	return urlResponse{
		ShortCode:   u.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", baseURL, u.ShortCode),
		OriginalURL: u.OriginalURL,
		IsCustom:    u.IsCustom,
		ExpiresAt:   formatTimePtr(u.ExpiresAt),
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		ClickCount:  u.ClickCount,
	}
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// cacheTTL returns how long to cache a URL.
// The TTL is capped at 24 hours and aligned to the URL's expiry so we never
// serve an expired URL from cache.
func cacheTTL(u *shortener.URL) time.Duration {
	const defaultTTL = 24 * time.Hour
	if u.ExpiresAt == nil {
		return defaultTTL
	}
	remaining := time.Until(*u.ExpiresAt)
	if remaining <= 0 {
		return 0
	}
	if remaining > defaultTTL {
		return defaultTTL
	}
	return remaining
}
