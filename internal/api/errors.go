// Package api implements the HTTP handler layer for GoShort.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, shortener.ErrNotFound):
		respondJSON(w, http.StatusNotFound, errorResponse{Error: errorDetail{
			Code:    "not_found",
			Message: "Short URL not found",
		}})
	case errors.Is(err, shortener.ErrExpired):
		respondJSON(w, http.StatusGone, errorResponse{Error: errorDetail{
			Code:    "expired",
			Message: "This short URL has expired",
		}})
	case errors.Is(err, shortener.ErrAliasTaken):
		respondJSON(w, http.StatusConflict, errorResponse{Error: errorDetail{
			Code:    "alias_taken",
			Message: "The requested alias is already in use",
		}})
	case errors.Is(err, shortener.ErrReservedPath):
		respondJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: errorDetail{
			Code:    "reserved_path",
			Message: "The alias is a reserved path",
		}})
	case errors.Is(err, shortener.ErrInvalidURL):
		respondJSON(w, http.StatusBadRequest, errorResponse{Error: errorDetail{
			Code:    "invalid_url",
			Message: "The URL format is invalid",
		}})
	case errors.Is(err, shortener.ErrInvalidAlias):
		respondJSON(w, http.StatusUnprocessableEntity, errorResponse{Error: errorDetail{
			Code:    "invalid_alias",
			Message: "The alias format is invalid",
		}})
	case errors.Is(err, shortener.ErrInvalidExpires):
		respondJSON(w, http.StatusBadRequest, errorResponse{Error: errorDetail{
			Code:    "invalid_expires",
			Message: "The expires_in duration is invalid",
		}})
	default:
		slog.Error("internal server error", "error", err)
		respondJSON(w, http.StatusInternalServerError, errorResponse{Error: errorDetail{
			Code:    "internal_error",
			Message: "An internal error occurred",
		}})
	}
}
