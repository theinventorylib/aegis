package core

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
)

// VerificationService manages temporary verification tokens for various flows:
//   - Email verification after signup
//   - Password reset tokens
//   - OTP (one-time password) codes
//   - Magic link tokens
//   - Custom verification workflows
//
// All verifications have:
//   - A unique token/code
//   - An identifier (email, phone, etc.)
//   - A type ("email", "reset", "otp", etc.)
//   - An expiration time
//
// Verifications are single-use and should be deleted or invalidated after use.
type VerificationService struct {
	// store persists verification tokens
	store auth.VerificationStore

	// auditLogger records verification events
	auditLogger AuditLogger
}

// newVerificationService creates a new verification service.
func newVerificationService(store auth.VerificationStore, auditLogger AuditLogger) *VerificationService {
	return &VerificationService{
		store:       store,
		auditLogger: auditLogger,
	}
}

// CreateVerification creates a new verification token for a specific purpose.
//
// Generates a cryptographically secure random token (or uses a custom token
// if provided). The verification is stored with an expiration time for automatic
// cleanup.
//
// Tokens are stored at rest as their SHA-256 hex hash (same scheme as session
// tokens) — a database read alone cannot impersonate a user. The returned
// verification carries the **raw** token so callers can deliver it to the user.
//
// Common use cases:
//   - Email verification: CreateVerification(ctx, email, "email", 24*time.Hour, nil)
//   - Password reset: CreateVerification(ctx, email, "reset", 1*time.Hour, nil)
//   - Custom OTP: CreateVerification(ctx, phone, "otp", 10*time.Minute, &customCode)
//
// Parameters:
//   - ctx: Request context
//   - identifier: Target being verified (email, phone, user ID, etc.)
//   - vType: Verification type ("email", "reset", "otp", etc.)
//   - expiry: How long the token is valid
//   - customToken: Optional custom token (if nil, random hex token is generated)
//
// Returns the created verification with populated token.
//
// Note: tokens issued before the hashing scheme are not matched anymore;
// verification tokens are short-lived, so a deploy simply invalidates any
// pending ones.
func (s *VerificationService) CreateVerification(ctx context.Context, identifier, vType string, expiry time.Duration, customToken *string) (auth.Verification, error) {
	// Sanitize identifier (could be email, phone, etc.)
	// We use SanitizeString as a catch-all for generic identifiers
	identifier = SanitizeString(identifier, nil)

	var token string
	if customToken != nil {
		token = *customToken
	} else {
		// Generate a secure random hex token (32 bytes = 64 hex characters)
		var err error
		token, err = generateHexToken(32)
		if err != nil {
			return auth.Verification{}, err
		}
	}

	verification := auth.Verification{
		ID:         GenerateID(),
		Identifier: identifier,
		Token:      token,
		Type:       vType,
		ExpiresAt:  time.Now().Add(expiry),
		CreatedAt:  time.Now(),
	}

	// Persist only the hash of the token; keep the raw token on the returned
	// struct for the caller to deliver (email link, SMS code, etc.).
	persisted := verification
	persisted.Token = hashTokenHex(token)
	if err := s.store.Create(ctx, persisted); err != nil {
		return auth.Verification{}, err
	}

	return verification, nil
}

// ValidateVerification validates a token and returns the verification if valid.
//
// Checks:
//  1. Token exists in storage
//  2. Token has not expired
//
// After successful validation, the caller should typically:
//   - Perform the verified action (activate account, reset password, etc.)
//   - Invalidate the token to prevent reuse
//
// Parameters:
//   - ctx: Request context
//   - token: The token string to validate
//
// Returns:
//   - The verification record if valid. Its Token field holds the stored
//     **hash**, not the raw token the caller passed in.
//   - AuthErrorCodeTokenExpired if expired
//   - Error if token not found
func (s *VerificationService) ValidateVerification(ctx context.Context, token string) (auth.Verification, error) {
	v, err := s.store.GetByToken(ctx, hashTokenHex(token))
	if err != nil {
		return auth.Verification{}, err
	}

	if time.Now().After(v.ExpiresAt) {
		return auth.Verification{}, NewAuthError(AuthErrorCodeTokenExpired, "verification token expired")
	}

	return v, nil
}

// ValidateVerificationFor validates a token scoped to a specific identifier and
// verification type, then consumes (deletes) it so it cannot be reused.
//
// This is the correct entry point for one-time codes (OTPs): binding to the
// identifier prevents a code sent to one email/phone from verifying another,
// binding to the type prevents cross-purpose reuse, and deletion enforces
// single use.
//
// Returns AuthErrorCodeTokenExpired for expired tokens, an invalid-credentials
// error if the token does not match the identifier/type, or the underlying
// store error if not found.
func (s *VerificationService) ValidateVerificationFor(ctx context.Context, identifier, vType, token string) (auth.Verification, error) {
	// Sanitize the identifier the same way CreateVerification does, so
	// programmatic callers passing raw emails/phones still match.
	identifier = SanitizeString(identifier, nil)

	v, err := s.ValidateVerification(ctx, token)
	if err != nil {
		return auth.Verification{}, err
	}

	if v.Identifier != identifier || v.Type != vType {
		return auth.Verification{}, NewAuthError(AuthErrorCodeInvalidCredentials, "verification token does not match identifier or purpose")
	}

	if err := s.DeleteVerification(ctx, v.ID); err != nil {
		return auth.Verification{}, err
	}

	return v, nil
}

// ponytail: validate-then-delete is not atomic — two concurrent requests
// with the same code can both pass before either deletes. Window is
// milliseconds and identifier+type-scoped; if single-use must be strictly
// atomic under load, switch the store to a "delete by token RETURNING row"
// query instead.

// InvalidateVerification marks all tokens of a specific type for an identifier as invalid.
//
// This prevents token reuse after successful verification. For example, after a user
// verifies their email, all pending email verification tokens for that address should
// be invalidated.
//
// Parameters:
//   - ctx: Request context
//   - identifier: The target identifier (email, phone, etc.)
//   - vType: The verification type to invalidate ("email", "reset", etc.)
func (s *VerificationService) InvalidateVerification(ctx context.Context, identifier, vType string) error {
	return s.store.InvalidateByIdentifier(ctx, identifier, vType)
}

// DeleteVerification permanently deletes a verification token.
//
// Use this for cleanup after successful verification or when canceling a
// verification flow.
//
// Parameters:
//   - ctx: Request context
//   - id: Verification ID to delete
func (s *VerificationService) DeleteVerification(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// generateHexToken creates a cryptographically secure random hex token.
// n is the number of random bytes (output will be 2*n hex characters).
func generateHexToken(n int) (string, error) {
	b, err := randomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
