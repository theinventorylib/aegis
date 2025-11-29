package jwt

import "time"

// JWK represents a JSON Web Key stored in the database.
type JWK struct {
	Kid       string     `json:"kid"`       // Key ID
	KeyData   []byte     `json:"keyData"`   // JSON-encoded JWK
	Algorithm string     `json:"algorithm"` // Algorithm (e.g., "RS256")
	Use       string     `json:"use"`       // Key use ("sig" for signing, "enc" for encryption)
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// Provider defines an interface for custom token generation and validation.
type Provider interface {
	GenerateTokenPair(userID string) (*TokenPair, error)
	ValidateToken(token string) (*Claims, error)
	RefreshTokens(refreshToken string) (*TokenPair, error)
	BlacklistToken(token string) error
}

// TokenPair represents a set of access and refresh tokens.
type TokenPair struct {
	AccessToken   string    `json:"access_token"`
	AccessExpiry  time.Time `json:"access_expiry"`
	RefreshToken  string    `json:"refresh_token"`
	RefreshExpiry time.Time `json:"refresh_expiry"`
}

// Claims defines the structure of JWT token claims.
type Claims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
}
