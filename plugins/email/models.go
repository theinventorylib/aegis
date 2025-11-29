package email

import "time"

// Verification represents an email verification record
type Verification struct {
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

// ========== Request DTOs ==========

// LoginWithEmailRequest represents email+password login
type LoginWithEmailRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegisterWithEmailRequest represents email+password registration
type RegisterWithEmailRequest struct {
	Avatar   *string `json:"avatar"`
	Name     *string `json:"name"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
}
