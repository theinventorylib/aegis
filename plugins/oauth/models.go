package oauth

import "time"

// Connection represents an OAuth provider connection for a user
type Connection struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"userId"`
	Provider       string                 `json:"provider"`
	ProviderUserID string                 `json:"providerUserId"`
	Email          string                 `json:"email,omitempty"`
	Name           string                 `json:"name,omitempty"`
	AvatarURL      string                 `json:"avatarUrl,omitempty"`
	AccessToken    string                 `json:"-"`                   // Never expose in JSON
	RefreshToken   string                 `json:"-"`                   // Never expose in JSON
	ExpiresAt      int64                  `json:"expiresAt,omitempty"` // Unix timestamp
	ProviderData   map[string]interface{} `json:"providerData,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}
