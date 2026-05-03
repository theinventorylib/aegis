// Package organizations provides multi-tenancy and team management for Aegis.
//
// This plugin enables SaaS applications to manage multiple organizations (workspaces,
// companies, tenants) with member roles and team hierarchies. It implements a complete
// RBAC (Role-Based Access Control) system for organizational resources.
//
// Multi-Tenancy Architecture:
//   - Organization: Top-level tenant (e.g., "Acme Corp", "Tech Startup")
//   - Members: Users with roles in an organization (owner, admin, member)
//   - Teams: Groups within an organization (e.g., "Engineering", "Sales")
//   - Team Members: Users with roles in a team (lead, member)
//
// Role Hierarchy:
//
//	Organization Roles:
//	  - owner: Full control, can delete organization, manage all members
//	  - admin: Can manage members, teams, but cannot delete organization
//	  - member: Read access to organization resources
//
//	Team Roles:
//	  - lead: Can manage team members and settings
//	  - member: Participate in team activities
//
// Common Use Cases:
//   - SaaS with company workspaces (Slack, Notion, GitHub)
//   - Project management tools with teams
//   - Enterprise apps with department hierarchies
//   - Multi-tenant platforms with access control
//
// Database Schema:
//   - organization: Stores organization metadata (id, name, slug)
//   - members: Links users to organizations with roles
//   - team: Stores team metadata within organizations
//   - team_member: Links users to teams with roles
//
// Example Setup:
//
//	// Create organization plugin
//	orgPlugin := organizations.New(nil, plugins.DialectPostgres)
//
//	// User creates organization
//	org, _ := orgPlugin.CreateOrganization(ctx, "Acme Corp", "acme", user.ID)
//	// User is automatically added as owner
//
//	// Owner adds admin
//	orgPlugin.AddOrganizationMember(ctx, org.ID, adminUserID, "admin")
//
//	// Admin creates team
//	team, _ := orgPlugin.CreateTeam(ctx, org.ID, "Engineering", "Dev team")
//
//	// Admin adds team member
//	orgPlugin.AddTeamMember(ctx, team.ID, devUserID, "member")
//
// Security Features:
//   - All routes require authentication (RequireAuthMiddleware)
//   - Role-based middleware (RequireOrganizationMember, RequireOrganizationAdmin, RequireOrganizationOwner)
//   - Foreign key constraints prevent orphaned records
//   - Cascade deletes when organization is deleted
package organizations

import (
	"context"
	"net/http"
	"time"

	"github.com/theinventorylib/aegis/core"
	iversion "github.com/theinventorylib/aegis/internal/version"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/plugins/openapi"
	orgdefaultstore "github.com/theinventorylib/aegis/plugins/organizations/default_store"
	orgtypes "github.com/theinventorylib/aegis/plugins/organizations/types"
	"github.com/theinventorylib/aegis/router"
)

// Plugin implements multi-tenant organization and team management.
//
// This plugin provides complete CRUD operations for organizations, members,
// teams, and team members with role-based access control.
//
// Components:
//   - sessionService: User authentication for protected routes
//   - store: Database persistence for organizations, members, teams
//   - dialect: SQL dialect (PostgreSQL, MySQL, SQLite)
//
// Endpoints Provided:
//
//	Organizations: POST, GET, PUT, DELETE /organizations
//	Members: POST, GET, PATCH, DELETE /organizations/:id/members
//	Teams: POST, GET, PUT, DELETE /teams, /organizations/:id/teams
//	Team Members: POST, GET, PATCH, DELETE /teams/:teamId/members
type Plugin struct {
	sessionService *core.SessionService
	store          orgtypes.OrganizationStore
	dialect        plugins.Dialect
	aegis          plugins.Aegis
}

// New creates a new organizations plugin for multi-tenancy management.
//
// Parameters:
//   - store: Organization storage implementation (nil = use DefaultOrganizationStore)
//   - dialect: Database dialect (defaults to PostgreSQL)
//
// Returns:
//   - *Plugin: Initialized plugin ready for Init() call
//
// Example:
//
//	plugin := organizations.New(nil, plugins.DialectPostgres)
func New(store orgtypes.OrganizationStore, dialect ...plugins.Dialect) *Plugin {
	d := plugins.DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}
	return &Plugin{
		store:   store,
		dialect: d,
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "organizations"
}

