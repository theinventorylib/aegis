package router

import (
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// MountRoutes mounts all core Aegis authentication routes to the provided router.
//
// This function registers handlers for:
//
// Email+Password Routes (if config.EnableEmailPassword is true):
//
//	POST {prefix}/login - Authenticate with email+password
//	POST {prefix}/signup - Register new account
//
// Protected Routes (require authentication):
//
//	POST {prefix}/logout - Invalidate session
//	GET  {prefix}/session - Get current session with user data
//	GET  {prefix}/sessions - List all user sessions
//	DELETE {prefix}/sessions/:id - Revoke specific session
//	DELETE {prefix}/sessions - Revoke all sessions
//
// Public Routes:
//
//	POST {prefix}/session/refresh - Refresh session with refresh token
//
// Route Grouping:
//
// All core authentication routes are grouped under "default" for OpenAPI
// documentation. Session management routes are additionally tagged with "Session".
//
// Note: Route metadata for login/signup is always registered for OpenAPI documentation,
// but the actual handlers are only mounted when EnableEmailPassword is true.
// User data is returned with session endpoints, not as a separate /user endpoint.
//
// Rate Limiting:
//
// If a rateLimiter is provided, it's applied to the refresh endpoint to prevent
// token brute force attacks. Other routes can be rate-limited by plugins or
// custom middleware.
//
// Parameters:
//   - router: HTTP router implementing the Router interface
//   - authService: Core authentication service with all sub-services
//   - config: Authentication configuration (controls which routes are enabled)
//   - prefix: URL prefix for all routes (e.g., "/auth", "/api/v1/auth")
//   - rateLimiter: Optional rate limiter for public endpoints (can be nil)
//
// Example:
//
//	router := chi.NewRouter()
//	authService := core.NewAuthService(...)
//	config := &core.AuthConfig{EnableEmailPassword: true}
//
//	router.MountRoutes(router, authService, config, "/auth", nil)
//	// Routes mounted:
//	//   POST /auth/login
//	//   POST /auth/signup
//	//   POST /auth/logout
//	//   GET  /auth/user
//	//   ... etc
func MountRoutes(router Router, authService *core.AuthService, config *core.AuthConfig, prefix string, rateLimiter *core.RateLimiter) {
	handlers := NewHandlers(authService)

	// Create route group for core authentication routes
	authGroup := router.Group(prefix, "default")

	// Mount Email/Password routes
	authGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/login",
		Summary:     "Email/Password Login",
		Description: "Authenticate a user with email and password",
		Tags:        []string{"default"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Login credentials",
			Required:    true,
			Schema:      core.LoginRequest{},
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Login successful", Schema: core.SchemaSessionWithUser},
			"401": {Description: "Invalid credentials", Schema: core.SchemaError},
		},
	})

	authGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/signup",
		Summary:     "Email/Password Registration",
		Description: "Register a new user with email and password",
		Tags:        []string{"default"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Registration details",
			Required:    true,
			Schema:      core.RegisterRequest{},
		},
		Responses: map[string]*core.ResponseMeta{
			"201": {Description: "Registration successful", Schema: core.SchemaSessionWithUser},
			"400": {Description: "Invalid input or user already exists", Schema: core.SchemaError},
		},
	})

	// Mount Email/Password route handlers (conditionally based on config)
	if config == nil || config.EnableEmailPassword {
		authGroup.POST("/login", handlers.loginHandler)
		authGroup.POST("/signup", handlers.registerHandler)
	}

	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(authService.Session)

	// Protected auth routes - require active session/cookie authentication
	authGroup.POST("/logout", requireAuth(http.HandlerFunc(handlers.logoutHandler)).ServeHTTP)
	authGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/logout",
		Summary:     "Logout",
		Description: "Invalidate the current session and log out the user",
		Tags:        []string{"default"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Successfully logged out", Schema: "Success"},
			"401": {Description: "Not authenticated", Schema: "Error"},
		},
	})

	// Protected session management routes - require active session
	// Create a session group for better OpenAPI organization
	sessionGroup := router.Group(prefix, "Session")

	sessionGroup.GET("/session", requireAuth(http.HandlerFunc(handlers.getCurrentSessionHandler)).ServeHTTP)
	sessionGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/session",
		Summary:     "Get current session",
		Description: "Retrieve the current session information with user data",
		Tags:        []string{"Session"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Current session information", Schema: core.SchemaSessionWithUser},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
		},
	})

	sessionGroup.GET("/sessions", requireAuth(http.HandlerFunc(handlers.listSessionsHandler)).ServeHTTP)
	sessionGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/sessions",
		Summary:     "List user sessions",
		Description: "Retrieve all active sessions for the current user",
		Tags:        []string{"Session"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "List of active sessions", Schema: core.SchemaSessionList},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
		},
	})

	sessionGroup.DELETE("/sessions/:id", requireAuth(http.HandlerFunc(handlers.revokeSessionHandler)).ServeHTTP)
	sessionGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "DELETE",
		Path:        NormalizePathToOpenAPI(prefix + "/sessions/:id"),
		Summary:     "Revoke session",
		Description: "Revoke a specific session by ID",
		Tags:        []string{"Session"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Session revoked successfully", Schema: core.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"404": {Description: "Session not found", Schema: core.SchemaError},
		},
	})

	sessionGroup.DELETE("/sessions", requireAuth(http.HandlerFunc(handlers.revokeAllSessionsHandler)).ServeHTTP)
	sessionGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "DELETE",
		Path:        prefix + "/sessions",
		Summary:     "Revoke all sessions",
		Description: "Revoke all active sessions for the current user",
		Tags:        []string{"Session"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "All sessions revoked successfully", Schema: core.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
		},
	})

	// Public route - refresh token is its own authentication
	// Apply rate limiting if available to prevent token abuse
	if rateLimiter != nil {
		sessionGroup.POST("/session/refresh", core.RateLimitMiddleware(rateLimiter)(http.HandlerFunc(handlers.refreshSessionHandler)).ServeHTTP)
	} else {
		sessionGroup.POST("/session/refresh", handlers.refreshSessionHandler)
	}
	sessionGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/session/refresh",
		Summary:     "Refresh session",
		Description: "Refresh an existing session using a refresh token",
		Tags:        []string{"Session"},
		Protected:   false,
		RequestBody: &core.RequestBodyMeta{
			Description: "Refresh token",
			Required:    true,
			Schema:      core.SchemaRefreshTokenRequest,
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Session refreshed successfully", Schema: core.SchemaSession},
			"400": {Description: "Invalid request", Schema: core.SchemaError},
			"401": {Description: "Invalid or expired refresh token", Schema: core.SchemaError},
		},
	})
}
