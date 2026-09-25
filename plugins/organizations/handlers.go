package organizations

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/theinventorylib/aegis/v2/core"
	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

// ========== ORGANIZATION HANDLERS ==========
//
// These HTTP handlers implement organization CRUD operations with role-based
// access control. All handlers require authentication via RequireAuthMiddleware.
//
// Permission Requirements:
//   - Create: Any authenticated user
//   - List/Get: Organization member (any role)
//   - Update: Admin or owner
//   - Delete: Owner only

// validateOrgAccess validates user authentication and organization membership.
//
// This helper method checks if the authenticated user has access to the organization
// specified in the URL path. It's used by handlers that require member-level access.
//
// Parameters:
//   - w: HTTP response writer
//   - r: HTTP request with user context and path parameter ":id"
//
// Returns:
//   - orgID: Organization ID from path if valid
//   - ok: true if user is authenticated and is organization member
func (p *Plugin) validateOrgAccess(w http.ResponseWriter, r *http.Request) (orgID string, ok bool) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return "", false
	}

	orgID = core.GetSanitizedPathParam(r, "id")
	if orgID == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Organization ID required")
		return "", false
	}

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermOrgView) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return "", false
	}

	return orgID, true
}

// CreateOrganizationHandler creates a new organization with the user as owner.
//
// This endpoint allows any authenticated user to create an organization. The creator
// is automatically assigned the "owner" role with full administrative privileges.
//
// Endpoint:
//   - Method: POST
//   - Path: /organizations
//   - Auth: Required (any authenticated user)
//
// Request Body:
//
//	{
//	  "name": "Acme Corporation",
//	  "slug": "acme-corp"
//	}
//
// Validation:
//   - name: Required, 1-100 characters
//   - slug: Required, 3-50 characters, lowercase alphanumeric with hyphens only
//
// Response (201 Created):
//
//	{
//	  "success": true,
//	  "message": "Organization created successfully",
//	  "data": {
//	    "id": "org_abc123",
//	    "name": "Acme Corporation",
//	    "slug": "acme-corp",
//	    "createdAt": "2024-01-01T00:00:00Z",
//	    "updatedAt": "2024-01-01T00:00:00Z"
//	  }
//	}
func (p *Plugin) CreateOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateOrganizationRequest

	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize inputs
	req.Name = core.SanitizeString(req.Name, nil)
	req.Slug = core.SanitizeString(req.Slug, nil)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	org, err := p.CreateOrganization(r.Context(), req.Name, req.Slug, user.ID)
	if err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Message: "Organization created successfully",
		Data:    org,
	})
}

// ListOrganizationsHandler lists all organizations the user is a member of.
//
// This endpoint returns all organizations where the user has any membership
// (owner, admin, or member role).
//
// Endpoint:
//   - Method: GET
//   - Path: /organizations
//   - Auth: Required
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "items": [
//	    {"id": "org_1", "name": "Acme Corp", "slug": "acme", ...},
//	    {"id": "org_2", "name": "Tech Inc", "slug": "tech", ...}
//	  ],
//	  "totalCount": 10,
//	  "page": 1,
//	  "offset": 0,
//	  "limit": 20
//	}
func (p *Plugin) ListOrganizationsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	pagination := core.ParsePagination(r)

	orgs, totalCount, err := p.GetUserOrganizations(r.Context(), user.ID, pagination.Offset, pagination.Limit)
	if err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[*orgtypes.Organization]{
		Items:      orgs,
		TotalCount: totalCount,
		Page:       pagination.Page,
		Offset:     pagination.Offset,
		Limit:      pagination.Limit,
	})
}

// GetOrganizationHandler retrieves details of a specific organization.
//
// This endpoint returns organization metadata. Requires membership in the organization.
//
// Endpoint:
//   - Method: GET
//   - Path: /organizations/:id
//   - Auth: Required (must be organization member)
//
// Path Parameters:
//   - id: Organization ID
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Organization retrieved successfully",
//	  "data": {
//	    "id": "org_abc123",
//	    "name": "Acme Corporation",
//	    "slug": "acme-corp",
//	    "createdAt": "2024-01-01T00:00:00Z",
//	    "updatedAt": "2024-01-01T00:00:00Z"
//	  }
//	}
func (p *Plugin) GetOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	orgID, ok := p.validateOrgAccess(w, r)
	if !ok {
		return
	}

	org, err := p.GetOrganization(r.Context(), orgID)
	if err != nil {
		core.WriteJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Organization retrieved successfully",
		Data:    org,
	})
}

