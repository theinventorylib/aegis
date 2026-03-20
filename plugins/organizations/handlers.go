package organizations

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
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
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}

	orgID = core.GetSanitizedPathParam(r, "id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return "", false
	}

	if !p.IsOrganizationMember(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
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
//	  "organization": {
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
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateOrganizationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	core.WriteJSON(w, http.StatusCreated, map[string]any{
		"success":      true,
		"organization": org,
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
//	  "organizations": [
//	    {"id": "org_1", "name": "Acme Corp", "slug": "acme", ...},
//	    {"id": "org_2", "name": "Tech Inc", "slug": "tech", ...}
//	  ]
//	}
func (p *Plugin) ListOrganizationsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	pagination := core.ParsePagination(r)

	orgs, totalCount, err := p.GetUserOrganizations(r.Context(), user.ID, pagination.Offset, pagination.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[*Organization]{
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
//	  "organization": {
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	core.WriteJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"organization": org,
	})
}

// UpdateOrganizationHandler updates an organization
func (p *Plugin) UpdateOrganizationHandler(w http.ResponseWriter, r *http.Request) {
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

	if !p.IsOwnerOrAdmin(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req UpdateOrganizationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	core.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Organization updated",
	})
}

// DeleteOrganizationHandler deletes an organization
func (p *Plugin) DeleteOrganizationHandler(w http.ResponseWriter, r *http.Request) {
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

	if !p.IsOwner(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Owner role required", http.StatusForbidden)
		return
	}

	if err := p.DeleteOrganization(r.Context(), orgID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	core.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Organization deleted",
	})
}

// ========== ORGANIZATION MEMBER HANDLERS ==========

// AddOrganizationMemberHandler adds a member to an organization
func (p *Plugin) AddOrganizationMemberHandler(w http.ResponseWriter, r *http.Request) {
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

	if !p.IsOwnerOrAdmin(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req AddOrganizationMemberRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize inputs
	req.UserID = core.SanitizeString(req.UserID, nil)
	req.Role = core.SanitizeString(req.Role, nil)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	if req.Role == "owner" {
		http.Error(w, "Cannot assign owner role", http.StatusBadRequest)
		return
	}

	if err := p.AddOrganizationMember(r.Context(), orgID, req.UserID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	core.WriteJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"message": "Member added to organization",
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[*Member]{
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
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orgID := core.GetSanitizedPathParam(r, "id")
	userID := core.GetSanitizedPathParam(r, "userId")

	if orgID == "" || userID == "" {
		http.Error(w, "Organization ID and User ID required", http.StatusBadRequest)
		return
	}

	if !p.IsOwner(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Owner role required", http.StatusForbidden)
		return
	}

	var req UpdateMemberRoleRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize inputs
	req.Role = core.SanitizeString(req.Role, nil)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	if req.Role == "owner" {
		http.Error(w, "Cannot assign owner role", http.StatusBadRequest)
		return
	}

	if err := p.UpdateMemberRole(r.Context(), orgID, userID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	core.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Role updated",
	})
}

// RemoveOrganizationMemberHandler removes a member from an organization
func (p *Plugin) RemoveOrganizationMemberHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orgID := core.GetSanitizedPathParam(r, "id")
	userID := core.GetSanitizedPathParam(r, "userId")

	if orgID == "" || userID == "" {
		http.Error(w, "Organization ID and User ID required", http.StatusBadRequest)
		return
	}

	if !p.IsOwnerOrAdmin(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	if p.IsOwner(r.Context(), userID, orgID) {
		http.Error(w, "Cannot remove owner", http.StatusBadRequest)
		return
	}

	if err := p.RemoveOrganizationMember(r.Context(), orgID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	core.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Member removed from organization",
	})
}

// ========== TEAM HANDLERS ==========

// CreateTeamHandler creates a new team within an organization
func (p *Plugin) CreateTeamHandler(w http.ResponseWriter, r *http.Request) {
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

	if !p.IsOwnerOrAdmin(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req CreateTeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	core.WriteJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"team":    team,
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[*Team]{
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
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	teamID := core.GetSanitizedPathParam(r, "teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.IsOrganizationMember(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	core.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"team":    team,
	})
}

// UpdateTeamHandler updates a team
func (p *Plugin) UpdateTeamHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	teamID := core.GetSanitizedPathParam(r, "teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.IsOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req UpdateTeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	core.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Team updated",
	})
}

// DeleteTeamHandler deletes a team
func (p *Plugin) DeleteTeamHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	teamID := core.GetSanitizedPathParam(r, "teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.IsOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	if err := p.DeleteTeam(r.Context(), teamID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	core.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Team deleted",
	})
}

// ========== TEAM MEMBER HANDLERS ==========

// AddTeamMemberHandler adds a member to a team
func (p *Plugin) AddTeamMemberHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	teamID := core.GetSanitizedPathParam(r, "teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.IsOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req AddTeamMemberRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize inputs
	req.UserID = core.SanitizeString(req.UserID, nil)
	req.Role = core.SanitizeString(req.Role, nil)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	// User must be organization member before joining team
	if !p.IsOrganizationMember(r.Context(), req.UserID, team.OrganizationID) {
		http.Error(w, "User must be organization member first", http.StatusBadRequest)
		return
	}

	if err := p.AddTeamMember(r.Context(), teamID, req.UserID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	core.WriteJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"message": "Member added to team",
	})
}

// ListTeamMembersHandler lists team members
func (p *Plugin) ListTeamMembersHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	teamID := core.GetSanitizedPathParam(r, "teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.IsOrganizationMember(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	pagination := core.ParsePagination(r)

	members, totalCount, err := p.ListTeamMembers(r.Context(), teamID, pagination.Offset, pagination.Limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[*TeamMember]{
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
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	teamID := core.GetSanitizedPathParam(r, "teamId")
	userID := core.GetSanitizedPathParam(r, "userId")

	if teamID == "" || userID == "" {
		http.Error(w, "Team ID and User ID required", http.StatusBadRequest)
		return
	}

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.IsOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req UpdateTeamMemberRoleRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Sanitize inputs
	req.Role = core.SanitizeString(req.Role, nil)

	if err := req.Validate(); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{Success: false, Error: err.Error()})
		return
	}

	if err := p.UpdateTeamMemberRole(r.Context(), teamID, userID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	core.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Team member role updated",
	})
}

// RemoveTeamMemberHandler removes a member from a team
func (p *Plugin) RemoveTeamMemberHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	teamID := core.GetSanitizedPathParam(r, "teamId")
	userID := core.GetSanitizedPathParam(r, "userId")

	if teamID == "" || userID == "" {
		http.Error(w, "Team ID and User ID required", http.StatusBadRequest)
		return
	}

	team, err := p.GetTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.IsOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	if err := p.RemoveTeamMember(r.Context(), teamID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{Success: true, Message: "Member removed from team"})
}
