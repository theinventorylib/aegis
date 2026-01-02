package emailotp

import (
	"context"
)

// EmailOTPStore defines the interface for Email OTP verification storage operations.
//
// This interface provides email-specific user management operations:
//   - User creation with email addresses
//   - Email lookup for authentication
//   - Email verification status updates
//
// Thread Safety:
// Implementations must be safe for concurrent use.
type EmailOTPStore interface {
	// ========== User Management ==========

	// CreateUser creates a new user with an email address.
	//
	// The email is initially unverified (emailVerified: false).
	//
	// Parameters:
	//   - ctx: Request context
	//   - user: User with email field populated
	//
	// Returns:
	//   - *User: Created user
	//   - error: If user creation fails (e.g., duplicate email)
	CreateUser(ctx context.Context, user User) (*User, error)

	// ========== Email Management ==========

	// GetUserByEmail retrieves a user by email address.
	//
	// Used for:
	//   - Email+password login
	//   - Checking if email already exists during registration
	//   - Email verification lookup
	//
	// Parameters:
	//   - ctx: Request context
	//   - email: Email address to lookup
	//
	// Returns:
	//   - *User: User with matching email
	//   - error: If user not found or database error
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// UpdateUserEmail updates a user's email address and verification status.
	//
	// Used after successful OTP verification to mark email as verified.
	//
	// Parameters:
	//   - ctx: Request context
	//   - userID: User ID
	//   - email: New email address
	//   - verified: Email verification status
	//
	// Returns:
	//   - error: If update fails
	UpdateUserEmail(ctx context.Context, userID, email string, verified bool) error
}
