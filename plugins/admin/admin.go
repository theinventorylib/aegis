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
//   - PUT    /admin/users/:id/role    - Update user role
//   - GET    /admin/stats             - Get platform statistics
package admin

import (
	"context"
	"net/http"
	"time"

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
	// aegis is the main framework instance
	aegis plugins.Aegis
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
func New(store Store, dialect ...plugins.Dialect) *Plugin {
	d := plugins.DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}
	return &Plugin{
		store:   store,
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
	a.aegis = aegis

	// Build schema requirements: basic table existence from RequiresTables + detailed checks
	tables := a.RequiresTables()
	requirements := make([]plugins.SchemaRequirement, 0, len(tables))
	for _, table := range tables {
		requirements = append(requirements, plugins.ValidateTableExists(table))
	}
	requirements = append(requirements, GetSchemaRequirements(a.dialect)...)

	// Validate admin plugin schema requirements
	if err := aegis.ValidateSchemaRequirements(ctx, requirements); err != nil {
		return err
	}

	// Schema registration moved to MountRoutes to ensure all plugins are initialized
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
	// Register schemas with OpenAPI if available
	if plugin, ok := a.aegis.GetPlugin("openapi"); ok {
		if oapi, ok := plugin.(interface {
			RegisterSchemaFromType(name string, example any)
		}); ok {
			// Register request schemas
			oapi.RegisterSchemaFromType(SchemaBanUserRequest, BanUserRequest{})
			oapi.RegisterSchemaFromType(SchemaUpdateRoleRequest, UpdateRoleRequest{})

			// Register response schemas
			oapi.RegisterSchemaFromType(SchemaAdminUser, User{})
			oapi.RegisterSchemaFromType(SchemaUserListResponse, UserListResponse{})
			oapi.RegisterSchemaFromType(SchemaAdminStats, StatsResponse{})
		}
	}

	// Create admin middleware - ALL admin routes require admin role
	requireAdmin := a.RequireAdminMiddleware()

	// Create route group for Admin plugin
	adminGroup := r.Group(prefix, "Admin")

	// User management - all protected
	adminGroup.GET("/users", requireAdmin(http.HandlerFunc(a.listUsersHandler)).ServeHTTP)
	adminGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/users",
		Summary:     "List users",
		Description: "List all users (admin only)",
		Tags:        []string{"Admin"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "List of users", Schema: SchemaUserListResponse},
			"401": {Description: "Not authorized", Schema: core.SchemaError},
		},
	})

	adminGroup.GET("/users/:id", requireAdmin(http.HandlerFunc(a.getUserHandler)).ServeHTTP)
	adminGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        router.NormalizePathToOpenAPI(prefix + "/users/:id"),
		Summary:     "Get user",
		Description: "Get user details by ID (admin only)",
		Tags:        []string{"Admin"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "User details", Schema: SchemaAdminUser},
			"401": {Description: "Not authorized", Schema: core.SchemaError},
			"404": {Description: "User not found", Schema: core.SchemaError},
		},
	})

	adminGroup.POST("/users/:id/disable", requireAdmin(http.HandlerFunc(a.disableUserHandler)).ServeHTTP)
	adminGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        router.NormalizePathToOpenAPI(prefix + "/users/:id/disable"),
		Summary:     "Disable user",
		Description: "Disable a user account (admin only)",
		Tags:        []string{"Admin"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "User disabled", Schema: core.SchemaSuccess},
			"401": {Description: "Not authorized", Schema: core.SchemaError},
		},
	})

	adminGroup.POST("/users/:id/enable", requireAdmin(http.HandlerFunc(a.enableUserHandler)).ServeHTTP)
	adminGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        router.NormalizePathToOpenAPI(prefix + "/users/:id/enable"),
		Summary:     "Enable user",
		Description: "Enable a user account (admin only)",
		Tags:        []string{"Admin"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "User enabled", Schema: core.SchemaSuccess},
			"401": {Description: "Not authorized", Schema: core.SchemaError},
		},
	})

	adminGroup.DELETE("/users/:id", requireAdmin(http.HandlerFunc(a.deleteUserHandler)).ServeHTTP)
	adminGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "DELETE",
		Path:        router.NormalizePathToOpenAPI(prefix + "/users/:id"),
		Summary:     "Delete user",
		Description: "Delete a user account (admin only)",
		Tags:        []string{"Admin"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "User deleted", Schema: core.SchemaSuccess},
			"401": {Description: "Not authorized", Schema: core.SchemaError},
			"404": {Description: "User not found", Schema: core.SchemaError},
		},
	})

	// Ban management - protected
	adminGroup.POST("/users/:id/ban", requireAdmin(http.HandlerFunc(a.banUserHandler)).ServeHTTP)
	adminGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        router.NormalizePathToOpenAPI(prefix + "/users/:id/ban"),
		Summary:     "Ban user",
		Description: "Ban a user with reason and expiry (admin only)",
		Tags:        []string{"Admin"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Ban details",
			Required:    true,
			Schema:      SchemaBanUserRequest,
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "User banned", Schema: core.SchemaSuccess},
			"401": {Description: "Not authorized", Schema: core.SchemaError},
		},
	})

	adminGroup.POST("/users/:id/unban", requireAdmin(http.HandlerFunc(a.unbanUserHandler)).ServeHTTP)
	adminGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        router.NormalizePathToOpenAPI(prefix + "/users/:id/unban"),
		Summary:     "Unban user",
		Description: "Remove ban from user (admin only)",
		Tags:        []string{"Admin"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "User unbanned", Schema: core.SchemaSuccess},
			"401": {Description: "Not authorized", Schema: core.SchemaError},
		},
	})

	// Role management - protected
	adminGroup.PUT("/users/:id/role", requireAdmin(http.HandlerFunc(a.updateRoleHandler)).ServeHTTP)
	adminGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "PUT",
		Path:        router.NormalizePathToOpenAPI(prefix + "/users/:id/role"),
		Summary:     "Update user role",
		Description: "Update a user's role (admin only)",
		Tags:        []string{"Admin"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Role update details",
			Required:    true,
			Schema:      SchemaUpdateRoleRequest,
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Role updated", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request", Schema: core.SchemaError},
			"401": {Description: "Not authorized", Schema: core.SchemaError},
			"404": {Description: "User not found", Schema: core.SchemaError},
		},
	})

	// Stats and analytics - protected
	adminGroup.GET("/stats", requireAdmin(http.HandlerFunc(a.getStatsHandler)).ServeHTTP)
	adminGroup.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/stats",
		Summary:     "Admin stats",
		Description: "Get platform statistics (admin only)",
		Tags:        []string{"Admin"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Statistics", Schema: SchemaAdminStats},
			"401": {Description: "Not authorized", Schema: core.SchemaError},
		},
	})
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

	adminUser, err := a.store.GetByID(ctx, user.ID)
	if err != nil {
		// Even if lookup fails, we don't want to fail the whole enrichment process
		// Silently ignore role lookup errors as they're not critical
		return err
	}

	if adminUser.Role != "" {
		user.Set(ExtKeyRole, adminUser.Role)
	}
	// TODO: enrich with the other fields

	return nil
}

