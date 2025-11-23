package server

import (
	"github.com/theinventorylib/aegis/core"
)

// MountRoutes mounts all authentication routes to the router
func MountRoutes(router Router, auth *core.AuthService, session *core.SessionService, prefix string) {
	handlers := NewHandlers(auth, session)

	// Auth routes
	router.POST(prefix+"/auth/logout", handlers.LogoutHandler)
	router.GET(prefix+"/auth/user", handlers.UserHandler)

	// Session management routes
	router.POST(prefix+"/auth/session/refresh", handlers.RefreshSessionHandler)
	router.GET(prefix+"/auth/session/validate", handlers.ValidateSessionHandler)
	router.GET(prefix+"/auth/sessions", handlers.ListSessionsHandler)
	router.DELETE(prefix+"/auth/sessions/:id", handlers.RevokeSessionHandler)
	router.DELETE(prefix+"/auth/sessions", handlers.RevokeAllSessionsHandler)
}
