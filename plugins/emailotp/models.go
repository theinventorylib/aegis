package emailotp

import (
	"github.com/theinventorylib/aegis/auth"
)

// ========== Request DTOs ==========

// LoginWithEmailRequest represents email+password login request.
//
// Example:
//
//	{
//	  "email": "user@example.com",
//	  "password": "SecurePassword123!"
//	}
type LoginWithEmailRequest struct {
	Email    string `json:"email"`    // User email address
	Password string `json:"password"` // User password
}

// RegisterWithEmailRequest represents email+password registration request.
//
// Example:
//
//	{
//	  "name": "John Doe",
//	  "email": "john@example.com",
//	  "password": "SecurePassword123!",
//	  "avatar": "https://example.com/avatar.jpg"
//	}
type RegisterWithEmailRequest struct {
	Avatar   *string `json:"avatar"`   // Optional avatar URL
	Name     *string `json:"name"`     // User display name (required)
	Email    string  `json:"email"`    // User email address (required)
	Password string  `json:"password"` // User password (required)
}

// SendOTPRequest represents the request to send an OTP code.
//
// Example:
//
//	{
//	  "email": "user@example.com",
//	  "userId": "user_123",
//	  "purpose": "email_verification"
//	}
type SendOTPRequest struct {
	Email   string `json:"email"`   // Email address to send OTP to
	UserID  string `json:"userId"`  // User ID requesting OTP
	Purpose string `json:"purpose"` // OTP purpose ("email_verification", "password_reset", "login_mfa")
}

// VerifyOTPRequest represents the request to verify an OTP code.
//
// Example:
//
//	{
//	  "email": "user@example.com",
//	  "code": "123456",
//	  "purpose": "email_verification"
//	}
type VerifyOTPRequest struct {
	Email   string `json:"email"`   // Email address to verify
	Code    string `json:"code"`    // OTP code to verify
	Purpose string `json:"purpose"` // OTP purpose
}

// ========== Extended User Model ==========

// User extends the core User model with email-specific fields.
//
// This model adds email verification status for display in API responses.
//
// Use this when:
//   - Returning user data after email registration
//   - Displaying email verification status in user profiles
//   - Checking if user has verified their email
//
// Example:
//
//	user := emailotp.User{
//	  User: auth.User{ID: "user_123", Name: "John Doe"},
//	  Email: ptr("john@example.com"),
//	  EmailVerified: true,
//	}
type User struct {
	auth.User
	// TODO: email is alredy provided by the user, look into this
	Email         *string `json:"email,omitempty"` // User email address
	EmailVerified bool    `json:"emailVerified"`   // Email verification status
}
