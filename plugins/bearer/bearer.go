// Package bearer provides Bearer token authentication support for Aegis.
//
// This plugin enables Bearer token authentication by activating token extraction
// from the Authorization header in the core AuthMiddleware. Without this plugin,
// Aegis only supports cookie-based session authentication.
//
// Bearer Authentication Flow:
//  1. Client obtains session token (via login, OAuth, etc.)
//  2. Client includes token in Authorization header: "Authorization: Bearer <token>"
//  3. AuthMiddleware extracts token from header (if bearer plugin is registered)
//  4. SessionService validates token against database + Redis cache
//  5. User is authenticated and injected into request context
//
// Use Cases:
//   - Mobile apps: No cookie support, need token-based auth
//   - API clients: cURL, Postman, automated scripts
//   - Server-to-server: Microservices, webhooks, integrations
//   - Single-page apps (SPAs): JavaScript clients with fetch/axios
//
// Security Notes:
//   - Tokens are the same session tokens used for cookie auth (shared infrastructure)
//   - Tokens should be stored securely (iOS keychain, Android keystore, secure storage)
//   - Use HTTPS to prevent token interception
//   - Tokens don't have CSRF protection (unlike cookies), but not vulnerable to CSRF
//
// Example:
//
//	package main
//
//	import (
//		"context"
//		"github.com/theinventorylib/aegis"
//		"github.com/theinventorylib/aegis/plugins/bearer"
//	)
//
//	func main() {
//		a, _ := aegis.New(context.Background(), ...)
//
//		// Enable Bearer token authentication
//		a.Use(context.Background(), bearer.New(nil))
//
//		a.MountRoutes("/auth")
//		// Now clients can use: Authorization: Bearer <session_token>
//	}
package bearer

import (
	"context"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/router"
)

// Plugin represents the Bearer token authentication plugin.
//
// This is a lightweight feature toggle plugin that enables Authorization header
// token extraction in the core AuthMiddleware. It doesn't add routes, migrations,
// or database tables - it simply activates existing session validation for Bearer tokens.
//
// Architecture:
//   - No database schema: Uses existing session tables
//   - No HTTP routes: Token validation is handled by core middleware
//   - No configuration: Zero-config feature toggle
//   - No dependencies: Works with any session backend
//
// When NOT to use this plugin:
//   - Cookie-only authentication: Don't register this plugin
//   - Custom token formats: Use JWT plugin for custom token structure
//   - OAuth only: OAuth plugin handles its own token formats
type Plugin struct {
	// sessionService stores reference to session service for feature activation
	sessionService *core.SessionService
}

// Config holds Bearer plugin configuration.
//
// Currently unused - reserved for future extensions:
//   - Custom token extraction patterns
//   - Token validation hooks
//   - Header name customization
//   - Token prefix customization (currently hardcoded to "Bearer ")
type Config struct {
	// Future: Could add configuration for custom token extraction, validation hooks, etc.
}

// New creates a new Bearer authentication plugin.
//
// The config parameter is currently unused (reserved for future features).
// Pass nil for now.
//
// Example:
//
//	bearerPlugin := bearer.New(nil)
//	aegis.Use(ctx, bearerPlugin)
func New(_ *Config) *Plugin {
	// Config is reserved for future use
	return &Plugin{}
}

// Name returns the plugin identifier used for registration and lookup.
//
// This name is used when:
//   - Registering the plugin with Aegis
//   - Looking up the plugin via GetPlugin("bearer")
//   - Logging and debugging
func (p *Plugin) Name() string {
	return "bearer"
}

// Version returns the plugin version following semantic versioning.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description for documentation and introspection.
func (p *Plugin) Description() string {
	return "Bearer token authentication support (feature toggle)"
}

// Init initializes the plugin and enables Bearer token authentication.
//
// This is called during plugin registration (aegis.Use) and performs the key
// activation: enabling Bearer token support in the SessionService.
//
// After Init completes:
//   - AuthMiddleware checks Authorization header for "Bearer <token>"
//   - Session tokens can be passed via header instead of cookies
//   - Mobile apps and API clients can authenticate
//
// Parameters:
//   - ctx: Context for initialization (can be cancelled)
//   - aegis: Framework instance providing access to services
//
// Returns:
//   - error: Always nil (initialization cannot fail for this plugin)
func (p *Plugin) Init(_ context.Context, aegis plugins.Aegis) error {
	// Store session service reference
	p.sessionService = aegis.GetAuthService().Session

	// Enable Bearer token authentication in core middleware
	// This is the key feature toggle - without this plugin registered,
	// the AuthMiddleware will NOT check Authorization headers
	p.sessionService.EnableBearerAuth()

	return nil
}

// GetMigrations returns the plugin database migrations.
//
// Bearer plugin has no migrations because it doesn't introduce new database tables.
// It reuses the existing session infrastructure.
//
// Returns empty slice.
func (p *Plugin) GetMigrations() []plugins.Migration {
	// No migrations needed - stateless plugin
	return []plugins.Migration{}
}

// GetSchemas returns database schemas for all supported dialects.
//
// Bearer plugin has no schema because it doesn't add database tables.
// It relies entirely on the core session tables.
//
// Returns empty slice.
func (p *Plugin) GetSchemas() []plugins.Schema {
	// Bearer plugin doesn't have its own schema
	return []plugins.Schema{}
}

// MountRoutes registers HTTP routes for the plugin.
//
// Bearer plugin has no routes because token validation is handled by the core
// AuthMiddleware automatically. The plugin is purely a feature toggle.
//
// Future considerations:
//   - Token introspection endpoint: /bearer/introspect
//   - Token refresh endpoint: /bearer/refresh
//   - Token revocation endpoint: /bearer/revoke
//
// For now, these features are available through core routes (/auth/logout, etc.).
func (p *Plugin) MountRoutes(_ router.Router, _ string) {
	// No routes needed - Bearer auth is handled by core middleware
	// Future: Could add /bearer/introspect endpoint for token introspection
}

// RequiresTables returns required database tables for schema validation.
//
// Bearer plugin requires the core session table to exist for token validation.
// This is validated during plugin initialization.
func (p *Plugin) RequiresTables() []string {
	// Uses core session tables
	return []string{"auth.session"}
}

// ProvidesAuthMethods returns authentication methods provided by this plugin.
//
// Returns ["bearer"] to indicate Bearer token authentication is available.
// This is used for introspection and capability discovery.
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{"bearer"}
}

// Dependencies returns plugin dependencies.
//
// Bearer plugin has no dependencies - it works standalone with core Aegis.
// It doesn't require any other plugins to function.
//
// Returns empty slice.
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}
