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
		Mux: mux,
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

// Group creates a sub-router with a prefix for grouping related routes.
//
// Routes registered on the returned GroupRouter will be prefixed with the
// given path. The groupName is used for organizational purposes.
//
// Example:
//
//	jwtGroup := router.Group("/jwt", "JWT")
//	jwtGroup.POST("/token", tokenHandler)     // Mounted at /jwt/token
//	jwtGroup.POST("/refresh", refreshHandler) // Mounted at /jwt/refresh
//
// Implements Router.Group.
func (r *ChiRouter) Group(path string, groupName string) GroupRouter {
	return &ChiGroupRouter{
		prefix:    path,
		groupName: groupName,
		parent:    r,
	}
}

// ServeHTTP handles HTTP requests by delegating to the underlying chi.Mux.
// This allows ChiRouter to satisfy the http.Handler interface.
//
// Implements Router.ServeHTTP (inherited from chi.Mux).

// ChiGroupRouter wraps a ChiRouter to provide route grouping functionality.
//
// Routes registered on a ChiGroupRouter are automatically prefixed with
// the group's path.
//
// ChiGroupRouter is created via ChiRouter.Group() and should not be
// instantiated directly.
type ChiGroupRouter struct {
	// prefix is the path prefix for all routes in this group
	prefix string

	// groupName is used for organizational purposes
	groupName string

	// parent is the ChiRouter that owns this group
	parent *ChiRouter
}

// GET registers a GET route handler within this group.
// The path is automatically prefixed with the group's prefix.
func (g *ChiGroupRouter) GET(path string, handler http.HandlerFunc) {
	g.parent.Get(g.prefix+path, handler)
}

// POST registers a POST route handler within this group.
// The path is automatically prefixed with the group's prefix.
func (g *ChiGroupRouter) POST(path string, handler http.HandlerFunc) {
	g.parent.Post(g.prefix+path, handler)
}

// PUT registers a PUT route handler within this group.
// The path is automatically prefixed with the group's prefix.
func (g *ChiGroupRouter) PUT(path string, handler http.HandlerFunc) {
	g.parent.Put(g.prefix+path, handler)
}

// PATCH registers a PATCH route handler within this group.
// The path is automatically prefixed with the group's prefix.
func (g *ChiGroupRouter) PATCH(path string, handler http.HandlerFunc) {
	g.parent.Patch(g.prefix+path, handler)
}

// DELETE registers a DELETE route handler within this group.
// The path is automatically prefixed with the group's prefix.
func (g *ChiGroupRouter) DELETE(path string, handler http.HandlerFunc) {
	g.parent.Delete(g.prefix+path, handler)
}

// Use adds middleware to this group.
//
// Note: Due to chi's routing model, middleware added via Use() on a group
// affects the parent router. For group-scoped middleware, wrap handlers
// individually when registering routes.
func (g *ChiGroupRouter) Use(_ func(http.Handler) http.Handler) {
	// Chi doesn't have true group-scoped middleware when using this pattern.
	// Handlers should be wrapped individually for group-specific middleware.
	// This is a no-op to maintain API compatibility.
	// For group-scoped middleware, use chi's Route() or Group() directly.
}

// Group creates a nested group under this group by combining prefixes.
// The returned GroupRouter uses the same ChiRouter parent so routes
// are collected on the same root router.
func (g *ChiGroupRouter) Group(path string, groupName string) GroupRouter {
	return &ChiGroupRouter{
		prefix:    g.prefix + path,
		groupName: groupName,
		parent:    g.parent,
	}
}
