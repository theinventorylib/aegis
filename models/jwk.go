package models

import "time"

// JWK represents a JSON Web Key for JWT signing/verification
type JWK struct {
	Kid       string                 `json:"kid"` // Key ID
	KeyData   map[string]interface{} `json:"keyData"`
	Algorithm string                 `json:"algorithm"` // "RS256", "ES256", etc.
	Use       string                 `json:"use"`       // "sig" (signing) or "enc" (encryption)
	CreatedAt time.Time              `json:"createdAt"`
	ExpiresAt *time.Time             `json:"expiresAt,omitempty"`
}
