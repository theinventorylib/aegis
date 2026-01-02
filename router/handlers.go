package router

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/core"
)

// Handlers provides HTTP handlers for core Aegis authentication routes.
//
// This struct contains all the standard HTTP handlers that Aegis provides
// out of the box:
//   - Logout: Invalidate session and clear cookie
//   - User: Get current user data (with plugin extensions)
//   - Sessions: List, get, refresh, revoke sessions
//
// Handlers is created by MountRoutes and integrates with the Router interface.
// For custom handlers, you can create additional handler structs that use
// core.AuthService directly.
//
// Example:
//
//	handlers := router.NewHandlers(authService)
//	r.POST("/auth/logout", handlers.LogoutHandler)
type Handlers struct {
	// auth provides access to all authentication services
	auth *core.AuthService
}

// NewHandlers creates a new Handlers instance with the given AuthService.
//
// The AuthService provides access to User, Account, Session, and Verification
// services needed by the handlers.
func NewHandlers(auth *core.AuthService) *Handlers {
	return &Handlers{
		auth: auth,
	}
}

// LogoutHandler handles user logout by invalidating the session and clearing the cookie.
//
// Logout flow:
//  1. Extract session token from cookie (primary) or Authorization header (fallback)
//  2. Delete the session from database and Redis cache
//  3. Clear the session cookie (set MaxAge=-1)
//  4. Return success response
//
// This handler is "best-effort" - even if the session deletion fails (e.g., session
// already expired, database error), the cookie is still cleared and success is returned.
// This prevents client-side logout failures while still attempting server-side cleanup.
//
// Security notes:
//   - Client-side cookie is always cleared (prevents reuse of expired tokens)
//   - Logout is idempotent (multiple logouts don't cause errors)
//   - Works even if session is already invalid (graceful degradation)
//
// Request:
//
//	POST /auth/logout
//	Cookie: aegis_session=<token>
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Logged out successfully"
//	}
func (h *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie or header using CookieManager (respects configured cookie name)
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
		if err := h.auth.Session.Logout(r.Context(), token); err != nil {
			// Logout is best-effort - log but don't fail the request
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

// UserHandler returns the current authenticated user with plugin extensions.
//
// This handler returns the enriched user data, which includes:
//   - Base user fields (id, email, name, created_at, etc.)
//   - Plugin extension fields (organization_id from org plugin, roles from admin plugin, etc.)
//
// Plugin extensions are flattened into the top-level JSON response using the
// EnrichedUser.ToAPIResponseFiltered() method, which respects UserFieldsConfig
// for whitelisting/blacklisting fields.
//
// Security notes:
//   - Requires authentication (protected route with RequireAuthMiddleware)
//   - Only returns data for the authenticated user (no user enumeration)
//   - Respects field visibility configuration (sensitive fields can be hidden)
//
// Request:
//
//	GET /auth/user
//	Cookie: aegis_session=<token>
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
//	    "email": "user@example.com",
//	    "name": "John Doe",
//	    "organization_id": "org_123",  // from organization plugin
//	    "role": "admin",  // from admin plugin
//	    "created_at": "2024-01-01T00:00:00Z",
//	    "updated_at": "2024-01-01T00:00:00Z"
//	  }
//	}
//
// Response (401 Unauthorized - Not authenticated):
//
//	{
//	  "success": false,
//	  "error": "Not authenticated"
//	}
func (h *Handlers) UserHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Try to get enriched user first (includes plugin extensions)
	if enriched := core.GetEnrichedUser(ctx); enriched != nil {
		config := h.auth.GetUserFieldsConfig()
		core.WriteJSON(w, http.StatusOK, &core.Response{
			Success: true,
			Data:    enriched.ToAPIResponseFiltered(config),
		})
		return
	}

	// Fallback to basic user
	user, err := core.GetUser(ctx)
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data:    user,
	})
}

// ========== SESSION MANAGEMENT HANDLERS ==========

