package types

import (
	"context"
	"time"
)

// Store defines the persistence contract for the TOTP plugin.
//
// Implementations must be safe for concurrent use.
type Store interface {
	// GetCredential returns the user's TOTP credential. Secret is empty when
	// enrollment has not started. Returns a not-found error for unknown users.
	GetCredential(ctx context.Context, userID string) (Credential, error)

	// SetCredential writes the credential. A nil secret clears the stored
	// secret (disable); enabled flips the trusted flag.
	SetCredential(ctx context.Context, userID string, secret *string, enabled bool) error

	// MarkSessionVerified records that sessionID completed second-factor
	// verification for userID at the given time.
	MarkSessionVerified(ctx context.Context, sessionID, userID string, at time.Time) error

	// IsSessionVerified reports whether sessionID was verified after since.
	IsSessionVerified(ctx context.Context, sessionID, userID string, since time.Time) (bool, error)

	// DeleteUserSessionVerifications clears every session verification for a
	// user (used when TOTP is disabled).
	DeleteUserSessionVerifications(ctx context.Context, userID string) error

	// DeleteSessionVerification clears one session's verification.
	DeleteSessionVerification(ctx context.Context, sessionID string) error
}
