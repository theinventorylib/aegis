// Package server provides HTTP server and routing implementations for Aegis.
package server

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/theinventorylib/aegis/models"
)

// Router is the interface for HTTP routers.
type Router interface {
	GET(path string, handler http.HandlerFunc)
	POST(path string, handler http.HandlerFunc)
	PUT(path string, handler http.HandlerFunc)
	PATCH(path string, handler http.HandlerFunc)
	DELETE(path string, handler http.HandlerFunc)
	Use(middleware func(http.Handler) http.Handler)
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// Route metadata for automatic documentation
	RegisterRouteMetadata(metadata models.RouteMetadata)
	GetRouteMetadata() []models.RouteMetadata
}

// ChiRouter adapts chi.Router to our Router interface.
type ChiRouter struct {
	mux           chi.Router
	routeMetadata []models.RouteMetadata
	mu            sync.RWMutex // Protects routeMetadata
}

// NewChiRouter creates a new Chi router adapter.
func NewChiRouter(mux chi.Router) *ChiRouter {
	return &ChiRouter{mux: mux}
}

// GET registers a GET route.
func (r *ChiRouter) GET(path string, handler http.HandlerFunc) {
	r.mux.Get(path, handler)
}

// POST registers a POST route.
func (r *ChiRouter) POST(path string, handler http.HandlerFunc) {
	r.mux.Post(path, handler)
}

// PUT registers a PUT route.
func (r *ChiRouter) PUT(path string, handler http.HandlerFunc) {
	r.mux.Put(path, handler)
}

// PATCH registers a PATCH route.
func (r *ChiRouter) PATCH(path string, handler http.HandlerFunc) {
	r.mux.Patch(path, handler)
}

// DELETE registers a DELETE route.
func (r *ChiRouter) DELETE(path string, handler http.HandlerFunc) {
	r.mux.Delete(path, handler)
}

// Use registers middleware.
func (r *ChiRouter) Use(middleware func(http.Handler) http.Handler) {
	r.mux.Use(middleware)
}

// ServeHTTP implements http.Handler.
func (r *ChiRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// RegisterRouteMetadata registers documentation metadata for a route.
func (r *ChiRouter) RegisterRouteMetadata(metadata models.RouteMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routeMetadata = append(r.routeMetadata, metadata)
}

// GetRouteMetadata returns all registered route metadata.
func (r *ChiRouter) GetRouteMetadata() []models.RouteMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]models.RouteMetadata, len(r.routeMetadata))
	copy(result, r.routeMetadata)
	return result
}
