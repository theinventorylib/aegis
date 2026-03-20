// Package defaults provides the core Aegis route mounting with OpenAPI documentation.
//
// This package exists as a sub-package of router to break the circular import
// between router and plugins/openapi:
//
//	defaults → router       (for Router interface and MountRoutes)
//	defaults → plugins/openapi (for Doc/Route types)
//	plugins/openapi → router  (for MountRoutes Router param)
//
// No cycle: router does NOT import defaults, plugins/openapi does NOT import defaults.
package defaults

import (
	"net/http"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins/openapi"
	"github.com/theinventorylib/aegis/router"
)

// MountRoutes mounts all core Aegis authentication routes and registers
// their OpenAPI documentation.
//
// This function:
//  1. Registers HTTP handlers directly
//  2. Calls openapi.Doc() for each route to register API documentation
//
// Routes mounted:
//
//	POST   {prefix}/login           - Email+password login (if enabled)
//	POST   {prefix}/signup          - Email+password registration (if enabled)
//	POST   {prefix}/logout          - Invalidate session (auth required)
//	GET    {prefix}/session         - Get current session with user data (auth required)
//	GET    {prefix}/sessions        - List all user sessions (auth required)
//	DELETE {prefix}/sessions/{id}   - Revoke specific session (auth required)
//	DELETE {prefix}/sessions        - Revoke all sessions (auth required)
//	POST   {prefix}/session/refresh - Refresh session with refresh token
//
// Parameters:
//   - r: HTTP router implementing the Router interface
//   - authService: Core authentication service with all sub-services
//   - config: Authentication configuration (controls which routes are enabled)
//   - prefix: URL prefix for all routes (e.g., "/auth/default")
//   - rateLimiter: Optional rate limiter for public endpoints (can be nil)
func MountRoutes(r router.Router, authService *core.AuthService, config *core.AuthConfig, prefix string, rateLimiter *core.RateLimiter) {
	handlers := NewHandlers(authService)

	// Create route groups for organization
	authGroup := r.Group(prefix, "Default")
	sessionGroup := r.Group(prefix, "Session")

	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(authService.Session)

	// ── Email+Password Routes (conditionally based on config) ──

	if config == nil || config.EnableEmailPassword {
		// Login
		authGroup.POST("/login", handlers.loginHandler)
		openapi.Doc(openapi.Route{
			Method:      "POST",
			Path:        prefix + "/login",
			Summary:     "Email Login",
			Description: "Authenticate a user with email and password",
			Tags:        []string{"Default"},
			Body:        openapi.BodyOf[core.LoginRequest](),
			Responses: openapi.Responses{
				200: openapi.DataResponseOf[core.SessionWithUser]("Login successful, session created"),
				401: openapi.RefResponse("Invalid credentials", "Error"),
			},
		})

		// Signup
		authGroup.POST("/signup", handlers.registerHandler)
		openapi.Doc(openapi.Route{
			Method:      "POST",
			Path:        prefix + "/signup",
			Summary:     "Email Registration",
			Description: "Register a new user with email and password",
			Tags:        []string{"Default"},
			Body:        openapi.BodyOf[core.RegisterRequest](),
			Responses: openapi.Responses{
				201: openapi.DataResponseOf[core.SessionWithUser]("Registration successful, session created"),
				400: openapi.RefResponse("Invalid input or user already exists", "Error"),
			},
		})
	}

	// ── Protected Routes (require active session) ──

	// Logout
	authGroup.POST("/logout", requireAuth(http.HandlerFunc(handlers.logoutHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/logout",
		Summary:     "Logout",
		Description: "Invalidate the current session and log out the user",
		Tags:        []string{"Default"},
		Auth:        true,
		Responses: openapi.Responses{
			200: openapi.RefResponse("Successfully logged out", "Success"),
			401: openapi.RefResponse("Not authenticated", "Error"),
		},
	})

	// Get current session
	sessionGroup.GET("/session", requireAuth(http.HandlerFunc(handlers.getCurrentSessionHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/session",
		Summary:     "Get current session",
		Description: "Retrieve the current session information with user data",
		Tags:        []string{"Session"},
		Auth:        true,
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[core.SessionWithUser]("Current session information with user data"),
			401: openapi.RefResponse("Not authenticated", "Error"),
		},
	})

	// List sessions
	sessionGroup.GET("/sessions", requireAuth(http.HandlerFunc(handlers.listSessionsHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/sessions",
		Summary:     "List user sessions",
		Description: "Retrieve all active sessions for the current user",
		Tags:        []string{"Session"},
		Auth:        true,
		Responses: openapi.Responses{
			200: openapi.PaginatedResponseOf[core.PaginatedResponse[auth.Session]]("List of active sessions"),
			401: openapi.RefResponse("Not authenticated", "Error"),
		},
	})

	// Revoke session
	sessionGroup.DELETE("/sessions/:id", requireAuth(http.HandlerFunc(handlers.revokeSessionHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "DELETE",
		Path:        prefix + "/sessions/{id}",
		Summary:     "Revoke session",
		Description: "Revoke a specific session by ID",
		Tags:        []string{"Session"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.RefResponse("Session revoked successfully", "Success"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			404: openapi.RefResponse("Session not found", "Error"),
		},
	})

	// Revoke all sessions
	sessionGroup.DELETE("/sessions", requireAuth(http.HandlerFunc(handlers.revokeAllSessionsHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "DELETE",
		Path:        prefix + "/sessions",
		Summary:     "Revoke all sessions",
		Description: "Revoke all active sessions for the current user",
		Tags:        []string{"Session"},
		Auth:        true,
		Responses: openapi.Responses{
			200: openapi.RefResponse("All sessions revoked successfully", "Success"),
			401: openapi.RefResponse("Not authenticated", "Error"),
		},
	})

	// ── Public Routes ──

	// Refresh session (rate limited if available)
	if rateLimiter != nil {
		sessionGroup.POST("/session/refresh", core.RateLimitMiddleware(rateLimiter)(http.HandlerFunc(handlers.refreshSessionHandler)).ServeHTTP)
	} else {
		sessionGroup.POST("/session/refresh", handlers.refreshSessionHandler)
	}
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/session/refresh",
		Summary:     "Refresh session",
		Description: "Refresh an existing session using a refresh token",
		Tags:        []string{"Session"},
		Body:        openapi.RefBody("RefreshTokenRequest"),
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[core.SessionRefreshResponse]("Session refreshed successfully"),
			400: openapi.RefResponse("Invalid request", "Error"),
			401: openapi.RefResponse("Invalid or expired refresh token", "Error"),
		},
	})
}
