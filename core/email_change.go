package core

import (
	"context"
	"strings"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
)

// RequestEmailChange issues a confirmation token for moving userID's account
// to newEmail and returns the raw token for delivery to that address.
//
// The token is stored hashed and scoped to the new address (identifier =
// newEmail, type = email_change), so redeeming it proves control of that
// mailbox. The account keeps its current address until ConfirmEmailChange is
// called. Any previously issued change token for the same address is
// invalidated first.
//
// Returns ErrEmailAlreadyExists when another account already owns newEmail.
func (s *UserService) RequestEmailChange(ctx context.Context, userID, newEmail string) (string, error) {
	newEmail = SanitizeEmail(newEmail)
	if err := ValidateEmail(newEmail); err != nil {
		return "", err
	}
	if s.verification == nil {
		return "", NewAuthError(AuthErrorCodeInternal, "email change is not configured")
	}

	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if user.Email == newEmail {
		return "", ValidationError{Field: "email", Message: "is already the account email"}
	}

	// Friendly duplicate check; the unique constraint stays the backstop.
	if existing, err := s.userStore.GetByEmail(ctx, newEmail); err == nil {
		if existing.GetID() != userID {
			return "", ErrEmailAlreadyExists
		}
	} else if !isNotFound(err) {
		return "", err
	}

	if err := s.verification.InvalidateVerification(ctx, newEmail, VerificationTypeEmailChange); err != nil {
		return "", err
	}

	v, err := s.verification.CreateVerification(ctx, newEmail, VerificationTypeEmailChange, s.emailChangeExpiry(), nil)
	if err != nil {
		return "", err
	}

	return v.Token, nil
}

// ConfirmEmailChange redeems an email-change token and moves the account's
// email. userID is the authenticated caller whose account changes; the token
// itself is scoped to the new address, so possession of it proves control of
// that mailbox.
//
// UpdateUserEmail clears the plugin-side verification flag for the new address
// before switching; because the caller just proved control of the address, the
// verified marker (when an email plugin wired one) is set again afterwards.
//
// The token is consumed only after the change is applied, so a failed
// uniqueness/validation check leaves it redeemable.
func (s *UserService) ConfirmEmailChange(ctx context.Context, userID, token string) (auth.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return auth.User{}, ErrInvalidToken
	}
	if s.verification == nil {
		return auth.User{}, NewAuthError(AuthErrorCodeInternal, "email change is not configured")
	}

	v, err := s.verification.ValidateVerification(ctx, token)
	if err != nil {
		return auth.User{}, err
	}
	if v.Type != VerificationTypeEmailChange {
		return auth.User{}, ErrInvalidToken
	}

	if err := s.UpdateUserEmail(ctx, userID, v.Identifier); err != nil {
		return auth.User{}, err
	}

	// Consume the redeemed token and any siblings for the same address. A
	// failure here would leave the token replayable until expiry, so surface it.
	if err := s.verification.DeleteVerification(ctx, v.ID); err != nil {
		return auth.User{}, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to consume email-change token", err)
	}
	if err := s.verification.InvalidateVerification(ctx, v.Identifier, VerificationTypeEmailChange); err != nil {
		s.logger.Error("email change: failed to invalidate sibling tokens", "user_id", userID, "error", err)
	}

	if s.emailVerifiedMarker != nil {
		if err := s.emailVerifiedMarker(ctx, userID, v.Identifier); err != nil {
			s.logger.Error("user: failed to mark changed email verified", "user_id", userID, "error", err)
		}
	}

	// The email_changed audit event is emitted by UpdateUserEmail, the single
	// owner of the address mutation.
	user, err := s.GetUserByID(ctx, userID)
	if err != nil {
		return auth.User{}, err
	}
	return user, nil
}

// emailChangeExpiry resolves the configured change-token lifetime, falling
// back to DefaultEmailChangeExpiry when unset.
func (s *UserService) emailChangeExpiry() time.Duration {
	if s.authConfig != nil && s.authConfig.EmailChangeExpiry > 0 {
		return s.authConfig.EmailChangeExpiry
	}
	return DefaultEmailChangeExpiry
}
