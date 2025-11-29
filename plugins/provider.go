// Package plugins defines the plugin interface and core plugin types for Aegis.
package plugins

import (
	"context"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/server"
)

// Aegis is the interface that plugins use to interact with the Aegis framework.
// This is a forward declaration to avoid import cycles.
// The actual implementation is in the main aegis package.
type Aegis interface {
	GetSessionService() *core.SessionService // Returns the session service for plugin integration
}

// Migration represents a database migration for a plugin.
type Migration struct {
	Version     string // Migration version (e.g., "001", "002")
	Description string // Human-readable description
	Up          string // SQL for applying migration
	Down        string // SQL for reverting migration
}

// Dependency represents an external package dependency.
type Dependency struct {
	Package string // Go package import path
	Version string // Required version (e.g., "v1.2.3" or "latest")
	Purpose string // Why this dependency is needed
}

// Plugin is the interface that all plugins must implement
type Plugin interface {
	// Identity
	Name() string        // Plugin identifier (e.g., "sms", "oauth", "email")
	Version() string     // Plugin version
	Description() string // Human-readable description

	// Lifecycle
	Init(ctx context.Context, a Aegis) error // Initialize plugin with Aegis instance
	GetMigrations() []Migration              // Return plugin-specific migrations

	// Routing
	MountRoutes(router server.Router, prefix string) // Register HTTP routes

	// Metadata
	// Dependencies returns external Go package dependencies required by this plugin.
	// This is INFORMATIONAL ONLY - Aegis does NOT validate or enforce these dependencies.
	// Users are responsible for ensuring dependencies are available (via go.mod or build tools).
	// Use this to document what packages must be imported for the plugin to function.
	Dependencies() []Dependency

	// RequiresTables returns the names of database tables this plugin depends on.
	// Format: "schema.table" (e.g., "auth.user", "auth.accounts")
	// This is INFORMATIONAL ONLY - Aegis does NOT validate table existence.
	// Users are responsible for running migrations in correct order.
	// The exporter uses this to document dependencies but does NOT reorder migrations.
	RequiresTables() []string

	// ProvidesAuthMethods returns authentication method names provided by this plugin.
	// Examples: "oauth_google", "sms_otp", "email_magic_link", "password"
	// This is used for documentation and plugin discovery.
	ProvidesAuthMethods() []string
}
