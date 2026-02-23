package openapi

import (
	"html"
	"net/http"
	"path"
	"strings"

	"github.com/theinventorylib/aegis/router"
)

// Handler handles HTTP requests for OpenAPI documentation.
//
// This handler serves:
//   - OpenAPI specification in JSON format
//   - Scalar interactive documentation UI
type Handler struct {
	// plugin holds the OpenAPI plugin instance
	plugin *Plugin
	// router provides route metadata for spec generation
	router router.Router
}

// NewHandler creates a new OpenAPI handler.
//
// Parameters:
//   - plugin: Initialized OpenAPI plugin
//   - router: Router instance for metadata collection
//
// Returns:
//   - *Handler: Handler ready for route registration
func NewHandler(plugin *Plugin, router router.Router) *Handler {
	return &Handler{
		plugin: plugin,
		router: router,
	}
}

// ServeSpec serves the OpenAPI specification as JSON.
//
// This endpoint:
//  1. Collects latest route metadata from router
//  2. Updates OpenAPI spec with new routes
//  3. Serializes spec to JSON
//  4. Serves with CORS headers for documentation tools
//
// Endpoint:
//   - Method: GET
//   - Path: /openapi.json
//   - Auth: Public
//
// Response:
//   - Content-Type: application/json
//   - Access-Control-Allow-Origin: * (for Swagger UI, Postman, etc.)
//
// Use Cases:
//   - Generate client SDKs
//   - Import into Postman/Insomnia
//   - Validate API contracts
//   - Feed documentation generators
func (h *Handler) ServeSpec(w http.ResponseWriter, _ *http.Request) {
	// Get latest metadata from router
	metadata := h.router.GetRouteMetadata()

	// Update spec with metadata
	h.plugin.UpdateSpec(metadata)

	spec := h.plugin.GetSpec()

	jsonBytes, err := spec.ToJSON()
	if err != nil {
		http.Error(w, "Failed to generate OpenAPI spec", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS for documentation tools
	w.WriteHeader(http.StatusOK)
	_, writeErr := w.Write(jsonBytes)
	_ = writeErr
}

// ServeScalarUI serves the Scalar documentation UI.
//
// Scalar Features:
//   - Interactive API explorer
//   - Request/response examples
//   - Try-it-out functionality
//   - Authentication testing
//   - Schema visualization
//
// Endpoint:
//   - Method: GET
//   - Path: /docs
//   - Auth: Public
//
// Implementation:
// Uses CDN-hosted Scalar UI (https://cdn.jsdelivr.net/npm/@scalar/api-reference)
// Loads spec from ./openapi.json endpoint.
//
// Example:
// Navigate to http://localhost:8080/auth/docs to view interactive documentation.
func (h *Handler) ServeScalarUI(w http.ResponseWriter, req *http.Request) {
	// Compute the correct spec URL relative to the current request path.
	// If docs are served at /.../openapi/docs then the spec is at /.../openapi/openapi.json
	p := strings.TrimSuffix(req.URL.Path, "/")
	specURL := path.Join(path.Dir(p), "openapi.json")

	// Scalar UI HTML with CDN-hosted assets
	// Escape user-controllable values to prevent reflected XSS
	escapedTitle := html.EscapeString(h.plugin.config.Title)
	escapedSpecURL := html.EscapeString(specURL)

	html := `<!DOCTYPE html>
<html>
<head>
	<title>` + escapedTitle + ` - API Documentation</title>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
	<script
		id="api-reference"
		data-url="` + escapedSpecURL + `"></script>
	<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	//nolint:gosec // HTML variables are escaped securely above
	_, err := w.Write([]byte(html))
	_ = err
}
