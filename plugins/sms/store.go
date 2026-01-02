package sms

import (
	"context"
)

// SMSStore defines the interface for SMS verification storage operations.
//
// This interface provides phone-specific user management operations:
//   - User creation with phone numbers
//   - Phone lookup for authentication
//   - Phone verification status updates
//
// Thread Safety:
// Implementations must be safe for concurrent use.
type SMSStore interface {
	// ========== User Management ==========

	// CreateUser creates a new user with a phone number.
	//
	// The phone is initially unverified (phoneVerified: false).
	//
	// Parameters:
	//   - ctx: Request context
	//   - user: User with phone field populated in E.164 format
	//
	// Returns:
	//   - *User: Created user
	//   - error: If user creation fails (e.g., duplicate phone)
	CreateUser(ctx context.Context, user User) (*User, error)

	// ========== Phone Management ==========

	// GetUserByPhone retrieves a user by phone number.
	//
	// Used for:
	//   - Phone+password login
	//   - Checking if phone already exists during registration
	//   - Phone verification lookup
	//
	// Parameters:
	//   - ctx: Request context
	//   - phone: Phone number to lookup (E.164 format)
	//
	// Returns:
	//   - *User: User with matching phone
	//   - error: If user not found or database error
	GetUserByPhone(ctx context.Context, phone string) (*User, error)

	// UpdateUserPhone updates a user's phone number and verification status.
	//
	// Used after successful OTP verification to mark phone as verified.
	//
	// Parameters:
	//   - ctx: Request context
	//   - userID: User ID
	//   - phone: New phone number (E.164 format)
	//   - verified: Phone verification status
	//
	// Returns:
	//   - error: If update fails
	UpdateUserPhone(ctx context.Context, userID, phone string, verified bool) error
}
