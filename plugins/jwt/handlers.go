package jwt

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/theinventorylib/aegis/core"
	jwttypes "github.com/theinventorylib/aegis/plugins/jwt/types"
)

// Handler manages HTTP handlers for the JWT plugin.
//
// All handlers have been made private (lowercase) to encourage programmatic
// use of the underlying Plugin methods. This struct serves as a mounting point
// for the router.
type Handler struct {
	// plugin references the parent JWT plugin for token operations
	plugin *Plugin
}

// NewHandler creates a new JWT HTTP handler.
func NewHandler(plugin *Plugin) *Handler {
	return &Handler{plugin: plugin}
}

// TokenRequest represents a request to get a JWT token.
type TokenRequest struct {
	UserID string `json:"user_id"`
}

// RefreshTokenRequest represents a token refresh request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// handleGetToken generates JWT access + refresh tokens for an authenticated user.
func (h *Handler) handleGetToken(w http.ResponseWriter, r *http.Request) {
	// Get current user from context (must be authenticated)
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	// Generate token pair for the authenticated user
	tokenPair, err := h.plugin.GenerateTokenPair(user.ID)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to generate tokens",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Tokens generated successfully",
		Data:    tokenPair,
	})
}

// handleGetAccessToken generates a new access token for an authenticated user.
func (h *Handler) handleGetAccessToken(w http.ResponseWriter, r *http.Request) {
	// Get current user from context (must be authenticated)
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	// Generate token pair
	tokenPair, err := h.plugin.GenerateTokenPair(user.ID)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to generate access token",
		})
		return
	}

	// Return only access token info
	accessToken := &jwttypes.AccessToken{
		AccessToken:  tokenPair.AccessToken,
		AccessExpiry: tokenPair.AccessExpiry,
		TokenType:    "Bearer",
	}
	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Access token generated successfully",
		Data:    accessToken,
	})
}

// handleRefreshToken generates a new access token using a refresh token.
func (h *Handler) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := core.ReadJSON(r, &req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Trim whitespace only — the token is parsed, never stored or rendered,
	// and JWTs routinely exceed the default SanitizeString length limit
	// (which would silently truncate them and break refresh).
	req.RefreshToken = strings.TrimSpace(req.RefreshToken)

	// Use plugin RefreshTokens
	tokenPair, err := h.plugin.RefreshTokens(req.RefreshToken)
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Invalid or expired refresh token",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Tokens refreshed successfully",
		Data:    tokenPair,
	})
}

// handleLogout invalidates the current token. JWTs are blacklisted; opaque
// session tokens (from the session cookie) are deleted via the session
// service, since they are not JWTs and cannot be parsed/blacklisted.
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get current user from context
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	// Get token from cookie or header
	var token string
	cookieToken, err := h.plugin.sessionService.GetCookieManager().GetSessionCookie(r)
	if err == nil && cookieToken != "" {
		token = cookieToken
	} else {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// Scheme is case-insensitive per RFC 7235; accept "bearer".
			if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
				token = authHeader[7:]
			} else {
				token = authHeader
			}
		}
	}

	if token == "" {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "No session token found",
		})
		return
	}

	// A JWT has the compact JWS form (header.payload.signature); opaque
	// session tokens do not. Route each kind to its own invalidation path.
	if strings.Count(token, ".") == 2 {
		if err := h.plugin.Logout(token); err != nil {
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to logout",
			})
			return
		}
	} else {
		if err := h.plugin.sessionService.DeleteSession(r.Context(), token); err != nil {
			core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
				Success: false,
				Error:   "Failed to logout",
			})
			return
		}
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Logged out successfully",
		Data: &jwttypes.LogoutResponse{
			UserID: user.ID,
		},
	})
}

// handleJWKS serves the JSON Web Key Set (JWKS).
func (h *Handler) handleJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		core.WriteJSON(w, http.StatusMethodNotAllowed, &core.Response{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	// Get all non-expired public keys from store
	keys, err := h.plugin.store.ListJWKS(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to retrieve keys",
		})
		return
	}

	set := jwk.NewSet()
	for _, dbKey := range keys {
		// Only expose access-token signing keys. Refresh keys are stored
		// under a distinct "use" tag and must never be published.
		if dbKey.Use != useSigAccess {
			continue
		}
		key, err := jwk.ParseKey(dbKey.KeyData)
		if err != nil {
			continue
		}
		pubKey, err := key.PublicKey()
		if err != nil {
			continue
		}
		err = set.AddKey(pubKey)
		_ = err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	if err := json.NewEncoder(w).Encode(set); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to encode response",
		})
		return
	}
}
