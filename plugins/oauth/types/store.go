package types

import "context"

// Store defines the interface for OAuth connection storage operations.
//
// Thread Safety: Implementations must be safe for concurrent use.
type Store interface {
	// CreateConnection creates a new OAuth provider connection.
	CreateConnection(ctx context.Context, conn Connection) (*Connection, error)

	// GetConnectionByProviderUserID retrieves a connection by provider and provider user ID.
	GetConnectionByProviderUserID(ctx context.Context, provider, providerUserID string) (*Connection, error)

	// GetConnectionsByUserID retrieves all OAuth connections for a user.
	GetConnectionsByUserID(ctx context.Context, userID string) ([]Connection, error)

	// UpdateConnection updates an existing OAuth connection.
	UpdateConnection(ctx context.Context, conn Connection) error

	// DeleteConnection removes an OAuth provider link from a user account.
	DeleteConnection(ctx context.Context, provider, userID string) error
}
