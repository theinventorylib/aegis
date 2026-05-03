// Package plugins defines the plugin interface and core plugin types for Aegis.
package plugins

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

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

// ValidateColumnSpec checks column presence + type/nullability for postgres/mysql
var ValidateColumnSpec = core.ValidateColumnSpec

// Dialect-aware variants — prefer these in plugin code so SQLite deployments
// validate against pragma_table_info instead of the missing information_schema.
var (
	ValidateTableExistsForDialect  = core.ValidateTableExistsForDialect
	ValidateColumnExistsForDialect = core.ValidateColumnExistsForDialect
	ValidateColumnSpecForDialect   = core.ValidateColumnSpecForDialect
)

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
	GetPlugin(name string) (Plugin, bool)                                                   // Returns a registered plugin by name (for inter-plugin communication)
}

// Migration represents a database migration for a plugin.
type Migration struct {
	Version     int    // Migration version (e.g., 001, 002)
	Description string // Human-readable description
	Up          string // SQL for applying migration
	Down        string // SQL for reverting migration
}

// Dialect is an alias for config.Dialect.
type Dialect = config.Dialect

// Database dialect constants
const (
	// DialectPostgres represents PostgreSQL database
	DialectPostgres = config.DialectPostgres
	// DialectMySQL represents MySQL database
	DialectMySQL = config.DialectMySQL
	// DialectSQLite represents SQLite database
	DialectSQLite = config.DialectSQLite
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

	// Routing
	MountRoutes(router router.Router, prefix string) // Register HTTP routes

	// Metadata
	Dependencies() []Dependency    // External Go package dependencies (library-level, not inter-plugin)
	RequiresTables() []string      // Core tables this plugin reads from (used for schema validation in Init)
	ProvidesAuthMethods() []string // Auth method identifiers this plugin enables (e.g. "oauth_google", "jwt")
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

// PluginShutdown is an optional interface for plugins that hold resources requiring
// cleanup (goroutines, connections, timers). Aegis calls Shutdown on all plugins
// that implement this interface when Aegis.Shutdown is invoked, in reverse
// priority order (highest priority number first).
//
// Example implementation (JWT key-rotation goroutine):
//
//	func (p *Plugin) Shutdown(ctx context.Context) error {
//	    p.stopKeyRotation()
//	    return nil
//	}
type PluginShutdown interface {
	Shutdown(ctx context.Context) error
}

// PluginRequires is an optional interface for plugins that depend on other
// plugins being registered first. When a plugin implementing this interface is
// registered via Aegis.Use / Aegis.UseWithPriority, Aegis validates that every
// name returned by Requires is already registered.
//
// Example:
//
//	func (a *Plugin) Requires() []string {
//	    return []string{"organizations"}
//	}
type PluginRequires interface {
	Requires() []string
}

// HealthChecker is an optional interface for plugins that can report their own
// health. Aegis itself does not poll this automatically; it is intended for use
// by readiness / liveness handlers or monitoring integrations.
//
// Example:
//
//	func (p *Plugin) Health(ctx context.Context) error {
//	    return p.store.Ping(ctx)
//	}
type HealthChecker interface {
	Health(ctx context.Context) error
}

// VersionedRequirement specifies a plugin dependency with a minimum version constraint.
// Used by PluginVersionRequires to declare version-aware inter-plugin dependencies.
type VersionedRequirement struct {
	Plugin     string // Name of the required plugin (matches Plugin.Name())
	MinVersion string // Minimum acceptable semver, e.g. "2.0.0"
}

// PluginVersionRequires is an optional interface for plugins that need a specific
// minimum version of another plugin. Aegis checks this at Use/UseWithPriority time:
// the required plugin must already be registered AND its Version() must satisfy
// the declared MinVersion.
//
// Use this instead of (or alongside) PluginRequires when the API contract of the
// required plugin changed between versions and you need to enforce compatibility.
//
// Example:
//
//	func (p *Plugin) VersionedRequires() []plugins.VersionedRequirement {
//	    return []plugins.VersionedRequirement{
//	        {Plugin: "oauth", MinVersion: "2.0.0"},
//	    }
//	}
type PluginVersionRequires interface {
	VersionedRequires() []VersionedRequirement
}

// PluginMinAegisVersion is an optional interface for plugins that require a
// minimum version of the Aegis framework itself. Aegis checks this at
// Use/UseWithPriority time and returns an error if the running aegis.Version
// is below the declared minimum.
//
// Example:
//
//	func (p *Plugin) MinAegisVersion() string { return "1.1.0" }
type PluginMinAegisVersion interface {
	MinAegisVersion() string
}

// MeetsMinVersion reports whether `have` satisfies the semver lower-bound `need`.
// Both strings must be "major.minor.patch" (e.g. "2.1.0"). Returns true when
// have >= need by comparing each numeric component left-to-right.
// Malformed version strings are treated as "0.0.0" (never satisfies a non-zero need).
func MeetsMinVersion(have, need string) bool {
	parseSemver := func(v string) [3]int {
		parts := strings.SplitN(v, ".", 3)
		var parsed [3]int
		for i := 0; i < 3 && i < len(parts); i++ {
			n, err := strconv.Atoi(parts[i])
			if err != nil {
				return [3]int{}
			}
			parsed[i] = n
		}
		return parsed
	}
	h := parseSemver(have)
	n := parseSemver(need)
	for i := range h {
		if h[i] > n[i] {
			return true
		}
		if h[i] < n[i] {
			return false
		}
	}
	return true // equal
}

// CheckVersionedRequirements validates versioned plugin dependencies against a
// lookup function. Returns a descriptive error if any requirement is unmet.
// Called by Aegis internals; exposed so tests and custom hosts can reuse it.
func CheckVersionedRequirements(pluginName string, reqs []VersionedRequirement, lookup func(string) (Plugin, bool)) error {
	for _, req := range reqs {
		dep, found := lookup(req.Plugin)
		if !found {
			return fmt.Errorf("plugin %q requires plugin %q (>= %s) to be registered first",
				pluginName, req.Plugin, req.MinVersion)
		}
		if !MeetsMinVersion(dep.Version(), req.MinVersion) {
			return fmt.Errorf("plugin %q requires plugin %q version >= %s, but found %s",
				pluginName, req.Plugin, req.MinVersion, dep.Version())
		}
	}
	return nil
}
