package api

import (
	"net/http"

	"github.com/anIcedAntFA/goshort/docs"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RouterConfig holds middleware options for the router.
// Zero value disables all optional middleware (auth disabled, rate limit disabled).
type RouterConfig struct {
	APIKey           string
	RateLimitEnabled bool
	RateLimitRPM     int
}

// NewRouter creates and configures the Chi router with all application routes.
// cfg configures optional middleware; pass zero value to disable all.
func NewRouter(h *Handler, cfg RouterConfig) chi.Router {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware(h.logger))
	r.Use(MetricsMiddleware())

	r.Get("/health", h.HealthCheck)
	r.Handle("/metrics", promhttp.Handler())

	// API docs — Scalar UI + raw OpenAPI spec
	r.Get("/docs", serveDocs)
	r.Get("/docs/openapi.yaml", serveOpenAPISpec)

	// API v1 — auth before rate limiting so bad keys get 401, not 429.
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(AuthMiddleware(cfg.APIKey))
		r.Use(RateLimitMiddleware(cfg.RateLimitEnabled, cfg.RateLimitRPM))
		r.Post("/urls", h.CreateURL)
		r.Post("/urls/batch", h.BatchCreateURL)
		r.Get("/urls", h.ListURLs)
		r.Get("/urls/{code}/qr", h.GetQRCode)
		r.Get("/urls/{code}", h.GetURL)
		r.Delete("/urls/{code}", h.DeleteURL)
	})

	// Redirect — registered last so static routes take priority.
	r.Get("/{code}", h.Redirect)

	return r
}

func serveDocs(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html>
<head>
    <title>GoShort API</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
    <script id="api-reference" data-url="/docs/openapi.yaml"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`))
}

func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	_, _ = w.Write(docs.OpenAPISpec)
}
