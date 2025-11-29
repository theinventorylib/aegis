// Package sms provides SMS-based authentication and verification.
package sms

import (
	"context"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
)

// DB provides database operations for SMS plugin
// This uses the core auth.verification table
type DB struct {
	provider db.Provider
}

// NewDB creates a new SMS plugin database instance
func NewDB(provider db.Provider) *DB {
	return &DB{provider: provider}
}

// CreateVerification creates a new SMS verification record in auth.verification
func (d *DB) CreateVerification(ctx context.Context, verification *Verification) error {
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
func (d *DB) GetVerification(ctx context.Context, phoneNumber, verificationType string) (*Verification, error) {
	verification := &Verification{}

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

// GetUserByPhone retrieves a user by phone number
func (d *DB) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	err := d.provider.QueryRow(ctx, `
		SELECT id, created_at, updated_at
		FROM auth.user
		WHERE phone_number = ?
	`, phone).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return &user, nil
}

// UpdateUserPhone updates a user's phone number and verification status
func (d *DB) UpdateUserPhone(ctx context.Context, userID, phone string, verified bool) error {
	_, err := d.provider.Exec(ctx, `
		UPDATE auth.user
		SET phone_number = ?, phone_verified = ?
		WHERE id = ?
	`, phone, verified, userID)

	if err != nil {
		return fmt.Errorf("failed to update user phone: %w", err)
	}
	return nil
}

// VerifyOTP verifies an OTP code and deletes it if valid
func (d *DB) VerifyOTP(ctx context.Context, phoneNumber, code, purpose string) (bool, error) {
	var id string
	var token string
	var expiresAt time.Time

	err := d.provider.QueryRow(ctx, `
		SELECT id, token, expires_at
		FROM auth.verification
		WHERE identifier = ? AND type = ? AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`, phoneNumber, purpose).Scan(&id, &token, &expiresAt)

	if err != nil {
		return false, fmt.Errorf("OTP not found or expired")
	}

	// Verify code
	if token != code {
		return false, nil
	}

	// Delete the used verification token
	return true, d.DeleteVerification(ctx, id)
}
