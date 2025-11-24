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