// ========== PROGRAMMATIC FUNCTIONS ==========

// ListUsers lists all users programmatically.
func (a *Plugin) ListUsers(ctx context.Context, offset, limit int) ([]User, error) {
	return a.store.List(ctx, offset, limit)
}

// ListUsersRaw lists all users as raw map data programmatically.
func (a *Plugin) ListUsersRaw(ctx context.Context, offset, limit int) ([]map[string]any, error) {
	return a.store.ListUsersRaw(ctx, offset, limit)
}

// GetUser retrieves detailed information for a specific user programmatically.
func (a *Plugin) GetUser(ctx context.Context, userID string) (User, error) {
	return a.store.GetByID(ctx, userID)
}

// GetUserRaw retrieves detailed information for a specific user as raw map data programmatically.
func (a *Plugin) GetUserRaw(ctx context.Context, userID string) (map[string]any, error) {
	return a.store.GetUserRaw(ctx, userID)
}

// DisableUser disables a user account programmatically.
func (a *Plugin) DisableUser(ctx context.Context, userID string) error {
	user, err := a.store.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	user.Disabled = true
	return a.store.Update(ctx, user)
}

// EnableUser re-enables a user account programmatically.
func (a *Plugin) EnableUser(ctx context.Context, userID string) error {
	user, err := a.store.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	user.Disabled = false
	return a.store.Update(ctx, user)
}

// DeleteUser permanently deletes a user account programmatically.
func (a *Plugin) DeleteUser(ctx context.Context, userID string) error {
	return a.store.Delete(ctx, userID)
}

// BanUser bans a user programmatically.
func (a *Plugin) BanUser(ctx context.Context, userID, reason string, expiresAt *time.Time) error {
	// Sanitize ban reason
	reason = core.SanitizeString(reason, nil)

	return a.store.BanUser(ctx, userID, reason, expiresAt)
}

// UnbanUser removes the ban from a user account programmatically.
func (a *Plugin) UnbanUser(ctx context.Context, userID string) error {
	return a.store.UnbanUser(ctx, userID)
}

// GetStats returns platform statistics programmatically.
func (a *Plugin) GetStats(ctx context.Context) (StatsResponse, error) {
	return a.store.GetStats(ctx)
}

// AssignRole assigns a role to a user programmatically.
func (a *Plugin) AssignRole(ctx context.Context, userID string, role string) error {
	return a.store.AssignRole(ctx, userID, role)
}

// GetUserRole retrieves the role of a user programmatically.
func (a *Plugin) GetUserRole(ctx context.Context, userID string) (string, error) {
	return a.store.GetRole(ctx, userID)
}

// RemoveRole removes a role from a user programmatically.
func (a *Plugin) RemoveRole(ctx context.Context, userID string, role string) error {
	return a.store.RemoveRole(ctx, userID, role)
}

// GetAdminUser retrieves a user with admin-specific information.
func (a *Plugin) GetAdminUser(ctx context.Context, userID string) (User, error) {
	return a.store.GetByID(ctx, userID)
}

// Ensure Admin implements UserEnricher
var _ plugins.UserEnricher = (*Plugin)(nil)

// Ensure Admin implements Plugin
var _ plugins.Plugin = (*Plugin)(nil)
