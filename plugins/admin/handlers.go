package admin

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// ========== USER MANAGEMENT HANDLERS ==========

// listUsersHandler lists all users with pagination.
func (a *Plugin) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	pagination := core.ParsePagination(r)

	users, err := a.ListUsers(r.Context(), pagination.Offset, pagination.Limit)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to list users",
		})
		return
	}

	totalCount, err := a.store.Count(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to count users",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.PaginatedResponse[User]{
		Items:      users,
		TotalCount: totalCount,
		Page:       pagination.Page,
		Offset:     pagination.Offset,
		Limit:      pagination.Limit,
	})

}

// getUserHandler retrieves detailed information for a specific user.
func (a *Plugin) getUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetSanitizedPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	user, err := a.GetUser(r.Context(), userID)
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

// disableUserHandler disables a user account.
func (a *Plugin) disableUserHandler(w http.ResponseWriter, r *http.Request) {
	a.setUserDisabledStatus(w, r, true, "disable", "disabled")
}

// enableUserHandler re-enables a disabled user account.
func (a *Plugin) enableUserHandler(w http.ResponseWriter, r *http.Request) {
	a.setUserDisabledStatus(w, r, false, "enable", "enabled")
}

// setUserDisabledStatus is a helper function that sets the disabled status of a user.
func (a *Plugin) setUserDisabledStatus(w http.ResponseWriter, r *http.Request, disabled bool, action, pastTense string) {
	userID := core.GetSanitizedPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	var err error
	if disabled {
		err = a.DisableUser(r.Context(), userID)
	} else {
		err = a.EnableUser(r.Context(), userID)
	}

	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to " + action + " user",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "User " + pastTense,
	})
}

// deleteUserHandler permanently deletes a user account.
func (a *Plugin) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetSanitizedPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	if err := a.DeleteUser(r.Context(), userID); err != nil {
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

// ========== ROLE MANAGEMENT HANDLERS ==========

// updateRoleHandler updates a user's role.
func (a *Plugin) updateRoleHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetSanitizedPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Sanitize role input
	req.Role = core.SanitizeString(req.Role, nil)

	if req.Role == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Role is required",
		})
		return
	}

	// Verify user exists before assigning role
	if _, err := a.GetUser(r.Context(), userID); err != nil {
		core.WriteJSON(w, http.StatusNotFound, &core.Response{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	if err := a.AssignRole(r.Context(), userID, req.Role); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to update role",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Role updated",
	})
}

// ========== BAN MANAGEMENT HANDLERS ==========

// banUserHandler bans a user with a reason and optional expiry date.
func (a *Plugin) banUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetSanitizedPathParam(r, "id")
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

	// Sanitize inputs
	req.Reason = core.SanitizeString(req.Reason, nil)

	if req.Reason == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Ban reason is required",
		})
		return
	}

	if err := a.BanUser(r.Context(), userID, req.Reason, req.ExpiresAt); err != nil {
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

// unbanUserHandler removes the ban from a user account.
func (a *Plugin) unbanUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := core.GetSanitizedPathParam(r, "id")
	if userID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "User ID required",
		})
		return
	}

	if err := a.UnbanUser(r.Context(), userID); err != nil {
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

// getStatsHandler returns platform statistics.
func (a *Plugin) getStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := a.GetStats(r.Context())
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
