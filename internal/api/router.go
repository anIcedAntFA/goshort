package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter creates and configures the Chi router with all application routes.
func NewRouter(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware(h.logger))
	r.Use(MetricsMiddleware())

	r.Get("/health", h.HealthCheck)
	r.Handle("/metrics", promhttp.Handler())

	// API docs — Scalar UI + raw OpenAPI spec
	r.Get("/docs", serveDocs)
	r.Get("/docs/openapi.yaml", serveOpenAPISpec)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/urls", h.CreateURL)
		r.Get("/urls", h.ListURLs)
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

func serveOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "docs/openapi.yaml")
}
