// Package bearer provides Bearer token authentication support for Aegis.
// When registered, this plugin enables Bearer token authentication by allowing
// the core AuthMiddleware to check for tokens in the Authorization header.
package bearer

import (
	"context"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// Plugin represents the Bearer authentication plugin.
// This plugin acts as a feature toggle - when registered, it enables
// Bearer token authentication in the core AuthMiddleware.
type Plugin struct {
	sessionService *core.SessionService
}

// Config holds Bearer plugin configuration.
type Config struct {
	// Future: Could add configuration for custom token extraction, validation hooks, etc.
}

// New creates a new Bearer plugin.
func New(_ *Config) *Plugin {
	// Config is reserved for future use
	return &Plugin{}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "bearer"
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "Bearer token authentication support (feature toggle)"
}

// Init initializes the plugin and enables Bearer authentication.
func (p *Plugin) Init(_ context.Context, aegis plugins.Aegis) error {
	// Store session service reference
	p.sessionService = aegis.GetSessionService()

	// Enable Bearer token authentication in core middleware
	// This is the key feature toggle - without this plugin registered,
	// the AuthMiddleware will NOT check Authorization headers
	p.sessionService.EnableBearerAuth()

	return nil
}

// GetMigrations returns the plugin migrations.
func (p *Plugin) GetMigrations() []plugins.Migration {
	// No migrations needed - stateless plugin
	return []plugins.Migration{}
}

// MountRoutes registers HTTP routes for the plugin.
func (p *Plugin) MountRoutes(_ server.Router, _ string) {
	// No routes needed - Bearer auth is handled by core middleware
	// Future: Could add /bearer/introspect endpoint for token introspection
}

// RequiresTables returns required database tables.
func (p *Plugin) RequiresTables() []string {
	// Uses core session tables
	return []string{"auth.session"}
}

// ProvidesAuthMethods returns authentication methods provided.
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{"bearer"}
}

// Dependencies returns plugin dependencies.
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}
