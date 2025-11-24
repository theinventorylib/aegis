// Package email provides email-based authentication and verification.
package email

import (
	"context"
	"fmt"

	"github.com/theinventorylib/aegis/db"
)

// DB provides database operations for Email plugin
// This uses the core auth.verification table
type DB struct {
	provider db.Provider
}

// NewDB creates a new Email plugin database instance
func NewDB(provider db.Provider) *DB {
	return &DB{provider: provider}
}

// CreateVerification creates a new email verification record in auth.verification
func (d *DB) CreateVerification(ctx context.Context, verification *Verification) error {
	_, err := d.provider.Exec(ctx, `
		INSERT INTO auth.verification (id, identifier, token, type, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, verification.ID, verification.Email, verification.Code, verification.Purpose, verification.ExpiresAt, verification.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create email verification: %w", err)
	}
	return nil
}

// GetVerificationByEmail retrieves the most recent valid verification for an email/type
func (d *DB) GetVerificationByEmail(ctx context.Context, email, verificationType string) (*Verification, error) {
	verification := &Verification{}

	err := d.provider.QueryRow(ctx, `
		SELECT id, identifier, token, type, expires_at, created_at
		FROM auth.verification
		WHERE identifier = ? AND type = ? AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, email, verificationType).Scan(
		&verification.ID, &verification.Email, &verification.Code,
		&verification.Purpose, &verification.ExpiresAt, &verification.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("verification not found or expired: %w", err)
	}

	return verification, nil
}

// GetVerificationByToken retrieves verification by token from auth.verification
func (d *DB) GetVerificationByToken(ctx context.Context, token string) (*Verification, error) {
	verification := &Verification{}

	err := d.provider.QueryRow(ctx, `
		SELECT id, identifier, token, type, expires_at, created_at
		FROM auth.verification
		WHERE token = ? AND expires_at > NOW()
		LIMIT 1
	`, token).Scan(
		&verification.ID, &verification.Email, &verification.Code,
		&verification.Purpose, &verification.ExpiresAt, &verification.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("verification not found or expired: %w", err)
	}

	return verification, nil
}

// DeleteVerification deletes a verification record from auth.verification
func (d *DB) DeleteVerification(ctx context.Context, id string) error {
	_, err := d.provider.Exec(ctx, `
		DELETE FROM auth.verification WHERE id = ?
	`, id)

	if err != nil {
		return fmt.Errorf("failed to delete verification: %w", err)
	}
	return nil
}

// InvalidateVerifications deletes all verifications for an identifier/type
func (d *DB) InvalidateVerifications(ctx context.Context, identifier, verificationType string) error {
	_, err := d.provider.Exec(ctx, `
		DELETE FROM auth.verification
		WHERE identifier = ? AND type = ?
	`, identifier, verificationType)

	if err != nil {
		return fmt.Errorf("failed to invalidate verifications: %w", err)
	}
	return nil
}
