// Package admin provides role-based access control (RBAC) and administrative user management.
//
// This plugin extends the auth.User model with:
//   - Role management (admin role assignment)
//   - User ban/unban functionality with expiry dates
//   - Platform statistics and analytics
//   - Administrative CRUD operations for user accounts
//
// All admin endpoints require the 'admin' role via RequireAdminMiddleware.
//
// Key Features:
//   - Role-based authorization (admin role)
//   - User ban management with reasons and expiry dates
//   - User enable/disable controls
//   - User listing with pagination
//   - Platform statistics
//
// Database Extensions:
// Adds columns to the 'user' table:
//   - role (VARCHAR): User role (e.g., "admin")
//   - banned (BOOLEAN): Ban status
//   - ban_reason (TEXT): Ban reason text
//   - ban_expiry (TIMESTAMP): Ban expiration date (NULL for permanent)
//   - ban_counter (INTEGER): Number of times user has been banned
//
// Route Structure:
//   - GET    /admin/users            - List all users (paginated)
//   - GET    /admin/users/:id        - Get user details
//   - POST   /admin/users/:id/disable - Disable user account
//   - POST   /admin/users/:id/enable  - Enable user account
//   - DELETE /admin/users/:id        - Delete user account
//   - POST   /admin/users/:id/ban     - Ban user with reason
//   - POST   /admin/users/:id/unban   - Unban user
//   - GET    /admin/stats             - Get platform statistics
package admin

import (
	"context"
	"net/http"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/router"
)

// Plugin provides role-based access control and administrative user management.
//
// This plugin integrates with the core authentication system to provide:
//   - Plugin role verification via middleware
//   - User account management (enable/disable/delete)
//   - Ban management with expiry dates and reasons
//   - Platform statistics and analytics
//
// The plugin implements plugins.UserEnricher to automatically add role information
// to authenticated users, making it available in API responses.
type Plugin struct {
	// store handles admin-specific database operations
	store Store
	// dialect specifies the database dialect (postgres, mysql)
	dialect plugins.Dialect
	// sessionService provides authentication verification
	sessionService *core.SessionService
}

// New creates a new Admin plugin instance.
//
// Parameters:
//   - store: AdminStore implementation for database operations (can be nil, will use DefaultAdminStore)
//   - dialect: Optional database dialect (defaults to PostgreSQL)
//
// Returns:
//   - *Admin: Configured admin plugin ready for initialization
//
// Example:
//
//	admin := admin.New(nil, plugins.DialectPostgres)
//	aegis.RegisterPlugin(admin)
func New(store *Store, dialect ...plugins.Dialect) *Plugin {
	d := plugins.DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}
	return &Plugin{
		store:   *store,
		dialect: d,
	}
}

// Name returns the plugin identifier.
func (a *Plugin) Name() string {
	return "admin"
}

