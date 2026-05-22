// Package routers provides router adapters for the Aegis authentication framework.
//
// This package contains implementations of the Router interface for popular
// Go HTTP routers, allowing Aegis to integrate with various routing libraries.
//
// Currently supported routers:
//   - Chi:  A lightweight, idiomatic HTTP router for Go (NewChiRouter)
//   - Gin:  A high-performance HTTP web framework (NewGinRouter)
//   - Echo: A high-performance, extensible web framework (NewEchoRouter)
//
// All adapters bridge standard net/http handlers and middleware to the
// native types of each framework, so Aegis plugins work transparently
// regardless of which router is chosen.
//
// Usage (Chi):
//
//	mux := chi.NewRouter()
//	router := routers.NewChiRouter(mux)
//	aegis.New(ctx, config.WithRouter(router), ...)
//
// Usage (Gin):
//
//	engine := gin.Default()
//	router := routers.NewGinRouter(engine)
//	aegis.New(ctx, config.WithRouter(router), ...)
//
// Usage (Echo):
//
//	e := echo.New()
//	router := routers.NewEchoRouter(e)
//	aegis.New(ctx, config.WithRouter(router), ...)
package routers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/theinventorylib/aegis/core"
	aegisrouter "github.com/theinventorylib/aegis/router"
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
	// groups caches sub-routers by path so repeated Group() calls with the
	// same path reuse the existing chi sub-router instead of calling Route()
	// again, which would panic with "attempting to Mount() on an existing path".
	groups map[string]chi.Router
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
	mux.Use(chiPathParamMiddleware())
	return &ChiRouter{
		Mux:    mux,
		groups: make(map[string]chi.Router),
	}
}

// GET registers a GET route handler.
// Implements Router.GET by delegating to chi.Mux.Get.
func (r *ChiRouter) GET(path string, handler http.HandlerFunc) {
	r.Get(aegisrouter.NormalizePath(path), handler)
}

// POST registers a POST route handler.
// Implements Router.POST by delegating to chi.Mux.Post.
func (r *ChiRouter) POST(path string, handler http.HandlerFunc) {
	r.Post(aegisrouter.NormalizePath(path), handler)
}

// PUT registers a PUT route handler.
// Implements Router.PUT by delegating to chi.Mux.Put.
func (r *ChiRouter) PUT(path string, handler http.HandlerFunc) {
	r.Put(aegisrouter.NormalizePath(path), handler)
}

// PATCH registers a PATCH route handler.
// Implements Router.PATCH by delegating to chi.Mux.Patch.
func (r *ChiRouter) PATCH(path string, handler http.HandlerFunc) {
	r.Patch(aegisrouter.NormalizePath(path), handler)
}

// DELETE registers a DELETE route handler.
// Implements Router.DELETE by delegating to chi.Mux.Delete.
//
// Note: This adapts chi's lowercase Delete method to the uppercase DELETE
// method required by the Router interface.
func (r *ChiRouter) DELETE(path string, handler http.HandlerFunc) {
	r.Delete(aegisrouter.NormalizePath(path), handler)
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
func (r *ChiRouter) Group(path string, groupName string) aegisrouter.GroupRouter {
	norm := aegisrouter.NormalizePath(path)
	if sub, ok := r.groups[norm]; ok {
		return &ChiGroupRouter{groupName: groupName, router: sub, groups: make(map[string]chi.Router)}
	}
	var sub chi.Router
	r.Route(norm, func(r chi.Router) {
		sub = r
	})
	r.groups[norm] = sub
	return &ChiGroupRouter{
		groupName: groupName,
		router:    sub,
		groups:    make(map[string]chi.Router),
	}
}

// ServeHTTP handles HTTP requests by delegating to the underlying chi.Mux.
// This allows ChiRouter to satisfy the http.Handler interface.
//
// Implements Router.ServeHTTP (inherited from chi.Mux).

// ChiGroupRouter wraps a chi.Router sub-router to provide route grouping functionality.
//
// It is backed by chi's native Route(), which creates a sub-router that
// inherits the full parent middleware chain. Middleware added via Use() is
// scoped only to routes registered on this group.
//
// ChiGroupRouter is created via ChiRouter.Group() and should not be
// instantiated directly.
type ChiGroupRouter struct {
	// groupName is used for organizational purposes
	groupName string

	// router is the chi sub-router created by Route()
	router chi.Router

	// groups caches nested sub-routers by path to prevent double-mount panics
	// when Group() is called multiple times with the same path.
	groups map[string]chi.Router
}

// GET registers a GET route handler within this group.
func (g *ChiGroupRouter) GET(path string, handler http.HandlerFunc) {
	g.router.Get(aegisrouter.NormalizePath(path), handler)
}

// POST registers a POST route handler within this group.
func (g *ChiGroupRouter) POST(path string, handler http.HandlerFunc) {
	g.router.Post(aegisrouter.NormalizePath(path), handler)
}

// PUT registers a PUT route handler within this group.
func (g *ChiGroupRouter) PUT(path string, handler http.HandlerFunc) {
	g.router.Put(aegisrouter.NormalizePath(path), handler)
}

// PATCH registers a PATCH route handler within this group.
func (g *ChiGroupRouter) PATCH(path string, handler http.HandlerFunc) {
	g.router.Patch(aegisrouter.NormalizePath(path), handler)
}

// DELETE registers a DELETE route handler within this group.
func (g *ChiGroupRouter) DELETE(path string, handler http.HandlerFunc) {
	g.router.Delete(aegisrouter.NormalizePath(path), handler)
}

// Use adds middleware scoped to this group.
// Delegates to chi's native sub-router Use — middleware is inherited by
// any nested groups created from this group.
func (g *ChiGroupRouter) Use(middleware func(http.Handler) http.Handler) {
	g.router.Use(middleware)
}

// Group creates a nested sub-group under this group.
// Uses chi's Route() so the nested group inherits this group's middleware chain.
func (g *ChiGroupRouter) Group(path string, groupName string) aegisrouter.GroupRouter {
	norm := aegisrouter.NormalizePath(path)
	if sub, ok := g.groups[norm]; ok {
		return &ChiGroupRouter{groupName: groupName, router: sub, groups: make(map[string]chi.Router)}
	}
	var sub chi.Router
	g.router.Route(norm, func(r chi.Router) {
		sub = r
	})
	if g.groups == nil {
		g.groups = make(map[string]chi.Router)
	}
	g.groups[norm] = sub
	return &ChiGroupRouter{
		groupName: groupName,
		router:    sub,
		groups:    make(map[string]chi.Router),
	}
}

// chiPathParamMiddleware injects a PathParamFunc into the request context that
// bridges chi's parameter extraction (chi.URLParam) to core.GetPathParam.
//
// Chi stores path parameters in its own RouteContext, not accessible via
// Go 1.22's r.PathValue(). Without this middleware, core.GetPathParam would
// fall through to r.PathValue(), which chi does not populate.
//
// Because chi's Route() sub-routers inherit the parent middleware chain,
// this middleware is automatically active for all groups and nested groups.
func chiPathParamMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fn := core.PathParamFunc(func(r *http.Request, name string) string {
				return chi.URLParam(r, name)
			})
			ctx := core.WithPathParamFunc(r.Context(), fn)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
