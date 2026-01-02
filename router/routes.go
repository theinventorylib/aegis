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
//	POST {prefix}/email/login - Authenticate with email+password
//	POST {prefix}/email/register - Register new account
//
// Protected Routes (require authentication):
//
//	POST {prefix}/logout - Invalidate session
//	GET  {prefix}/user - Get current user data
//	GET  {prefix}/session - Get current session info
//	GET  {prefix}/sessions - List all user sessions
//	DELETE {prefix}/sessions/:id - Revoke specific session
//	DELETE {prefix}/sessions - Revoke all sessions
//
// Public Routes:
//
//	POST {prefix}/session/refresh - Refresh session with refresh token
//
// Route Metadata:
//
// All routes are automatically registered with OpenAPI metadata (summary, description,
// request/response schemas). This metadata is used by the OpenAPI plugin to generate
// API documentation.
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
//	//   POST /auth/email/login
//	//   POST /auth/email/register
//	//   POST /auth/logout
//	//   GET  /auth/user
//	//   ... etc
func MountRoutes(router Router, authService *core.AuthService, config *core.AuthConfig, prefix string, rateLimiter *core.RateLimiter) {
	handlers := NewHandlers(authService)

	// Mount Email+Password routes if enabled
	if config != nil && config.EnableEmailPassword {
		emailHandlers := core.NewEmailPasswordHandlers(authService)

		router.POST(prefix+"/email/login", emailHandlers.LoginHandler)
		router.RegisterRouteMetadata(core.RouteMetadata{
			Method:      "POST",
			Path:        prefix + "/email/login",
			Summary:     "Login with email and password",
			Description: "Authenticate using email address and password",
			Tags:        []string{"Authentication"},
			Protected:   false,
			RequestBody: &core.RequestBodyMeta{
				Description: "Email and password credentials",
				Required:    true,
				Schema:      core.SchemaLoginRequest,
			},
			Responses: map[string]*core.ResponseMeta{
				"200": {Description: "Login successful, session created", Schema: core.SchemaSession},
				"400": {Description: "Invalid request", Schema: core.SchemaError},
				"401": {Description: "Invalid credentials", Schema: core.SchemaError},
			},
		})

		router.POST(prefix+"/email/register", emailHandlers.RegisterHandler)
		router.RegisterRouteMetadata(core.RouteMetadata{
			Method:      "POST",
			Path:        prefix + "/email/register",
			Summary:     "Register with email and password",
			Description: "Create a new account using email address and password",
			Tags:        []string{"Authentication"},
			Protected:   false,
			RequestBody: &core.RequestBodyMeta{
				Description: "Email and password credentials",
				Required:    true,
				Schema:      core.SchemaRegisterRequest,
			},
			Responses: map[string]*core.ResponseMeta{
				"201": {Description: "Registration successful, session created", Schema: core.SchemaSession},
				"400": {Description: "Invalid request or email already exists", Schema: core.SchemaError},
			},
		})
	}

	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(authService.Session)

	// Protected auth routes - require active session/cookie authentication
	router.POST(prefix+"/logout", requireAuth(http.HandlerFunc(handlers.LogoutHandler)).ServeHTTP)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/logout",
		Summary:     "Logout",
		Description: "Invalidate the current session and log out the user",
		Tags:        []string{"Authentication"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Successfully logged out", Schema: "Success"},
			"401": {Description: "Not authenticated", Schema: "Error"},
		},
	})

	router.GET(prefix+"/user", requireAuth(http.HandlerFunc(handlers.UserHandler)).ServeHTTP)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/user",
		Summary:     "Get current user",
		Description: "Retrieve the currently authenticated user's information",
		Tags:        []string{"Authentication"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "User information", Schema: core.SchemaUser},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
		},
	})

	// Protected session management routes - require active session
	router.GET(prefix+"/session", requireAuth(http.HandlerFunc(handlers.GetCurrentSessionHandler)).ServeHTTP)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/session",
		Summary:     "Get current session",
		Description: "Retrieve the current session information",
		Tags:        []string{"Session"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Current session information", Schema: core.SchemaSession},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
		},
	})

	router.GET(prefix+"/sessions", requireAuth(http.HandlerFunc(handlers.ListSessionsHandler)).ServeHTTP)
	router.RegisterRouteMetadata(core.RouteMetadata{
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

	router.DELETE(prefix+"/sessions/:id", requireAuth(http.HandlerFunc(handlers.RevokeSessionHandler)).ServeHTTP)
	router.RegisterRouteMetadata(core.RouteMetadata{
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

	router.DELETE(prefix+"/sessions", requireAuth(http.HandlerFunc(handlers.RevokeAllSessionsHandler)).ServeHTTP)
	router.RegisterRouteMetadata(core.RouteMetadata{
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
		router.POST(prefix+"/session/refresh", core.RateLimitMiddleware(rateLimiter)(http.HandlerFunc(handlers.RefreshSessionHandler)).ServeHTTP)
	} else {
		router.POST(prefix+"/session/refresh", handlers.RefreshSessionHandler)
	}
	router.RegisterRouteMetadata(core.RouteMetadata{
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
