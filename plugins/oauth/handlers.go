package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/markbates/goth"
	"github.com/theinventorylib/aegis/core"
	oauthtypes "github.com/theinventorylib/aegis/plugins/oauth/types"
)

// Handlers provides HTTP endpoint handlers for OAuth authentication.
//
// All handlers have been made private (lowercase) to encourage programmatic
// use of the underlying Plugin methods. This struct serves as a mounting point
// for the router.
type Handlers struct {
	plugin *Plugin
}

// NewHandlers creates OAuth plugin HTTP handlers.
func NewHandlers(plugin *Plugin) *Handlers {
	return &Handlers{plugin: plugin}
}

// beginAuthHandler starts the OAuth flow by redirecting to the provider.
func (h *Handlers) beginAuthHandler(w http.ResponseWriter, r *http.Request) {
	// Provider comes from URL path parameter
	provider := core.GetSanitizedPathParam(r, "provider")
	if provider == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Provider required",
		})
		return
	}

	// Sanitize provider name
	provider = core.SanitizeString(provider, nil)

	// Begin OAuth flow using plugin method
	if err := h.plugin.BeginAuth(w, r, provider); err != nil {
		if h.plugin.logger != nil {
			h.plugin.logger.Error("oauth: BeginAuth failed",
				"provider", provider,
				"error", err)
		}
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Unable to start OAuth flow",
		})
		return
	}
}

// callbackHandler handles the OAuth provider callback and creates a session.
func (h *Handlers) callbackHandler(w http.ResponseWriter, r *http.Request) {
	user, session, err := h.plugin.CompleteAuth(r.Context(), w, r)
	if err != nil {
		if h.plugin.logger != nil {
			h.plugin.logger.Error("oauth: CompleteAuth failed",
				"provider", core.GetSanitizedPathParam(r, "provider"),
				"error", err)
		}
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "OAuth authentication failed",
		})
		return
	}

	// Set session cookie using CookieManager
	if h.plugin.sessionService != nil {
		h.plugin.sessionService.GetCookieManager().SetSessionCookie(w, session.Token)
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "OAuth login successful",
		Data: &core.SessionWithUser{
			Session: session,
			User:    core.NewEnrichedUser(&user.User),
		},
	})
}

// logoutHandler clears OAuth state and session cookies.
func (h *Handlers) logoutHandler(w http.ResponseWriter, _ *http.Request) {
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

// refreshTokenHandler refreshes the OAuth access token for the authenticated user.
func (h *Handlers) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	provider := core.GetSanitizedPathParam(r, "provider")
	if provider == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Provider required",
		})
		return
	}

	provider = core.SanitizeString(provider, nil)

	conn, err := h.plugin.RefreshConnection(r.Context(), user.ID, provider)
	if err != nil {
		if h.plugin.logger != nil {
			h.plugin.logger.Error("oauth: RefreshConnection failed",
				"user_id", user.ID, "provider", provider, "error", err)
		}
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Unable to refresh OAuth token",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data: &oauthtypes.TokenRefreshResponse{
			Provider:  conn.Provider,
			ExpiresAt: conn.ExpiresAt,
		},
	})
}

// linkAccountHandler links an OAuth provider to the current authenticated user.
func (h *Handlers) linkAccountHandler(w http.ResponseWriter, r *http.Request) {
	// Get current user from context
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	var req oauthtypes.LinkAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Sanitize provider name before using in error message
	providerName := core.SanitizeString(req.Provider, nil)

	// Get provider
	provider, err := goth.GetProvider(req.Provider)
	if err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   fmt.Sprintf("Provider %s not found", providerName),
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
		if h.plugin.logger != nil {
			h.plugin.logger.Error("oauth: Authorize failed",
				"user_id", user.ID, "provider", providerName, "error", err)
		}
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Authorization failed",
		})
		return
	}

	// Fetch user from provider
	gothUser, err := provider.FetchUser(sess)
	if err != nil {
		if h.plugin.logger != nil {
			h.plugin.logger.Error("oauth: FetchUser failed",
				"user_id", user.ID, "provider", providerName, "error", err)
		}
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Failed to fetch user from provider",
		})
		return
	}

	// Link the account
	oauthUser := GothUserToUser(gothUser)
	if err := h.plugin.LinkAccount(r.Context(), user.ID, oauthUser, gothUser.Provider); err != nil {
		if h.plugin.logger != nil {
			h.plugin.logger.Error("oauth: LinkAccount failed",
				"user_id", user.ID, "provider", providerName, "error", err)
		}
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to link OAuth account",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "OAuth account linked successfully",
	})
}
