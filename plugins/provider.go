package plugins

import (
	"context"

	"github.com/theinventorylib/aegis/server"
)

// Aegis is a forward declaration to avoid import cycles.
// The actual type is defined in the main aegis package.
type Aegis interface{}

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
	Dependencies() []Dependency    // External dependencies used
	RequiresTables() []string      // Core tables this plugin depends on
	ProvidesAuthMethods() []string // Auth methods provided (e.g., "oauth_google", "sms_otp")
}
