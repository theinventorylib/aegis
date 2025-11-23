package password

import (
	"context"
	"fmt"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// Plugin implements password authentication via auth.accounts table
type Plugin struct {
	db     *DB
	userDB db.DBProvider
	hasher *core.PasswordHasherConfig
}

// Config for password plugin
type Config struct {
	DB     db.DBProvider
	UserDB db.DBProvider
	Hasher *core.PasswordHasherConfig // Optional, uses defaults if nil
}

// New creates a new password plugin
func New(cfg *Config) *Plugin {
	if cfg.Hasher == nil {
		cfg.Hasher = core.DefaultPasswordHasherConfig()
	}

	return &Plugin{
		db:     NewDB(cfg.DB),
		userDB: cfg.UserDB,
		hasher: cfg.Hasher,
	}
}

// Plugin interface implementation

func (p *Plugin) Name() string {
	return "password"
}

func (p *Plugin) Version() string {
	return "1.0.0"
}

func (p *Plugin) Description() string {
	return "Password authentication using auth.accounts table"
}

func (p *Plugin) Init(ctx context.Context, a plugins.Aegis) error {
	// No initialization needed - uses core auth.accounts table
	return nil
}

func (p *Plugin) GetMigrations() []plugins.Migration {
	// No migrations needed - uses core auth.accounts table
	return []plugins.Migration{}
}

func (p *Plugin) MountRoutes(router server.Router, prefix string) {
	handlers := NewHandlers(p)

	// Password management routes
	router.POST(prefix+"/password/change", handlers.ChangePasswordHandler)
}

func (p *Plugin) RequiresTables() []string {
	return []string{"auth.user", "auth.accounts"}
}

func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{"password"}
}

func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}

// Password authentication methods

// CreateAccount creates a password account for a user
func (p *Plugin) CreateAccount(ctx context.Context, userID, password string) error {
	// Hash password
	passwordHash, err := core.HashPassword(
		password,
		p.hasher.Argon2Time,
		p.hasher.Argon2Memory,
		p.hasher.Argon2Threads,
		p.hasher.Argon2KeyLength,
	)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create password account
	return p.db.CreatePasswordAccount(ctx, userID, passwordHash)
}

// VerifyPassword verifies a password for a user ID
func (p *Plugin) VerifyPassword(ctx context.Context, userID, password string) (bool, error) {
	// Get password account
	account, err := p.db.GetPasswordAccount(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("invalid credentials")
	}

	// Verify password
	valid, err := core.VerifyPassword(password, account.PasswordHash)
	if err != nil || !valid {
		return false, fmt.Errorf("invalid credentials")
	}

	return true, nil
}

// GetPasswordAccount retrieves a password account by user ID (public for other plugins)
func (p *Plugin) GetPasswordAccount(ctx context.Context, userID string) (*PasswordAccount, error) {
	return p.db.GetPasswordAccount(ctx, userID)
}

// ChangePassword changes a user's password after verifying the old one
func (p *Plugin) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	// Get password account
	account, err := p.db.GetPasswordAccount(ctx, userID)
	if err != nil {
		return fmt.Errorf("password account not found")
	}

	// Verify old password
	valid, err := core.VerifyPassword(oldPassword, account.PasswordHash)
	if err != nil || !valid {
		return fmt.Errorf("invalid current password")
	}

	// Hash new password
	newHash, err := core.HashPassword(
		newPassword,
		p.hasher.Argon2Time,
		p.hasher.Argon2Memory,
		p.hasher.Argon2Threads,
		p.hasher.Argon2KeyLength,
	)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	return p.db.UpdatePasswordHash(ctx, userID, newHash)
}

// ResetPassword resets a user's password without verification (for password reset flows)
func (p *Plugin) ResetPassword(ctx context.Context, userID, newPassword string) error {
	// Hash new password
	newHash, err := core.HashPassword(
		newPassword,
		p.hasher.Argon2Time,
		p.hasher.Argon2Memory,
		p.hasher.Argon2Threads,
		p.hasher.Argon2KeyLength,
	)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	return p.db.UpdatePasswordHash(ctx, userID, newHash)
}

// HasPassword checks if a user has a password account
func (p *Plugin) HasPassword(ctx context.Context, userID string) (bool, error) {
	return p.db.HasPasswordAccount(ctx, userID)
}

// DeleteAccount deletes a user's password account
func (p *Plugin) DeleteAccount(ctx context.Context, userID string) error {
	return p.db.DeletePasswordAccount(ctx, userID)
}
