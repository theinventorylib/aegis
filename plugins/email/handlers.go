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

// LoginWithEmailHandler handles email+password login
func (h *Handlers) LoginWithEmailHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.authService == nil {
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
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Verify password via core AuthService
	valid, err := h.plugin.authService.VerifyPassword(r.Context(), user.ID, req.Password)
	if err != nil || !valid {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid credentials",
		})
		return
	}

	// Create session
	if h.plugin.sessionService != nil {
		session, err := h.plugin.sessionService.CreateSession(r.Context(), user, r.RemoteAddr, r.UserAgent())
		if err != nil {
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to create session",
			})
			return
		}

		// Set session cookie using central helper to respect configured cookie settings
		if h.plugin.sessionService != nil {
			cfg := h.plugin.sessionService.GetConfig()
			core.SetSessionCookie(w, session.Token, cfg)
		}
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"user": user,
		},
	})
}

// RegisterWithEmailHandler handles email+password registration
func (h *Handlers) RegisterWithEmailHandler(w http.ResponseWriter, r *http.Request) {
	if h.plugin.authService == nil {
		http.Error(w, "Password authentication not configured", http.StatusNotImplemented)
		return
	}

	var req RegisterWithEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create user with email and password
	user, err := h.plugin.CreateUserWithEmailAndPassword(r.Context(), req.Email, req.Password)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Create session (auto-login)
	if h.plugin.sessionService != nil {
		session, err := h.plugin.sessionService.CreateSession(r.Context(), user, r.RemoteAddr, r.UserAgent())
		if err != nil {
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to create session",
			})
			return
		}

		// Set session cookie
		cfg := h.plugin.sessionService.GetConfig()
		core.SetSessionCookie(w, session.Token, cfg)
	}

	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Message: "Registration successful",
		Data: map[string]interface{}{
			"user": user,
		},
	})
}
