package core

import (
	"context"
	"strings"
	"time"

	"github.com/theinventorylib/aegis/v2/auth"
)

// RequestPasswordReset issues a single-use password-reset token for the
// account that owns email and returns the raw token for delivery.
//
// The token is stored hashed; only the value returned here can be redeemed
// through ConfirmPasswordReset. Any previously issued reset token for the
// same address is invalidated first, so at most one reset is in flight per
// account.
//
// Returns ErrUserNotFound when no user owns that address. HTTP layers that
// must not reveal whether an account exists should treat that error as
// success and return the same response as a delivered reset.
func (s *AccountService) RequestPasswordReset(ctx context.Context, email string) (auth.User, string, error) {
	email = SanitizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return auth.User{}, "", err
	}
	if s.userStore == nil || s.verification == nil {
		return auth.User{}, "", NewAuthError(AuthErrorCodeInternal, "password reset is not configured")
	}

	user, err := s.userStore.GetByEmail(ctx, email)
	if err != nil {
		return auth.User{}, "", err
	}

	if err := s.verification.InvalidateVerification(ctx, email, VerificationTypePasswordReset); err != nil {
		return auth.User{}, "", err
	}

	v, err := s.verification.CreateVerification(ctx, email, VerificationTypePasswordReset, s.passwordResetExpiry(), nil)
	if err != nil {
		return auth.User{}, "", err
	}

	return user, v.Token, nil
}

// ConfirmPasswordReset redeems a password-reset token and sets the account's
// new password. UpdatePassword invalidates every existing session, so any
// attacker holding a stolen session loses access once the reset completes.
//
// The token is consumed only after the password is updated: a rejected new
// password (policy, unknown token) leaves the token redeemable, so a user who
// mistypes a password does not have to restart the flow. The reverse risk —
// a DB failure after the update but before the token delete — is surfaced as
// an internal error rather than silently swallowed.
func (s *AccountService) ConfirmPasswordReset(ctx context.Context, token, newPassword string) (auth.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return auth.User{}, ErrInvalidToken
	}
	if s.userStore == nil || s.verification == nil {
		return auth.User{}, NewAuthError(AuthErrorCodeInternal, "password reset is not configured")
	}
	// Validate before consuming so a policy rejection does not burn the token.
	if err := validatePassword(newPassword, s.authConfig.PasswordPolicy); err != nil {
		return auth.User{}, err
	}

	v, err := s.verification.ValidateVerification(ctx, token)
	if err != nil {
		return auth.User{}, err
	}
	if v.Type != VerificationTypePasswordReset {
		return auth.User{}, ErrInvalidToken
	}

	user, err := s.userStore.GetByEmail(ctx, v.Identifier)
	if err != nil {
		// The address no longer maps to an account: the token is stale.
		return auth.User{}, ErrInvalidToken
	}

	if err := s.UpdatePassword(ctx, user.GetID(), newPassword); err != nil {
		return auth.User{}, err
	}

	// Consume the redeemed token and any siblings for the same address. A
	// failure here would leave the token replayable until expiry, so surface it.
	if err := s.verification.DeleteVerification(ctx, v.ID); err != nil {
		return auth.User{}, NewAuthErrorWithCause(AuthErrorCodeInternal, "failed to consume password-reset token", err)
	}
	_ = s.verification.InvalidateVerification(ctx, v.Identifier, VerificationTypePasswordReset)

	logAuthEvent(ctx, s.logger, s.auditLogger, AuditEventPasswordReset, user.GetID(), true, nil)
	return user, nil
}

// passwordResetExpiry resolves the configured reset-token lifetime, falling
// back to DefaultPasswordResetExpiry when unset.
func (s *AccountService) passwordResetExpiry() time.Duration {
	if s.authConfig != nil && s.authConfig.PasswordResetExpiry > 0 {
		return s.authConfig.PasswordResetExpiry
	}
	return DefaultPasswordResetExpiry
}