// UpdateOrganizationHandler updates an organization
func (p *Plugin) UpdateOrganizationHandler(w http.ResponseWriter, r *http.Request) {
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

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermOrgManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	var req UpdateOrganizationRequest

	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize inputs
	req.Name = core.SanitizeString(req.Name, nil)
	req.Slug = core.SanitizeString(req.Slug, nil)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	if err := p.UpdateOrganization(r.Context(), orgID, req.Name, req.Slug); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Organization updated",
	})
}

// DeleteOrganizationHandler deletes an organization
func (p *Plugin) DeleteOrganizationHandler(w http.ResponseWriter, r *http.Request) {
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

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermOrgDelete) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Owner role required")
		return
	}

	if err := p.DeleteOrganization(r.Context(), orgID); err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Organization deleted",
	})
}

// ========== ORGANIZATION MEMBER HANDLERS ==========

// AddOrganizationMemberHandler adds a member to an organization
func (p *Plugin) AddOrganizationMemberHandler(w http.ResponseWriter, r *http.Request) {
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

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermMemberManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	var req AddOrganizationMemberRequest

	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize inputs
	req.UserID = core.SanitizeString(req.UserID, nil)
	req.Role = core.SanitizeString(req.Role, nil)

	if err := p.ValidateAddMember(r.Context(), orgID, req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	if req.Role == "owner" {
		core.WriteJSONError(w, http.StatusBadRequest, "Cannot assign owner role")
		return
	}

	if err := p.AddOrganizationMember(r.Context(), orgID, req.UserID, req.Role); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Message: "Member added to organization",
	})
}

// ListOrganizationMembersHandler lists organization members.
func (p *Plugin) ListOrganizationMembersHandler(w http.ResponseWriter, r *http.Request) {
	orgID, ok := p.validateOrgAccess(w, r)
	if !ok {
		return
	}

	pagination := core.ParsePagination(r)

	members, totalCount, err := p.ListOrganizationMembers(r.Context(), orgID, pagination.Offset, pagination.Limit)
	if err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[*orgtypes.Member]{
		Items:      members,
		TotalCount: totalCount,
		Page:       pagination.Page,
		Offset:     pagination.Offset,
		Limit:      pagination.Limit,
	})
}

// UpdateMemberRoleHandler updates a member's role
func (p *Plugin) UpdateMemberRoleHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	orgID := core.GetSanitizedPathParam(r, "id")
	userID := core.GetSanitizedPathParam(r, "userId")

	if orgID == "" || userID == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Organization ID and User ID required")
		return
	}

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermMemberAssignRoles) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Owner role required")
		return
	}

	// Ownership must not be demotable: an admin (who also grants
	// PermMemberAssignRoles) could otherwise seize the organization by
	// changing the owner's role. Ownership transfer needs its own flow.
	if p.IsOwner(r.Context(), userID, orgID) {
		core.WriteJSONError(w, http.StatusBadRequest, "Cannot change the owner's role")
		return
	}

	var req UpdateMemberRoleRequest

	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize inputs
	req.Role = core.SanitizeString(req.Role, nil)

	if err := p.ValidateUpdateMemberRole(r.Context(), orgID, req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	if req.Role == "owner" {
		core.WriteJSONError(w, http.StatusBadRequest, "Cannot assign owner role")
		return
	}

	if err := p.UpdateMemberRole(r.Context(), orgID, userID, req.Role); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Role updated",
	})
}

// GetMemberPermissionsHandler returns a member's role, overrides and resolved
// permission set.
func (p *Plugin) GetMemberPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	orgID := core.GetSanitizedPathParam(r, "id")
	userID := core.GetSanitizedPathParam(r, "userId")
	if orgID == "" || userID == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Organization ID and User ID required")
		return
	}
	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermMemberAssignRoles) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return
	}

	perms, err := p.GetMemberPermissions(r.Context(), userID, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			core.WriteJSONError(w, http.StatusNotFound, "Member not found")
			return
		}
		core.WriteJSONError(w, http.StatusInternalServerError, "Failed to load permissions")
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Data: perms})
}

