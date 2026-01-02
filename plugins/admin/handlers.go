package admin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// ========== USER MANAGEMENT HANDLERS ==========
//
// All handlers require admin role via RequireAdminMiddleware.
// They provide CRUD operations for user account management.

// ListUsersHandler lists all users with pagination.
//
// Returns raw database records to support flexible admin UIs without schema changes.
//
// Endpoint:
//   - Method: GET
//   - Path: /admin/users?page=1&limit=20
//   - Auth: Required (admin role)
//
// Query Parameters:
//   - page: Page number (default: 1)
//   - limit: Page size (default: 20, max: 100)
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "users": [{"id": "user_1", "email": "user@example.com", ...}],
//	    "page": 1,
//	    "limit": 20
//	  }
//	}
func (a *Admin) ListUsersHandler(w http.ResponseWriter, r *http.Request) {
	pagination := core.ParsePagination(r)

	// Use DB method
	users, err := a.store.ListUsersRaw(r.Context(), pagination.Offset, pagination.Limit)
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

// GetUserHandler retrieves detailed information for a specific user.
//
// Returns raw database record for flexible admin UI rendering.
//
// Endpoint:
//   - Method: GET
//   - Path: /admin/users/:id
//   - Auth: Required (admin role)
//
// Path Parameters:
//   - id: User ID
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {"id": "user_123", "email": "user@example.com", "role": "admin", ...}
//	}
func (a *Admin) GetUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	user, err := a.store.GetUserRaw(r.Context(), userID)
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

// DisableUserHandler disables a user account.
//
// Disabled users cannot log in but their data is preserved.
// This is a soft action compared to deletion.
//
// Endpoint:
//   - Method: POST
//   - Path: /admin/users/:id/disable
//   - Auth: Required (admin role)
//
// Path Parameters:
//   - id: User ID
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "User disabled"
//	}
func (a *Admin) DisableUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	// Get user first
	user, err := a.store.GetByID(r.Context(), userID)
	if err != nil {
		core.WriteJSON(w, http.StatusNotFound, &core.Response{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	// Update disabled status
	user.Disabled = true
	if err := a.store.Update(r.Context(), user); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to disable user",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "User disabled",
	})
}

// EnableUserHandler re-enables a disabled user account.
//
// Allows previously disabled users to log in again.
//
// Endpoint:
//   - Method: POST
//   - Path: /admin/users/:id/enable
//   - Auth: Required (admin role)
//
// Path Parameters:
//   - id: User ID
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "User enabled"
//	}
func (a *Admin) EnableUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	user, err := a.store.GetByID(r.Context(), userID)
	if err != nil {
		core.WriteJSON(w, http.StatusNotFound, &core.Response{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	user.Disabled = false
	if err := a.store.Update(r.Context(), user); err != nil {
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

// DeleteUserHandler permanently deletes a user account.
//
// This is a destructive operation. Consider disabling users instead to preserve data.
//
// Endpoint:
//   - Method: DELETE
//   - Path: /admin/users/:id
//   - Auth: Required (admin role)
//
// Path Parameters:
//   - id: User ID
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "User deleted"
//	}
func (a *Admin) DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	if err := a.store.Delete(r.Context(), userID); err != nil {
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

// ========== BAN MANAGEMENT HANDLERS ==========

// BanUserHandler bans a user with a reason and optional expiry date.
//
// Bans prevent users from accessing the platform. Increments ban_counter for
// tracking repeat offenders.
//
// Endpoint:
//   - Method: POST
//   - Path: /admin/users/:id/ban
//   - Auth: Required (admin role)
//
// Path Parameters:
//   - id: User ID
//
// Request Body:
//
//	{
//	  "reason": "Spam",
//	  "expiresAt": "2024-12-31T23:59:59Z"  // Optional, null for permanent
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "User banned"
//	}
func (a *Admin) BanUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	var req BanUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	if req.Reason == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Ban reason is required",
		})
		return
	}

	if err := a.store.BanUser(r.Context(), userID, req.Reason, req.ExpiresAt); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to ban user",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "User banned",
	})
}

// UnbanUserHandler removes the ban from a user account.
//
// Allows banned users to access the platform again.
//
// Endpoint:
//   - Method: POST
//   - Path: /admin/users/:id/unban
//   - Auth: Required (admin role)
//
// Path Parameters:
//   - id: User ID
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "User unbanned"
//	}
func (a *Admin) UnbanUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	if err := a.store.UnbanUser(r.Context(), userID); err != nil {
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

// ========== STATISTICS HANDLERS ==========

// GetStatsHandler returns platform statistics.
//
// Provides high-level metrics for admin dashboards.
//
// Endpoint:
//   - Method: GET
//   - Path: /admin/stats
//   - Auth: Required (admin role)
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "totalUsers": 1234
//	  }
//	}
func (a *Admin) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := a.store.GetStats(r.Context())
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

// ========== ROLE MANAGEMENT HELPERS ==========
//
// These methods provide programmatic access to role management,
// useful for integration with other plugins or custom logic.

// AssignRole assigns a role to a user programmatically.
//
// Parameters:
//   - ctx: Request context
//   - userID: Target user ID
//   - role: Role to assign (e.g., "admin", "moderator")
//
// Returns:
//   - error: If database operation fails
func (a *Admin) AssignRole(ctx context.Context, userID string, role string) error {
	return a.store.AssignRole(ctx, userID, role)
}

// GetUserRole retrieves the role of a user programmatically.
//
// Parameters:
//   - ctx: Request context
//   - userID: Target user ID
//
// Returns:
//   - string: User's role (empty string if no role assigned)
//   - error: If database operation fails
func (a *Admin) GetUserRole(ctx context.Context, userID string) (string, error) {
	return a.store.GetRole(ctx, userID)
}

// RemoveRole removes a role from a user programmatically.
//
// Parameters:
//   - ctx: Request context
//   - userID: Target user ID
//   - role: Role to remove
//
// Returns:
//   - error: If database operation fails
func (a *Admin) RemoveRole(ctx context.Context, userID string, role string) error {
	return a.store.RemoveRole(ctx, userID, role)
}

// GetAdminUser retrieves a user with admin-specific information.
//
// This includes role, ban status, and ban details.
//
// Parameters:
//   - ctx: Request context
//   - userID: Target user ID
//
// Returns:
//   - User: User with admin fields populated
//   - error: If user not found or database error
func (a *Admin) GetAdminUser(ctx context.Context, userID string) (User, error) {
	// Get user from store
	user, err := a.store.GetByID(ctx, userID)
	if err != nil {
		return User{}, err
	}

	return user, nil
}
