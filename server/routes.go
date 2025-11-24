package server

import (
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// MountRoutes mounts all authentication routes to the router with auth middleware protection.
func MountRoutes(router Router, auth *core.AuthService, session *core.SessionService, prefix string) {
	handlers := NewHandlers(auth, session)

	// Create auth middleware for protected routes
	requireAuth := core.RequireAuthMiddleware(session)

	// Protected auth routes - require active session/cookie authentication
	router.POST(prefix+"/logout", requireAuth(http.HandlerFunc(handlers.LogoutHandler)).ServeHTTP)
	router.GET(prefix+"/user", requireAuth(http.HandlerFunc(handlers.UserHandler)).ServeHTTP)

	// Protected session management routes - require active session
	router.GET(prefix+"/session/validate", requireAuth(http.HandlerFunc(handlers.ValidateSessionHandler)).ServeHTTP)
	router.GET(prefix+"/sessions", requireAuth(http.HandlerFunc(handlers.ListSessionsHandler)).ServeHTTP)
	router.DELETE(prefix+"/sessions/:id", requireAuth(http.HandlerFunc(handlers.RevokeSessionHandler)).ServeHTTP)
	router.DELETE(prefix+"/sessions", requireAuth(http.HandlerFunc(handlers.RevokeAllSessionsHandler)).ServeHTTP)

	// Public route - refresh token is its own authentication
	router.POST(prefix+"/session/refresh", handlers.RefreshSessionHandler)
}
