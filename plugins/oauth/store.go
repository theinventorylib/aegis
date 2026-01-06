package oauth

import (
	"context"
)

// Store defines the interface for OAuth connection storage operations.
//
// This interface abstracts database operations for managing OAuth provider
// connections. Implementations should use transactions for consistency and
// handle duplicate connection errors appropriately.
//
// Thread Safety:
// Implementations must be safe for concurrent use from multiple goroutines.
type Store interface {
	// CreateConnection creates a new OAuth provider connection.
	//
	// This method links an OAuth provider to an Aegis user account. It stores
	// the provider's user ID, access tokens, and user profile data.
	//
	// Constraints:
	//   - (provider, provider_user_id) must be unique
	//   - user_id must reference a valid auth.users.id
	//
	// Parameters:
	//   - ctx: Request context
	//   - conn: Connection data (ID, UserID, Provider, tokens, etc.)
	//
	// Returns:
	//   - *Connection: Created connection with generated ID
	//   - error: Duplicate connection or foreign key violation
	CreateConnection(ctx context.Context, conn Connection) (*Connection, error)

	// GetConnectionByProviderUserID retrieves a connection by provider and provider user ID.
	//
	// This method is used during OAuth login to check if a provider account is
	// already linked to an Aegis user. The provider user ID is the unique identifier
	// from the OAuth provider (e.g., Google's "108...", GitHub's "12345").
	//
	// Parameters:
	//   - ctx: Request context
	//   - provider: Provider name ("google", "github", etc.)
	//   - providerUserID: Provider's unique user ID
	//
	// Returns:
	//   - *Connection: Existing connection if found
	//   - error: ErrNotFound if no connection exists
	GetConnectionByProviderUserID(ctx context.Context, provider, providerUserID string) (*Connection, error)

	// GetConnectionsByUserID retrieves all OAuth connections for a user.
	//
	// This method returns all linked provider accounts for a user, useful for
	// displaying connected accounts in user settings.
	//
	// Parameters:
	//   - ctx: Request context
	//   - userID: Aegis user ID
	//
	// Returns:
	//   - []Connection: List of connections (may be empty)
	//   - error: Database query error
	GetConnectionsByUserID(ctx context.Context, userID string) ([]Connection, error)

	// UpdateConnection updates an existing OAuth connection.
	//
	// This method refreshes tokens or updates user profile data from the provider.
	// It updates the updated_at timestamp automatically.
	//
	// Parameters:
	//   - ctx: Request context
	//   - conn: Updated connection data (must have matching ID)
	//
	// Returns:
	//   - error: Connection not found or database error
	UpdateConnection(ctx context.Context, conn Connection) error

	// DeleteConnection removes an OAuth provider link from a user account.
	//
	// This method unlinks the specified provider from the user's account. The
	// user's Aegis account remains active.
	//
	// Parameters:
	//   - ctx: Request context
	//   - provider: Provider name ("google", "github", etc.)
	//   - userID: Aegis user ID
	//
	// Returns:
	//   - error: Connection not found or database error
	DeleteConnection(ctx context.Context, provider, userID string) error
}
