package jwt

import "time"

// JWK represents a JSON Web Key stored in the database.
//
// JWKs are cryptographic keys used for signing and verifying JWT tokens.
// They are stored in the database to support:
//   - Key rotation: Generate new keys periodically
//   - Key retention: Keep old keys for verifying existing tokens
//   - Multi-server: Share keys across multiple application instances
//
// Key Lifecycle:
//  1. Generate: Create RSA key pair (private + public)
//  2. Store: Save to database with expiry time
//  3. Use: Sign tokens with private key, verify with public key
//  4. Rotate: Generate new key, keep old key for verification
//  5. Expire: Delete keys after retention period
//
// Database Schema:
//
//	CREATE TABLE jwk_keys (
//	  kid VARCHAR(255) PRIMARY KEY,
//	  key_data BYTEA NOT NULL,
//	  algorithm VARCHAR(50) NOT NULL,
//	  use VARCHAR(50) NOT NULL,
//	  created_at TIMESTAMP NOT NULL,
//	  expires_at TIMESTAMP
//	);
type JWK struct {
	// Kid is the Key ID (unique identifier for this key)
	// Used in JWT header to identify which key was used for signing
	// Format: timestamp-based or UUID
	Kid string `json:"kid"`

	// KeyData is the JSON-encoded JWK (includes public/private key material)
	// Stored as BYTEA/BLOB in database
	// For RSA: Contains n (modulus), e (exponent), d (private exponent)
	KeyData []byte `json:"keyData"`

	// Algorithm is the cryptographic algorithm (e.g., "RS256", "ES256")
	// Currently only RS256 (RSA with SHA-256) is supported
	Algorithm string `json:"algorithm"`

	// Use indicates the key purpose:
	//   - "sig": Signature/verification (most common)
	//   - "enc": Encryption/decryption (future feature)
	Use string `json:"use"`

	// CreatedAt is when the key was generated
	CreatedAt time.Time `json:"createdAt"`

	// ExpiresAt is when the key should be deleted from storage
	// Set to CreatedAt + KeyRetention duration
	// nil means the key never expires (not recommended)
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// Provider defines an interface for custom JWT token generation and validation.
//
// Implement this interface to customize JWT token handling:
//   - Custom claims structure
//   - Custom signing algorithms
//   - Custom token validation logic
//   - Integration with external JWT services
//
// The default implementation (Plugin) uses RSA signing with database-backed keys.
// Most applications don't need a custom provider.
type Provider interface {
	// GenerateTokenPair creates an access token + refresh token for a user.
	// Returns token strings and expiry times.
	GenerateTokenPair(userID string) (*TokenPair, error)

	// ValidateToken validates a JWT token and returns the claims.
	// Checks signature, expiry, issuer, and token type.
	ValidateToken(token string) (*Claims, error)

	// RefreshTokens validates a refresh token and generates a new token pair.
	// Old refresh token is invalidated (single-use refresh tokens).
	RefreshTokens(refreshToken string) (*TokenPair, error)

	// BlacklistToken adds a token to the blacklist (for logout/revocation).
	// Requires Redis for distributed blacklist.
	BlacklistToken(token string) error
}

// TokenPair represents a complete set of access and refresh tokens.
//
// This is returned by token generation and refresh endpoints.
// Clients should:
//   - Store access token for API requests (short-lived)
//   - Store refresh token securely for obtaining new access tokens (long-lived)
//   - Use refresh token before access token expires
//
// Example response:
//
//	{
//	  "access_token": "eyJhbGc...",
//	  "access_expiry": "2024-01-01T12:15:00Z",
//	  "refresh_token": "eyJhbGc...",
//	  "refresh_expiry": "2024-01-08T12:00:00Z"
//	}
type TokenPair struct {
	// AccessToken is the JWT access token (use for API requests)
	AccessToken string `json:"access_token"`

	// AccessExpiry is when the access token expires (UTC)
	AccessExpiry time.Time `json:"access_expiry"`

	// RefreshToken is the JWT refresh token (use to get new access token)
	RefreshToken string `json:"refresh_token"`

	// RefreshExpiry is when the refresh token expires (UTC)
	RefreshExpiry time.Time `json:"refresh_expiry"`
}

// Claims defines the structure of JWT token claims.
//
// JWT claims are the payload embedded in the token. These are NOT encrypted,
// only signed - anyone can decode and read them. Don't include sensitive data.
//
// Standard claims (JWT spec):
//   - iss: Issuer (from Config.Issuer)
//   - sub: Subject (UserID)
//   - exp: Expiration time
//   - iat: Issued at time
//   - jti: JWT ID (for revocation tracking)
//
// Custom claims (Aegis):
//   - user_id: User ID for quick lookup
//   - token_type: "access" or "refresh"
//
// Example token payload:
//
//	{
//	  "iss": "aegis",
//	  "sub": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
//	  "exp": 1704114900,
//	  "iat": 1704114000,
//	  "user_id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
//	  "token_type": "access"
//	}
type Claims struct {
	// UserID is the authenticated user's ID
	// Used for quick user lookup without database query
	UserID string `json:"user_id"`

	// TokenType indicates if this is an "access" or "refresh" token
	// Access tokens can be used for API requests
	// Refresh tokens can only be used to get new access tokens
	TokenType string `json:"token_type"`
}

// AccessToken represents a single access token response.
//
// This is returned by the /getAccessToken endpoint when only an access token
// is needed (without a refresh token).
//
// Example response:
//
//	{
//	  "access_token": "eyJhbGc...",
//	  "access_expiry": "2024-01-01T12:15:00Z"
//	}
type AccessToken struct {
	// AccessToken is the JWT access token (use for API requests)
	AccessToken string `json:"access_token"`

	// AccessExpiry is when the access token expires (UTC)
	AccessExpiry time.Time `json:"access_expiry"`

	// TokenType is the auth scheme name returned to clients.
	// Aegis returns the constant "Bearer".
	TokenType string `json:"token_type"`
}

// LogoutResponse is the data payload returned by POST /logout.
type LogoutResponse struct {
	UserID string `json:"user_id"`
}

// JWKS represents a JSON Web Key Set response.
//
// This is returned by the /.well-known/jwks.json endpoint and contains
// the public keys used to verify JWT signatures.
//
// Example response:
//
//	{
//	  "keys": [
//	    {
//	      "kty": "RSA",
//	      "use": "sig",
//	      "kid": "access-1234567890",
//	      "n": "...",
//	      "e": "AQAB"
//	    }
//	  ]
//	}
type JWKS struct {
	// Keys is the array of public keys
	Keys []map[string]any `json:"keys"`
}
