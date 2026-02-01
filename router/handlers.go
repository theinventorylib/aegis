package router

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/core"
)

// Handlers provides HTTP handlers for core Aegis authentication routes.
//
// All handlers have been made private (lowercase) to encourage programmatic
// use of the underlying core services. This struct serves as a mounting point
// for the router.
type Handlers struct {
	// auth provides access to all authentication services
	auth *core.AuthService
}

// NewHandlers creates a new Handlers instance with the given AuthService.
func NewHandlers(auth *core.AuthService) *Handlers {
	return &Handlers{
		auth: auth,
	}
}

// loginHandler handles email+password authentication.
func (h *Handlers) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Sanitize inputs
	req.Email = core.SanitizeEmail(req.Email)

	result, err := h.auth.EmailPassword.Login(r.Context(), req.Email, req.Password, r.RemoteAddr, r.UserAgent())
	if err != nil {
		status := http.StatusUnauthorized
		msg := err.Error()
		if core.IsAuthError(err) {
			ae, ok := err.(*core.AuthError)
			if ok && ae.Code == core.AuthErrorCodeRateLimit {
				status = http.StatusTooManyRequests
			}
		}
		core.WriteJSON(w, status, &core.Response{
			Success: false,
			Error:   msg,
		})
		return
	}

	// Set session cookie
	h.auth.Session.GetCookieManager().SetSessionCookie(w, result.Token)

	// Convert user to EnrichedUser for consistent response format
	enriched := core.NewEnrichedUser(&result.User)

	// Return session with user data
	config := h.auth.GetUserFieldsConfig()
	swu := &core.SessionWithUser{
		Session: result.Session,
		User:    enriched,
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Login successful",
		Data:    swu.ToAPIResponseFiltered(config),
	})
}

// registerHandler handles new user registration with email+password.
func (h *Handlers) registerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Sanitize inputs
	req.Name = core.SanitizeString(req.Name, nil)
	req.Email = core.SanitizeEmail(req.Email)

	if req.Name == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Name is required",
		})
		return
	}

	result, err := h.auth.EmailPassword.Register(r.Context(), req.Name, req.Email, req.Password, r.RemoteAddr, r.UserAgent())
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Set session cookie
	h.auth.Session.GetCookieManager().SetSessionCookie(w, result.Token)

	// Convert user to EnrichedUser for consistent response format
	enriched := core.NewEnrichedUser(&result.User)

	// Return session with user data
	config := h.auth.GetUserFieldsConfig()
	swu := &core.SessionWithUser{
		Session: result.Session,
		User:    enriched,
	}
	core.WriteJSON(w, http.StatusCreated, &core.Response{
		Success: true,
		Message: "Registration successful",
		Data:    swu.ToAPIResponseFiltered(config),
	})
}

// logoutHandler handles user logout by invalidating the session and clearing the cookie.
func (h *Handlers) logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie or header using CookieManager
	var token string

	cookieToken, err := h.auth.Session.GetCookieManager().GetSessionCookie(r)
	if err == nil && cookieToken != "" {
		token = cookieToken
	} else {
		// Fallback to Authorization header
		token = r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token != "" {
		// Use core SessionService.Logout
		if err := h.auth.Session.Logout(r.Context(), token); err != nil {
			log.Printf("logout error (non-fatal): %v", err)
		}
	}

	// Clear session cookie using CookieManager
	h.auth.Session.GetCookieManager().ClearSessionCookie(w)

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Logged out successfully",
	})
}

// refreshSessionHandler refreshes an existing session using a refresh token.
func (h *Handlers) refreshSessionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	if req.RefreshToken == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Refresh token is required",
		})
		return
	}

	// Sanitize inputs
	req.RefreshToken = core.SanitizeString(req.RefreshToken, nil)

	// Use core SessionService.RefreshSession
	newSession, err := h.auth.Session.RefreshSession(r.Context(), req.RefreshToken)
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid or expired refresh token",
		})
		return
	}

	// Set new session cookie using CookieManager
	token := ""
	expiresAt := time.Time{}
	if sm, ok := any(newSession).(core.SessionModel); ok {
		token = sm.GetToken()
		expiresAt = sm.GetExpiresAt()
	} else if sm2, ok := any(newSession).(*auth.Session); ok {
		token = sm2.Token
		expiresAt = sm2.ExpiresAt
	}

	if token != "" {
		h.auth.Session.GetCookieManager().SetSessionCookie(w, token)
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Session refreshed",
		Data: map[string]any{
			"expiresAt": expiresAt.String(),
		},
	})
}

// getCurrentSessionHandler returns the current session information.
func (h *Handlers) getCurrentSessionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get session from context
	session := core.GetSession(ctx)
	if session == nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	// Get enriched user
	enriched := core.GetEnrichedUser(ctx)
	if enriched == nil {
		// Fallback to basic user
		user, err := core.GetUser(ctx)
		if err != nil {
			core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
				Success: false,
				Error:   "Invalid session",
			})
			return
		}
		core.WriteJSON(w, http.StatusOK, &core.Response{
			Success: true,
			Message: "Session valid",
			Data: map[string]any{
				"session": session,
				"user":    user,
			},
		})
		return
	}

	// Return session with enriched user
	config := h.auth.GetUserFieldsConfig()
	swu := &core.SessionWithUser{
		Session: session,
		User:    enriched,
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    swu.ToAPIResponseFiltered(config),
	})
}

// listSessionsHandler lists all active sessions for the current user.
func (h *Handlers) listSessionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := core.GetUser(ctx)
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	uid := ""
	if um, ok := any(user).(core.UserModel); ok {
		uid = um.GetID()
	} else if um2, ok := any(user).(*auth.User); ok {
		uid = um2.ID
	}

	// Use core SessionService.GetUserSessions
	sessions, err := h.auth.Session.GetUserSessions(ctx, uid)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to retrieve sessions",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    sessions,
	})
}

// revokeSessionHandler revokes a specific session.
func (h *Handlers) revokeSessionHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := core.GetUser(ctx)
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	// Get session ID from path parameter
	sessionID := core.GetSanitizedPathParam(r, "id")
	if sessionID == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Session ID required",
		})
		return
	}

	uid := ""
	if um, ok := any(user).(core.UserModel); ok {
		uid = um.GetID()
	} else if um2, ok := any(user).(*auth.User); ok {
		uid = um2.ID
	}

	// Use core SessionService.RevokeSessionByID
	if err := h.auth.Session.RevokeSessionByID(ctx, uid, sessionID); err != nil {
		if errors.Is(err, core.ErrSessionNotFound) {
			core.WriteJSON(w, http.StatusNotFound, &core.Response{
				Success: false,
				Error:   "Session not found or does not belong to user",
			})
			return
		}
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to revoke session",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Session revoked",
	})
}

// revokeAllSessionsHandler revokes all sessions for the current user.
func (h *Handlers) revokeAllSessionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := core.GetUser(ctx)
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	uid := ""
	if um, ok := any(user).(core.UserModel); ok {
		uid = um.GetID()
	} else if um2, ok := any(user).(*auth.User); ok {
		uid = um2.ID
	}

	// Use core SessionService.DeleteUserSessions
	if err := h.auth.Session.DeleteUserSessions(ctx, uid); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to revoke sessions",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "All sessions revoked successfully",
	})
}

// ========== END OF HANDLERS ==========
