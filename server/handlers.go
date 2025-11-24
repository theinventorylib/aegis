package server

import (
	"encoding/json"
	"net/http"

	"github.com/theinventorylib/aegis/core"
)

// Handlers struct contains dependencies for HTTP handlers.
type Handlers struct {
	auth    *core.AuthService
	session *core.SessionService
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(auth *core.AuthService, session *core.SessionService) *Handlers {
	return &Handlers{
		auth:    auth,
		session: session,
	}
}

// LogoutHandler handles user logout.
func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie or header.
	var token string

	cookie, err := r.Cookie("aegis_session")
	if err == nil {
		token = cookie.Value
	} else {
		token = r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token != "" {
		_ = h.auth.Logout(r.Context(), token) // Ignore error, logout is best-effort
	}

	// Clear session cookie.
	core.ClearSessionCookie(w, h.session.GetConfig())

	_ = core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Logged out successfully",
	})
}

// UserHandler returns the current user.
func (h *Handlers) UserHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
}

// ========== SESSION MANAGEMENT HANDLERS ==========

// RefreshSessionHandler refreshes a session using refresh token.
func (h *Handlers) RefreshSessionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Refresh the session.
	newSession, err := h.session.RefreshSession(r.Context(), req.RefreshToken)
	if err != nil {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid or expired refresh token",
		})
		return
	}

	// Set new session cookie.
	core.SetSessionCookie(w, newSession.Token, h.session.GetConfig())

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Session refreshed",
		"session": map[string]interface{}{
			"token":        newSession.Token,
			"refreshToken": newSession.RefreshToken,
			"expiresAt":    newSession.ExpiresAt,
		},
	})
}

// ValidateSessionHandler validates the current session.
func (h *Handlers) ValidateSessionHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid session",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Session valid",
		"user":    user,
	})
}

// ListSessionsHandler lists all active sessions for the current user.
func (h *Handlers) ListSessionsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	sessions, err := h.session.GetDB().GetUserSessions(r.Context(), user.ID)
	if err != nil {
		_ = core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to retrieve sessions",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"sessions": sessions,
	})
}

// RevokeSessionHandler revokes a specific session.
func (h *Handlers) RevokeSessionHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	// Delete session by token (treating ID as token).
	if err := h.session.DeleteSession(r.Context(), sessionID); err != nil {
		_ = core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Failed to revoke session",
		})
		return
	}

	_ = user
	_ = core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Session revoked",
	})
}

// RevokeAllSessionsHandler revokes all sessions for the current user.
func (h *Handlers) RevokeAllSessionsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		_ = core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	if err := h.session.GetDB().DeleteUserSessions(r.Context(), user.ID); err != nil {
		_ = core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to revoke sessions",
		})
		return
	}

	_ = core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "All sessions revoked successfully",
	})
}