// Version returns the plugin version
func (p *Plugin) Version() string {
	return iversion.Version
}

// Description returns the plugin description
func (p *Plugin) Description() string {
	return "Organization and team management plugin"
}

// Init initializes the organizations plugin with Aegis services.
//
// This method validates database schema requirements and stores the session
// service for authentication middleware.
//
// Initialization Steps:
//  1. Initialize store if not provided (DefaultOrganizationStore)
//  2. Build schema validation requirements (tables, foreign keys)
//  3. Validate schema via Aegis
//  4. Store session service for protected routes
//
// Required Tables:
//   - organization: Organization metadata
//   - members: Organization membership with roles
//   - team: Team metadata within organizations
//   - team_member: Team membership with roles
//
// Parameters:
//   - ctx: Initialization context
//   - aegis: Aegis interface providing services and DB
//
// Returns:
//   - error: Schema validation error if tables don't exist
func (p *Plugin) Init(ctx context.Context, aegis plugins.Aegis) error {
	// Initialize store if not provided
	if p.store == nil {
		store, err := orgdefaultstore.NewDefaultOrganizationStore(aegis.DB(), p.dialect)
		if err != nil {
			return err
		}
		p.store = store
	}

	// Build schema requirements
	tables := p.RequiresTables()
	requirements := make([]plugins.SchemaRequirement, 0, len(tables))
	for _, table := range tables {
		requirements = append(requirements, plugins.ValidateTableExists(table))
	}
	requirements = append(requirements, GetSchemaRequirements(p.dialect)...)

	// Validate schema requirements
	if err := aegis.ValidateSchemaRequirements(ctx, requirements); err != nil {
		return err
	}

	// Store session service for auth middleware
	p.sessionService = aegis.GetAuthService().Session
	p.aegis = aegis

	return nil
}

// GetMigrations returns the plugin migrations
func (p *Plugin) GetMigrations() []plugins.Migration {
	migs, err := GetMigrations(p.dialect)
	if err != nil {
		return []plugins.Migration{}
	}
	return migs
}

