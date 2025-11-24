// Package core provides core authentication functionality.
package core

import (
	"context"
	"fmt"

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
