package oauth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/markbates/goth/gothic"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/models"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// Plugin represents the OAuth plugin for Aegis
// Supports Goth (default/recommended) and custom OAuth provider implementations
type Plugin struct {
	db             *DB
	sessionService *core.SessionService
	userDB         interface{} // For creating Aegis users
}

// Config holds OAuth plugin configuration
type Config struct {
	DBPool         *pgxpool.Pool
	SessionService *core.SessionService
	UserDB         interface{} // db.Provider for user operations
}

// New creates a new OAuth plugin
// Goth providers should be configured before creating the plugin using goth.UseProviders()
func New(cfg *Config) *Plugin {
	var db *DB
	if cfg.DBPool != nil {
		db = NewDB(cfg.DBPool)
	}

	return &Plugin{
		db:             db,
		sessionService: cfg.SessionService,
		userDB:         cfg.UserDB,
	}
}

// Name returns the plugin identifier
func (p *Plugin) Name() string {
	return "oauth"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description
func (p *Plugin) Description() string {
	return "OAuth authentication plugin supporting multiple providers (Google, GitHub, Apple, etc.)"
}

// Init initializes the plugin.
func (p *Plugin) Init(_ context.Context, _ plugins.Aegis) error {
	// Plugin initialization logic
	return nil
}

// MountRoutes registers HTTP routes for the OAuth plugin
func (p *Plugin) MountRoutes(router server.Router, prefix string) {
	handlers := NewHandlers(p)

	// OAuth authentication routes
	router.GET(prefix+"/oauth/:provider", handlers.BeginAuthHandler)
	router.GET(prefix+"/oauth/:provider/callback", handlers.CallbackHandler)
}

// Dependencies returns external package dependencies
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{
		{
			Package: "github.com/markbates/goth",
			Version: "latest",
			Purpose: "OAuth provider integration (recommended, but optional via abstraction)",
		},
	}
}

// RequiresTables returns core tables this plugin depends on
func (p *Plugin) RequiresTables() []string {
	return []string{"auth.user", "auth.session"}
}

// ProvidesAuthMethods returns authentication methods provided
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{"oauth_google", "oauth_github", "oauth_apple", "oauth_custom"}
}

// BeginAuth starts the OAuth authentication flow
// This uses Goth's gothic package as the default/recommended implementation
func (p *Plugin) BeginAuth(w http.ResponseWriter, r *http.Request, provider string) {
	// Set the provider in the URL for gothic
	q := r.URL.Query()
	q.Set("provider", provider)
	r.URL.RawQuery = q.Encode()

	// Use gothic to begin auth
	gothic.BeginAuthHandler(w, r)
}

// CompleteAuth completes the OAuth authentication flow using Goth
func (p *Plugin) CompleteAuth(ctx context.Context, w http.ResponseWriter, r *http.Request) (*models.User, *models.Session, error) {
	// Get Goth user from callback
	gothUser, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to complete OAuth: %w", err)
	}

	// Convert Goth user to our abstraction
	oauthUser := GothUserToUser(gothUser)

	// Get or create Aegis user
	user, err := p.getOrCreateUser(ctx, gothUser.Provider, oauthUser)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	// Create session
	ipAddress := r.RemoteAddr
	userAgent := r.Header.Get("User-Agent")
	session, err := p.sessionService.CreateSession(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return user, session, nil
}

// getOrCreateUser gets an existing user or creates a new one from OAuth data
func (p *Plugin) getOrCreateUser(ctx context.Context, provider string, oauthUser *User) (*models.User, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	// Check if OAuth connection already exists
	connection, err := p.db.GetConnectionByProviderUserID(ctx, provider, oauthUser.ID)
	if err == nil && connection != nil {
		// User exists, get and return them
		// Note: This requires access to core user DB - would need to be passed in Config
		// For now, we'll return a basic user with the connection's user ID
		return &models.User{
			ID: connection.UserID,
		}, nil
	}

	// Check if user with this email already exists
	var user *models.User
	if oauthUser.Email != "" {
		// This would require access to core user DB
		// For demo purposes, create a new user
		user = &models.User{
			ID:        core.GenerateID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// In real implementation, you'd call userDB.CreateUser here
	} else {
		// No email provided, create user with provider-based email
		user = &models.User{
			ID:        core.GenerateID(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	// Save OAuth connection
	conn := &Connection{
		ID:             core.GenerateID(),
		UserID:         user.ID,
		Provider:       provider,
		ProviderUserID: oauthUser.ID,
		Email:          oauthUser.Email,
		Name:           oauthUser.Name,
		AvatarURL:      oauthUser.AvatarURL,
		AccessToken:    oauthUser.AccessToken,
		RefreshToken:   oauthUser.RefreshToken,
		ExpiresAt:      oauthUser.ExpiresAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := p.db.SaveConnection(ctx, conn); err != nil {
		return nil, fmt.Errorf("failed to save OAuth connection: %w", err)
	}

	return user, nil
}

// LinkAccount links an OAuth provider to an existing user account
func (p *Plugin) LinkAccount(ctx context.Context, userID string, oauthUser *User, provider string) error {
	if p.db == nil {
		return fmt.Errorf("database not configured")
	}

	conn := &Connection{
		ID:             core.GenerateID(),
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: oauthUser.ID,
		Email:          oauthUser.Email,
		Name:           oauthUser.Name,
		AvatarURL:      oauthUser.AvatarURL,
		AccessToken:    oauthUser.AccessToken,
		RefreshToken:   oauthUser.RefreshToken,
		ExpiresAt:      oauthUser.ExpiresAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return p.db.SaveConnection(ctx, conn)
}

// GetUserConnections retrieves all OAuth connections for a user
func (p *Plugin) GetUserConnections(ctx context.Context, userID string) ([]*Connection, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not configured")
	}
	return p.db.GetUserConnections(ctx, userID)
}

// UnlinkAccount removes an OAuth provider link from a user account
func (p *Plugin) UnlinkAccount(ctx context.Context, userID, provider string) error {
	if p.db == nil {
		return fmt.Errorf("database not configured")
	}
	return p.db.DeleteConnection(ctx, provider, userID)
}