// Version returns the plugin version for compatibility tracking.
func (a *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description for logging and diagnostics.
func (a *Plugin) Description() string {
	return "Admin plugin for user role management"
}

// Init initializes the admin plugin and validates database schema.
//
// Initialization Steps:
//  1. Create DefaultAdminStore if custom store not provided
//  2. Store session service reference for authentication middleware
//  3. Build schema validation requirements (table + column checks)
//  4. Validate admin schema extensions exist in database
//
// Schema Validation:
// Checks for required columns in 'user' table:
//   - role: VARCHAR for role assignment
//   - banned, ban_reason, ban_expiry, ban_counter: Ban management fields
//
// Parameters:
//   - ctx: Context for database operations
//   - aegis: Main Aegis instance providing DB access and services
//
// Returns:
//   - error: If schema validation fails or initialization errors occur
func (a *Plugin) Init(ctx context.Context, aegis plugins.Aegis) error {
	// Initialize store if not provided
	if a.store == nil {
		a.store = NewDefaultAdminStore(aegis.DB())
	}

	// Store session service for auth middleware
	a.sessionService = aegis.GetAuthService().Session

	// Build schema requirements: basic table existence from RequiresTables + detailed checks
	requirements := []plugins.SchemaRequirement{}
	for _, table := range a.RequiresTables() {
		requirements = append(requirements, plugins.ValidateTableExists(table))
	}
	requirements = append(requirements, GetSchemaRequirements(a.dialect)...)

	// Validate admin plugin schema requirements
	if err := aegis.ValidateSchemaRequirements(ctx, requirements); err != nil {
		return err
	}

	return nil
}

// GetMigrations returns the plugin migrations
func (a *Plugin) GetMigrations() []plugins.Migration {
	migs, err := GetMigrations(a.dialect)
	if err != nil {
		return []plugins.Migration{}
	}
	return migs
}

// MountRoutes registers administrative management endpoints.
func (a *Plugin) MountRoutes(r router.Router, prefix string) {
	// Create admin middleware - ALL admin routes require admin role
	requireAdmin := a.RequireAdminMiddleware()

	// User management - all protected
	r.GET(prefix+"/admin/users", requireAdmin(http.HandlerFunc(a.ListUsersHandler)).ServeHTTP)
	r.GET(prefix+"/admin/users/:id", requireAdmin(http.HandlerFunc(a.GetUserHandler)).ServeHTTP)
	r.POST(prefix+"/admin/users/:id/disable", requireAdmin(http.HandlerFunc(a.DisableUserHandler)).ServeHTTP)
	r.POST(prefix+"/admin/users/:id/enable", requireAdmin(http.HandlerFunc(a.EnableUserHandler)).ServeHTTP)
	r.DELETE(prefix+"/admin/users/:id", requireAdmin(http.HandlerFunc(a.DeleteUserHandler)).ServeHTTP)

	// Ban management - protected
	r.POST(prefix+"/admin/users/:id/ban", requireAdmin(http.HandlerFunc(a.BanUserHandler)).ServeHTTP)
	r.POST(prefix+"/admin/users/:id/unban", requireAdmin(http.HandlerFunc(a.UnbanUserHandler)).ServeHTTP)

	// Stats and analytics - protected
	r.GET(prefix+"/admin/stats", requireAdmin(http.HandlerFunc(a.GetStatsHandler)).ServeHTTP)
}

// Dependencies returns the plugin dependencies
func (a *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}

// RequiresTables returns the required tables
func (a *Plugin) RequiresTables() []string {
	return []string{"user"}
}

// ProvidesAuthMethods returns the provided auth methods
func (a *Plugin) ProvidesAuthMethods() []string {
	return []string{}
}

// GetSchemas returns all schemas for all supported dialects
func (a *Plugin) GetSchemas() []plugins.Schema {
	dialects := []plugins.Dialect{plugins.DialectPostgres, plugins.DialectMySQL}
	schemas := make([]plugins.Schema, 0, len(dialects))

	for _, dialect := range dialects {
		schema, err := GetSchema(dialect)
		if err != nil {
			continue
		}
		schemas = append(schemas, *schema)
	}

	return schemas
}

// EnrichUser implements plugins.UserEnricher to add admin-specific fields to user responses.
//
// This method is called automatically by the authentication system after user lookup.
// It adds the user's role to the EnrichedUser, making it available in API responses
// without requiring separate queries.
//
// Fields Added:
//   - "role" (string): User's role (e.g., "admin", empty if no role)
//
// The enriched role is accessible via:
//   - In API responses: {"user": {"id": "...", "role": "admin", ...}}
//   - In middleware: plugins.GetUserExtensionString(ctx, ExtKeyRole)
//   - In handlers: enrichedUser.GetString("role")
//
// Parameters:
//   - ctx: Request context
//   - user: EnrichedUser to populate with admin data
//
// Returns:
//   - error: Always nil (role lookup failure is not an error)
func (a *Plugin) EnrichUser(ctx context.Context, user *core.EnrichedUser) error {
	if user == nil || user.User == nil {
		return nil
	}

	role, err := a.store.GetRole(ctx, user.ID)
	if err != nil {
		// Even if lookup fails, we don't want to fail the whole enrichment process
		// unless it's a critical error. For now, we return nil to maintain existing behavior
		// but we blank the error to show it's intentional.
		_ = err
		return nil
	}

	if role != "" {
		user.Set(ExtKeyRole, role)
	}

	return nil
}

// Ensure Admin implements UserEnricher
var _ plugins.UserEnricher = (*Plugin)(nil)
