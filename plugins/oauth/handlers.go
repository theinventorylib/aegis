package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/markbates/goth/gothic"
	"github.com/theinventorylib/aegis/core"
)

// Handlers for OAuth plugin
type Handlers struct {
	plugin *Plugin
}

// NewHandlers creates OAuth plugin handlers
func NewHandlers(plugin *Plugin) *Handlers {
	return &Handlers{plugin: plugin}
}

// BeginAuthHandler starts the OAuth flow
func (h *Handlers) BeginAuthHandler(w http.ResponseWriter, r *http.Request) {
	// Provider comes from URL path parameter
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		http.Error(w, "Provider required", http.StatusBadRequest)
		return
	}

	// Goth/Gothic handles the redirect
	h.plugin.BeginAuth(w, r, provider)
}

// CallbackHandler handles the OAuth callback
func (h *Handlers) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	user, session, err := h.plugin.CompleteAuth(r.Context(), w, r)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Set session cookie using central helper to respect configured cookie settings
	if h.plugin.sessionService != nil {
		cfg := h.plugin.sessionService.GetConfig()
		core.SetSessionCookie(w, session.Token, cfg)
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data: map[string]interface{}{
			"user":    user,
			"session": session,
		},
	})
}

// LogoutHandler handles OAuth logout
func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Goth logout - error not critical
	_ = gothic.Logout(w, r)

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Logged out successfully",
	})
}

// LinkAccountRequest represents linking an OAuth account
type LinkAccountRequest struct {
	Provider string `json:"provider"`
}

// LinkAccountHandler links an OAuth account to the current user
func (h *Handlers) LinkAccountHandler(w http.ResponseWriter, r *http.Request) {
	// Get current user from context
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	var req LinkAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Complete OAuth and link to existing account
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Link the account
	oauthUser := GothUserToUser(gothUser)
	if err := h.plugin.LinkAccount(r.Context(), user.ID, oauthUser, gothUser.Provider); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to link OAuth account: %v", err),
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "OAuth account linked successfully",
	})
}
