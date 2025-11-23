package models

import "time"

// Verification represents a generic verification token
// Used for email verification, password reset, magic links, etc.
type Verification struct {
	ID         string    `json:"id"`
	Identifier string    `json:"identifier"` // email, phone, user_id
	Token      string    `json:"token"`
	Type       string    `json:"type"` // "email_verification", "password_reset", "magic_link"
	ExpiresAt  time.Time `json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
}
