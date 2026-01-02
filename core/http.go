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
	Data interface{} `json:"data,omitempty"`
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

// HTTPLogger is an optional interface for logging HTTP helper errors.
// This is a subset of structured logging interfaces (zap, logrus, slog).
type HTTPLogger interface {
	Error(msg string, keysAndValues ...interface{})
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

// RequestBodyMeta describes the expected request body for an API endpoint.
// Used by the OpenAPI plugin for automatic documentation generation.
type RequestBodyMeta struct {
	// Description explains what the request body contains
	Description string

	// Required indicates if a request body must be provided
	Required bool

	// Schema is either:
	//   - A string with the schema name (e.g., "CreateUserRequest")
	//   - An inline schema definition (struct or map)
	Schema interface{}
}

// ResponseMeta describes a possible response for an API endpoint.
// Used by the OpenAPI plugin for automatic documentation generation.
type ResponseMeta struct {
	// Description explains what this response represents
	Description string

	// Schema is either:
	//   - A string with the schema name (e.g., "User", "Error")
	//   - An inline schema definition (struct or map)
	Schema interface{}
}

// RouteMetadata contains OpenAPI documentation metadata for a route.
//
// This metadata enables automatic API documentation generation via the OpenAPI
// plugin. Developers annotate routes with this metadata, and the OpenAPI spec
// is automatically generated.
//
// Example:
//
//	metadata := &core.RouteMetadata{
//		Method:      "POST",
//		Path:        "/auth/login",
//		Summary:     "Authenticate user",
//		Description: "Login with email and password",
//		Tags:        []string{"Authentication"},
//		Protected:   false,
//		RequestBody: &core.RequestBodyMeta{
//			Description: "Login credentials",
//			Required:    true,
//			Schema:      "LoginRequest",
//		},
//		Responses: map[string]*core.ResponseMeta{
//			"200": {Description: "Successful login", Schema: "LoginResponse"},
//			"401": {Description: "Invalid credentials", Schema: "Error"},
//		},
//	}
type RouteMetadata struct {
	// Method is the HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	Method string

	// Path is the route path (e.g., "/auth/logout", "/users/:id")
	Path string

	// Summary is a short one-line description of the endpoint
	Summary string

	// Description is a detailed explanation of what the endpoint does
	Description string

	// Tags are used for grouping operations in documentation (e.g., ["Auth", "Users"])
	Tags []string

	// Protected indicates if the endpoint requires authentication
	Protected bool

	// RequestBody describes the expected request body (optional)
	RequestBody *RequestBodyMeta

	// Responses maps HTTP status codes to response descriptions
	// Example: {"200": {...}, "401": {...}, "500": {...}}
	Responses map[string]*ResponseMeta
}

// WriteJSON writes a JSON response with the given status code and data.
//
// Automatically sets Content-Type header to application/json. If JSON encoding
// fails and an HTTPLogger is configured (via SetHTTPLogger), the error is logged.
//
// Example:
//
//	core.WriteJSON(w, 200, map[string]string{"status": "ok"})
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
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
//
// Example:
//
//	core.WriteJSONError(w, 400, "Invalid request")
//	// Output: {"error": "Invalid request"}
func WriteJSONError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, map[string]string{"error": message})
}
