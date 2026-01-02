package router

import (
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// Router is the interface for HTTP routers in Aegis.
//
// This abstraction allows Aegis to work with different HTTP routing libraries
// (chi, mux, gin, echo, etc.) without coupling to a specific implementation.
//
// Implementations must support:
//   - Standard HTTP methods (GET, POST, PUT, PATCH, DELETE)
//   - Middleware chain using standard http.Handler interface
//   - Route metadata registration for OpenAPI documentation
//   - http.Handler compliance for ServeHTTP
//
// Example implementation (using chi):
//
//	type ChiRouter struct {
//		*chi.Mux
//		metadata []core.RouteMetadata
//	}
//
//	func (r *ChiRouter) RegisterRouteMetadata(m core.RouteMetadata) {
//		r.metadata = append(r.metadata, m)
//	}
//
//	func (r *ChiRouter) GetRouteMetadata() []core.RouteMetadata {
//		return r.metadata
//	}
type Router interface {
	// GET registers a GET route handler
	GET(path string, handler http.HandlerFunc)

	// POST registers a POST route handler
	POST(path string, handler http.HandlerFunc)

	// PUT registers a PUT route handler
	PUT(path string, handler http.HandlerFunc)

	// PATCH registers a PATCH route handler
	PATCH(path string, handler http.HandlerFunc)

	// DELETE registers a DELETE route handler
	DELETE(path string, handler http.HandlerFunc)

	// Use adds middleware to the router
	// Middleware should follow the func(http.Handler) http.Handler pattern
	Use(middleware func(http.Handler) http.Handler)

	// ServeHTTP implements http.Handler for serving requests
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// RegisterRouteMetadata registers OpenAPI metadata for a route
	// This metadata is used by the OpenAPI plugin for automatic documentation
	RegisterRouteMetadata(metadata core.RouteMetadata)

	// GetRouteMetadata returns all registered route metadata
	// Used by the OpenAPI plugin to generate specifications
	GetRouteMetadata() []core.RouteMetadata
}
