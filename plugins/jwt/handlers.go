package jwt

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/theinventorylib/aegis/core"
)

// Handler manages HTTP handlers for the JWT plugin.
type Handler struct {
	plugin *Plugin
}

// NewHandler creates a new JWT handler.
func NewHandler(plugin *Plugin) *Handler {
	return &Handler{plugin: plugin}
}

// TokenRequest represents a request to get a JWT token for an authenticated user.
type TokenRequest struct {
	UserID string `json:"user_id"`
}

// RefreshTokenRequest represents a token refresh request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// HandleGetToken generates JWT tokens for an authenticated user.
// POST /token
func (h *Handler) HandleGetToken(w http.ResponseWriter, r *http.Request) {
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

// HandleGetAccessToken generates a new access token for an authenticated user.
// POST /getAccessToken
func (h *Handler) HandleGetAccessToken(w http.ResponseWriter, r *http.Request) {
	// Get current user from context (must be authenticated)
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	// Generate token pair (includes both access and refresh tokens)
	tokenPair, err := h.plugin.GenerateTokenPair(user.ID)
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to generate access token",
		})
		return
	}

	// Return only access token info
	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Access token generated successfully",
		Data: map[string]interface{}{
			"access_token":  tokenPair.AccessToken,
			"access_expiry": tokenPair.AccessExpiry,
			"token_type":    "Bearer",
		},
	})
}

// HandleRefreshToken refreshes JWT tokens using a refresh token.
// POST /refreshToken
func (h *Handler) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.WriteJSON(w, http.StatusBadRequest, &core.Response{
			Success: false,
			Error:   "Invalid request",
		})
		return
	}

	// Validate refresh token and generate new token pair
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

// HandleLogout blacklists the current access token (logout).
// POST /logout
func (h *Handler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get current user from context (must be authenticated via middleware)
	user, err := core.GetUser(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusUnauthorized, &core.Response{
			Success: false,
			Error:   "Not authenticated",
		})
		return
	}

	// Get token from either cookie or Authorization header (same logic as middleware)
	var token string

	// Try cookie first (preferred method)
	cookie, err := r.Cookie("aegis_session")
	if err == nil && cookie.Value != "" {
		token = cookie.Value
	} else {
		// Fall back to Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			// Remove "Bearer " prefix if present
			token = strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				// If no "Bearer " prefix was found, use the header as-is
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

	// Blacklist the token
	if err := h.plugin.BlacklistToken(token); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to logout",
		})
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Logged out successfully",
		Data: map[string]interface{}{
			"user_id": user.ID,
		},
	})
}

// HandleJWKS serves the JWKS endpoint (RFC 7517).
// This endpoint exposes public keys in JWKS format for JWT verification.
// GET /jwks or /.well-known/jwks.json
func (h *Handler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all non-expired public keys
	keys, err := h.plugin.db.ListJWKS(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve keys", http.StatusInternalServerError)
		return
	}

	// Create JWKS set with public keys only
	set := jwk.NewSet()
	for _, key := range keys {
		// Get public key from each JWK
		pubKey, err := key.PublicKey()
		if err != nil {
			// Skip keys that can't be converted to public keys
			continue
		}

		// Add to key set
		if err := set.AddKey(pubKey); err != nil {
			continue
		}
	}

	// Marshal the key set to JSON
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	if err := json.NewEncoder(w).Encode(set); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
