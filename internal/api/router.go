package api

import "github.com/go-chi/chi/v5"

// NewRouter creates and configures the Chi router with all application routes.
func NewRouter(h *Handler) chi.Router {
	r := chi.NewRouter()

	r.Get("/health", h.HealthCheck)

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
