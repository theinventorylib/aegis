package core

import (
	"context"
	"fmt"

	"github.com/theinventorylib/aegis/db"
)

// AuthService handles authentication operations
type AuthService struct {
	db         db.DBProvider
	session    *SessionService
	hashConfig *PasswordHasherConfig
}

// NewAuthService creates a new auth service
func NewAuthService(database db.DBProvider, sessionService *SessionService, hashConfig *PasswordHasherConfig) *AuthService {
	if hashConfig == nil {
		hashConfig = DefaultPasswordHasherConfig()
	}
	return &AuthService{
		db:         database,
		session:    sessionService,
		hashConfig: hashConfig,
	}
}

// Signup creates a new user account
// Deprecated: Use plugin-specific signup (e.g. email.Signup)
func (a *AuthService) Signup(ctx context.Context) (*User, *Session, error) {
	return nil, nil, fmt.Errorf("use plugin-specific signup")
}

// Login authenticates a user
// Deprecated: Use plugin-specific login (e.g. email.Login)
func (a *AuthService) Login(ctx context.Context) (*User, *Session, error) {
	return nil, nil, fmt.Errorf("use plugin-specific login")
}

// Logout invalidates a user's session
func (a *AuthService) Logout(ctx context.Context, token string) error {
	return a.session.DeleteSession(ctx, token)
}

// RequestPasswordReset generates a secure token for password reset
// Deprecated: Use plugin-specific password reset
func (a *AuthService) RequestPasswordReset(ctx context.Context, email string) (string, error) {
	return "", fmt.Errorf("use plugin-specific password reset")
}
