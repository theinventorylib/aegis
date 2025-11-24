package password

import (
	"context"
	"fmt"
	"net/http"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// Plugin implements password authentication via auth.accounts table
type Plugin struct {
	db             *DB
	userDB         db.Provider
	hasher         *core.PasswordHasherConfig
	sessionService *core.SessionService
}

// Config for password plugin
type Config struct {
	DB     db.Provider
	UserDB db.Provider
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

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "password"
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Password authentication using auth.accounts table"
}

// Init initializes the plugin.
func (p *Plugin) Init(_ context.Context, aegis plugins.Aegis) error {
	// Store session service for auth middleware
	p.sessionService = aegis.GetSessionService()
	return nil
}

// GetMigrations returns the plugin migrations.
func (p *Plugin) GetMigrations() []plugins.Migration {
	// No migrations needed - uses core auth.accounts table
	return []plugins.Migration{}
}

// MountRoutes registers HTTP routes for the plugin.
func (p *Plugin) MountRoutes(router server.Router, prefix string) {
	handlers := NewHandlers(p)

	// Create auth middleware - password changes require authentication
	requireAuth := core.RequireAuthMiddleware(p.sessionService)

	// Protected password management routes
	router.POST(prefix+"/password/change", requireAuth(http.HandlerFunc(handlers.ChangePasswordHandler)).ServeHTTP)
}

// RequiresTables returns required database tables.
func (p *Plugin) RequiresTables() []string {
	return []string{"auth.user", "auth.accounts"}
}

// ProvidesAuthMethods returns authentication methods provided.
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{"password"}
}

// Dependencies returns plugin dependencies.
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
	return p.db.CreateAccount(ctx, userID, passwordHash)
}

// VerifyPassword verifies a password for a user ID
func (p *Plugin) VerifyPassword(ctx context.Context, userID, password string) (bool, error) {
	// Get password account
	account, err := p.db.GetAccount(ctx, userID)
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

// GetAccount retrieves a password account by user ID (public for other plugins)
func (p *Plugin) GetAccount(ctx context.Context, userID string) (*Account, error) {
	return p.db.GetAccount(ctx, userID)
}

// ChangePassword changes a user's password after verifying the old one
func (p *Plugin) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	// Get password account
	account, err := p.db.GetAccount(ctx, userID)
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
	return p.db.HasAccount(ctx, userID)
}

// DeleteAccount deletes a user's password account
func (p *Plugin) DeleteAccount(ctx context.Context, userID string) error {
	return p.db.DeleteAccount(ctx, userID)
}
