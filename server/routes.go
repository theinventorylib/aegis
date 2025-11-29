package server

import (
	"net/http"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/models"
)

// MountRoutes mounts all authentication routes to the router with auth middleware protection.
func MountRoutes(router Router, auth *core.AuthService, session *core.SessionService, prefix string) {
	handlers := NewHandlers(auth, session)

	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(session)

	// Protected auth routes - require active session/cookie authentication
	router.POST(prefix+"/logout", requireAuth(http.HandlerFunc(handlers.LogoutHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/logout",
		Summary:     "Logout",
		Description: "Invalidate the current session and log out the user",
		Tags:        []string{"Authentication"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Successfully logged out", Schema: "Success"},
			"401": {Description: "Not authenticated", Schema: "Error"},
		},
	})

	router.GET(prefix+"/user", requireAuth(http.HandlerFunc(handlers.UserHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/user",
		Summary:     "Get current user",
		Description: "Retrieve the currently authenticated user's information",
		Tags:        []string{"Authentication"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "User information", Schema: models.SchemaUser},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
		},
	})

	// Protected session management routes - require active session
	router.GET(prefix+"/session/validate", requireAuth(http.HandlerFunc(handlers.ValidateSessionHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/session/validate",
		Summary:     "Validate session",
		Description: "Check if the current session is valid and active",
		Tags:        []string{"Session"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Session is valid", Schema: models.SchemaSession},
			"401": {Description: "Session is invalid or expired", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/sessions", requireAuth(http.HandlerFunc(handlers.ListSessionsHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/sessions",
		Summary:     "List user sessions",
		Description: "Retrieve all active sessions for the current user",
		Tags:        []string{"Session"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "List of active sessions", Schema: models.SchemaSessionList},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
		},
	})

	router.DELETE(prefix+"/sessions/:id", requireAuth(http.HandlerFunc(handlers.RevokeSessionHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "DELETE",
		Path:        NormalizePathToOpenAPI(prefix + "/sessions/:id"),
		Summary:     "Revoke session",
		Description: "Revoke a specific session by ID",
		Tags:        []string{"Session"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Session revoked successfully", Schema: models.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"404": {Description: "Session not found", Schema: models.SchemaError},
		},
	})

	router.DELETE(prefix+"/sessions", requireAuth(http.HandlerFunc(handlers.RevokeAllSessionsHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "DELETE",
		Path:        prefix + "/sessions",
		Summary:     "Revoke all sessions",
		Description: "Revoke all active sessions for the current user",
		Tags:        []string{"Session"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "All sessions revoked successfully", Schema: models.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
		},
	})

	// Public route - refresh token is its own authentication
	router.POST(prefix+"/session/refresh", handlers.RefreshSessionHandler)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/session/refresh",
		Summary:     "Refresh session",
		Description: "Refresh an existing session using a refresh token",
		Tags:        []string{"Session"},
		Protected:   false,
		RequestBody: &models.RequestBodyMeta{
			Description: "Refresh token",
			Required:    true,
			Schema:      models.SchemaRefreshTokenRequest,
		},
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Session refreshed successfully", Schema: models.SchemaSession},
			"400": {Description: "Invalid request", Schema: models.SchemaError},
			"401": {Description: "Invalid or expired refresh token", Schema: models.SchemaError},
		},
	})
}
