package organizations

import (
	"context"
	"net/http"

	"github.com/theinventorylib/aegis/core"
	orgtypes "github.com/theinventorylib/aegis/plugins/organizations/types"
)

// orgRoleChecker is a function type for checking organization role requirements.
type orgRoleChecker func(ctx context.Context, userID, orgID string) (bool, error)

// requireOrganizationRole is a helper function that creates middleware for organization role checks.
// It handles the common logic for member, admin, and owner middleware.
func (p *Plugin) requireOrganizationRole(checker orgRoleChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := core.GetUser(r.Context())
			if err != nil {
				core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			orgID := core.GetSanitizedPathParam(r, "id")
			if orgID == "" {
				core.WriteJSONError(w, http.StatusBadRequest, "Organization ID required")
				return
			}

			hasRole, err := checker(r.Context(), user.ID, orgID)
			if err != nil || !hasRole {
				core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrgRole creates middleware that requires the authenticated user to have
// at least one of the specified org-level roles on the organization identified
// by the ":id" path parameter.
//
// Example:
//
//	r.GET("/organizations/:id/settings",
//	    requireAuth(
//	        plugin.RequireOrgRole(orgtypes.RoleOwner, orgtypes.RoleAdmin)(
//	            http.HandlerFunc(handler),
//	        ),
//	    ),
//	)
func (p *Plugin) RequireOrgRole(roles ...string) func(http.Handler) http.Handler {
	return p.requireOrganizationRole(func(ctx context.Context, userID, orgID string) (bool, error) {
		return p.caps.HasOrgRole(ctx, userID, orgID, roles...)
	})
}

// RequireOrgPermission creates middleware that requires the authenticated
// user's organization role to grant perm on the organization identified by the
// ":id" path parameter.
//
// Prefer this over RequireOrgRole so custom roles are honored: authority comes
// from the permission a role grants, not its name.
//
// Example:
//
//	r.PUT("/organizations/:id",
//	    requireAuth(plugin.RequireOrgPermission(PermOrgManage)(handler)),
//	)
func (p *Plugin) RequireOrgPermission(perm Permission) func(http.Handler) http.Handler {
	return p.requireOrganizationRole(func(ctx context.Context, userID, orgID string) (bool, error) {
		return p.HasOrgPermission(ctx, userID, orgID, perm)
	})
}

// RequireTeamPermission creates middleware that requires the authenticated
// user to be able to access the team (via CanAccessTeam) and to have a team
// role that grants perm.
func (p *Plugin) RequireTeamPermission(perm Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := core.GetUser(r.Context())
			if err != nil {
				core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			teamID := core.GetSanitizedPathParam(r, "teamId")
			if teamID == "" {
				core.WriteJSONError(w, http.StatusBadRequest, "Team ID required")
				return
			}

			canAccess, err := p.caps.CanAccessTeam(r.Context(), user.ID, teamID)
			if err != nil || !canAccess {
				core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
				return
			}

			hasPerm, err := p.HasTeamPermission(r.Context(), user.ID, teamID, perm)
			if err != nil || !hasPerm {
				core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireTeamRole creates middleware that requires the authenticated user to have
// at least one of the specified team-level roles on the team identified by the
// ":teamId" path parameter.
//
// Unlike RequireOrgRole, this middleware checks membership in the parent organization
// as well (via CanAccessTeam), so a user must be both an org member and a team member.
func (p *Plugin) RequireTeamRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := core.GetUser(r.Context())
			if err != nil {
				core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			teamID := core.GetSanitizedPathParam(r, "teamId")
			if teamID == "" {
				core.WriteJSONError(w, http.StatusBadRequest, "Team ID required")
				return
			}

			canAccess, err := p.caps.CanAccessTeam(r.Context(), user.ID, teamID)
			if err != nil || !canAccess {
				core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
				return
			}

			if len(roles) > 0 {
				hasRole, err := p.caps.HasTeamRole(r.Context(), user.ID, teamID, roles...)
				if err != nil || !hasRole {
					core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireOrganizationMemberMiddleware enforces organization membership.
//
// Deprecated: Use RequireOrgRole instead.
func (p *Plugin) RequireOrganizationMemberMiddleware() func(http.Handler) http.Handler {
	return p.RequireOrgRole(orgtypes.RoleOwner, orgtypes.RoleAdmin, orgtypes.RoleMember)
}

// RequireOrganizationAdminMiddleware enforces admin or owner privileges.
//
// Deprecated: Use RequireOrgRole instead.
func (p *Plugin) RequireOrganizationAdminMiddleware() func(http.Handler) http.Handler {
	return p.RequireOrgRole(orgtypes.RoleOwner, orgtypes.RoleAdmin)
}

// RequireOrganizationOwnerMiddleware enforces owner-only access.
func (p *Plugin) RequireOrganizationOwnerMiddleware() func(http.Handler) http.Handler {
	return p.RequireOrgRole(orgtypes.RoleOwner)
}
