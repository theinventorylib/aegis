// Package core provides core authentication functionality.
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
)

// AuthService handles authentication operations.
type AuthService struct {
	db         db.Provider
	session    *SessionService
	hashConfig *PasswordHasherConfig
}

// NewAuthService creates a new auth service
func NewAuthService(database db.Provider, sessionService *SessionService, hashConfig *PasswordHasherConfig) *AuthService {
	if hashConfig == nil {
		hashConfig = DefaultPasswordHasherConfig()
	}
	return &AuthService{
		db:         database,
		session:    sessionService,
		hashConfig: hashConfig,
	}
}

// CreateUser creates a new user in the database.
// It returns the created user or an error if the operation fails.
func (a *AuthService) CreateUser(ctx context.Context) (*models.User, error) {
	return a.db.CreateUser(ctx)
}

// CreateUserWithPassword creates a new user and a password account for that user.
// It attempts to clean up the user if creating the password account fails.
func (a *AuthService) CreateUserWithPassword(ctx context.Context, password string) (*models.User, error) {
	user, err := a.db.CreateUser(ctx)
	if err != nil {
		return nil, err
	}

	// Try to create password account; if it fails delete the user to avoid orphaned user row.
	if err := a.CreatePasswordAccount(ctx, user.ID, password); err != nil {
		_ = a.db.DeleteUser(ctx, user.ID) // best-effort cleanup
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by their unique ID.
// It returns the user if found, or an error if not found or if a database error occurs.
func (a *AuthService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return a.db.GetUserByID(ctx, id)
}

// UpdateUser updates an existing user's information.
// It returns an error if the update fails.
func (a *AuthService) UpdateUser(ctx context.Context, user *models.User) error {
	return a.db.UpdateUser(ctx, user)
}

// DeleteUser deletes a user and all their associated sessions.
// This operation is transactional: sessions are deleted first, then the user.
func (a *AuthService) DeleteUser(ctx context.Context, id string) error {
	// First delete all user sessions
	if err := a.db.DeleteUserSessions(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	// Then delete the user
	return a.db.DeleteUser(ctx, id)
}

// ListUsers retrieves a paginated list of users.
// offset specifies the number of users to skip, and limit specifies the maximum number of users to return.
func (a *AuthService) ListUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	return a.db.ListUsers(ctx, offset, limit)
}

// CountUsers returns the total number of registered users.
func (a *AuthService) CountUsers(ctx context.Context) (int, error) {
	return a.db.CountUsers(ctx)
}

// ========== Password account operations (migrated from plugins/password) ==========

// CreatePasswordAccount creates a password account for a user with a hashed password.
func (a *AuthService) CreatePasswordAccount(ctx context.Context, userID, password string) error {
	// Hash password using configured hasher
	passwordHash, err := HashPassword(
		password,
		a.hashConfig.Argon2Time,
		a.hashConfig.Argon2Memory,
		a.hashConfig.Argon2Threads,
		a.hashConfig.Argon2KeyLength,
	)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	id := GenerateID()
	now := time.Now()

	_, err = a.db.Exec(ctx, `
		INSERT INTO auth.accounts (id, user_id, provider, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, userID, "password", passwordHash, now, now)

	if err != nil {
		return fmt.Errorf("failed to create password account: %w", err)
	}

	return nil
}

// GetPasswordAccount retrieves the password account for a user.
func (a *AuthService) GetPasswordAccount(ctx context.Context, userID string) (*models.Account, error) {
	var acc models.Account
	var passwordHash string

	err := a.db.QueryRow(ctx, `
		SELECT id, user_id, provider, password_hash, created_at, updated_at
		FROM auth.accounts
		WHERE user_id = ? AND provider = ?
	`, userID, "password").Scan(
		&acc.ID, &acc.UserID, &acc.Provider,
		&passwordHash, &acc.CreatedAt, &acc.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("password account not found")
	}

	acc.Password = &passwordHash
	return &acc, nil
}

// UpdatePassword updates the stored password hash for a user.
func (a *AuthService) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	result, err := a.db.Exec(ctx, `
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

// VerifyPassword verifies a cleartext password against the stored hash for a user.
func (a *AuthService) VerifyPassword(ctx context.Context, userID, password string) (bool, error) {
	acc, err := a.GetPasswordAccount(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("invalid credentials")
	}

	if acc.Password == nil {
		return false, fmt.Errorf("invalid credentials")
	}

	valid, err := VerifyPassword(password, *acc.Password)
	if err != nil || !valid {
		return false, fmt.Errorf("invalid credentials")
	}
	return true, nil
}

// ChangePassword verifies the current password and updates it to a new password.
func (a *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	acc, err := a.GetPasswordAccount(ctx, userID)
	if err != nil {
		return fmt.Errorf("password account not found")
	}

	if acc.Password == nil {
		return fmt.Errorf("password account not found")
	}

	valid, err := VerifyPassword(oldPassword, *acc.Password)
	if err != nil || !valid {
		return fmt.Errorf("invalid current password")
	}

	newHash, err := HashPassword(
		newPassword,
		a.hashConfig.Argon2Time,
		a.hashConfig.Argon2Memory,
		a.hashConfig.Argon2Threads,
		a.hashConfig.Argon2KeyLength,
	)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	return a.UpdatePassword(ctx, userID, newHash)
}

// ResetPassword updates the password without verifying the old one (used by reset flows).
func (a *AuthService) ResetPassword(ctx context.Context, userID, newPassword string) error {
	newHash, err := HashPassword(
		newPassword,
		a.hashConfig.Argon2Time,
		a.hashConfig.Argon2Memory,
		a.hashConfig.Argon2Threads,
		a.hashConfig.Argon2KeyLength,
	)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}
	return a.UpdatePassword(ctx, userID, newHash)
}

// HasPassword checks whether a user has a password account.
func (a *AuthService) HasPassword(ctx context.Context, userID string) (bool, error) {
	var count int
	err := a.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM auth.accounts
		WHERE user_id = ? AND provider = ?
	`, userID, "password").Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check password account: %w", err)
	}
	return count > 0, nil
}

// DeletePasswordAccount deletes a user's password account.
func (a *AuthService) DeletePasswordAccount(ctx context.Context, userID string) error {
	_, err := a.db.Exec(ctx, `
		DELETE FROM auth.accounts
		WHERE user_id = ? AND provider = ?
	`, userID, "password")
	if err != nil {
		return fmt.Errorf("failed to delete password account: %w", err)
	}
	return nil
}

// Logout invalidates a specific session by its token.
func (a *AuthService) Logout(ctx context.Context, token string) error {
	return a.session.DeleteSession(ctx, token)
}

// LogoutAllSessions invalidates all active sessions for a specific user.
// This is useful for security events like password changes or account suspension.
func (a *AuthService) LogoutAllSessions(ctx context.Context, userID string) error {
	return a.db.DeleteUserSessions(ctx, userID)
}

// GetUserSessions retrieves all active sessions for a specific user.
func (a *AuthService) GetUserSessions(ctx context.Context, userID string) ([]*models.Session, error) {
	return a.db.GetUserSessions(ctx, userID)
}
