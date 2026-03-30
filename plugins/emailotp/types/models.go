// Package types defines the domain models and request/response types used by the emailotp plugin.
package types

import (
	"github.com/theinventorylib/aegis/auth"
)

// ========== Request DTOs ==========

// LoginWithEmailRequest is the request body for the email+password login endpoint.
// Authenticates an existing user using their registered email address and password.
type LoginWithEmailRequest struct {
	Email    string `json:"email"`    // User email address
	Password string `json:"password"` // User password
}

// RegisterWithEmailRequest is the request body for the email+password registration endpoint.
// Creates a new user account. Callers should trigger an email verification OTP
// (via SendOTPRequest with Purpose "email_verification") after a successful response.
type RegisterWithEmailRequest struct {
	Avatar   *string `json:"avatar"`   // Optional avatar URL
	Name     *string `json:"name"`     // User display name (required)
	Email    string  `json:"email"`    // User email address (required)
	Password string  `json:"password"` // User password (required)
}

// SendOTPRequest is the request body for the OTP dispatch endpoint.
// Triggers a one-time passcode to be sent to the given email address.
// Purpose controls which flow the OTP belongs to:
//   - "email_verification" — confirm a newly registered email address
//   - "password_reset"     — initiate a password reset flow
//   - "login_mfa"          — second-factor login challenge
type SendOTPRequest struct {
	Email   string `json:"email"`   // Email address to send OTP to
	UserID  string `json:"userId"`  // Aegis user ID requesting the OTP
	Purpose string `json:"purpose"` // OTP purpose ("email_verification", "password_reset", "login_mfa")
}

// VerifyOTPRequest is the request body for the OTP verification endpoint.
// Submits a one-time passcode to confirm a given email address or complete a flow.
// Purpose must match the value used in the corresponding SendOTPRequest.
type VerifyOTPRequest struct {
	Email   string `json:"email"`   // Email address to verify
	Code    string `json:"code"`    // OTP code to verify
	Purpose string `json:"purpose"` // OTP purpose
}

// ========== Extended User Model ==========

// User extends the core auth.User model with the email-specific verification flag.
// EmailVerified is false until the user completes an email verification flow
// (e.g., by verifying an OTP sent to their address via SendOTPRequest).
type User struct {
	auth.User
	EmailVerified bool `json:"emailVerified"` // Whether the user's email address has been verified
}
