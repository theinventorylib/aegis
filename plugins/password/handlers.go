package password

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// Handlers for password management endpoints
type Handlers struct {
	plugin *Plugin
}

// NewHandlers creates password management handlers
func NewHandlers(plugin *Plugin) *Handlers {
	return &Handlers{plugin: plugin}
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ChangePasswordHandler handles password change requests
func (h *Handlers) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	// Get current user
	user, err := core.GetUser(r.Context())
	if err != nil {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Change password
	if err := h.plugin.ChangePassword(r.Context(), user.ID, req.CurrentPassword, req.NewPassword); err != nil {
		_ = core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	_ = core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Password changed successfully",
	})
}
