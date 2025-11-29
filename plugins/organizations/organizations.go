package organizations

import (
	"context"
	"net/http"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// Plugin implements organization and team management
type Plugin struct {
	database       db.Provider
	sessionService *core.SessionService
}

// New creates a new organizations plugin
func New(database db.Provider) *Plugin {
	return &Plugin{
		database: database,
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "organizations"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Organization and team management plugin"
}

// Init initializes the organizations plugin
func (p *Plugin) Init(_ context.Context, aegis plugins.Aegis) error {
	// Store session service for auth middleware
	p.sessionService = aegis.GetSessionService()
	return nil
}

// MountRoutes registers HTTP routes for the organizations plugin
func (p *Plugin) MountRoutes(router server.Router, prefix string) {
	// Create auth middleware - ALL organization routes require authentication
	requireAuth := core.RequireAuthMiddleware(p.sessionService)

	// Organization CRUD - all protected
	router.POST(prefix+"/organizations", requireAuth(http.HandlerFunc(p.CreateOrganizationHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/organizations",
		Summary:     "Create organization",
		Description: "Create a new organization with the authenticated user as owner",
		Tags:        []string{"Organizations"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "Organization details",
			Required:    true,
			Schema:      SchemaCreateOrganizationRequest,
		},
		Responses: map[string]*models.ResponseMeta{
			"201": {Description: "Organization created successfully", Schema: SchemaOrganization},
			"400": {Description: "Invalid request or validation error", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/organizations", requireAuth(http.HandlerFunc(p.ListOrganizationsHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/organizations",
		Summary:     "List user organizations",
		Description: "Retrieve all organizations the authenticated user is a member of",
		Tags:        []string{"Organizations"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "List of organizations", Schema: SchemaOrganizationList},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"500": {Description: "Internal server error", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/organizations/:id", requireAuth(http.HandlerFunc(p.GetOrganizationHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        server.NormalizePathToOpenAPI(prefix + "/organizations/:id"),
		Summary:     "Get organization",
		Description: "Retrieve details of a specific organization",
		Tags:        []string{"Organizations"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Organization details", Schema: SchemaOrganization},
			"400": {Description: "Invalid organization ID", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: models.SchemaError},
			"404": {Description: "Organization not found", Schema: models.SchemaError},
		},
	})

	router.PUT(prefix+"/organizations/:id", requireAuth(http.HandlerFunc(p.UpdateOrganizationHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "PUT",
		Path:        prefix + "/organizations/{id}",
		Summary:     "Update organization",
		Description: "Update organization details (requires owner or admin role)",
		Tags:        []string{"Organizations"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "Updated organization details",
			Required:    true,
			Schema:      UpdateOrganizationRequest{},
		},
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Organization updated successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: models.SchemaError},
		},
	})

	router.DELETE(prefix+"/organizations/:id", requireAuth(http.HandlerFunc(p.DeleteOrganizationHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "DELETE",
		Path:        prefix + "/organizations/{id}",
		Summary:     "Delete organization",
		Description: "Delete an organization (requires owner role)",
		Tags:        []string{"Organizations"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Organization deleted successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid organization ID", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Only owner can delete organization", Schema: models.SchemaError},
			"500": {Description: "Internal server error", Schema: models.SchemaError},
		},
	})

	// Organization Member Management - all protected
	router.POST(prefix+"/organizations/:id/members", requireAuth(http.HandlerFunc(p.AddOrganizationMemberHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/organizations/{id}/members",
		Summary:     "Add organization member",
		Description: "Add a new member to the organization (requires admin role)",
		Tags:        []string{"Organizations", "Members"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "Member details (userId and role)",
			Required:    true,
			Schema:      AddOrganizationMemberRequest{},
		},
		Responses: map[string]*models.ResponseMeta{
			"201": {Description: "Member added successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/organizations/:id/members", requireAuth(http.HandlerFunc(p.ListOrganizationMembersHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/organizations/{id}/members",
		Summary:     "List organization members",
		Description: "Retrieve all members of an organization",
		Tags:        []string{"Organizations", "Members"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "List of organization members", Schema: SchemaMemberList},
			"400": {Description: "Invalid organization ID", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: models.SchemaError},
			"500": {Description: "Internal server error", Schema: models.SchemaError},
		},
	})

	router.PATCH(prefix+"/organizations/:id/members/:userId", requireAuth(http.HandlerFunc(p.UpdateMemberRoleHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "PATCH",
		Path:        prefix + "/organizations/{id}/members/{userId}",
		Summary:     "Update member role",
		Description: "Update a member's role in the organization (requires owner role)",
		Tags:        []string{"Organizations", "Members"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "New role for the member",
			Required:    true,
			Schema:      UpdateMemberRoleRequest{},
		},
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Role updated successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Only owner can update roles", Schema: models.SchemaError},
		},
	})

	router.DELETE(prefix+"/organizations/:id/members/:userId", requireAuth(http.HandlerFunc(p.RemoveOrganizationMemberHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "DELETE",
		Path:        prefix + "/organizations/{id}/members/{userId}",
		Summary:     "Remove organization member",
		Description: "Remove a member from the organization (requires admin role, cannot remove owner)",
		Tags:        []string{"Organizations", "Members"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Member removed successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request or cannot remove owner", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: models.SchemaError},
		},
	})

	// Team CRUD - all protected
	router.POST(prefix+"/organizations/:id/teams", requireAuth(http.HandlerFunc(p.CreateTeamHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/organizations/{id}/teams",
		Summary:     "Create team",
		Description: "Create a new team within an organization (requires admin role)",
		Tags:        []string{"Teams"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "Team details",
			Required:    true,
			Schema:      CreateTeamRequest{},
		},
		Responses: map[string]*models.ResponseMeta{
			"201": {Description: "Team created successfully", Schema: "Team"},
			"400": {Description: "Invalid request or validation error", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/organizations/:id/teams", requireAuth(http.HandlerFunc(p.ListTeamsHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/organizations/{id}/teams",
		Summary:     "List organization teams",
		Description: "Retrieve all teams in an organization",
		Tags:        []string{"Teams"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "List of teams", Schema: SchemaTeamList},
			"400": {Description: "Invalid organization ID", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: models.SchemaError},
			"500": {Description: "Internal server error", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/teams/:teamId", requireAuth(http.HandlerFunc(p.GetTeamHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/teams/{teamId}",
		Summary:     "Get team",
		Description: "Retrieve details of a specific team",
		Tags:        []string{"Teams"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Team details", Schema: "Team"},
			"400": {Description: "Invalid team ID", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: models.SchemaError},
			"404": {Description: "Team not found", Schema: models.SchemaError},
		},
	})

	router.PUT(prefix+"/teams/:teamId", requireAuth(http.HandlerFunc(p.UpdateTeamHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "PUT",
		Path:        prefix + "/teams/{teamId}",
		Summary:     "Update team",
		Description: "Update team details (requires admin role)",
		Tags:        []string{"Teams"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "Updated team details",
			Required:    true,
			Schema:      UpdateTeamRequest{},
		},
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Team updated successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: models.SchemaError},
			"404": {Description: "Team not found", Schema: models.SchemaError},
		},
	})

	router.DELETE(prefix+"/teams/:teamId", requireAuth(http.HandlerFunc(p.DeleteTeamHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "DELETE",
		Path:        prefix + "/teams/{teamId}",
		Summary:     "Delete team",
		Description: "Delete a team (requires admin role)",
		Tags:        []string{"Teams"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Team deleted successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid team ID", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: models.SchemaError},
			"404": {Description: "Team not found", Schema: models.SchemaError},
			"500": {Description: "Internal server error", Schema: models.SchemaError},
		},
	})

	// Team Member Management - all protected
	router.POST(prefix+"/teams/:teamId/members", requireAuth(http.HandlerFunc(p.AddTeamMemberHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/teams/{teamId}/members",
		Summary:     "Add team member",
		Description: "Add a member to a team (requires admin role, user must be organization member)",
		Tags:        []string{"Teams", "Members"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "Team member details (userId and role)",
			Required:    true,
			Schema:      AddTeamMemberRequest{},
		},
		Responses: map[string]*models.ResponseMeta{
			"201": {Description: "Member added to team successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request or user not organization member", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: models.SchemaError},
			"404": {Description: "Team not found", Schema: models.SchemaError},
		},
	})

	router.GET(prefix+"/teams/:teamId/members", requireAuth(http.HandlerFunc(p.ListTeamMembersHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/teams/{teamId}/members",
		Summary:     "List team members",
		Description: "Retrieve all members of a team",
		Tags:        []string{"Teams", "Members"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "List of team members", Schema: SchemaTeamMemberList},
			"400": {Description: "Invalid team ID", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: models.SchemaError},
			"404": {Description: "Team not found", Schema: models.SchemaError},
			"500": {Description: "Internal server error", Schema: models.SchemaError},
		},
	})

	router.PATCH(prefix+"/teams/:teamId/members/:userId", requireAuth(http.HandlerFunc(p.UpdateTeamMemberRoleHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "PATCH",
		Path:        prefix + "/teams/{teamId}/members/{userId}",
		Summary:     "Update team member role",
		Description: "Update a team member's role (requires admin role)",
		Tags:        []string{"Teams", "Members"},
		Protected:   true,
		RequestBody: &models.RequestBodyMeta{
			Description: "New role for the team member",
			Required:    true,
			Schema:      UpdateTeamMemberRoleRequest{},
		},
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Team member role updated successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: models.SchemaError},
			"404": {Description: "Team not found", Schema: models.SchemaError},
		},
	})

	router.DELETE(prefix+"/teams/:teamId/members/:userId", requireAuth(http.HandlerFunc(p.RemoveTeamMemberHandler)).ServeHTTP)
	router.RegisterRouteMetadata(models.RouteMetadata{
		Method:      "DELETE",
		Path:        prefix + "/teams/{teamId}/members/{userId}",
		Summary:     "Remove team member",
		Description: "Remove a member from a team (requires admin role)",
		Tags:        []string{"Teams", "Members"},
		Protected:   true,
		Responses: map[string]*models.ResponseMeta{
			"200": {Description: "Member removed from team successfully", Schema: models.SchemaSuccess},
			"400": {Description: "Invalid request", Schema: models.SchemaError},
			"401": {Description: "Not authenticated", Schema: models.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: models.SchemaError},
			"404": {Description: "Team not found", Schema: models.SchemaError},
		},
	})
}

// Dependencies returns plugin dependencies
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}

// RequiresTables returns required tables
func (p *Plugin) RequiresTables() []string {
	return []string{"auth.user", "auth.organizations", "auth.user_organizations", "auth.teams", "auth.team_members"}
}

// ProvidesAuthMethods returns auth methods
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{}
}
