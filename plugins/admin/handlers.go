package admin

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/theinventorylib/aegis/core"
)

// ListUsersHandler lists all users
func (p *Plugin) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	pagination := core.ParsePagination(r)

	// Use DB method
	users, err := p.db.ListUsersRaw(r.Context(), pagination.Offset, pagination.Limit)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to list users",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data: map[string]interface{}{
			"users": users,
			"page":  pagination.Page,
			"limit": pagination.Limit,
		},
	})
}

// GetUserHandler gets a specific user
func (p *Plugin) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	user, err := p.db.GetUserRaw(r.Context(), userID)
	if err != nil {
		core.WriteJSON(w, http.StatusNotFound, &core.Response{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    user,
	})
}

// DisableUserHandler disables a user
func (p *Plugin) DisableUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	// Get user first
	user, err := p.db.GetUserByID(r.Context(), userID)
	if err != nil {
		core.WriteJSON(w, http.StatusNotFound, &core.Response{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	// Update disabled status
	user.Disabled = true
	if err := p.db.UpdateUser(r.Context(), user); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to disable user",
		})
		return
	}

	// Kill sessions
	_ = p.db.DeleteUserSessions(r.Context(), userID)

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "User disabled",
	})
}

// EnableUserHandler enables a user
func (p *Plugin) EnableUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	user, err := p.db.GetUserByID(r.Context(), userID)
	if err != nil {
		core.WriteJSON(w, http.StatusNotFound, &core.Response{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	user.Disabled = false
	if err := p.db.UpdateUser(r.Context(), user); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to enable user",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "User enabled",
	})
}

// DeleteUserHandler deletes a user
func (p *Plugin) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	if err := p.db.DeleteUser(r.Context(), userID); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to delete user",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "User deleted",
	})
}

// BanUserHandler bans a user
func (p *Plugin) BanUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Reason    string    `json:"reason"`
		ExpiresAt time.Time `json:"expiresAt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Reason == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Ban reason is required",
		})
		return
	}

	var expiry interface{}
	if !req.ExpiresAt.IsZero() {
		expiry = req.ExpiresAt
	}

	if err := p.db.BanUser(r.Context(), userID, req.Reason, expiry); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to ban user",
		})
		return
	}

	// Kill sessions
	_ = p.db.DeleteUserSessions(r.Context(), userID)

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "User banned",
	})
}

// UnbanUserHandler unbans a user
func (p *Plugin) UnbanUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	if err := p.db.UnbanUser(r.Context(), userID); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to unban user",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "User unbanned",
	})
}

// AddOrganizationHandler creates a new organization
func (p *Plugin) AddOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Slug    string `json:"slug"`
		OwnerID string `json:"ownerId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Slug == "" || req.OwnerID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Name, slug, and ownerId are required",
		})
		return
	}

	id := core.GenerateID()
	org, err := p.db.CreateOrganization(r.Context(), id, req.Name, req.Slug, req.OwnerID)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to create organization",
		})
		return
	}

	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Data:    org,
	})
}

// ListOrganizationsHandler lists all organizations
func (p *Plugin) ListOrganizationsHandler(w http.ResponseWriter, r *http.Request) {
	pagination := core.ParsePagination(r)

	orgs, err := p.db.ListOrganizations(r.Context(), pagination.Offset, pagination.Limit)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to list organizations",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data: map[string]interface{}{
			"organizations": orgs,
			"page":          pagination.Page,
			"limit":         pagination.Limit,
		},
	})
}

// GetOrganizationHandler gets a specific organization
func (p *Plugin) GetOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	org, err := p.db.GetOrganization(r.Context(), orgID)
	if err != nil {
		core.WriteJSON(w, http.StatusNotFound, &core.Response{
			Success: false,
			Error:   "Organization not found",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    org,
	})
}

// BanOrganizationHandler bans an organization
func (p *Plugin) BanOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if err := p.db.BanOrganization(r.Context(), orgID); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to ban organization",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Organization banned",
	})
}

// DeleteOrganizationHandler deletes an organization
func (p *Plugin) DeleteOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if err := p.db.DeleteOrganization(r.Context(), orgID); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to delete organization",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Organization deleted",
	})
}

// GetStatsHandler returns platform statistics
func (p *Plugin) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := p.db.GetStats(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to get stats",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    stats,
	})
}
