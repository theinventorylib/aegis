package types

import "context"

// Store defines the interface for Email OTP verification storage operations.
//
// This interface provides email-specific user management operations:
//   - User creation with email addresses
//   - Email lookup for authentication
//   - Email verification status updates
//
// Thread Safety:
// Implementations must be safe for concurrent use.
type Store interface {
	// CreateUser creates a new user with an email address.
	//
	// The email is initially unverified (emailVerified: false).
	CreateUser(ctx context.Context, user User) (*User, error)

	// GetUserByEmail retrieves a user by email address.
	GetUserByEmail(ctx context.Context, email string) (*User, error)

	// UpdateUserEmail updates a user's email address and verification status.
	UpdateUserEmail(ctx context.Context, userID, email string, verified bool) error
}