// UpdateMemberPermissionsHandler replaces a member's permission overrides.
//
// The set is replaced, not merged: delete-then-insert. A failure part-way
// through leaves the member with a partial set, which the caller can correct
// by re-sending the desired list.
func (p *Plugin) UpdateMemberPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	orgID := core.GetSanitizedPathParam(r, "id")
	userID := core.GetSanitizedPathParam(r, "userId")
	if orgID == "" || userID == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Organization ID and User ID required")
		return
	}
	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermMemberAssignRoles) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return
	}

	var req UpdateMemberPermissionsRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	// Duplicate permissions would violate the (org, user, permission) unique
	// constraint part-way through the insert loop; reject them up front.
	seen := make(map[string]bool, len(req.Overrides))
	overrides := make([]orgtypes.MemberPermissionOverride, 0, len(req.Overrides))
	now := time.Now()
	for _, o := range req.Overrides {
		perm := core.SanitizeString(o.Permission, nil)
		if seen[perm] {
			core.WriteJSONError(w, http.StatusBadRequest, "Duplicate permission: "+perm)
			return
		}
		seen[perm] = true
		overrides = append(overrides, orgtypes.MemberPermissionOverride{
			ID:             core.GenerateID(),
			OrganizationID: orgID,
			UserID:         userID,
			Permission:     perm,
			Effect:         orgtypes.PermissionEffect(o.Effect),
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	// The member must exist before we touch overrides.
	if _, err := p.store.GetMember(r.Context(), userID, orgID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			core.WriteJSONError(w, http.StatusNotFound, "Member not found")
			return
		}
		core.WriteJSONError(w, http.StatusInternalServerError, "Failed to load member")
		return
	}

	if err := p.store.DeleteMemberPermissionOverrides(r.Context(), orgID, userID); err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Failed to clear permissions")
		return
	}
	for _, override := range overrides {
		if err := p.store.CreateMemberPermissionOverride(r.Context(), override); err != nil {
			core.WriteJSONError(w, http.StatusInternalServerError, "Failed to save permissions")
			return
		}
	}

	perms, err := p.GetMemberPermissions(r.Context(), userID, orgID)
	if err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Failed to load permissions")
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Message: "Permissions updated", Data: perms})
}

// ListRolesHandler returns the compiled and custom roles for the organization.
func (p *Plugin) ListRolesHandler(w http.ResponseWriter, r *http.Request) {
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
	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermOrgView) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return
	}
	roles, err := p.ListRoles(r.Context(), orgID)
	if err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Failed to list roles")
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Data: roles})
}

// CreateRoleHandler creates a custom organization role.
func (p *Plugin) CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
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
	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermOrgManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return
	}

	var req CreateRoleRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	info, err := p.CreateRole(r.Context(), orgID, req.Name, permissionsFromStrings(req.Permissions))
	switch {
	case errors.Is(err, ErrRoleReserved):
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, ErrRoleExists):
		core.WriteJSONError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		core.WriteJSONError(w, http.StatusInternalServerError, "Failed to create role")
		return
	}
	core.WriteJSON(w, http.StatusCreated, &core.Response{Success: true, Data: info})
}

// UpdateRoleHandler replaces a custom role's permissions.
func (p *Plugin) UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	orgID := core.GetSanitizedPathParam(r, "id")
	name := core.GetSanitizedPathParam(r, "name")
	if orgID == "" || name == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Organization ID and role name required")
		return
	}
	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermOrgManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return
	}

	var req UpdateRoleRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	info, err := p.UpdateRole(r.Context(), orgID, name, permissionsFromStrings(req.Permissions))
	switch {
	case errors.Is(err, ErrRoleReserved):
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, sql.ErrNoRows):
		core.WriteJSONError(w, http.StatusNotFound, "Role not found")
		return
	case err != nil:
		core.WriteJSONError(w, http.StatusInternalServerError, "Failed to update role")
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Data: info})
}

// DeleteRoleHandler removes a custom role that is not in use.
func (p *Plugin) DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	orgID := core.GetSanitizedPathParam(r, "id")
	name := core.GetSanitizedPathParam(r, "name")
	if orgID == "" || name == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Organization ID and role name required")
		return
	}
	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermOrgManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return
	}

	err = p.DeleteRole(r.Context(), orgID, name)
	switch {
	case errors.Is(err, ErrRoleReserved):
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, ErrRoleInUse):
		core.WriteJSONError(w, http.StatusConflict, err.Error())
		return
	case errors.Is(err, sql.ErrNoRows):
		core.WriteJSONError(w, http.StatusNotFound, "Role not found")
		return
	case err != nil:
		core.WriteJSONError(w, http.StatusInternalServerError, "Failed to delete role")
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Message: "Role deleted"})
}

