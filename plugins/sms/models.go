package sms

import "time"

// Verification represents a phone number verification record
type Verification struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId,omitempty"`
	PhoneNumber string    `json:"phoneNumber"`
	Code        string    `json:"-"` // Never expose in JSON
	Purpose     string    `json:"purpose"`
	Verified    bool      `json:"verified"`
	ExpiresAt   time.Time `json:"expiresAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ========== Request DTOs ==========

// LoginWithPhoneRequest represents phone+password login
type LoginWithPhoneRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Password    string `json:"password"`
}

// RegisterWithPhoneRequest represents phone+password registration
type RegisterWithPhoneRequest struct {
	Avatar      *string `json:"avatar"`
	Name        *string `json:"name"`
	PhoneNumber string  `json:"phoneNumber"`
	Password    string  `json:"password"`
}

// SendOTPRequest represents the request to send an OTP
type SendOTPRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	UserID      string `json:"userId"`
	Purpose     string `json:"purpose"` // "phone_verification", "password_reset", "login_mfa"
}

// VerifyOTPRequest represents the request to verify an OTP
type VerifyOTPRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Code        string `json:"code"`
	Purpose     string `json:"purpose"`
}
