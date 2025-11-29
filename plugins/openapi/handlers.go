package openapi

import (
	"net/http"

	"github.com/theinventorylib/aegis/server"
)

// Handler handles HTTP requests for OpenAPI documentation.
type Handler struct {
	plugin *Plugin
	router server.Router
}

// NewHandler creates a new OpenAPI handler.
func NewHandler(plugin *Plugin, router server.Router) *Handler {
	return &Handler{
		plugin: plugin,
		router: router,
	}
}

// ServeSpec serves the OpenAPI specification as JSON.
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
	_, _ = w.Write(jsonBytes)
}

// ServeScalarUI serves the Scalar documentation UI.
func (h *Handler) ServeScalarUI(w http.ResponseWriter, _ *http.Request) {
	// Scalar UI HTML with CDN-hosted assets
	html := `<!DOCTYPE html>
<html>
<head>
    <title>` + h.plugin.config.Title + ` - API Documentation</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
    <script
        id="api-reference"
        data-url="./openapi.json"></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}
