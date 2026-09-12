package core

import (
	"context"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
)

// This file defines model interfaces that allow for flexible data layer implementations.
// While the auth package provides concrete User, Account, Session, and Verification
// models, these interfaces enable custom model implementations if needed.
//
// Most applications will use the default auth.* models and won't need to implement
// these interfaces directly.

// UserIDOf extracts a user ID from the two concrete shapes a context user
// can take (a UserModel implementation or *auth.User). Single owner of that
// type switch. Returns "" when user is neither.
func UserIDOf(user any) string {
	if ua, ok := user.(UserModel); ok {
		return ua.GetID()
	}
	if ua, ok := user.(*auth.User); ok {
		return ua.ID
	}
	return ""
}

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

// BearerTokenValidator is an optional interface for validating non-session bearer
// tokens such as JWT access tokens. When registered with SessionService via
// SetBearerTokenValidator, it is called by AuthMiddleware before falling back to
// the opaque session-token database lookup.
//
// Implementations (e.g., the JWT plugin) should:
//   - Cryptographically verify the token
//   - Return the authenticated user and, if applicable, a synthetic session
//   - Return a non-nil error if the token is invalid or expired
//
// A nil *auth.Session return value is valid for fully-stateless token schemes.
type BearerTokenValidator interface {
	ValidateBearerToken(ctx context.Context, token string) (*auth.User, *auth.Session, error)
}
