package email

import "time"

// EmailVerification represents an email verification record
type EmailVerification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId,omitempty"`
	Email     string    `json:"email"`
	Code      string    `json:"-"` // OTP code - never expose in JSON
	Token     string    `json:"-"` // Token for link-based - never expose in JSON
	Purpose   string    `json:"purpose"`
	Verified  bool      `json:"verified"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}
