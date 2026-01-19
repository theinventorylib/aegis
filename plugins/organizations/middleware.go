package organizations

import (
	"context"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// orgRoleChecker is a function type for checking organization role requirements.
type orgRoleChecker func(ctx context.Context, userID, orgID string) (bool, error)

// requireOrganizationRole is a helper function that creates middleware for organization role checks.
// It handles the common logic for member, admin, and owner middleware.
func (p *Plugin) requireOrganizationRole(checker orgRoleChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user from context (populated by AuthMiddleware)
			user, err := core.GetUser(r.Context())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			orgID := core.GetSanitizedPathParam(r, "id")
			if orgID == "" {
				http.Error(w, "Organization ID required", http.StatusBadRequest)
				return
			}

			// Check role using the provided checker function
			hasRole, err := checker(r.Context(), user.ID, orgID)
			if err != nil || !hasRole {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrganizationMemberMiddleware enforces organization membership.
//
// This middleware ensures the authenticated user is a member of the organization
// specified in the URL path parameter ":id". It allows any role (owner, admin, member).
//
// Use Cases:
//   - Viewing organization details
//   - Listing organization members
//   - Viewing teams
//
// Request Flow:
//  1. Get authenticated user from context (set by RequireAuthMiddleware)
//  2. Extract organization ID from path parameter ":id"
//  3. Check if user is a member (any role)
//  4. If yes → continue to handler, if no → 403 Forbidden
//
// Example:
//
//	r.GET("/organizations/:id",
//	    requireAuth(
//	        plugin.RequireOrganizationMemberMiddleware()(
//	            http.HandlerFunc(handler),
//	        ),
//	    ),
//	)
func (p *Plugin) RequireOrganizationMemberMiddleware() func(http.Handler) http.Handler {
	return p.requireOrganizationRole(p.store.IsOrganizationMember)
}

// RequireOrganizationAdminMiddleware enforces admin or owner privileges.
//
// This middleware ensures the authenticated user has administrative privileges
// (owner or admin role) in the organization. Regular members are denied access.
//
// Use Cases:
//   - Adding/removing organization members
//   - Creating/deleting teams
//   - Updating organization settings
//
// Permission Requirements:
//   - Owner role: Allowed ✓
//   - Admin role: Allowed ✓
//   - Member role: Denied ✗
//
// Request Flow:
//  1. Get authenticated user from context
//  2. Extract organization ID from path parameter ":id"
//  3. Check if user has owner OR admin role
//  4. If yes → continue, if no → 403 Forbidden
func (p *Plugin) RequireOrganizationAdminMiddleware() func(http.Handler) http.Handler {
	return p.requireOrganizationRole(p.store.IsOwnerOrAdmin)
}

// RequireOrganizationOwnerMiddleware enforces owner-only access.
//
// This middleware ensures the authenticated user is the organization owner.
// This is the highest privilege level and is required for destructive operations.
//
// Use Cases:
//   - Deleting organization (permanent)
//   - Transferring ownership
//   - Changing other members' roles to admin
//
// Permission Requirements:
//   - Owner role: Allowed ✓
//   - Admin role: Denied ✗
//   - Member role: Denied ✗
//
// Request Flow:
//  1. Get authenticated user from context
//  2. Extract organization ID from path parameter ":id"
//  3. Check if user has owner role (exact match)
//  4. If yes → continue, if no → 403 Forbidden
//
// Best Practice:
// Only one owner per organization is recommended. Multiple owners complicate
// permission management and deletion workflows.
func (p *Plugin) RequireOrganizationOwnerMiddleware() func(http.Handler) http.Handler {
	return p.requireOrganizationRole(p.store.IsOwner)
}
