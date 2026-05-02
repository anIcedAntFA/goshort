package api

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goshort_http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "goshort_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"method", "path"},
	)

	urlsCreatedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goshort_urls_created_total",
			Help: "Total URLs created",
		},
		[]string{"type"},
	)
	redirectsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "goshort_redirects_total",
			Help: "Total redirects by status",
		},
		[]string{"status"},
	)
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// LoggingMiddleware logs each request after completion, skipping /health and /metrics.
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if path == "/health" || path == "/metrics" || path == "/docs" || path == "/docs/openapi.yaml" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			ua := r.UserAgent()
			if len(ua) > 100 {
				ua = ua[:100]
			}

			attrs := []any{
				"method", r.Method,
				"path", path,
				"status", rw.statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", ua,
			}

			switch {
			case rw.statusCode >= 500:
				logger.Error("request", attrs...)
			case rw.statusCode >= 400:
				logger.Warn("request", attrs...)
			default:
				logger.Info("request", attrs...)
			}
		})
	}
}

// AuthMiddleware enforces X-API-Key authentication on protected routes.
// When apiKey is empty the middleware is a no-op, preserving backward compatibility
// for deployments that have not configured auth.
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}
			header := r.Header.Get("X-API-Key")
			if header == "" {
				respondJSON(w, http.StatusUnauthorized, errorResponse{Error: errorDetail{
					Code:    "unauthorized",
					Message: "API key is required",
				}})
				return
			}
			if subtle.ConstantTimeCompare([]byte(header), []byte(apiKey)) != 1 {
				respondJSON(w, http.StatusUnauthorized, errorResponse{Error: errorDetail{
					Code:    "unauthorized",
					Message: "Invalid API key",
				}})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// extractIP extracts the client IP from r.RemoteAddr.
// X-Forwarded-For is intentionally ignored — it can be spoofed without proxy
// configuration. Proxy-aware extraction is a Phase 3.5 concern (Nginx/Caddy).
func extractIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// startLimiterCleanup launches a background goroutine that evicts idle IP entries
// from limiters every 5 minutes. Entries unseen for 10 minutes are removed.
func startLimiterCleanup(limiters map[string]*ipLimiter, mu *sync.Mutex) {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			for ip, entry := range limiters {
				if time.Since(entry.lastSeen) > 10*time.Minute {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()
}

// RateLimitMiddleware enforces per-IP token bucket rate limiting.
// When enabled is false the middleware is a no-op passthrough.
// Each IP gets a burst equal to requestsPerMinute, then throttled to RPM/60 per second.
func RateLimitMiddleware(enabled bool, requestsPerMinute int) func(http.Handler) http.Handler {
	if !enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	var mu sync.Mutex
	limiters := make(map[string]*ipLimiter)
	startLimiterCleanup(limiters, &mu)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)

			mu.Lock()
			entry, exists := limiters[ip]
			if !exists {
				entry = &ipLimiter{
					limiter: rate.NewLimiter(rate.Limit(float64(requestsPerMinute)/60.0), requestsPerMinute),
				}
				limiters[ip] = entry
			}
			entry.lastSeen = time.Now()
			lim := entry.limiter
			mu.Unlock()

			if !lim.Allow() {
				w.Header().Set("Retry-After", "60")
				respondJSON(w, http.StatusTooManyRequests, errorResponse{Error: errorDetail{
					Code:    "rate_limited",
					Message: "Too many requests. Please try again later.",
				}})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MetricsMiddleware records Prometheus request count and duration metrics.
func MetricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			// Use the route pattern to avoid high cardinality from real short codes.
			pattern := chi.RouteContext(r.Context()).RoutePattern()
			if pattern == "" {
				pattern = "unmatched"
			}

			statusStr := fmt.Sprintf("%d", rw.statusCode)

			httpRequestsTotal.WithLabelValues(r.Method, pattern, statusStr).Inc()
			httpRequestDuration.WithLabelValues(r.Method, pattern).Observe(time.Since(start).Seconds())
		})
	}
}