// RefreshSessionHandler refreshes an existing session using a refresh token.
//
// Refresh tokens are long-lived tokens (default: 7 days) that can be used to
// obtain new session tokens without re-authentication. This allows "remember me"
// functionality and persistent logins.
//
// Refresh flow:
//  1. Validate the refresh token
//  2. Check if the refresh token is still valid (not expired/revoked)
//  3. Create a new session token with updated expiry
//  4. Optionally rotate the refresh token (create new refresh token)
//  5. Set new session cookie
//  6. Return new session data
//
// Security notes:
//   - Refresh tokens are single-use (refresh token rotation prevents replay attacks)
//   - Rate limiting recommended (prevent token brute force)
//   - Refresh tokens can be revoked (session deletion also invalidates refresh tokens)
//
// Request:
//
//	POST /auth/session/refresh
//	Content-Type: application/json
//	{
//	  "refreshToken": "<refresh_token>"
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Session refreshed",
//	  "data": {
//	    "expiresAt": "2024-01-08T00:00:00Z"
//	  }
//	}
//
// Response (401 Unauthorized - Invalid/expired token):
//
//	{
//	  "success": false,
//	  "error": "Invalid or expired refresh token"
//	}
func (h *Handlers) RefreshSessionHandler(w http.ResponseWriter, r *http.Request) {
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

	// Refresh the session.
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
	expiresAt := ""
	if sm, ok := any(newSession).(core.SessionModel); ok {
		token = sm.GetToken()
		expiresAt = sm.GetExpiresAt().String()
	} else if sm2, ok := any(newSession).(*auth.Session); ok {
		token = sm2.Token
		expiresAt = sm2.ExpiresAt.String()
	}

	if token != "" {
		h.auth.Session.GetCookieManager().SetSessionCookie(w, token)
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Session refreshed",
		Data: map[string]interface{}{
			"expiresAt": expiresAt,
		},
	})
}

// GetCurrentSessionHandler returns the current session information.
//
// This endpoint returns ONLY session data (token, expiry, IP, user agent, etc.)
// without user information. Use /auth/user to get user data.
//
// Useful for:
//   - Checking session expiry time
//   - Verifying session IP address and user agent
//   - Debugging authentication issues
//   - Displaying "logged in from" information to users
//
// Request:
//
//	GET /auth/session
//	Cookie: aegis_session=<token>
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "id": "session_123",
//	    "user_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
//	    "expires_at": "2024-01-08T00:00:00Z",
//	    "ip_address": "192.168.1.1",
//	    "user_agent": "Mozilla/5.0 ...",
//	    "created_at": "2024-01-01T00:00:00Z"
//	  }
//	}
func (h *Handlers) GetCurrentSessionHandler(w http.ResponseWriter, r *http.Request) {
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
			Data: map[string]interface{}{
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

// ListSessionsHandler lists all active sessions for the current user.
func (h *Handlers) ListSessionsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
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

	sessions, err := h.auth.Session.GetUserSessions(r.Context(), uid)
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

// RevokeSessionHandler revokes a specific session.
func (h *Handlers) RevokeSessionHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	// Get session ID from path parameter
	sessionID := core.GetPathParam(r, "id")
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

	// Verify the session belongs to the current user before deleting
	sessions, err := h.auth.Session.GetUserSessions(r.Context(), uid)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to verify session ownership",
		})
		return
	}

	// Find the session and get its token for deletion
	var sessionToken string
	for _, s := range sessions {
		sid := ""
		stoken := ""
		if sm, ok := any(&s).(core.SessionModel); ok {
			sid = sm.GetID()
			stoken = sm.GetToken()
		} else if sm2, ok := any(&s).(*auth.Session); ok {
			sid = sm2.ID
			stoken = sm2.Token
		}

		if sid == sessionID {
			sessionToken = stoken
			break
		}
	}

	if sessionToken == "" {
		core.WriteJSON(w, http.StatusNotFound, &core.Response{
			Success: false,
			Error:   "Session not found or does not belong to user",
		})
		return
	}

	// Delete session by token
	if err := h.auth.Session.DeleteSession(r.Context(), sessionToken); err != nil {
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

// RevokeAllSessionsHandler revokes all sessions for the current user.
func (h *Handlers) RevokeAllSessionsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
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

	if err := h.auth.Session.DeleteUserSessions(r.Context(), uid); err != nil {
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
