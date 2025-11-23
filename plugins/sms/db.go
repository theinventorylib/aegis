package sms

import (
	"context"
	"fmt"

	"github.com/theinventorylib/aegis/db"
)

// DB provides database operations for SMS plugin
// This uses the core auth.verification table
type DB struct {
	provider db.DBProvider
}

// NewDB creates a new SMS plugin database instance
func NewDB(provider db.DBProvider) *DB {
	return &DB{provider: provider}
}

// CreateVerification creates a new SMS verification record in auth.verification
func (d *DB) CreateVerification(ctx context.Context, verification *SMSVerification) error {
	_, err := d.provider.Exec(ctx, `
		INSERT INTO auth.verification (id, identifier, token, type, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, verification.ID, verification.PhoneNumber, verification.Code, verification.Purpose, verification.ExpiresAt, verification.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create SMS verification: %w", err)
	}
	return nil
}

// GetVerification retrieves the most recent valid verification for a phone/type
func (d *DB) GetVerification(ctx context.Context, phoneNumber, verificationType string) (*SMSVerification, error) {
	verification := &SMSVerification{}

	err := d.provider.QueryRow(ctx, `
		SELECT id, identifier, token, type, expires_at, created_at
		FROM auth.verification
		WHERE identifier = ? AND type = ? AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, phoneNumber, verificationType).Scan(
		&verification.ID, &verification.PhoneNumber, &verification.Code,
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