// RemoveOrganizationMemberHandler removes a member from an organization
func (p *Plugin) RemoveOrganizationMemberHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	orgID := core.GetSanitizedPathParam(r, "id")
	userID := core.GetSanitizedPathParam(r, "userId")

	if orgID == "" || userID == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Organization ID and User ID required")
		return
	}

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermMemberManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	if p.IsOwner(r.Context(), userID, orgID) {
		core.WriteJSONError(w, http.StatusBadRequest, "Cannot remove owner")
		return
	}

	if err := p.RemoveOrganizationMember(r.Context(), orgID, userID); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Member removed from organization",
	})
}

// ========== TEAM HANDLERS ==========

// CreateTeamHandler creates a new team within an organization
func (p *Plugin) CreateTeamHandler(w http.ResponseWriter, r *http.Request) {
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

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermTeamManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	var req CreateTeamRequest

	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize inputs
	req.Name = core.SanitizeString(req.Name, nil)
	req.Description = core.SanitizeMultiline(req.Description, 500)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	team, err := p.CreateTeam(r.Context(), orgID, req.Name, req.Description)
	if err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Message: "Team created successfully",
		Data:    team,
	})
}

// ListTeamsHandler lists teams in an organization.
func (p *Plugin) ListTeamsHandler(w http.ResponseWriter, r *http.Request) {
	orgID, ok := p.validateOrgAccess(w, r)
	if !ok {
		return
	}

	pagination := core.ParsePagination(r)

	teams, totalCount, err := p.ListTeams(r.Context(), orgID, pagination.Offset, pagination.Limit)
	if err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[*orgtypes.Team]{
		Items:      teams,
		TotalCount: totalCount,
		Page:       pagination.Page,
		Offset:     pagination.Offset,
		Limit:      pagination.Limit,
	})
}

// GetTeamHandler gets a specific team
func (p *Plugin) GetTeamHandler(w http.ResponseWriter, r *http.Request) {
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

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		core.WriteJSONError(w, http.StatusNotFound, "Team not found")
		return
	}

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, team.OrganizationID, PermTeamView) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Team retrieved successfully",
		Data:    team,
	})
}

// UpdateTeamHandler updates a team
func (p *Plugin) UpdateTeamHandler(w http.ResponseWriter, r *http.Request) {
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

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		core.WriteJSONError(w, http.StatusNotFound, "Team not found")
		return
	}

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, team.OrganizationID, PermTeamManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	var req UpdateTeamRequest

	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize inputs
	req.Name = core.SanitizeString(req.Name, nil)
	req.Description = core.SanitizeMultiline(req.Description, 500)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	if err := p.UpdateTeam(r.Context(), teamID, req.Name, req.Description); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Team updated",
	})
}

// DeleteTeamHandler deletes a team
func (p *Plugin) DeleteTeamHandler(w http.ResponseWriter, r *http.Request) {
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

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		core.WriteJSONError(w, http.StatusNotFound, "Team not found")
		return
	}

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, team.OrganizationID, PermTeamManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	if err := p.DeleteTeam(r.Context(), teamID); err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Team deleted",
	})
}

// ========== TEAM MEMBER HANDLERS ==========

// AddTeamMemberHandler adds a member to a team
func (p *Plugin) AddTeamMemberHandler(w http.ResponseWriter, r *http.Request) {
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

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		core.WriteJSONError(w, http.StatusNotFound, "Team not found")
		return
	}

	if !p.canManageTeamMembers(r.Context(), user.ID, teamID, team.OrganizationID) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	var req AddTeamMemberRequest

	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize inputs
	req.UserID = core.SanitizeString(req.UserID, nil)
	req.Role = core.SanitizeString(req.Role, nil)

	if err := p.ValidateAddTeamMember(req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	// User must be organization member before joining team
	if !p.IsOrganizationMember(r.Context(), req.UserID, team.OrganizationID) {
		core.WriteJSONError(w, http.StatusBadRequest, "User must be organization member first")
		return
	}

	if err := p.AddTeamMember(r.Context(), teamID, req.UserID, req.Role); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Message: "Member added to team",
	})
}

