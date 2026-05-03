// Package mcp implements the GoShort MCP server with tools, resources, and prompts.
package mcp

import (
	"crypto/subtle"
	"net/http"
)

// apiKeyMiddleware wraps an HTTP handler with X-API-Key authentication.
// When apiKey is empty, all requests are allowed through.
func apiKeyMiddleware(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		got := r.Header.Get("X-API-Key")
		if subtle.ConstantTimeCompare([]byte(got), []byte(apiKey)) != 1 {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
