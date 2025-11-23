package admin

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// ListUsersHandler lists all users (paginated)
func (p *Plugin) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Extract pagination params
	// page := r.URL.Query().Get("page")
	// limit := r.URL.Query().Get("limit")

	users, err := p.listUsers(r.Context(), 0, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"users":   users,
	})
}

// GetUserHandler gets a specific user
func (p *Plugin) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	var id, email, role string
	var createdAt, updatedAt interface{}
	var disabled bool

	// Query extended fields
	err := p.database.QueryRow(r.Context(), `
		SELECT id, created_at, updated_at, 
		       COALESCE(email, '') as email, 
		       COALESCE(role, 'user') as role,
		       disabled
		FROM auth.user
		WHERE id = $1
	`, userID).Scan(&id, &createdAt, &updatedAt, &email, &role, &disabled)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	user := map[string]interface{}{
		"id":        id,
		"createdAt": createdAt,
		"updatedAt": updatedAt,
		"email":     email,
		"role":      role,
		"disabled":  disabled,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

// DisableUserHandler disables a user account
func (p *Plugin) DisableUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	// Get user
	user, err := p.database.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update user status
	user.Disabled = true
	if err := p.database.UpdateUser(r.Context(), user); err != nil {
		http.Error(w, "Failed to disable user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User disabled",
	})
}

// EnableUserHandler enables a user account
func (p *Plugin) EnableUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	// Get user
	user, err := p.database.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update user status
	user.Disabled = false
	if err := p.database.UpdateUser(r.Context(), user); err != nil {
		http.Error(w, "Failed to enable user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User enabled",
	})
}

// DeleteUserHandler deletes a user account
func (p *Plugin) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	// Delete user (cascades to sessions, OTPs, organization memberships)
	if err := p.database.DeleteUser(r.Context(), userID); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "User deleted successfully",
	})
}

// AddOrganizationHandler creates a new organization
func (p *Plugin) AddOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Slug    string `json:"slug"`
		OwnerID string `json:"owner_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Slug == "" || req.OwnerID == "" {
		http.Error(w, "Name, slug, and owner_id are required", http.StatusBadRequest)
		return
	}

	org, err := p.addOrganization(r.Context(), req.Name, req.Slug, req.OwnerID)
	if err != nil {
		http.Error(w, "Failed to create organization: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"organization": org,
	})
}

// ListOrganizationsHandler lists all organizations
func (p *Plugin) ListOrganizationsHandler(w http.ResponseWriter, r *http.Request) {
	orgs, err := p.listOrganizations(r.Context(), 0, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"organizations": orgs,
	})
}

// GetOrganizationHandler gets a specific organization
func (p *Plugin) GetOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	org, err := p.getOrganization(r.Context(), orgID)
	if err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"organization": org,
	})
}

// BanOrganizationHandler bans (disables) an organization
func (p *Plugin) BanOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if err := p.banOrganization(r.Context(), orgID); err != nil {
		http.Error(w, "Failed to ban organization: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Organization banned",
	})
}

// DeleteOrganizationHandler deletes an organization
func (p *Plugin) DeleteOrganizationHandler(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("id")
	if orgID == "" {
		http.Error(w, "Organization ID required", http.StatusBadRequest)
		return
	}

	if err := p.deleteOrganization(r.Context(), orgID); err != nil {
		http.Error(w, "Failed to delete organization: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Organization deleted",
	})
}

// GetStatsHandler returns platform statistics
func (p *Plugin) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := p.getStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

// AdminMiddleware checks if user has admin role
func (p *Plugin) AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := core.GetUser(r.Context())
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check if user has admin role by querying DB
		var role string
		err = p.database.QueryRow(r.Context(), "SELECT role FROM auth.user WHERE id = $1", user.ID).Scan(&role)
		if err != nil {
			// If role column doesn't exist or user not found (shouldn't happen if auth passed), fail
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if role != "admin" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
