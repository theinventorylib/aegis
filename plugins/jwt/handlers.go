package jwt

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/theinventorylib/aegis/core"
)

// Handler manages HTTP handlers for the JWT plugin.
//
// This struct provides HTTP endpoints for JWT token operations:
//   - Token generation: Get JWT tokens for authenticated users
//   - Token refresh: Exchange refresh token for new access token
//   - Token revocation: Blacklist tokens (logout)
//   - JWKS endpoint: Expose public keys for token verification
//
// All endpoints except JWKS require authentication via core AuthMiddleware.
type Handler struct {
	// plugin references the parent JWT plugin for token operations
	plugin *Plugin
}

// NewHandler creates a new JWT HTTP handler.
//
// This is called during plugin initialization to create the handler instance.
func NewHandler(plugin *Plugin) *Handler {
	return &Handler{plugin: plugin}
}

// TokenRequest represents a request to get a JWT token for an authenticated user.
//
// This struct is currently unused - token generation uses the authenticated
// user from context instead of accepting user_id in request body.
//
// Reserved for future use (admin token generation for other users).
type TokenRequest struct {
	// UserID is the user to generate tokens for
	UserID string `json:"user_id"`
}

// RefreshTokenRequest represents a token refresh request.
//
// Request body:
//
//	{
//	  "refresh_token": "eyJhbGciOiJSUzI1NiIs..."
//	}
type RefreshTokenRequest struct {
	// RefreshToken is the long-lived refresh token (from previous token generation)
	RefreshToken string `json:"refresh_token"`
}

// HandleGetToken generates JWT access + refresh tokens for an authenticated user.
//
// Authentication: REQUIRED (via core AuthMiddleware)
//
// Flow:
//  1. Get authenticated user from request context
//  2. Generate JWT access token (15min expiry)
//  3. Generate JWT refresh token (7d expiry)
//  4. Return token pair
//
// Use this after logging in with email/password, OAuth, etc. to get JWT tokens
// for subsequent API requests.
//
// Request:
//
//	POST /jwt/token
//	Cookie: aegis_session=<session_token>
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Tokens generated successfully",
//	  "data": {
//	    "access_token": "eyJhbGc...",
//	    "access_expiry": "2024-01-01T12:15:00Z",
//	    "refresh_token": "eyJhbGc...",
//	    "refresh_expiry": "2024-01-08T12:00:00Z"
//	  }
//	}
//
// Response (401 Unauthorized):
//
//	{
//	  "success": false,
//	  "error": "Not authenticated"
//	}
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
//
// This endpoint is similar to HandleGetToken but returns only the access token
// without the refresh token. Useful for clients that only need short-lived
// access tokens and don't want to manage refresh tokens.
//
// Authentication:
// Requires authentication via AuthMiddleware. User must have a valid session
// (cookie or JWT in Authorization header) to call this endpoint.
//
// Endpoint:
//   - Method: POST
//   - Path: /getAccessToken
//   - Auth: Required (AuthMiddleware)
//
// Request:
//   - No body required (uses authenticated user from context)
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Access token generated successfully",
//	  "data": {
//	    "access_token": "eyJhbGc...",
//	    "access_expiry": "2024-01-01T12:15:00Z",
//	    "token_type": "Bearer"
//	  }
//	}
//
// Response (401 Unauthorized):
//
//	{
//	  "success": false,
//	  "error": "Not authenticated"
//	}
//
// Use Cases:
//   - Mobile apps that use native refresh token storage
//   - Microservices requesting access tokens for API calls
//   - Clients that prefer separate endpoints for access vs. refresh tokens
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

// HandleRefreshToken generates a new access token using a refresh token.
//
// This endpoint implements the OAuth 2.0 refresh token flow for JWT-based authentication.
// When an access token expires, clients can use this endpoint to obtain a new token pair
// without re-authenticating.
//
// Token Rotation:
// This endpoint generates BOTH a new access token AND a new refresh token. The old
// refresh token is NOT blacklisted (can still be used once). For stricter security,
// consider implementing refresh token rotation with blacklisting.
//
// Endpoint:
//   - Method: POST
//   - Path: /refreshToken
//   - Auth: None (uses refresh token in request body)
//
// Request Body:
//
//	{
//	  "refresh_token": "eyJhbGc..."
//	}
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Tokens refreshed successfully",
//	  "data": {
//	    "access_token": "eyJhbGc...",
//	    "access_expiry": "2024-01-01T12:15:00Z",
//	    "refresh_token": "eyJhbGc...",
//	    "refresh_expiry": "2024-01-08T12:00:00Z"
//	  }
//	}
//
// Response (401 Unauthorized):
//
//	{
//	  "success": false,
//	  "error": "Invalid or expired refresh token"
//	}
//
// Security Considerations:
//   - Refresh tokens are long-lived (default 7 days) - store securely
//   - Validate token signature, expiration, and claims before issuing new tokens
//   - Consider rate limiting to prevent refresh token abuse
//   - Log refresh token usage for security monitoring
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

