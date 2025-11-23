package models

import "time"

// Account represents an authentication account
// Supports multiple auth methods: password, OAuth providers, etc.
type Account struct {
	ID                string                 `json:"id"`
	UserID            string                 `json:"userId"`
	Provider          string                 `json:"provider"` // "password", "google", "github", "apple"
	ProviderAccountID *string                `json:"providerAccountId,omitempty"`
	PasswordHash      *string                `json:"-"` // Never expose in JSON
	AccessToken       *string                `json:"-"` // Never expose in JSON
	RefreshToken      *string                `json:"-"` // Never expose in JSON
	ExpiresAt         *time.Time             `json:"expiresAt,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}
