// Package admin provides administrative functionality for user and organization management.
package admin

import (
	"context"
	"net/http"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// Plugin implements admin-level management
type Plugin struct {
	db             *DB
	sessionService *core.SessionService
}

// New creates a new admin plugin
func New(database db.Provider) *Plugin {
	return &Plugin{
		db: NewDB(database),
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "admin"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Admin dashboard and management API"
}

// Init initializes the admin plugin.
func (p *Plugin) Init(_ context.Context, aegis plugins.Aegis) error {
	// Store session service for auth middleware
	p.sessionService = aegis.GetSessionService()
	return nil
}

// MountRoutes registers HTTP routes
func (p *Plugin) MountRoutes(router server.Router, prefix string) {
	// Create admin middleware - ALL admin routes require admin role
	requireAdmin := RequireAdminMiddleware(p.sessionService, p.db)

	// User management - all protected
	router.GET(prefix+"/admin/users", requireAdmin(http.HandlerFunc(p.ListUsersHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/admin/users",
		Summary:     "List all users",
		Description: "Retrieve a paginated list of all users (admin only)",
		Tags:        []string{"Admin", "Users"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "List of users", Schema: SchemaUserList},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/admin/users/:id", requireAdmin(http.HandlerFunc(p.GetUserHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/admin/users/{id}",
		Summary:     "Get user details",
		Description: "Retrieve detailed information about a specific user (admin only)",
		Tags:        []string{"Admin", "Users"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "User details", Schema: models.SchemaUser},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
			"404": {Description: "User not found", Schema: models.SchemaError},
		},
	})

	router.POST(prefix+"/admin/users/:id/disable", requireAdmin(http.HandlerFunc(p.DisableUserHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/admin/users/{id}/disable",
		Summary:     "Disable user",
		Description: "Disable a user account (admin only)",
		Tags:        []string{"Admin", "Users"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "User disabled successfully", Schema: models.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
			"404": {Description: "User not found", Schema: models.SchemaError},
		},
	})

	router.POST(prefix+"/admin/users/:id/enable", requireAdmin(http.HandlerFunc(p.EnableUserHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/admin/users/{id}/enable",
		Summary:     "Enable user",
		Description: "Enable a previously disabled user account (admin only)",
		Tags:        []string{"Admin", "Users"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "User enabled successfully", Schema: models.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
			"404": {Description: "User not found", Schema: models.SchemaError},
		},
	})

	router.DELETE(prefix+"/admin/users/:id", requireAdmin(http.HandlerFunc(p.DeleteUserHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "DELETE",
		Path:        prefix + "/admin/users/{id}",
		Summary:     "Delete user",
		Description: "Permanently delete a user account (admin only)",
		Tags:        []string{"Admin", "Users"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "User deleted successfully", Schema: models.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
			"404": {Description: "User not found", Schema: models.SchemaError},
		},
	})

	// Organization management - all protected
	router.POST(prefix+"/admin/organizations", requireAdmin(http.HandlerFunc(p.AddOrganizationHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/admin/organizations",
		Summary:     "Create organization",
		Description: "Create a new organization (admin only)",
		Tags:        []string{"Admin", "Organizations"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "Organization details",
			Required:    true,
			Schema:      "CreateOrganizationRequest",
		},
		Responses: map[string]*models.ResponseMeta{
			"201": {Description: "Organization created successfully", Schema: "Organization"},
			"400": {Description: "Invalid request", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/admin/organizations", requireAdmin(http.HandlerFunc(p.ListOrganizationsHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/admin/organizations",
		Summary:     "List all organizations",
		Description: "Retrieve a paginated list of all organizations (admin only)",
		Tags:        []string{"Admin", "Organizations"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "List of user organizations", Schema: "OrganizationList"},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Admin access required", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/admin/organizations/:id", requireAdmin(http.HandlerFunc(p.GetOrganizationHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/admin/organizations/{id}",
		Summary:     "Get organization details",
		Description: "Retrieve detailed information about a specific organization (admin only)",
		Tags:        []string{"Admin", "Organizations"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Organization details", Schema: "Organization"},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
			"404": {Description: "Organization not found", Schema: models.SchemaError},
		},
	})

	router.POST(prefix+"/admin/organizations/:id/ban", requireAdmin(http.HandlerFunc(p.BanOrganizationHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/admin/organizations/{id}/ban",
		Summary:     "Ban organization",
		Description: "Ban an organization and disable all its members (admin only)",
		Tags:        []string{"Admin", "Organizations"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Organization banned successfully", Schema: models.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
			"404": {Description: "Organization not found", Schema: models.SchemaError},
		},
	})

	router.DELETE(prefix+"/admin/organizations/:id", requireAdmin(http.HandlerFunc(p.DeleteOrganizationHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "DELETE",
		Path:        prefix + "/admin/organizations/{id}",
		Summary:     "Delete organization",
		Description: "Permanently delete an organization (admin only)",
		Tags:        []string{"Admin", "Organizations"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Organization deleted successfully", Schema: models.SchemaSuccess},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
			"404": {Description: "Organization not found", Schema: models.SchemaError},
		},
	})

	// Stats and analytics - protected
	router.GET(prefix+"/admin/stats", requireAdmin(http.HandlerFunc(p.GetStatsHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/admin/stats",
		Summary:     "Get platform statistics",
		Description: "Retrieve platform-wide statistics and analytics (admin only)",
		Tags:        []string{"Admin", "Analytics"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Platform statistics", Schema: "Stats"},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
		},
	})

	// Ban management - protected
	router.POST(prefix+"/admin/users/:id/ban", requireAdmin(http.HandlerFunc(p.BanUserHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/admin/users/{id}/ban",
		Summary:     "Ban user",
		Description: "Ban a user with a reason and optional expiry (admin only)",
		Tags:        []string{"Admin", "Users", "Ban Management"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "Ban details (reason required, expiresAt optional)",
			Required:    true,
			Schema:      "BanUserRequest",
		},
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "User banned successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
			"404": {Description: "User not found", Schema: models.SchemaError},
		},
	})

	router.POST(prefix+"/admin/users/:id/unban", requireAdmin(http.HandlerFunc(p.UnbanUserHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/admin/users/{id}/unban",
		Summary:     "Unban user",
		Description: "Remove ban from a previously banned user (admin only)",
		Tags:        []string{"Admin", "Users", "Ban Management"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "User unbanned successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not authorized (requires admin role)", Schema: models.SchemaError},
			"404": {Description: "User not found", Schema: models.SchemaError},
		},
	})
}

// Dependencies returns plugin dependencies
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}

// RequiresTables returns required tables
func (p *Plugin) RequiresTables() []string {
	return []string{"auth.user", "auth.organizations"}
}

// ProvidesAuthMethods returns auth methods
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{}
}