// ListTeamMembersHandler lists team members
func (p *Plugin) ListTeamMembersHandler(w http.ResponseWriter, r *http.Request) {
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

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		core.WriteJSONError(w, http.StatusNotFound, "Team not found")
		return
	}

	if !p.hasOrgPermissionForUser(r.Context(), user.ID, team.OrganizationID, PermTeamView) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden")
		return
	}

	pagination := core.ParsePagination(r)

	members, totalCount, err := p.ListTeamMembers(r.Context(), teamID, pagination.Offset, pagination.Limit)
	if err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[*orgtypes.TeamMember]{
		Items:      members,
		TotalCount: totalCount,
		Page:       pagination.Page,
		Offset:     pagination.Offset,
		Limit:      pagination.Limit,
	})
}

// UpdateTeamMemberRoleHandler updates a team member's role
func (p *Plugin) UpdateTeamMemberRoleHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	teamID := core.GetSanitizedPathParam(r, "teamId")
	userID := core.GetSanitizedPathParam(r, "userId")

	if teamID == "" || userID == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Team ID and User ID required")
		return
	}

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		core.WriteJSONError(w, http.StatusNotFound, "Team not found")
		return
	}

	if !p.canManageTeamMembers(r.Context(), user.ID, teamID, team.OrganizationID) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	var req UpdateTeamMemberRoleRequest

	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Sanitize inputs
	req.Role = core.SanitizeString(req.Role, nil)

	if err := p.ValidateUpdateTeamMemberRole(req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	if err := p.UpdateTeamMemberRole(r.Context(), teamID, userID, req.Role); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Team member role updated",
	})
}

// RemoveTeamMemberHandler removes a member from a team
func (p *Plugin) RemoveTeamMemberHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	teamID := core.GetSanitizedPathParam(r, "teamId")
	userID := core.GetSanitizedPathParam(r, "userId")

	if teamID == "" || userID == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Team ID and User ID required")
		return
	}

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		core.WriteJSONError(w, http.StatusNotFound, "Team not found")
		return
	}

	if !p.canManageTeamMembers(r.Context(), user.ID, teamID, team.OrganizationID) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	if err := p.RemoveTeamMember(r.Context(), teamID, userID); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Message: "Member removed from team"})
}

// ========== INVITATION HANDLERS ==========
//
// These HTTP handlers implement invitation creation, listing, cancellation,
// acceptance, decline, and verification.
//
// Permission Requirements:
//   - Create/List/Cancel: Admin or owner of the organization
//   - Accept/Decline/Verify: No auth required (token-based)

// CreateInvitationHandler creates a new invitation.
//
// This handler accepts both org-level and team-level invitation requests.
// For org-level invites, use POST /organizations/:id/invitations.
// For team-level invites, use POST /teams/:teamId/invitations.
//
// Endpoint:
//   - Method: POST
//   - Path: /organizations/:id/invitations or /teams/:teamId/invitations
//   - Auth: Required (must be admin or owner)
//
// Request Body:
//
//	{
//	  "email": "user@example.com",
//	  "role": "member",
//	  "teamId": "team_abc123"
//	}
func (p *Plugin) CreateInvitationHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Determine org ID from path — can be under /organizations/:id or /teams/:teamId
	orgID := core.GetSanitizedPathParam(r, "id")
	var teamID *string

	if orgID == "" {
		// Team-level invitation: get team and its org
		tid := core.GetSanitizedPathParam(r, "teamId")
		if tid == "" {
			core.WriteJSONError(w, http.StatusBadRequest, "Organization ID or Team ID required")
			return
		}
		team, err := p.GetTeam(r.Context(), tid)
		if err != nil {
			core.WriteJSONError(w, http.StatusNotFound, "Team not found")
			return
		}
		orgID = team.OrganizationID
		teamID = &tid
	}

	// Verify permission: manage invitations
	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermInvitationManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	var req CreateInvitationRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Email = core.SanitizeString(req.Email, nil)
	if p := req.TeamID; p != nil {
		s := core.SanitizeString(*p, nil)
		req.TeamID = &s
	}

	// Path-based teamID wins for /teams/:teamId routes; reflect it on the
	// request so role validation uses the team role set.
	if teamID != nil {
		req.TeamID = teamID
	} else if req.TeamID != nil {
		teamID = req.TeamID
	}

	if err := p.ValidateCreateInvitation(r.Context(), orgID, req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	inv, rawToken, err := p.CreateInvitation(r.Context(), orgID, teamID, req.Email, req.Role, user.ID, req.ExpiresIn)
	if err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Message: "Invitation created",
		Data: InvitationResponse{
			ID:             inv.ID,
			OrganizationID: inv.OrganizationID,
			TeamID:         inv.TeamID,
			Email:          inv.Email,
			Role:           inv.Role,
			Token:          rawToken,
			Status:         inv.Status,
			ExpiresAt:      inv.ExpiresAt.Format(time.RFC3339),
			CreatedAt:      inv.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      inv.UpdatedAt.Format(time.RFC3339),
		},
	})
}

