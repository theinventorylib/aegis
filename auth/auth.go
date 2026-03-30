// Package auth provides the core data layer for Aegis authentication system.
// It defines the fundamental storage interfaces and models for users, accounts,
// sessions, and verification tokens that form the foundation of authentication.
//
// The package follows a repository pattern with four primary storage interfaces:
//   - UserStore: Core user identity management
//   - AccountStore: Provider-specific account linking (OAuth, email/password, etc.)
//   - VerificationStore: Temporary tokens for email verification, password reset, etc.
//   - SessionStore: Active user session tracking with token management
//
// A default SQL-based implementation using sqlc is provided, but consumers can
// supply custom implementations for any store to integrate with different backends.
//
// Example usage:
//
//	auth := auth.New(auth.Config{
//		DB: db, // Uses default SQL stores for all operations
//	})
//
//	// Or with custom stores:
//	auth := auth.New(auth.Config{
//		DB:        db,
//		UserStore: myCustomUserStore, // Custom implementation
//		// Other stores fall back to defaults
//	})
package auth

import (
	"database/sql"

	defaultstore "github.com/theinventorylib/aegis/auth/default_store"
)

// Config holds the configuration for the auth system.
// Only the User model is generic.
//
// If any store is nil, the default SQL-based implementation will be used
// automatically using the provided DB connection. This allows mixing custom
// and default stores as needed.
type Config struct {
	// DB is the SQL database connection used for default store implementations.
	// Required if any store field is left nil.
	DB *sql.DB

	// Dialect selects which sqlc-generated code is used for the default stores.
	// Defaults to DialectPostgres when not set.
	Dialect Dialect

	// UserStore handles user identity storage operations.
	// If nil, uses default SQL implementation.
	UserStore UserStore

	// AccountStore manages provider-linked accounts (OAuth, credentials, etc.).
	// If nil, uses default SQL implementation.
	AccountStore AccountStore

	// VerificationStore manages temporary verification tokens.
	// If nil, uses default SQL implementation.
	VerificationStore VerificationStore

	// SessionStore handles active session persistence.
	// If nil, uses default SQL implementation.
	SessionStore SessionStore
}

// Auth represents the core authentication system and provides access to all
// configured storage backends. It acts as a central registry for user, account,
// session, and verification data operations.
//
// Auth is safe for concurrent use and should be initialized once at application
// startup and shared across the application.
type Auth struct {
	userStore         UserStore
	accountStore      AccountStore
	verificationStore VerificationStore
	sessionStore      SessionStore
}

// New creates a new Auth instance with the provided configuration.
// For any store that is nil in the Config, a default SQL-based implementation
// is automatically created using the provided database connection.
//
// Returns an error if DB is nil and any store is also nil, since default stores
// cannot be created without a database connection.
func New(cfg Config) (*Auth, error) {
	defaultStore, err := defaultstore.NewDefaultStore(cfg.DB, cfg.Dialect)
	if err != nil {
		return nil, err
	}

	userStore := cfg.UserStore
	if any(userStore) == nil {
		if us, ok := any(defaultStore.UserStore()).(UserStore); ok {
			userStore = us
		}
	}

	accountStore := cfg.AccountStore
	if accountStore == nil {
		accountStore = defaultStore.AccountStore()
	}

	verificationStore := cfg.VerificationStore
	if verificationStore == nil {
		verificationStore = defaultStore.VerificationStore()
	}

	sessionStore := cfg.SessionStore
	if sessionStore == nil {
		sessionStore = defaultStore.SessionStore()
	}

	return &Auth{
		userStore:         userStore,
		accountStore:      accountStore,
		verificationStore: verificationStore,
		sessionStore:      sessionStore,
	}, nil
}

// UserStore returns the configured user store implementation.
// Use this to perform CRUD operations on user identities.
func (a *Auth) UserStore() UserStore {
	return a.userStore
}

// AccountStore returns the configured account store implementation.
// Use this to manage provider-specific account associations and credentials.
func (a *Auth) AccountStore() AccountStore {
	return a.accountStore
}

// VerificationStore returns the configured verification store implementation.
// Use this to manage temporary verification tokens for email confirmation,
// password resets, OTP codes, and other time-limited verification flows.
func (a *Auth) VerificationStore() VerificationStore {
	return a.verificationStore
}

// SessionStore returns the configured session store implementation.
// Use this to manage active user sessions, including token-based and
// refresh token authentication flows.
func (a *Auth) SessionStore() SessionStore {
	return a.sessionStore
}
