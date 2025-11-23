package organizations

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// ========== ORGANIZATION HANDLERS ==========

// CreateOrganizationHandler creates a new organization
func (p *Plugin) CreateOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	org, err := p.createOrganization(r.Context(), req.Name, req.Slug, user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success":      true,
		"organization": org,
	})
}

// ListOrganizationsHandler lists user's organizations
func (p *Plugin) ListOrganizationsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orgs, err := p.getUserOrganizations(r.Context(), user.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"organizations": orgs,
	})
}

// GetOrganizationHandler gets a specific organization
func (p *Plugin) GetOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if !p.isOrganizationMember(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	org, err := p.getOrganization(r.Context(), orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if !p.isOwnerOrAdmin(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := p.updateOrganization(r.Context(), orgID, req.Name, req.Slug); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if !p.isOwner(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Owner role required", http.StatusForbidden)
		return
	}

	if err := p.deleteOrganization(r.Context(), orgID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if !p.isOwnerOrAdmin(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req struct {
		UserID string `json:"userId"`
		Role   string `json:"role"` // "admin" or "member"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Role == "owner" {
		http.Error(w, "Cannot assign owner role", http.StatusBadRequest)
		return
	}

	if err := p.addOrganizationMember(r.Context(), orgID, req.UserID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Member added to organization",
	})
}

// ListOrganizationMembersHandler lists organization members
func (p *Plugin) ListOrganizationMembersHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if !p.isOrganizationMember(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	members, err := p.getOrganizationMembers(r.Context(), orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"members": members,
	})
}

// UpdateMemberRoleHandler updates a member's role
func (p *Plugin) UpdateMemberRoleHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orgID := r.URL.Query().Get("id")
	userID := r.URL.Query().Get("userId")

	if orgID == "" || userID == "" {
		http.Error(w, "Organization ID and User ID required", http.StatusBadRequest)
		return
	}

	if !p.isOwner(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Owner role required", http.StatusForbidden)
		return
	}

	var req struct {
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Role == "owner" {
		http.Error(w, "Cannot assign owner role", http.StatusBadRequest)
		return
	}

	if err := p.updateOrganizationMemberRole(r.Context(), orgID, userID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	orgID := r.URL.Query().Get("id")
	userID := r.URL.Query().Get("userId")

	if orgID == "" || userID == "" {
		http.Error(w, "Organization ID and User ID required", http.StatusBadRequest)
		return
	}

	if !p.isOwnerOrAdmin(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	if p.isOwner(r.Context(), userID, orgID) {
		http.Error(w, "Cannot remove owner", http.StatusBadRequest)
		return
	}

	if err := p.removeOrganizationMember(r.Context(), orgID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if !p.isOwnerOrAdmin(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	team, err := p.createTeam(r.Context(), orgID, req.Name, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"team":    team,
	})
}

// ListTeamsHandler lists teams in an organization
func (p *Plugin) ListTeamsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if !p.isOrganizationMember(r.Context(), user.ID, orgID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	teams, err := p.getOrganizationTeams(r.Context(), orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"teams":   teams,
	})
}

// GetTeamHandler gets a specific team
func (p *Plugin) GetTeamHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	teamID := r.URL.Query().Get("teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.getTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.isOrganizationMember(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	teamID := r.URL.Query().Get("teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.getTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.isOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := p.updateTeam(r.Context(), teamID, req.Name, req.Description); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	teamID := r.URL.Query().Get("teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.getTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.isOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	if err := p.deleteTeam(r.Context(), teamID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	teamID := r.URL.Query().Get("teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.getTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.isOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req struct {
		UserID string `json:"userId"`
		Role   string `json:"role"` // "lead" or "member"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// User must be organization member before joining team
	if !p.isOrganizationMember(r.Context(), req.UserID, team.OrganizationID) {
		http.Error(w, "User must be organization member first", http.StatusBadRequest)
		return
	}

	if err := p.addTeamMember(r.Context(), teamID, req.UserID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
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

	teamID := r.URL.Query().Get("teamId")
	if teamID == "" {
		http.Error(w, "Team ID required", http.StatusBadRequest)
		return
	}

	team, err := p.getTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.isOrganizationMember(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	members, err := p.getTeamMembers(r.Context(), teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"members": members,
	})
}

// UpdateTeamMemberRoleHandler updates a team member's role
func (p *Plugin) UpdateTeamMemberRoleHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	teamID := r.URL.Query().Get("teamId")
	userID := r.URL.Query().Get("userId")

	if teamID == "" || userID == "" {
		http.Error(w, "Team ID and User ID required", http.StatusBadRequest)
		return
	}

	team, err := p.getTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.isOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	var req struct {
		Role string `json:"role"` // "lead" or "member"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := p.updateTeamMemberRole(r.Context(), teamID, userID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

	teamID := r.URL.Query().Get("teamId")
	userID := r.URL.Query().Get("userId")

	if teamID == "" || userID == "" {
		http.Error(w, "Team ID and User ID required", http.StatusBadRequest)
		return
	}

	team, err := p.getTeam(r.Context(), teamID)
	if err != nil {
		http.Error(w, "Team not found", http.StatusNotFound)
		return
	}

	if !p.isOwnerOrAdmin(r.Context(), user.ID, team.OrganizationID) {
		http.Error(w, "Forbidden - Admin role required", http.StatusForbidden)
		return
	}

	if err := p.removeTeamMember(r.Context(), teamID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Member removed from team",
	})
}

// Helper function
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
