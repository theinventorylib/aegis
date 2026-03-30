package admin

import (
	"net/http"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
)

// Admin plugin context keys for EnrichedUser extensions.
//
// These keys are used to store admin-specific data in the user context,
// making them available throughout the request lifecycle and in API responses.
//
// Simple field names are used - they become top-level JSON fields in responses.
const (
	// ExtKeyRole is the key for user role in EnrichedUser extensions.
	//
	// Available as:
	//   - In handlers: enriched.GetString("role")
	//   - In middleware: plugins.GetUserExtensionString(ctx, ExtKeyRole)
	//   - In JSON: {"role": "admin"}
	ExtKeyRole = "role"

	// ExtKeyPermissions is the key for user permissions in EnrichedUser extensions.
	//
	// Available as:
	//   - In handlers: enriched.GetStringSlice("permissions")
	//   - In JSON: {"permissions": [...]}
	//
	// Note: Currently not implemented - reserved for future use.
	ExtKeyPermissions = "permissions"

	// RoleAdmin is the default role for administrative users.
	RoleAdmin = "admin"
)

// EnrichUserMiddleware fetches the user's role and adds it to the EnrichedUser.
//
// This middleware should be used after RequireAuthMiddleware to add admin-specific
// data to the user context. The enriched data is automatically included in API responses.
//
// Enrichment Process:
//  1. Retrieve authenticated user from context
//  2. Fetch user's role from database
//  3. Add role to EnrichedUser via plugins.ExtendUser
//  4. Role becomes available as {"role": "admin"} in JSON responses
//
// Usage:
//
//	router.Use(auth.RequireAuthMiddleware())
//	router.Use(admin.EnrichUserMiddleware())
//
// The role is then accessible via:
//   - core.GetUserExtensionString(ctx, "role")
//   - JSON responses: {"user": {"id": "...", "role": "admin", ...}}
func (a *Plugin) EnrichUserMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, err := core.GetUser(ctx)
			if err != nil {
				// Not authenticated, skip enrichment
				next.ServeHTTP(w, r)
				return
			}

			// Fetch role from store and add to enriched user
			role, err := a.store.GetRole(ctx, user.ID)
			if err == nil && role != "" {
				plugins.ExtendUser(ctx, ExtKeyRole, role)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// checkUserRole verifies the user has the required role.
// It first checks the enriched user context, then falls back to database lookup.
// Returns true if the user has the required role, false otherwise.
func (a *Plugin) checkUserRole(ctx *http.Request, w http.ResponseWriter, requiredRole string) bool {
	user, err := core.GetUser(ctx.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return false
	}

	// First check if role is already in enriched user (avoid DB call)
	if role := plugins.GetUserExtensionString(ctx.Context(), ExtKeyRole); role != "" {
		if role != requiredRole {
			core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
			return false
		}
		return true
	}

	// Fallback: fetch from DB and enrich for future use
	role, err := a.store.GetRole(ctx.Context(), user.ID)
	if err != nil || role != requiredRole {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return false
	}

	// Enrich user for subsequent middleware/handlers
	plugins.ExtendUser(ctx.Context(), ExtKeyRole, role)
	return true
}

// RequireAdminMiddleware ensures the user has the 'admin' role.
//
// This middleware enforces admin-only access to protected routes. It checks for
// the admin role in two ways:
//  1. First checks EnrichedUser context (if already enriched) - fast path
//  2. Falls back to database lookup if not enriched - slow path
//
// Authentication Flow:
//  1. Retrieve authenticated user from context (via RequireAuthMiddleware)
//  2. Check if role is already in EnrichedUser ("role" extension)
//  3. If not found, fetch role from database
//  4. Verify role is "admin"
//  5. Enrich user context for subsequent handlers
//
// Usage:
//
//	adminRouter := router.NewGroup("/admin")
//	adminRouter.Use(auth.RequireAuthMiddleware())
//	adminRouter.Use(admin.RequireAdminMiddleware())
//
// Response Codes:
//   - 401 Unauthorized: User not authenticated
//   - 403 Forbidden: User authenticated but not admin
func (a *Plugin) RequireAdminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if a.checkUserRole(r, w, RoleAdmin) {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// RequireRoleMiddleware ensures the user has a specific role.
//
// This is a generalized version of RequireAdminMiddleware for custom role requirements.
// It follows the same enrichment pattern (check context first, then database).
//
// Usage:
//
//	// Require moderator role
//	router.Use(admin.RequireRoleMiddleware("moderator"))
//
// Parameters:
//   - requiredRole: Role string to check (e.g., "admin", "moderator", "editor")
//
// Response Codes:
//   - 401 Unauthorized: User not authenticated
//   - 403 Forbidden: User authenticated but lacks required role
func (a *Plugin) RequireRoleMiddleware(requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if a.checkUserRole(r, w, requiredRole) {
				next.ServeHTTP(w, r)
			}
		})
	}
}
