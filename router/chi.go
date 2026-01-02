// Package router provides router adapters for the Aegis authentication framework.
//
// This package contains implementations of the Router interface for popular
// Go HTTP routers, allowing Aegis to integrate with various routing libraries.
//
// Currently supported routers:
//   - Chi: A lightweight, idiomatic HTTP router for Go
//
// Usage:
//
//	mux := chi.NewRouter()
//	router := router.NewChiRouter(mux)
//	aegis.New(ctx, config.WithRouter(router), ...)
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/theinventorylib/aegis/core"
)

// ChiRouter wraps chi.Mux to implement the Router interface.
//
// This adapter allows the popular chi router to be used with Aegis while
// maintaining the abstraction provided by the Router interface.
//
// Usage:
//
//	mux := chi.NewRouter()
//	mux.Use(middleware.Logger)
//	router := router.NewChiRouter(mux)
//
//	aegis.New(ctx,
//	    config.WithRouter(router),
//	    // ... other options
//	)
type ChiRouter struct {
	*chi.Mux
	metadata []core.RouteMetadata
}

// NewChiRouter creates a new ChiRouter that wraps the provided chi.Mux.
//
// The chi.Mux should be configured with any middleware before wrapping,
// as middleware added via the Router.Use method will be applied to the mux.
//
// Example:
//
//	mux := chi.NewRouter()
//	mux.Use(middleware.Logger)
//	mux.Use(middleware.Recoverer)
//	router := router.NewChiRouter(mux)
func NewChiRouter(mux *chi.Mux) *ChiRouter {
	return &ChiRouter{
		Mux:      mux,
		metadata: []core.RouteMetadata{},
	}
}

// GET registers a GET route handler.
// Implements Router.GET by delegating to chi.Mux.Get.
func (r *ChiRouter) GET(path string, handler http.HandlerFunc) {
	r.Get(path, handler)
}

// POST registers a POST route handler.
// Implements Router.POST by delegating to chi.Mux.Post.
func (r *ChiRouter) POST(path string, handler http.HandlerFunc) {
	r.Post(path, handler)
}

// PUT registers a PUT route handler.
// Implements Router.PUT by delegating to chi.Mux.Put.
func (r *ChiRouter) PUT(path string, handler http.HandlerFunc) {
	r.Put(path, handler)
}

// PATCH registers a PATCH route handler.
// Implements Router.PATCH by delegating to chi.Mux.Patch.
func (r *ChiRouter) PATCH(path string, handler http.HandlerFunc) {
	r.Patch(path, handler)
}

// DELETE registers a DELETE route handler.
// Implements Router.DELETE by delegating to chi.Mux.Delete.
//
// Note: This adapts chi's lowercase Delete method to the uppercase DELETE
// method required by the Router interface.
func (r *ChiRouter) DELETE(path string, handler http.HandlerFunc) {
	r.Delete(path, handler)
}

// Use adds middleware to the router.
// Implements Router.Use by delegating to chi.Mux.Use.
//
// Middleware should follow the func(http.Handler) http.Handler pattern.
func (r *ChiRouter) Use(middleware func(http.Handler) http.Handler) {
	r.Mux.Use(middleware)
}

// RegisterRouteMetadata stores OpenAPI metadata for a route.
// This metadata is collected and used by the OpenAPI plugin to generate
// API documentation automatically.
//
// Implements Router.RegisterRouteMetadata.
func (r *ChiRouter) RegisterRouteMetadata(metadata core.RouteMetadata) {
	r.metadata = append(r.metadata, metadata)
}

// GetRouteMetadata returns all registered route metadata.
// Used by the OpenAPI plugin to retrieve route documentation.
//
// Implements Router.GetRouteMetadata.
func (r *ChiRouter) GetRouteMetadata() []core.RouteMetadata {
	return r.metadata
}

// ServeHTTP handles HTTP requests by delegating to the underlying chi.Mux.
// This allows ChiRouter to satisfy the http.Handler interface.
//
// Implements Router.ServeHTTP (inherited from chi.Mux).
