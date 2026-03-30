package routers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	aegisrouter "github.com/theinventorylib/aegis/router"
)

// GinRouter wraps gin.Engine to implement the Router interface.
//
// This adapter allows the popular Gin web framework to be used with Aegis
// while maintaining the abstraction provided by the Router interface.
//
// Standard net/http middleware and handlers are automatically bridged to
// Gin's native types via gin.WrapH and a custom middleware adapter.
//
// Usage:
//
//	engine := gin.Default()
//	engine.Use(gin.Recovery())
//	router := routers.NewGinRouter(engine)
//
//	aegis.New(ctx,
//	    config.WithRouter(router),
//	    // ... other options
//	)
type GinRouter struct {
	*gin.Engine
}

// NewGinRouter creates a new GinRouter that wraps the provided gin.Engine.
//
// The engine should be configured with any global middleware before wrapping,
// or middleware can be added later via the Router.Use method.
//
// Example:
//
//	engine := gin.New()
//	engine.Use(gin.Logger())
//	engine.Use(gin.Recovery())
//	router := routers.NewGinRouter(engine)
func NewGinRouter(engine *gin.Engine) *GinRouter {
	return &GinRouter{Engine: engine}
}

// GET registers a GET route handler.
// Implements Router.GET by bridging http.HandlerFunc to gin.HandlerFunc via gin.WrapF.
func (r *GinRouter) GET(path string, handler http.HandlerFunc) {
	r.Engine.GET(path, gin.WrapF(handler))
}

// POST registers a POST route handler.
// Implements Router.POST by bridging http.HandlerFunc to gin.HandlerFunc via gin.WrapF.
func (r *GinRouter) POST(path string, handler http.HandlerFunc) {
	r.Engine.POST(path, gin.WrapF(handler))
}

// PUT registers a PUT route handler.
// Implements Router.PUT by bridging http.HandlerFunc to gin.HandlerFunc via gin.WrapF.
func (r *GinRouter) PUT(path string, handler http.HandlerFunc) {
	r.Engine.PUT(path, gin.WrapF(handler))
}

// PATCH registers a PATCH route handler.
// Implements Router.PATCH by bridging http.HandlerFunc to gin.HandlerFunc via gin.WrapF.
func (r *GinRouter) PATCH(path string, handler http.HandlerFunc) {
	r.Engine.PATCH(path, gin.WrapF(handler))
}

// DELETE registers a DELETE route handler.
// Implements Router.DELETE by bridging http.HandlerFunc to gin.HandlerFunc via gin.WrapF.
func (r *GinRouter) DELETE(path string, handler http.HandlerFunc) {
	r.Engine.DELETE(path, gin.WrapF(handler))
}

// Use adds standard net/http middleware to the engine.
// The middleware is adapted to Gin's handler chain via wrapGinMiddleware.
func (r *GinRouter) Use(middleware func(http.Handler) http.Handler) {
	r.Engine.Use(wrapGinMiddleware(middleware))
}

// Group creates a sub-router with a prefix for grouping related routes.
//
// Routes registered on the returned GinGroupRouter will be prefixed with the
// given path. The groupName is used for organizational purposes.
//
// Example:
//
//	jwtGroup := router.Group("/jwt", "JWT")
//	jwtGroup.POST("/token", tokenHandler)     // Mounted at /jwt/token
//	jwtGroup.POST("/refresh", refreshHandler) // Mounted at /jwt/refresh
//
// Implements Router.Group.
func (r *GinRouter) Group(path string, groupName string) aegisrouter.GroupRouter {
	return &GinGroupRouter{
		prefix:    path,
		groupName: groupName,
		group:     r.Engine.Group(path),
	}
}

// ServeHTTP handles HTTP requests by delegating to the underlying gin.Engine.
// Inherited from gin.Engine — satisfies the http.Handler interface.

// GinGroupRouter wraps gin.RouterGroup to provide route grouping functionality.
//
// Routes registered on a GinGroupRouter are automatically prefixed with
// the group's path by the underlying gin.RouterGroup.
//
// GinGroupRouter is created via GinRouter.Group() and should not be
// instantiated directly.
type GinGroupRouter struct {
	// prefix is the accumulated path prefix for this group
	prefix string

	// groupName is used for organizational purposes
	groupName string

	// group is the underlying gin router group
	group *gin.RouterGroup
}

// GET registers a GET route handler within this group.
func (g *GinGroupRouter) GET(path string, handler http.HandlerFunc) {
	g.group.GET(path, gin.WrapF(handler))
}

// POST registers a POST route handler within this group.
func (g *GinGroupRouter) POST(path string, handler http.HandlerFunc) {
	g.group.POST(path, gin.WrapF(handler))
}

// PUT registers a PUT route handler within this group.
func (g *GinGroupRouter) PUT(path string, handler http.HandlerFunc) {
	g.group.PUT(path, gin.WrapF(handler))
}

// PATCH registers a PATCH route handler within this group.
func (g *GinGroupRouter) PATCH(path string, handler http.HandlerFunc) {
	g.group.PATCH(path, gin.WrapF(handler))
}

// DELETE registers a DELETE route handler within this group.
func (g *GinGroupRouter) DELETE(path string, handler http.HandlerFunc) {
	g.group.DELETE(path, gin.WrapF(handler))
}

// Use adds standard net/http middleware to this group.
// The middleware is scoped to routes registered on this group.
func (g *GinGroupRouter) Use(middleware func(http.Handler) http.Handler) {
	g.group.Use(wrapGinMiddleware(middleware))
}

// Group creates a nested group under this group by combining path prefixes.
// The underlying gin.RouterGroup handles prefix concatenation automatically.
func (g *GinGroupRouter) Group(path string, groupName string) aegisrouter.GroupRouter {
	return &GinGroupRouter{
		prefix:    g.prefix + path,
		groupName: groupName,
		group:     g.group.Group(path),
	}
}

// wrapGinMiddleware converts a standard net/http middleware to a gin.HandlerFunc.
//
// The standard middleware receives a next handler that, when called, advances
// the gin handler chain via c.Next(). This preserves gin's handler chain
// while allowing standard http middleware to execute before and after it.
func wrapGinMiddleware(mw func(http.Handler) http.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	}
}
