// Package password provides password-based authentication.
package password

import (
	"context"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
)

// DB provides database operations for password accounts.
type DB struct {
	provider db.Provider
}

// NewDB creates a new password database instance.
func NewDB(provider db.Provider) *DB {
	return &DB{provider: provider}
}

// CreateAccount creates a new password account in auth.accounts.
func (d *DB) CreateAccount(ctx context.Context, userID, passwordHash string) error {
	now := time.Now()
	id := core.GenerateID()

	_, err := d.provider.Exec(ctx, `
		INSERT INTO auth.accounts (id, user_id, provider, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, userID, "password", passwordHash, now, now)

	if err != nil {
		return fmt.Errorf("failed to create password account: %w", err)
	}

	return nil
}

// GetAccount retrieves a password account by user ID
func (d *DB) GetAccount(ctx context.Context, userID string) (*Account, error) {
	account := &Account{}

	err := d.provider.QueryRow(ctx, `
		SELECT id, user_id, provider, password_hash, created_at, updated_at
		FROM auth.accounts
		WHERE user_id = ? AND provider = ?
	`, userID, "password").Scan(
		&account.ID, &account.UserID, &account.Provider,
		&account.PasswordHash, &account.CreatedAt, &account.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("password account not found")
	}

	return account, nil
}

// UpdatePasswordHash updates the password hash for a user
func (d *DB) UpdatePasswordHash(ctx context.Context, userID, passwordHash string) error {
	result, err := d.provider.Exec(ctx, `
		UPDATE auth.accounts
		SET password_hash = ?, updated_at = NOW()
		WHERE user_id = ? AND provider = ?
	`, passwordHash, userID, "password")

	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("password account not found")
	}

	return nil
}

// DeleteAccount deletes a password account for a user
func (d *DB) DeleteAccount(ctx context.Context, userID string) error {
	_, err := d.provider.Exec(ctx, `
		DELETE FROM auth.accounts
		WHERE user_id = ? AND provider = ?
	`, userID, "password")

	if err != nil {
		return fmt.Errorf("failed to delete password account: %w", err)
	}

	return nil
}

// HasAccount checks if a user has a password account
func (d *DB) HasAccount(ctx context.Context, userID string) (bool, error) {
	var count int

	err := d.provider.QueryRow(ctx, `
		SELECT COUNT(*) FROM auth.accounts
		WHERE user_id = ? AND provider = ?
	`, userID, "password").Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check password account: %w", err)
	}

	return count > 0, nil
}
