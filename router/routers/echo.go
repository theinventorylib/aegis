package routers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	aegisrouter "github.com/theinventorylib/aegis/router"
)

// EchoRouter wraps echo.Echo to implement the Router interface.
//
// This adapter allows the popular Echo web framework to be used with Aegis
// while maintaining the abstraction provided by the Router interface.
//
// Standard net/http middleware and handlers are automatically bridged to
// Echo's native types via echo.WrapHandler and echo.WrapMiddleware.
//
// Usage:
//
//	e := echo.New()
//	e.Use(middleware.Logger())
//	router := routers.NewEchoRouter(e)
//
//	aegis.New(ctx,
//	    config.WithRouter(router),
//	    // ... other options
//	)
type EchoRouter struct {
	*echo.Echo
}

// NewEchoRouter creates a new EchoRouter that wraps the provided echo.Echo instance.
//
// The Echo instance should be configured with any global middleware before
// wrapping, or middleware can be added later via the Router.Use method.
//
// Example:
//
//	e := echo.New()
//	e.Use(middleware.Logger())
//	e.Use(middleware.Recover())
//	router := routers.NewEchoRouter(e)
func NewEchoRouter(e *echo.Echo) *EchoRouter {
	return &EchoRouter{Echo: e}
}

// GET registers a GET route handler.
// Implements Router.GET by bridging http.HandlerFunc to echo.HandlerFunc via echo.WrapHandler.
func (r *EchoRouter) GET(path string, handler http.HandlerFunc) {
	r.Echo.GET(path, echo.WrapHandler(handler))
}

// POST registers a POST route handler.
// Implements Router.POST by bridging http.HandlerFunc to echo.HandlerFunc via echo.WrapHandler.
func (r *EchoRouter) POST(path string, handler http.HandlerFunc) {
	r.Echo.POST(path, echo.WrapHandler(handler))
}

// PUT registers a PUT route handler.
// Implements Router.PUT by bridging http.HandlerFunc to echo.HandlerFunc via echo.WrapHandler.
func (r *EchoRouter) PUT(path string, handler http.HandlerFunc) {
	r.Echo.PUT(path, echo.WrapHandler(handler))
}

// PATCH registers a PATCH route handler.
// Implements Router.PATCH by bridging http.HandlerFunc to echo.HandlerFunc via echo.WrapHandler.
func (r *EchoRouter) PATCH(path string, handler http.HandlerFunc) {
	r.Echo.PATCH(path, echo.WrapHandler(handler))
}

// DELETE registers a DELETE route handler.
// Implements Router.DELETE by bridging http.HandlerFunc to echo.HandlerFunc via echo.WrapHandler.
func (r *EchoRouter) DELETE(path string, handler http.HandlerFunc) {
	r.Echo.DELETE(path, echo.WrapHandler(handler))
}

// Use adds standard net/http middleware to the Echo instance.
// The middleware is adapted to Echo's MiddlewareFunc via echo.WrapMiddleware.
func (r *EchoRouter) Use(middleware func(http.Handler) http.Handler) {
	r.Echo.Use(echo.WrapMiddleware(middleware))
}

// Group creates a sub-router with a prefix for grouping related routes.
//
// Routes registered on the returned EchoGroupRouter will be prefixed with
// the given path. The groupName is used for organizational purposes.
//
// Example:
//
//	jwtGroup := router.Group("/jwt", "JWT")
//	jwtGroup.POST("/token", tokenHandler)     // Mounted at /jwt/token
//	jwtGroup.POST("/refresh", refreshHandler) // Mounted at /jwt/refresh
//
// Implements Router.Group.
func (r *EchoRouter) Group(path string, groupName string) aegisrouter.GroupRouter {
	return &EchoGroupRouter{
		prefix:    path,
		groupName: groupName,
		group:     r.Echo.Group(path),
	}
}

// ServeHTTP handles HTTP requests by delegating to the underlying echo.Echo instance.
// Inherited from echo.Echo — satisfies the http.Handler interface.

// EchoGroupRouter wraps echo.Group to provide route grouping functionality.
//
// Routes registered on an EchoGroupRouter are automatically prefixed with
// the group's path by the underlying echo.Group.
//
// EchoGroupRouter is created via EchoRouter.Group() and should not be
// instantiated directly.
type EchoGroupRouter struct {
	// prefix is the accumulated path prefix for this group
	prefix string

	// groupName is used for organizational purposes
	groupName string

	// group is the underlying echo group
	group *echo.Group
}

// GET registers a GET route handler within this group.
func (g *EchoGroupRouter) GET(path string, handler http.HandlerFunc) {
	g.group.GET(path, echo.WrapHandler(handler))
}

// POST registers a POST route handler within this group.
func (g *EchoGroupRouter) POST(path string, handler http.HandlerFunc) {
	g.group.POST(path, echo.WrapHandler(handler))
}

// PUT registers a PUT route handler within this group.
func (g *EchoGroupRouter) PUT(path string, handler http.HandlerFunc) {
	g.group.PUT(path, echo.WrapHandler(handler))
}

// PATCH registers a PATCH route handler within this group.
func (g *EchoGroupRouter) PATCH(path string, handler http.HandlerFunc) {
	g.group.PATCH(path, echo.WrapHandler(handler))
}

// DELETE registers a DELETE route handler within this group.
func (g *EchoGroupRouter) DELETE(path string, handler http.HandlerFunc) {
	g.group.DELETE(path, echo.WrapHandler(handler))
}

// Use adds standard net/http middleware to this group.
// The middleware is scoped to routes registered on this group.
func (g *EchoGroupRouter) Use(middleware func(http.Handler) http.Handler) {
	g.group.Use(echo.WrapMiddleware(middleware))
}

// Group creates a nested group under this group by combining path prefixes.
// The underlying echo.Group handles prefix concatenation automatically.
func (g *EchoGroupRouter) Group(path string, groupName string) aegisrouter.GroupRouter {
	return &EchoGroupRouter{
		prefix:    g.prefix + path,
		groupName: groupName,
		group:     g.group.Group(path),
	}
}
