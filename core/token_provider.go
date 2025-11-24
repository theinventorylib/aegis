package core

import "time"

// TokenProvider interface allows plugins to provide custom token generation and validation.
type TokenProvider interface {
	GenerateTokenPair(userID string) (*TokenPair, error)
	ValidateToken(token string) (*TokenClaims, error)
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

// TokenClaims defines the structure of token claims.
type TokenClaims struct {
	UserID    string `json:"user_id"`
	TokenType string `json:"token_type"`
}