// HandleLogout invalidates the current access token by adding it to the Redis blacklist.
//
// This endpoint provides logout functionality for JWT-based authentication. Since JWTs
// are stateless, we can't "delete" a token - instead, we add it to a Redis blacklist
// that the authentication middleware checks on each request.
//
// Token Extraction:
// The token is extracted from either:
//  1. Cookie (using SessionService.CookieManager configured cookie name)
//  2. Authorization header ("Bearer <token>" or raw token)
//
// Blacklist Storage:
// Tokens are stored in Redis with a TTL matching the token's expiration time.
// Once a token expires naturally, it's automatically removed from Redis.
//
// Endpoint:
//   - Method: POST
//   - Path: /logout
//   - Auth: Required (AuthMiddleware)
//
// Request:
//   - No body required (uses token from cookie or Authorization header)
//
// Response (200 OK):
//
//	{
//	  "success": true,
//	  "message": "Logged out successfully",
//	  "data": {
//	    "user_id": "user-123"
//	  }
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
//	  "error": "No session token found"
//	}
//
// Security Considerations:
//   - Requires Redis for blacklist storage (fails if Redis unavailable)
//   - Tokens remain valid until Redis blacklist check completes
//   - Consider clearing refresh tokens on logout (not currently implemented)
//   - Blacklist uses memory - monitor Redis usage for high-traffic apps
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

	// Try cookie first using the configured cookie name via CookieManager
	cookieToken, err := h.plugin.sessionService.GetCookieManager().GetSessionCookie(r)
	if err == nil && cookieToken != "" {
		token = cookieToken
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

// HandleJWKS serves the JSON Web Key Set (JWKS) endpoint per RFC 7517.
//
// This endpoint exposes PUBLIC keys only in JWKS format, allowing external services
// to verify JWTs issued by this server without sharing private keys. This is the
// standard mechanism for distributed JWT verification (e.g., microservices, SPAs).
//
// JWKS Format (RFC 7517):
// Returns a JSON object with a "keys" array containing JWK objects:
//
//	{
//	  "keys": [
//	    {
//	      "kty": "RSA",
//	      "use": "sig",
//	      "kid": "key-2024-01",
//	      "n": "base64url-encoded-modulus",
//	      "e": "AQAB"
//	    }
//	  ]
//	}
//
// Key Filtering:
//   - Only includes non-expired keys (expires_at > NOW() OR expires_at IS NULL)
//   - Extracts public key material only (private keys are NOT exposed)
//   - Skips keys that fail parsing or public key extraction
//
// Multiple Keys:
// During key rotation, multiple keys may be returned to support verification of
// tokens signed with both old and new keys during the transition period.
//
// Endpoint:
//   - Method: GET only (405 for other methods)
//   - Path: /jwks or /.well-known/jwks.json (RFC 8414 standard path)
//   - Auth: None (public endpoint)
//
// Response Headers:
//   - Content-Type: application/json
//   - Cache-Control: public, max-age=3600 (cache for 1 hour)
//
// Response (200 OK):
//   - Raw JWKS JSON (not wrapped in {success, data} envelope)
//
// Response (500 Internal Server Error):
//
//	{
//	  "success": false,
//	  "error": "Failed to retrieve keys"
//	}
//
// Client Usage:
//
//	// Fetch JWKS from server
//	resp, _ := http.Get("https://auth.example.com/.well-known/jwks.json")
//	var jwks map[string]interface{}
//	json.NewDecoder(resp.Body).Decode(&jwks)
//
//	// Verify JWT using fetched public keys
//	token, _ := jwt.Parse(tokenString, jwk.NewSet(jwks))
//
// Security Considerations:
//   - PUBLIC endpoint - no authentication required
//   - Only exposes public keys (safe to share)
//   - Cache response to reduce database load
//   - Monitor for excessive requests (potential DoS)
func (h *Handler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		core.WriteJSON(w, http.StatusMethodNotAllowed, &core.Response{
			Success: false,
			Error:   "Method not allowed",
		})
		return
	}

	// Get all non-expired public keys
	keys, err := h.plugin.store.ListJWKS(r.Context())
	if err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to retrieve keys",
		})
		return
	}

	// Create JWKS set with public keys only
	set := jwk.NewSet()
	for _, dbKey := range keys {
		// Parse the stored JWK from JSON
		key, err := jwk.ParseKey(dbKey.KeyData)
		if err != nil {
			// Skip keys that can't be parsed
			continue
		}

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

	// Marshal the key set to JSON (JWKS format, not wrapped in Response)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	if err := json.NewEncoder(w).Encode(set); err != nil {
		core.WriteJSON(w, http.StatusInternalServerError, &core.Response{
			Success: false,
			Error:   "Failed to encode response",
		})
		return
	}
}
