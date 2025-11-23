package models

import "time"

// Session represents an active user session
type Session struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"userId"`
	Token        string                 `json:"token"`
	RefreshToken string                 `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time              `json:"expiresAt"`
	CreatedAt    time.Time              `json:"createdAt"`
	IPAddress    string                 `json:"ipAddress,omitempty"`
	UserAgent    string                 `json:"userAgent,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
