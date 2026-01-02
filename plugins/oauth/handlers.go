package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/markbates/goth"
	"github.com/theinventorylib/aegis/core"
)

// Handlers provides HTTP endpoint handlers for OAuth authentication.
//
// Endpoints:
//   - GET /auth/oauth/:provider - Start OAuth flow
//   - GET /auth/oauth/:provider/callback - Complete OAuth flow
//   - POST /auth/oauth/logout - Logout (clear cookies)
//   - POST /auth/oauth/link - Link provider to existing account
type Handlers struct {
	plugin *Plugin
}

// NewHandlers creates OAuth plugin HTTP handlers.
//
// Parameters:
//   - plugin: Initialized OAuth plugin
//
// Returns:
//   - *Handlers: Handler struct for registering routes
func NewHandlers(plugin *Plugin) *Handlers {
	return &Handlers{plugin: plugin}
}

// BeginAuthHandler starts the OAuth flow by redirecting to the provider.
//
// This endpoint initiates OAuth authentication with the specified provider.
// It generates a CSRF state token, stores it in a signed cookie, and redirects
// the user to the provider's authorization page.
//
// Endpoint:
//   - Method: GET
//   - Path: /auth/oauth/:provider
//   - Auth: None (public endpoint)
//
// Path Parameters:
//   - provider: Provider name ("google", "github", "apple", etc.)
//
// Response (302 Redirect):
//   - Location: Provider's authorization URL (e.g., https://accounts.google.com/o/oauth2/auth?...)
//   - Set-Cookie: _aegis_oauth_state=<signed-state-data>
//
// Response (400 Bad Request):
//
//	{
//	  "success": false,
//	  "error": "Provider not found" | "Failed to begin OAuth flow"
//	}
func (h *Handlers) BeginAuthHandler(w http.ResponseWriter, r *http.Request) {
	// Provider comes from URL path parameter
	provider := core.GetPathParam(r, "provider")
	if provider == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Provider required",
		})
		return
	}

	// Begin OAuth flow using Aegis's state management
	if err := h.plugin.BeginAuth(w, r, provider); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
}

// CallbackHandler handles the OAuth provider callback and creates a session.
//
// This endpoint completes the OAuth flow by:
//  1. Validating the CSRF state token
//  2. Exchanging the authorization code for an access token
//  3. Fetching the user's profile from the provider
//  4. Creating/linking an Aegis user account
//  5. Creating a session and setting the session cookie
//
// Endpoint:
//   - Method: GET
//   - Path: /auth/oauth/:provider/callback
//   - Auth: None (OAuth callback from provider)
//
// Query Parameters:
//   - code: Authorization code from provider
//   - state: CSRF state token (must match cookie)
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "data": {
//	    "user": {"id": "...", "email": "...", "name": "..."},
//	    "session": {"token": "...", "expires_at": "..."}
//	  }
//	}
//
// Response (500 Internal Server Error):
//
//	{
//	  "success": false,
//	  "error": "State validation failed" | "Failed to create user"
//	}
func (h *Handlers) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	user, session, err := h.plugin.CompleteAuth(r.Context(), w, r)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Set session cookie using CookieManager
	if h.plugin.sessionService != nil {
		h.plugin.sessionService.GetCookieManager().SetSessionCookie(w, session.Token)
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data: map[string]interface{}{
			"user":    user,
			"session": session,
		},
	})
}

// LogoutHandler clears OAuth state and session cookies.
//
// This endpoint handles logout by clearing both the OAuth state cookie
// (if present) and the session cookie. It does NOT revoke OAuth tokens
// at the provider.
//
// Endpoint:
//   - Method: POST
//   - Path: /auth/oauth/logout
//   - Auth: None (can be called without authentication)
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Logged out successfully"
//	}
func (h *Handlers) LogoutHandler(w http.ResponseWriter, _ *http.Request) {
	// Clear OAuth state cookie if present
	if h.plugin.stateStore != nil {
		h.plugin.stateStore.ClearState(w)
	}

	// Clear session cookie using CookieManager
	if h.plugin.sessionService != nil {
		h.plugin.sessionService.GetCookieManager().ClearSessionCookie(w)
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Logged out successfully",
	})
}

// LinkAccountRequest represents a request to link an OAuth account.
//
// This struct defines the request body for the account linking endpoint.
type LinkAccountRequest struct {
	Provider string `json:"provider"` // Provider to link ("google", "github", etc.)
}

// LinkAccountHandler links an OAuth provider to the current authenticated user.
//
// This endpoint allows users to add additional OAuth providers to their account.
// For example, a user who signed up with email/password can link their Google
// account for easier login.
//
// OAuth Flow:
//  1. User initiates OAuth flow (GET /auth/oauth/google)
//  2. User authenticates with provider
//  3. Provider redirects to this endpoint with code and state
//  4. Endpoint validates state, exchanges code, links to current user
//
// Endpoint:
//   - Method: POST
//   - Path: /auth/oauth/link (custom path, not auto-registered)
//   - Auth: Required (must be authenticated to link account)
//
// Request Body:
//
//	{
//	  "provider": "google"
//	}
//
// Query Parameters (from OAuth callback):
//   - code: Authorization code from provider
//   - state: CSRF state token (must match cookie)
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "OAuth account linked successfully"
//	}
//
// Response (401 Unauthorized):
//
//	{
//	  "success": false,
//	  "error": "Not authenticated"
//	}
//
// Response (400 Bad Request):
//
//	{
//	  "success": false,
//	  "error": "Provider not found" | "Invalid state"
//	}
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
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Get provider
	provider, err := goth.GetProvider(req.Provider)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   fmt.Sprintf("Provider %s not found", req.Provider),
		})
		return
	}

	// Validate state from callback
	callbackState := r.URL.Query().Get("state")
	stateData, err := h.plugin.stateStore.ValidateState(r, callbackState)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid state",
		})
		return
	}

	// Clear the state cookie
	h.plugin.stateStore.ClearState(w)

	// Unmarshal session and authorize
	sess, err := provider.UnmarshalSession(stateData.SessionData)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid session data",
		})
		return
	}

	params := r.URL.Query()
	if _, err := sess.Authorize(provider, params); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   fmt.Sprintf("Authorization failed: %v", err),
		})
		return
	}

	// Fetch user from provider
	gothUser, err := provider.FetchUser(sess)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   fmt.Sprintf("Failed to fetch user: %v", err),
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
