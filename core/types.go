package core

import (
	"github.com/theinventorylib/aegis/models"
)

// Re-export commonly used types for convenience
type (
	User    = models.User
	Session = models.Session
	Account = models.Account
	JWK     = models.JWK
)

// SignupRequest represents a signup request payload
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a login request payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// OTPRequest represents an OTP generation request
type OTPRequest struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"` // "email_verification", "password_reset", "login"
}

// OTPVerifyRequest represents an OTP verification request
type OTPVerifyRequest struct {
	Email   string `json:"email"`
	Code    string `json:"code"`
	Purpose string `json:"purpose"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetConfirm represents a password reset confirmation
type PasswordResetConfirm struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

// Response represents a generic API response
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