// MountRoutes registers HTTP routes for the organizations plugin
func (p *Plugin) MountRoutes(r router.Router, prefix string) {
	// Create auth middleware - ALL organization routes require authentication
	requireAuth := core.RequireAuthMiddleware(p.sessionService)

	// Organization routes grouped under plugin prefix
	orgGroup := r.Group(prefix, "Organizations")

	// Create organization (POST to prefix)
	orgGroup.POST("", requireAuth(http.HandlerFunc(p.CreateOrganizationHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix,
		Summary:     "Create organization",
		Description: "Create a new organization with the authenticated user as owner",
		Tags:        []string{"Organizations"},
		Auth:        true,
		Body:        openapi.BodyOf[CreateOrganizationRequest](),
		Responses: openapi.Responses{
			201: openapi.DataResponseOf[orgtypes.Organization]("Organization created successfully"),
			400: openapi.RefResponse("Invalid request or validation error", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
		},
	})

	// List organizations (GET to prefix)
	orgGroup.GET("", requireAuth(http.HandlerFunc(p.ListOrganizationsHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix,
		Summary:     "List user organizations",
		Description: "Retrieve all organizations the authenticated user is a member of",
		Tags:        []string{"Organizations"},
		Auth:        true,
		Responses: openapi.Responses{
			200: openapi.PaginatedResponseOf[core.PaginatedResponse[orgtypes.Organization]]("List of organizations"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			500: openapi.RefResponse("Internal server error", "Error"),
		},
	})

	// Organization detail routes
	orgGroup.GET("/:id", requireAuth(http.HandlerFunc(p.GetOrganizationHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/{id}",
		Summary:     "Get organization",
		Description: "Retrieve details of a specific organization",
		Tags:        []string{"Organizations"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[orgtypes.Organization]("Organization details"),
			400: openapi.RefResponse("Invalid organization ID", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Not a member of this organization", "Error"),
			404: openapi.RefResponse("Organization not found", "Error"),
		},
	})

	orgGroup.PUT("/:id", requireAuth(http.HandlerFunc(p.UpdateOrganizationHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "PUT",
		Path:        prefix + "/{id}",
		Summary:     "Update organization",
		Description: "Update organization details (requires owner or admin role)",
		Tags:        []string{"Organizations"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Body: openapi.BodyOf[UpdateOrganizationRequest](),
		Responses: openapi.Responses{
			200: openapi.RefResponse("Organization updated successfully", "Success"),
			400: openapi.RefResponse("Invalid request or validation error", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
		},
	})

	orgGroup.DELETE("/:id", requireAuth(http.HandlerFunc(p.DeleteOrganizationHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "DELETE",
		Path:        prefix + "/{id}",
		Summary:     "Delete organization",
		Description: "Delete an organization (requires owner role)",
		Tags:        []string{"Organizations"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.RefResponse("Organization deleted successfully", "Success"),
			400: openapi.RefResponse("Invalid organization ID", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Only owner can delete organization", "Error"),
			500: openapi.RefResponse("Internal server error", "Error"),
		},
	})

	// Organization Member Management - group under orgGroup
	membersGroup := orgGroup.Group("/:id/members", "Members")

	membersGroup.POST("", requireAuth(http.HandlerFunc(p.AddOrganizationMemberHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/{id}/members",
		Summary:     "Add organization member",
		Description: "Add a new member to the organization (requires admin role)",
		Tags:        []string{"Members"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Body: openapi.BodyOf[AddOrganizationMemberRequest](),
		Responses: openapi.Responses{
			201: openapi.RefResponse("Member added successfully", "Success"),
			400: openapi.RefResponse("Invalid request or validation error", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
		},
	})

	membersGroup.GET("", requireAuth(http.HandlerFunc(p.ListOrganizationMembersHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/{id}/members",
		Summary:     "List organization members",
		Description: "Retrieve all members of an organization",
		Tags:        []string{"Members"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.PaginatedResponseOf[core.PaginatedResponse[orgtypes.Member]]("List of organization members"),
			400: openapi.RefResponse("Invalid organization ID", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Not a member of this organization", "Error"),
			500: openapi.RefResponse("Internal server error", "Error"),
		},
	})

	membersGroup.PATCH("/:userId", requireAuth(http.HandlerFunc(p.UpdateMemberRoleHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "PATCH",
		Path:        prefix + "/{id}/members/{userId}",
		Summary:     "Update member role",
		Description: "Update a member's role in the organization (requires owner role)",
		Tags:        []string{"Members"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
			{Name: "userId", In: "path", Type: "string", Required: true},
		},
		Body: openapi.BodyOf[UpdateMemberRoleRequest](),
		Responses: openapi.Responses{
			200: openapi.RefResponse("Role updated successfully", "Success"),
			400: openapi.RefResponse("Invalid request or validation error", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Only owner can update roles", "Error"),
		},
	})

	membersGroup.DELETE("/:userId", requireAuth(http.HandlerFunc(p.RemoveOrganizationMemberHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "DELETE",
		Path:        prefix + "/{id}/members/{userId}",
		Summary:     "Remove organization member",
		Description: "Remove a member from the organization (requires admin role, cannot remove owner)",
		Tags:        []string{"Members"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
			{Name: "userId", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.RefResponse("Member removed successfully", "Success"),
			400: openapi.RefResponse("Invalid request or cannot remove owner", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
		},
	})

	// Organization-specific teams under orgGroup
	orgTeams := orgGroup.Group("/:id/teams", "Teams")

	orgTeams.POST("", requireAuth(http.HandlerFunc(p.CreateTeamHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/{id}/teams",
		Summary:     "Create team",
		Description: "Create a new team within an organization (requires admin role)",
		Tags:        []string{"Teams"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Body: openapi.BodyOf[CreateTeamRequest](),
		Responses: openapi.Responses{
			201: openapi.DataResponseOf[orgtypes.Team]("Team created successfully"),
			400: openapi.RefResponse("Invalid request or validation error", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
		},
	})

	orgTeams.GET("", requireAuth(http.HandlerFunc(p.ListTeamsHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/{id}/teams",
		Summary:     "List organization teams",
		Description: "Retrieve all teams in an organization",
		Tags:        []string{"Teams"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.PaginatedResponseOf[core.PaginatedResponse[orgtypes.Team]]("List of teams"),
			400: openapi.RefResponse("Invalid organization ID", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Not a member of this organization", "Error"),
			500: openapi.RefResponse("Internal server error", "Error"),
		},
	})

	// Team operations at /teams/:teamId under plugin prefix
	teamsGroup := orgGroup.Group("/teams", "Teams")

	teamsGroup.GET(":/teamId", requireAuth(http.HandlerFunc(p.GetTeamHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/teams/{teamId}",
		Summary:     "Get team",
		Description: "Retrieve details of a specific team",
		Tags:        []string{"Teams"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "teamId", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[orgtypes.Team]("Team details"),
			400: openapi.RefResponse("Invalid team ID", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Not a member of this organization", "Error"),
			404: openapi.RefResponse("Team not found", "Error"),
		},
	})

	teamsGroup.PUT("/:teamId", requireAuth(http.HandlerFunc(p.UpdateTeamHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "PUT",
		Path:        prefix + "/teams/{teamId}",
		Summary:     "Update team",
		Description: "Update team details (requires admin role)",
		Tags:        []string{"Teams"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "teamId", In: "path", Type: "string", Required: true},
		},
		Body: openapi.BodyOf[UpdateTeamRequest](),
		Responses: openapi.Responses{
			200: openapi.RefResponse("Team updated successfully", "Success"),
			400: openapi.RefResponse("Invalid request or validation error", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
			404: openapi.RefResponse("Team not found", "Error"),
		},
	})

	teamsGroup.DELETE("/:teamId", requireAuth(http.HandlerFunc(p.DeleteTeamHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "DELETE",
		Path:        prefix + "/teams/{teamId}",
		Summary:     "Delete team",
		Description: "Delete a team (requires admin role)",
		Tags:        []string{"Teams"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "teamId", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.RefResponse("Team deleted successfully", "Success"),
			400: openapi.RefResponse("Invalid team ID", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
			404: openapi.RefResponse("Team not found", "Error"),
			500: openapi.RefResponse("Internal server error", "Error"),
		},
	})

	// Team Member Management - all protected
	r.POST(prefix+"/teams/:teamId/members", requireAuth(http.HandlerFunc(p.AddTeamMemberHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/teams/{teamId}/members",
		Summary:     "Add team member",
		Description: "Add a member to a team (requires admin role, user must be organization member)",
		Tags:        []string{"Team Members"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "teamId", In: "path", Type: "string", Required: true},
		},
		Body: openapi.BodyOf[AddTeamMemberRequest](),
		Responses: openapi.Responses{
			201: openapi.RefResponse("Member added to team successfully", "Success"),
			400: openapi.RefResponse("Invalid request or user not organization member", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
			404: openapi.RefResponse("Team not found", "Error"),
		},
	})

	r.GET(prefix+"/teams/:teamId/members", requireAuth(http.HandlerFunc(p.ListTeamMembersHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/teams/{teamId}/members",
		Summary:     "List team members",
		Description: "Retrieve all members of a team",
		Tags:        []string{"Team Members"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "teamId", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.PaginatedResponseOf[core.PaginatedResponse[orgtypes.TeamMember]]("List of team members"),
			400: openapi.RefResponse("Invalid team ID", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Not a member of this organization", "Error"),
			404: openapi.RefResponse("Team not found", "Error"),
			500: openapi.RefResponse("Internal server error", "Error"),
		},
	})

	r.PATCH(prefix+"/teams/:teamId/members/:userId", requireAuth(http.HandlerFunc(p.UpdateTeamMemberRoleHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "PATCH",
		Path:        prefix + "/teams/{teamId}/members/{userId}",
		Summary:     "Update team member role",
		Description: "Update a team member's role (requires admin role)",
		Tags:        []string{"Team Members"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "teamId", In: "path", Type: "string", Required: true},
			{Name: "userId", In: "path", Type: "string", Required: true},
		},
		Body: openapi.BodyOf[UpdateTeamMemberRoleRequest](),
		Responses: openapi.Responses{
			200: openapi.RefResponse("Team member role updated successfully", "Success"),
			400: openapi.RefResponse("Invalid request or validation error", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
			404: openapi.RefResponse("Team not found", "Error"),
		},
	})

	r.DELETE(prefix+"/teams/:teamId/members/:userId", requireAuth(http.HandlerFunc(p.RemoveTeamMemberHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "DELETE",
		Path:        prefix + "/teams/{teamId}/members/{userId}",
		Summary:     "Remove team member",
		Description: "Remove a member from a team (requires admin role)",
		Tags:        []string{"Team Members"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "teamId", In: "path", Type: "string", Required: true},
			{Name: "userId", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.RefResponse("Member removed from team successfully", "Success"),
			400: openapi.RefResponse("Invalid request", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
			404: openapi.RefResponse("Team not found", "Error"),
		},
	})
}

// EnrichUser implements plugins.UserEnricher to add organization memberships.
//
// This method is called automatically by the authentication system after user lookup.
// It adds the user's organization memberships to the EnrichedUser, making them
// available in API responses without requiring separate queries.
//
// Fields Added:
//   - "organizations" ([]map[string]any): List of organizations the user belongs to,
//     each containing id, name, and slug fields.
//
// Parameters:
//   - ctx: Request context
//   - user: EnrichedUser to populate with organization data
//
// Returns:
//   - error: Always nil (organization lookup failure is not an error)
func (p *Plugin) EnrichUser(ctx context.Context, user *core.EnrichedUser) error {
	if user == nil || user.User == nil {
		return nil
	}

	orgs, _, err := p.GetUserOrganizations(ctx, user.ID, 0, 50)
	if err != nil {
		// Don't fail enrichment if lookup fails
		return err
	}

	// Convert to simple list for API response
	orgList := make([]map[string]any, len(orgs))
	for i, org := range orgs {
		orgList[i] = map[string]any{
			"id":   org.ID,
			"name": org.Name,
			"slug": org.Slug,
		}
	}

	user.Set("organizations", orgList)
	return nil
}

// ========== BUSINESS LOGIC METHODS ==========
//
// These methods implement the core organization management logic, separated
// from HTTP handlers for testability and reusability.
//
// Organization Lifecycle:
//  1. createOrganization: Create org + add creator as owner
//  2. User invites members via addOrganizationMember
//  3. Admin creates teams via createTeam
//  4. Admin adds team members via addTeamMember
//  5. Owner can deleteOrganization (cascade deletes members, teams)

// Organization operations

// CreateOrganization creates a new organization and adds the creator as owner.
//
// This method performs two database operations atomically:
//  1. Create organization record
//  2. Create member record with role="owner" for creator
//
// Parameters:
//   - ctx: Request context
//   - name: Organization display name (e.g., "Acme Corporation")
//   - slug: URL-friendly identifier (e.g., "acme-corp")
//   - ownerID: User ID of the organization creator
//
// Returns:
//   - *Organization: Created organization with metadata
//   - error: Database error or duplicate slug error
func (p *Plugin) CreateOrganization(ctx context.Context, name, slug, ownerID string) (*orgtypes.Organization, error) {
	// Sanitize inputs
	name = core.SanitizeString(name, nil)
	slug = core.SanitizeUsername(slug, 50) // Slugs follow username-like rules

	now := time.Now()
	id := core.GenerateID()

	err := p.store.CreateOrganization(ctx, id, name, slug, now, now)
	if err != nil {
		return nil, err
	}

	// Add owner as first member
	err = p.store.CreateMember(ctx, core.GenerateID(), ownerID, id, "owner", now, now)
	if err != nil {
		return nil, err
	}

	return &orgtypes.Organization{
		ID:        id,
		Name:      name,
		Slug:      slug,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetOrganization retrieves an organization by ID.
func (p *Plugin) GetOrganization(ctx context.Context, id string) (orgtypes.Organization, error) {
	return p.store.GetOrganization(ctx, id)
}

// UpdateOrganization updates an organization's name and slug.
func (p *Plugin) UpdateOrganization(ctx context.Context, id, name, slug string) error {
	// Sanitize inputs
	name = core.SanitizeString(name, nil)
	slug = core.SanitizeUsername(slug, 50)

	return p.store.UpdateOrganization(ctx, id, name, slug, time.Now())
}

// DeleteOrganization soft-deletes an organization.
func (p *Plugin) DeleteOrganization(ctx context.Context, id string) error {
	return p.store.DeleteOrganization(ctx, id, time.Now())
}

// GetUserOrganizations retrieves all organizations for a user.
func (p *Plugin) GetUserOrganizations(ctx context.Context, userID string, offset, limit int) ([]*orgtypes.Organization, int, error) {
	orgs, err := p.store.ListUserOrganizations(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := p.store.CountUserOrganizations(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*orgtypes.Organization, len(orgs))
	for i := range orgs {
		result[i] = &orgs[i]
	}
	return result, count, nil
}

// User Organization operations

// IsOrganizationMember checks if a user is a member of an organization.
//
// This method is used by middleware to enforce organization access control.
// Returns true only if the user has any role (owner, admin, or member).
//
// Parameters:
//   - ctx: Request context
//   - userID: User ID to check
//   - orgID: Organization ID
//
// Returns:
//   - bool: true if user is a member with any role
func (p *Plugin) IsOrganizationMember(ctx context.Context, userID, orgID string) bool {
	isMember, err := p.store.IsOrganizationMember(ctx, userID, orgID)
	return err == nil && isMember
}

// IsOwnerOrAdmin checks if a user is an owner or admin of an organization.
//
// This method enforces permission requirements for administrative actions:
//   - Updating organization settings
//   - Adding/removing members
//   - Creating/deleting teams
//
// Parameters:
//   - ctx: Request context
//   - userID: User ID to check
//   - orgID: Organization ID
//
// Returns:
//   - bool: true if user has owner or admin role
func (p *Plugin) IsOwnerOrAdmin(ctx context.Context, userID, orgID string) bool {
	isOwnerAdmin, err := p.store.IsOwnerOrAdmin(ctx, userID, orgID)
	return err == nil && isOwnerAdmin
}

// IsOwner checks if a user is the owner of an organization.
//
// This method enforces permission requirements for destructive actions:
//   - Deleting organization
//   - Transferring ownership
//   - Changing admin roles
//
// Parameters:
//   - ctx: Request context
//   - userID: User ID to check
//   - orgID: Organization ID
//
// Returns:
//   - bool: true if user has owner role
func (p *Plugin) IsOwner(ctx context.Context, userID, orgID string) bool {
	isOwner, err := p.store.IsOwner(ctx, userID, orgID)
	return err == nil && isOwner
}

// AddOrganizationMember adds a user to an organization with a specified role.
//
// This method creates a membership record linking the user to the organization.
// The caller must verify admin/owner permissions before calling this method.
//
// Valid Roles:
//   - "owner": Full control (only one owner per organization recommended)
//   - "admin": Can manage members and teams
//   - "member": Read-only access to organization resources
//
// Parameters:
//   - ctx: Request context
//   - orgID: Organization ID
//   - userID: User ID to add
//   - role: Membership role ("owner", "admin", "member")
//
// Returns:
//   - error: Database error or duplicate membership
func (p *Plugin) AddOrganizationMember(ctx context.Context, orgID, userID, role string) error {
	now := time.Now()
	return p.store.CreateMember(ctx, core.GenerateID(), userID, orgID, role, now, now)
}

// UpdateMemberRole updates a user's role in an organization.
func (p *Plugin) UpdateMemberRole(ctx context.Context, orgID, userID, role string) error {
	return p.store.UpdateMemberRole(ctx, userID, orgID, role, time.Now())
}

// RemoveOrganizationMember removes a user from an organization.
func (p *Plugin) RemoveOrganizationMember(ctx context.Context, userID, orgID string) error {
	return p.store.RemoveMember(ctx, userID, orgID)
}

// ListOrganizationMembers lists all members of an organization.
func (p *Plugin) ListOrganizationMembers(ctx context.Context, orgID string, offset, limit int) ([]*orgtypes.Member, int, error) {
	members, err := p.store.ListOrganizationMembers(ctx, orgID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := p.store.CountOrganizationMembers(ctx, orgID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*orgtypes.Member, len(members))
	for i := range members {
		result[i] = &members[i]
	}
	return result, count, nil
}

// Team operations

// CreateTeam creates a new team within an organization.
func (p *Plugin) CreateTeam(ctx context.Context, orgID, name, description string) (*orgtypes.Team, error) {
	// Sanitize inputs
	name = core.SanitizeString(name, nil)
	description = core.SanitizeMultiline(description, 500)

	now := time.Now()
	id := core.GenerateID()

	err := p.store.CreateTeam(ctx, id, orgID, name, description, now, now)
	if err != nil {
		return nil, err
	}

	return &orgtypes.Team{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Description:    description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

// GetTeam retrieves a team by ID.
func (p *Plugin) GetTeam(ctx context.Context, id string) (*orgtypes.Team, error) {
	team, err := p.store.GetTeam(ctx, id)
	return &team, err
}

// ListTeams lists all teams in an organization.
func (p *Plugin) ListTeams(ctx context.Context, orgID string, offset, limit int) ([]*orgtypes.Team, int, error) {
	teams, err := p.store.ListTeams(ctx, orgID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := p.store.CountTeams(ctx, orgID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*orgtypes.Team, len(teams))
	for i := range teams {
		result[i] = &teams[i]
	}
	return result, count, nil
}

// UpdateTeam updates a team's name and description.
func (p *Plugin) UpdateTeam(ctx context.Context, id, name, description string) error {
	// Sanitize inputs
	name = core.SanitizeString(name, nil)
	description = core.SanitizeMultiline(description, 500)

	return p.store.UpdateTeam(ctx, id, name, description, time.Now())
}

// DeleteTeam deletes a team.
func (p *Plugin) DeleteTeam(ctx context.Context, id string) error {
	return p.store.DeleteTeam(ctx, id)
}

// Team Member operations

// AddTeamMember adds a user to a team with a specified role.
func (p *Plugin) AddTeamMember(ctx context.Context, teamID, userID, role string) error {
	now := time.Now()
	return p.store.CreateTeamMember(ctx, core.GenerateID(), teamID, userID, role, now, now)
}

// UpdateTeamMemberRole updates a user's role in a team.
func (p *Plugin) UpdateTeamMemberRole(ctx context.Context, teamID, userID, role string) error {
	return p.store.UpdateTeamMemberRole(ctx, teamID, userID, role, time.Now())
}

// RemoveTeamMember removes a user from a team.
func (p *Plugin) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	return p.store.RemoveTeamMember(ctx, teamID, userID)
}

// ListTeamMembers lists all members of a team.
func (p *Plugin) ListTeamMembers(ctx context.Context, teamID string, offset, limit int) ([]*orgtypes.TeamMember, int, error) {
	members, err := p.store.ListTeamMembers(ctx, teamID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := p.store.CountTeamMembers(ctx, teamID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*orgtypes.TeamMember, len(members))
	for i := range members {
		result[i] = &members[i]
	}
	return result, count, nil
}

// Dependencies returns plugin dependencies
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}

// RequiresTables returns required tables
func (p *Plugin) RequiresTables() []string {
	return []string{"organization", "members", "team", "team_member"}
}

// ProvidesAuthMethods returns the provided auth methods
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{}
}

// Ensure Plugin implements UserEnricher
var _ plugins.UserEnricher = (*Plugin)(nil)

// Ensure Plugin implements Plugin
var _ plugins.Plugin = (*Plugin)(nil)