// ListInvitationsHandler lists pending invitations for an organization or team.
func (p *Plugin) ListInvitationsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Determine org ID from path
	orgID := core.GetSanitizedPathParam(r, "id")
	teamID := ""

	if orgID == "" {
		// Team-level listing
		tid := core.GetSanitizedPathParam(r, "teamId")
		if tid == "" {
			core.WriteJSONError(w, http.StatusBadRequest, "Organization ID or Team ID required")
			return
		}
		team, err := p.GetTeam(r.Context(), tid)
		if err != nil {
			core.WriteJSONError(w, http.StatusNotFound, "Team not found")
			return
		}
		orgID = team.OrganizationID
		teamID = tid
	}

	// Verify permission: manage invitations
	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermInvitationManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	pagination := core.ParsePagination(r)

	invitations, totalCount, err := p.ListInvitations(r.Context(), orgID, teamID, pagination.Offset, pagination.Limit)
	if err != nil {
		core.WriteJSONError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[*orgtypes.Invitation]{
		Items:      invitations,
		TotalCount: totalCount,
		Page:       pagination.Page,
		Offset:     pagination.Offset,
		Limit:      pagination.Limit,
	})
}

// CancelInvitationHandler cancels (deletes) a pending invitation.
func (p *Plugin) CancelInvitationHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Determine org ID from path
	orgID := core.GetSanitizedPathParam(r, "id")
	if orgID == "" {
		tid := core.GetSanitizedPathParam(r, "teamId")
		if tid == "" {
			core.WriteJSONError(w, http.StatusBadRequest, "Organization ID or Team ID required")
			return
		}
		team, err := p.GetTeam(r.Context(), tid)
		if err != nil {
			core.WriteJSONError(w, http.StatusNotFound, "Team not found")
			return
		}
		orgID = team.OrganizationID
	}

	// Verify permission: manage invitations
	if !p.hasOrgPermissionForUser(r.Context(), user.ID, orgID, PermInvitationManage) {
		core.WriteJSONError(w, http.StatusForbidden, "Forbidden - Admin role required")
		return
	}

	invitationID := core.GetSanitizedPathParam(r, "invitationId")
	if invitationID == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Invitation ID required")
		return
	}

	if err := p.CancelInvitation(r.Context(), invitationID); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Invitation canceled",
	})
}

// AcceptInvitationHandler accepts a pending invitation using the raw token.
//
// This endpoint is NOT authenticated — the token is the credential.
// The caller must provide the user ID of the accepting user in the request body.
func (p *Plugin) AcceptInvitationHandler(w http.ResponseWriter, r *http.Request) {
	var req AcceptInvitationRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Token = core.SanitizeString(req.Token, nil)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	// Hash the token for lookup
	tokenHash := hashTokenForLookup(req.Token)

	// Get the authenticated user (or require user ID in body)
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSONError(w, http.StatusUnauthorized, "Authentication required to accept invitation")
		return
	}

	inv, err := p.AcceptInvitation(r.Context(), tokenHash, user.ID)
	if err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Invitation accepted",
		Data:    inv,
	})
}

// DeclineInvitationHandler declines a pending invitation using the raw token.
//
// This endpoint is NOT authenticated — the token is the credential.
func (p *Plugin) DeclineInvitationHandler(w http.ResponseWriter, r *http.Request) {
	var req DeclineInvitationRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Token = core.SanitizeString(req.Token, nil)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	// Hash the token for lookup
	tokenHash := hashTokenForLookup(req.Token)

	inv, err := p.DeclineInvitation(r.Context(), tokenHash)
	if err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Invitation declined",
		Data:    inv,
	})
}

// VerifyInvitationHandler validates a raw invitation token and returns
// invitation details (for UI pre-fill). Unauthenticated.
func (p *Plugin) VerifyInvitationHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		core.WriteJSONError(w, http.StatusBadRequest, "Token required")
		return
	}

	inv, err := p.VerifyInvitation(r.Context(), token)
	if err != nil {
		core.WriteJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    inv,
	})
}
