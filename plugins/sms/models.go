package sms

import (
	"github.com/theinventorylib/aegis/auth"
)

// ========== Request DTOs ==========

// LoginWithPhoneRequest represents phone+password login request.
//
// Example:
//
//	{
//	  "phoneNumber": "+14155551234",
//	  "password": "SecurePassword123!"
//	}
type LoginWithPhoneRequest struct {
	PhoneNumber string `json:"phoneNumber"` // Phone number in E.164 format
	Password    string `json:"password"`    // User password
}

// RegisterWithPhoneRequest represents phone+password registration request.
//
// Example:
//
//	{
//	  "name": "John Doe",
//	  "phoneNumber": "+14155551234",
//	  "password": "SecurePassword123!",
//	  "avatar": "https://example.com/avatar.jpg"
//	}
type RegisterWithPhoneRequest struct {
	Avatar      *string `json:"avatar"`      // Optional avatar URL
	Name        *string `json:"name"`        // User display name (required)
	PhoneNumber string  `json:"phoneNumber"` // Phone number in E.164 format (required)
	Password    string  `json:"password"`    // User password (required)
}

// SendOTPRequest represents the request to send an OTP code.
//
// Example:
//
//	{
//	  "phoneNumber": "+14155551234",
//	  "userId": "user_123",
//	  "purpose": "phone_verification"
//	}
type SendOTPRequest struct {
	PhoneNumber string `json:"phoneNumber"` // Phone number to send OTP to
	UserID      string `json:"userId"`      // User ID requesting OTP
	Purpose     string `json:"purpose"`     // OTP purpose ("phone_verification", "password_reset", "login_mfa")
}

// VerifyOTPRequest represents the request to verify an OTP code.
//
// Example:
//
//	{
//	  "phoneNumber": "+14155551234",
//	  "code": "123456",
//	  "purpose": "phone_verification"
//	}
type VerifyOTPRequest struct {
	PhoneNumber string `json:"phoneNumber"` // Phone number to verify
	Code        string `json:"code"`        // OTP code to verify
	Purpose     string `json:"purpose"`     // OTP purpose
}

// ========== Extended User Model ==========

// User extends the core User model with phone-specific fields.
//
// This model adds phone verification status for display in API responses.
//
// Use this when:
//   - Returning user data after phone registration
//   - Displaying phone verification status in user profiles
//   - Checking if user has verified their phone
//
// Example:
//
//	user := sms.User{
//	  User: auth.User{ID: "user_123", Name: "John Doe"},
//	  Phone: ptr("+14155551234"),
//	  PhoneVerified: true,
//	}
type User struct {
	auth.User
	Phone         *string `json:"phone,omitempty"` // User phone number in E.164 format
	PhoneVerified bool    `json:"phoneVerified"`   // Phone verification status
}

// SMSAuthResponse is the data payload returned by SMS phone+password login
// and registration endpoints.
type SMSAuthResponse struct {
	User *auth.User `json:"user"`
}
