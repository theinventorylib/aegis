package openapi

import (
	"html"
	"net/http"
	"path"
	"strings"
)

// Handler handles HTTP requests for OpenAPI documentation.
//
// This handler serves:
//   - OpenAPI specification in JSON format
//   - Scalar interactive documentation UI
type Handler struct {
	// plugin holds the OpenAPI plugin instance
	plugin *Plugin
}

// NewHandler creates a new OpenAPI handler.
//
// Parameters:
//   - plugin: Initialized OpenAPI plugin
//
// Returns:
//   - *Handler: Handler ready for route registration
func NewHandler(plugin *Plugin) *Handler {
	return &Handler{
		plugin: plugin,
	}
}

// ServeSpec serves the OpenAPI specification as JSON.
//
// The spec is generated from all routes registered via Doc().
// No metadata collection from the router is needed — the spec
// is fully built during plugin initialization.
//
// Endpoint:
//   - Method: GET
//   - Path: Configured via Config.SpecPath (default: /openapi.json)
//   - Auth: Public
//
// Response:
//   - Content-Type: application/json
//   - Access-Control-Allow-Origin: * (for Swagger UI, Postman, etc.)
func (h *Handler) ServeSpec(w http.ResponseWriter, _ *http.Request) {
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
//   - Path: Configured via Config.DocsPath (default: /docs)
//   - Auth: Public
//
// Implementation:
// Uses CDN-hosted Scalar UI (https://cdn.jsdelivr.net/npm/@scalar/api-reference)
// Loads spec from the configured spec path.
func (h *Handler) ServeScalarUI(w http.ResponseWriter, req *http.Request) {
	// Compute the correct spec URL relative to the current request path.
	// We need to find the spec URL relative to the docs URL.
	p := strings.TrimSuffix(req.URL.Path, "/")
	specURL := path.Join(path.Dir(p), strings.TrimPrefix(h.plugin.config.SpecPath, "/"))

	// Scalar UI HTML with CDN-hosted assets
	// Escape user-controllable values to prevent reflected XSS
	escapedTitle := html.EscapeString(h.plugin.config.Title)
	escapedSpecURL := html.EscapeString(specURL)

	htmlContent := `<!DOCTYPE html>
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
	_, err := w.Write([]byte(htmlContent))
	_ = err
}
