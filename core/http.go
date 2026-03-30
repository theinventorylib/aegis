package core

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Response represents a standard JSON API response structure.
// This provides a consistent format across all Aegis endpoints.
type Response struct {
	// Success indicates if the request completed successfully
	Success bool `json:"success"`

	// Message contains a human-readable success message (optional)
	Message string `json:"message,omitempty"`

	// Error contains a human-readable error message (optional, Success=false)
	Error string `json:"error,omitempty"`

	// Data contains the response payload (optional, Success=true)
	Data any `json:"data,omitempty"`
}

// PaginationParams holds parsed and validated pagination parameters.
// Used with ParsePagination to extract pagination from query strings.
type PaginationParams struct {
	// Page is the 1-based page number (default: 1)
	Page int

	// Limit is the number of items per page (default: 20, max: 100)
	Limit int

	// Offset is the calculated skip offset for database queries
	// Automatically calculated as (Page-1) * Limit
	Offset int
}

// PaginatedResponse is a standard pagination envelope for list endpoints.
//
// It is intended to be wrapped as core.Response.Data (i.e. core.Response{Data: PaginatedResponse[T]}).
// This keeps pagination metadata consistent across endpoints without per-resource DTOs.
type PaginatedResponse[T any] struct {
	Items      []T `json:"items"`
	TotalCount int `json:"totalCount"`
	Page       int `json:"page"`
	Offset     int `json:"offset"`
	Limit      int `json:"limit"`
}

// SessionRefreshResponse represents the data payload returned by the session
// refresh endpoint. This is wrapped in a core.Response envelope.
type SessionRefreshResponse struct {
	// ExpiresAt is when the new session expires
	ExpiresAt string `json:"expiresAt"`
}

// HTTPLogger is an optional interface for logging HTTP helper errors.
// This is a subset of structured logging interfaces (zap, logrus, slog).
type HTTPLogger interface {
	Error(msg string, keysAndValues ...any)
}

// httpLogger is the global logger for HTTP helpers (optional)
var httpLogger HTTPLogger

// SetHTTPLogger sets the logger for HTTP helpers.
// If set, WriteJSON will log JSON encoding errors instead of silently failing.
//
// Example:
//
//	core.SetHTTPLogger(myZapLogger)
func SetHTTPLogger(l HTTPLogger) {
	httpLogger = l
}

// ParsePagination extracts and validates pagination parameters from request query string.
//
// Query parameters:
//   - page: Page number (1-based, default: 1)
//   - limit: Items per page (default: 20, max: 100)
//
// Invalid values are replaced with defaults:
//   - page < 1 becomes 1
//   - limit < 1 or limit > 100 becomes 20
//
// Example:
//
//	params := core.ParsePagination(r) // ?page=2&limit=50
//	users, _ := userStore.List(ctx, params.Offset, params.Limit)
func ParsePagination(r *http.Request) PaginationParams {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	return PaginationParams{
		Page:   page,
		Limit:  limit,
		Offset: offset,
	}
}

// WriteJSON writes a JSON response with the given status code and data.
//
// Automatically sets Content-Type header to application/json. If JSON encoding
// fails and an HTTPLogger is configured (via SetHTTPLogger), the error is logged.
//
// Example:
//
//	core.WriteJSON(w, 200, map[string]string{"status": "ok"})
func WriteJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		if httpLogger != nil {
			httpLogger.Error("failed to encode JSON response", "error", err.Error())
		}
	}
}

// WriteJSONError writes a JSON error response with the given status code and message.
//
// This is a convenience wrapper around WriteJSON for error responses.
// The response envelope matches the core.Response structure and the OpenAPI Error schema.
//
// Example:
//
//	core.WriteJSONError(w, 400, "Invalid request")
//	// Output: {"success": false, "error": "Invalid request"}
func WriteJSONError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, &Response{Success: false, Error: message})
}
