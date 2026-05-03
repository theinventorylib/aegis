package types

import "context"

// UserStore defines the interface for user storage operations.
// Implementations of this interface manage the persistence and retrieval
// of user identity data.
//
// All methods accept a context for cancellation and deadline support.
// Implementations should respect context cancellation and return
// context.Canceled or context.DeadlineExceeded when appropriate.
type UserStore interface {
	// Create persists a new user to storage.
	// Returns the created user with any storage-assigned fields populated.
	// Returns an error if a user with the same email already exists.
	Create(ctx context.Context, user User) (User, error)

	// GetByEmail retrieves a user by their email address.
	// Returns sql.ErrNoRows or equivalent if no user is found.
	GetByEmail(ctx context.Context, email string) (User, error)

	// GetByID retrieves a user by their unique identifier.
	// Returns sql.ErrNoRows or equivalent if no user is found.
	GetByID(ctx context.Context, id string) (User, error)

	// Update modifies an existing user's data.
	// Returns an error if the user does not exist.
	Update(ctx context.Context, user User) error

	// Delete removes a user from storage (may be soft or hard delete).
	// Returns an error if the user does not exist.
	Delete(ctx context.Context, id string) error

	// List retrieves a paginated list of users.
	// The offset and limit parameters control pagination.
	List(ctx context.Context, offset, limit int) ([]User, error)

	// Count returns the total number of users in storage.
	Count(ctx context.Context) (int, error)
}

// AccountStore defines the interface for account storage operations.
// Implementations manage provider-specific authentication accounts that
// are linked to users.
//
// A single user can have multiple accounts (one per authentication provider).
type AccountStore interface {
	// Create persists a new account to storage.
	// Returns an error if an account with the same provider and provider
	// account ID already exists.
	Create(ctx context.Context, account Account) error

	// GetByID retrieves an account by its unique identifier.
	// Returns sql.ErrNoRows or equivalent if no account is found.
	GetByID(ctx context.Context, id string) (Account, error)

	// GetByUserID retrieves all accounts belonging to a specific user.
	// Returns an empty slice if the user has no accounts.
	GetByUserID(ctx context.Context, userID string) ([]Account, error)

	// GetByProvider retrieves an account by provider name and provider-specific user ID.
	// This is used during login to find an existing account for a provider.
	// Returns sql.ErrNoRows or equivalent if no matching account is found.
	GetByProvider(ctx context.Context, provider, providerAccountID string) (Account, error)

	// Update modifies an existing account's data (e.g., updating OAuth tokens).
	// Returns an error if the account does not exist.
	Update(ctx context.Context, account Account) error

	// Delete removes an account from storage.
	// Returns an error if the account does not exist.
	Delete(ctx context.Context, id string) error
}

// VerificationStore defines the interface for verification token storage operations.
// Implementations manage temporary tokens used for email verification, password resets,
// OTP codes, and other time-limited verification flows.
type VerificationStore interface {
	// Create persists a new verification token to storage.
	Create(ctx context.Context, verification Verification) error

	// GetByToken retrieves a verification by its token value.
	// Returns sql.ErrNoRows or equivalent if no verification is found.
	GetByToken(ctx context.Context, token string) (Verification, error)

	// GetByIdentifier retrieves all verifications for a given identifier
	// (e.g., all pending verifications for an email address).
	// Returns an empty slice if no verifications exist for the identifier.
	GetByIdentifier(ctx context.Context, identifier string) ([]Verification, error)

	// InvalidateByIdentifier marks all verifications of a specific type for an
	// identifier as invalid/used. This is typically called after successful
	// verification to prevent token reuse.
	InvalidateByIdentifier(ctx context.Context, identifier, vType string) error

	// Delete removes a verification token from storage.
	Delete(ctx context.Context, id string) error

	// CleanupExpired removes all expired verification tokens.
	// This should be called periodically to prevent storage bloat.
	CleanupExpired(ctx context.Context) error
}

// SessionStore defines the interface for session storage operations.
// Implementations manage active user sessions and their authentication tokens.
type SessionStore interface {
	// Create persists a new session to storage.
	Create(ctx context.Context, session Session) error

	// Get retrieves a session by its unique identifier.
	// Returns sql.ErrNoRows or equivalent if no session is found.
	Get(ctx context.Context, id string) (Session, error)

	// GetByToken retrieves a session by its authentication token.
	// This is the primary method used during request authentication.
	// Returns sql.ErrNoRows or equivalent if no session is found.
	GetByToken(ctx context.Context, token string) (Session, error)

	// GetByRefreshToken retrieves a session by its refresh token.
	// Used in refresh token flows to generate new sessions.
	// Returns sql.ErrNoRows or equivalent if no session is found.
	GetByRefreshToken(ctx context.Context, refreshToken string) (Session, error)

	// GetByUserID retrieves a paginated list of active sessions for a specific user.
	// Returns an empty slice if the user has no active sessions.
	GetByUserID(ctx context.Context, userID string, offset, limit int) ([]Session, error)

	// CountByUserID returns the total number of active sessions for a user.
	CountByUserID(ctx context.Context, userID string) (int, error)

	// Update modifies an existing session's data (e.g., extending expiration).
	// Returns an error if the session does not exist.
	Update(ctx context.Context, session Session) error

	// Delete removes a specific session (used during logout).
	Delete(ctx context.Context, id string) error

	// DeleteByUserID removes all sessions for a user (used during password change
	// or when forcing all sessions to be logged out).
	DeleteByUserID(ctx context.Context, userID string) error

	// CleanupExpired removes all expired sessions.
	// This should be called periodically to prevent storage bloat.
	CleanupExpired(ctx context.Context) error
}

// Transactor is an optional capability that store implementations may
// expose to allow callers to compose multiple store operations in a single
// database transaction. Custom store implementations that do not back onto
// a single SQL database can omit this entirely; callers must treat
// Transactor support as optional and provide a non-transactional fallback.
type Transactor interface {
	// BeginTx opens a new transaction and returns a Tx that exposes
	// transaction-bound copies of the relevant stores. The returned Tx
	// MUST be either committed or rolled back by the caller.
	BeginTx(ctx context.Context) (Tx, error)
}

// Tx represents an in-flight database transaction across one or more
// stores. All store accessors on Tx return implementations bound to the
// transaction; writes are visible to other accessors on the same Tx but
// are only persisted once Commit is called. Rollback discards all writes.
//
// Tx is single-use: once Commit or Rollback is called the Tx must not be
// used again.
type Tx interface {
	// AccountStore returns an AccountStore bound to this transaction.
	AccountStore() AccountStore
	// SessionStore returns a SessionStore bound to this transaction.
	SessionStore() SessionStore
	// UserStore returns a UserStore bound to this transaction.
	UserStore() UserStore
	// VerificationStore returns a VerificationStore bound to this transaction.
	VerificationStore() VerificationStore
	// Commit persists all changes made via the Tx-bound stores.
	Commit() error
	// Rollback discards all changes made via the Tx-bound stores. It is
	// safe to call after Commit (it becomes a no-op) so callers can use
	// `defer tx.Rollback()` immediately after BeginTx.
	Rollback() error
}
