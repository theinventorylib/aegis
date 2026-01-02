// Package plugins defines the plugin interface and core plugin types for Aegis.
package plugins

import (
	"context"
	"database/sql"

	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/router"
)

// SchemaRequirement defines a schema validation requirement
type SchemaRequirement = core.SchemaRequirement

// ValidateTableExists creates a requirement to check if a table exists
var ValidateTableExists = core.ValidateTableExists

// ValidateColumnExists creates a requirement to check if a column exists in a table
var ValidateColumnExists = core.ValidateColumnExists

// Aegis is the interface that plugins use to interact with the Aegis framework.
// Uses one generic parameter for the User model (U).
// Account, Session, and Verification models are standard across the framework.
type Aegis interface {
	GetAuthService() *core.AuthService                                                      // Returns the auth service for user operations
	GetLogger() config.Logger                                                               // Returns the configured logger (may be nil)
	GetRateLimiter() *core.RateLimiter                                                      // Returns the rate limiter (may be nil)
	DeriveSecret(purpose string) []byte                                                     // Derives a purpose-specific secret from the master secret
	DB() *sql.DB                                                                            // Returns the database connection for schema validation
	ValidateSchemaRequirements(ctx context.Context, requirements []SchemaRequirement) error // Validates schema requirements
}

// Migration represents a database migration for a plugin.
type Migration struct {
	Version     int    // Migration version (e.g., 001, 002)
	Description string // Human-readable description
	Up          string // SQL for applying migration
	Down        string // SQL for reverting migration
}

// SchemaInfo contains metadata about a schema
type SchemaInfo struct {
	Package     string
	Version     int
	Description string
}

// Schema represents the complete schema for a dialect
type Schema struct {
	Dialect Dialect
	SQL     string
	Info    SchemaInfo
}

// Dialect represents a database dialect
type Dialect string

// Database dialect constants
const (
	// DialectPostgres represents PostgreSQL database
	DialectPostgres Dialect = "postgres"
	// DialectMySQL represents MySQL database
	DialectMySQL Dialect = "mysql"
	// DialectSQLite represents SQLite database
	DialectSQLite Dialect = "sqlite"
)

// Dependency represents an external package dependency.
type Dependency struct {
	Package string // Go package import path
	Version string // Required version (e.g., "v1.2.3" or "latest")
	Purpose string // Why this dependency is needed
}

// Plugin is the interface that all plugins must implement.
// Simplified to one generic parameter (U).
type Plugin interface {
	// Identity
	Name() string        // Plugin identifier (e.g., "sms", "oauth", "email")
	Version() string     // Plugin version
	Description() string // Human-readable description

	// Lifecycle
	Init(ctx context.Context, a Aegis) error // Initialize plugin with Aegis instance
	GetMigrations() []Migration              // Return plugin-specific migrations
	GetSchemas() []Schema                    // Return plugin-specific schemas for all dialects

	// Routing
	MountRoutes(router router.Router, prefix string) // Register HTTP routes

	// Metadata
	Dependencies() []Dependency    // Informational only
	RequiresTables() []string      // Informational only
	ProvidesAuthMethods() []string // Informational only
}

// UserEnricher is an optional interface that plugins can implement to add
// extension fields to the EnrichedUser. Plugins that implement this interface
// will have their EnrichUser method called after authentication to populate
// user-specific data (role, permissions, organizations, etc.).
//
// Example implementation:
//
//	func (a *Admin) EnrichUser(ctx context.Context, user *core.EnrichedUser) error {
//	    role, err := a.store.GetRole(ctx, user.ID)
//	    if err == nil && role != "" {
//	        user.Set("role", role)
//	    }
//	    return nil
//	}
type UserEnricher interface {
	// EnrichUser adds plugin-specific data to the enriched user.
	// Called after authentication to populate extension fields.
	// Use simple field names (e.g., "role", not "admin:role").
	EnrichUser(ctx context.Context, user *core.EnrichedUser) error
}

// IsUserEnricher checks if a plugin implements the UserEnricher interface.
func IsUserEnricher(p Plugin) bool {
	_, ok := p.(UserEnricher)
	return ok
}

// GetUserEnricher returns the UserEnricher interface if the plugin implements it.
func GetUserEnricher(p Plugin) (UserEnricher, bool) {
	ue, ok := p.(UserEnricher)
	return ue, ok
}
