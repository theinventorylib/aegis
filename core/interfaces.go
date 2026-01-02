package core

import "time"

// This file defines model interfaces that allow for flexible data layer implementations.
// While the auth package provides concrete User, Account, Session, and Verification
// models, these interfaces enable custom model implementations if needed.
//
// Most applications will use the default auth.* models and won't need to implement
// these interfaces directly.

// UserModel defines the required methods for a user model implementation.
// Any type implementing this interface can be used as a user in the authentication
// system.
type UserModel interface {
	// GetID returns the unique identifier for this user
	GetID() string

	// SetID assigns a unique identifier to this user
	SetID(string)

	// GetEmail returns the user's email address
	GetEmail() string

	// SetEmail assigns an email address to this user
	SetEmail(string)

	// GetName returns the user's display name
	GetName() string

	// SetName assigns a display name to this user
	SetName(string)

	// SetCreatedAt assigns the creation timestamp
	SetCreatedAt(time.Time)

	// SetUpdatedAt assigns the last modification timestamp
	SetUpdatedAt(time.Time)
}

// AccountModel defines the required methods for an account model implementation.
// Accounts link users to authentication providers (credentials, OAuth, etc.).
type AccountModel interface {
	// GetID returns the unique identifier for this account
	GetID() string

	// SetID assigns a unique identifier to this account
	SetID(string)

	// GetUserID returns the ID of the user this account belongs to
	GetUserID() string

	// SetUserID assigns the owning user's ID
	SetUserID(string)

	// GetProvider returns the authentication provider name (e.g., "credentials", "google")
	GetProvider() string

	// SetProvider assigns the authentication provider name
	SetProvider(string)

	// GetPasswordHash returns the hashed password (for credential-based accounts)
	GetPasswordHash() string

	// SetPasswordHash assigns the hashed password
	SetPasswordHash(string)

	// SetCreatedAt assigns the creation timestamp
	SetCreatedAt(time.Time)

	// SetUpdatedAt assigns the last modification timestamp
	SetUpdatedAt(time.Time)

	// GetExpiresAt returns when OAuth tokens expire (OAuth accounts only)
	GetExpiresAt() time.Time

	// SetExpiresAt assigns the OAuth token expiration time
	SetExpiresAt(time.Time)

	// GetAccessToken returns the OAuth access token (OAuth accounts only)
	GetAccessToken() string

	// SetAccessToken assigns the OAuth access token
	SetAccessToken(string)

	// GetRefreshToken returns the OAuth refresh token (OAuth accounts only)
	GetRefreshToken() string

	// SetRefreshToken assigns the OAuth refresh token
	SetRefreshToken(string)

	// GetProviderAccountID returns the provider-specific user identifier
	GetProviderAccountID() string

	// SetProviderAccountID assigns the provider-specific user identifier
	SetProviderAccountID(string)
}

// SessionModel defines the required methods for a session model implementation.
// Sessions track authenticated user activity and enable stateful authentication.
type SessionModel interface {
	// GetID returns the unique identifier for this session
	GetID() string

	// SetID assigns a unique identifier to this session
	SetID(string)

	// GetUserID returns the ID of the user this session belongs to
	GetUserID() string

	// SetUserID assigns the owning user's ID
	SetUserID(string)

	// GetToken returns the session authentication token
	GetToken() string

	// SetToken assigns the session authentication token
	SetToken(string)

	// GetRefreshToken returns the refresh token for session renewal
	GetRefreshToken() string

	// SetRefreshToken assigns the refresh token
	SetRefreshToken(string)

	// SetCreatedAt assigns the creation timestamp
	SetCreatedAt(time.Time)

	// GetExpiresAt returns when this session expires
	GetExpiresAt() time.Time

	// SetExpiresAt assigns the session expiration time
	SetExpiresAt(time.Time)

	// SetIPAddress assigns the client IP address for security tracking
	SetIPAddress(string)

	// SetUserAgent assigns the client user agent for security tracking
	SetUserAgent(string)
}

// VerificationModel defines the required methods for a verification model implementation.
// Verifications are temporary tokens used for email confirmation, password resets, etc.
type VerificationModel interface {
	// GetID returns the unique identifier for this verification
	GetID() string

	// SetID assigns a unique identifier to this verification
	SetID(string)

	// GetToken returns the verification token/code
	GetToken() string

	// SetToken assigns the verification token/code
	SetToken(string)

	// GetIdentifier returns the target of verification (e.g., email address)
	GetIdentifier() string

	// SetIdentifier assigns the target identifier
	SetIdentifier(string)

	// SetCreatedAt assigns the creation timestamp
	SetCreatedAt(time.Time)

	// GetExpiresAt returns when this verification expires
	GetExpiresAt() time.Time

	// SetExpiresAt assigns the verification expiration time
	SetExpiresAt(time.Time)
}
