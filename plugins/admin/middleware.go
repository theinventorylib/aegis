package admin

import (
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// Middleware ensures the user has the 'admin' role
func Middleware(sessionService *core.SessionService, db *DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Create the admin check handler
		adminCheck := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context (populated by AuthMiddleware)
			user, err := core.GetUser(r.Context())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check user role
			isAdmin, err := db.HasUserRole(r.Context(), user.ID, "admin")
			if err != nil || !isAdmin {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})

		// Wrap with AuthMiddleware to ensure context is populated
		return core.AuthMiddleware(sessionService)(adminCheck)
	}
}

// RequireRoleMiddleware ensures the user has a specific role
func RequireRoleMiddleware(sessionService *core.SessionService, db *DB, role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Create the role check handler
		roleCheck := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := core.GetUser(r.Context())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			hasRole, err := db.HasUserRole(r.Context(), user.ID, role)
			if err != nil || !hasRole {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})

		// Wrap with AuthMiddleware to ensure context is populated
		return core.AuthMiddleware(sessionService)(roleCheck)
	}
}

// RequireAdminMiddleware is a convenience wrapper for RequireRoleMiddleware("admin")
func RequireAdminMiddleware(sessionService *core.SessionService, db *DB) func(http.Handler) http.Handler {
	return RequireRoleMiddleware(sessionService, db, "admin")
}
