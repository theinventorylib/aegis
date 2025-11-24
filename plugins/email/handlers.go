package email

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// Handlers for email verification and auth endpoints
type Handlers struct {
	plugin *Plugin
}

// NewHandlers creates email handlers
func NewHandlers(plugin *Plugin) *Handlers {
	return &Handlers{plugin: plugin}
}

// LoginWithEmailRequest represents email+password login
type LoginWithEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginWithEmailHandler handles email+password login
func (h *Handlers) LoginWithEmailHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.passwordPlugin == nil {
		http.Error(w, "Password authentication not configured", http.StatusNotImplemented)
		return
	}

	var req LoginWithEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get user by email
	user, err := h.plugin.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Verify password via password plugin
	valid, err := h.plugin.passwordPlugin.VerifyPassword(r.Context(), user.ID, req.Password)
	if err != nil || !valid {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Create session
	if h.plugin.sessionService != nil {
		session, err := h.plugin.sessionService.CreateSession(r.Context(), user, r.RemoteAddr, r.UserAgent())
		if err != nil {
			_ = core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to create session",
			})
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    session.Token,
			Path:     "/",
			Expires:  session.ExpiresAt,
			HttpOnly: true,
			Secure:   true, // Should be configurable based on env
			SameSite: http.SameSiteLaxMode,
		})
	}

	_ = core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"user": user,
		},
	})
}
