package router

import (
	"net/http"
)

// Router is the interface for HTTP routers in Aegis.
//
// This abstraction allows Aegis to work with different HTTP routing libraries
// (chi, mux, gin, echo, etc.) without coupling to a specific implementation.
//
// Implementations must support:
//   - Standard HTTP methods (GET, POST, PUT, PATCH, DELETE)
//   - Middleware chain using standard http.Handler interface
//   - http.Handler compliance for ServeHTTP
//   - Route grouping for organizing related routes
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

	// Group creates a sub-router with a prefix for grouping related routes.
	// Routes mounted to the returned GroupRouter will be prefixed with the given path.
	// The groupName is used for organizational purposes.
	//
	// Example:
	//   jwtGroup := router.Group("/jwt", "JWT")
	//   jwtGroup.POST("/token", tokenHandler)     // Mounted at /jwt/token
	//   jwtGroup.POST("/refresh", refreshHandler) // Mounted at /jwt/refresh
	Group(path string, groupName string) GroupRouter
}

// GroupRouter represents a sub-router for grouping related routes.
//
// GroupRouter provides the same HTTP method registration as Router,
// but routes are prefixed with the group's path.
//
// Example:
//
//	adminGroup := router.Group("/admin", "Admin")
//	adminGroup.Use(requireAdminMiddleware)
//	adminGroup.GET("/users", listUsersHandler)
//	adminGroup.DELETE("/users/:id", deleteUserHandler)
type GroupRouter interface {
	// GET registers a GET route handler within this group
	GET(path string, handler http.HandlerFunc)

	// POST registers a POST route handler within this group
	POST(path string, handler http.HandlerFunc)

	// PUT registers a PUT route handler within this group
	PUT(path string, handler http.HandlerFunc)

	// PATCH registers a PATCH route handler within this group
	PATCH(path string, handler http.HandlerFunc)

	// DELETE registers a DELETE route handler within this group
	DELETE(path string, handler http.HandlerFunc)

	// Use adds middleware to this group only
	// Middleware is scoped to this group and does not affect parent or sibling routes
	Use(middleware func(http.Handler) http.Handler)

	// Group creates a nested sub-group within this group.
	// Nested groups inherit the parent's prefix and allow further
	// organization of routes.
	Group(path string, groupName string) GroupRouter
}
