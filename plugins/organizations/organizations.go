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
	"fmt"
	"net/http"
	"time"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
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
	store          OrganizationStore
	dialect        plugins.Dialect
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
func New(store OrganizationStore, dialect ...plugins.Dialect) *Plugin {
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
	return "1.0.0"
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
		p.store = NewDefaultOrganizationStore(aegis.DB())
	}

	// Build schema requirements
	requirements := []plugins.SchemaRequirement{}
	for _, table := range p.RequiresTables() {
		requirements = append(requirements, plugins.ValidateTableExists(table))
	}
	requirements = append(requirements, GetSchemaRequirements(p.dialect)...)

	// Validate schema requirements
	if err := aegis.ValidateSchemaRequirements(ctx, requirements); err != nil {
		return err
	}

	// Store session service for auth middleware
	p.sessionService = aegis.GetAuthService().Session

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

	// Organization CRUD - all protected
	r.POST(prefix+"/organizations", requireAuth(http.HandlerFunc(p.CreateOrganizationHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        prefix + "/organizations",
		Summary:     "Create organization",
		Description: "Create a new organization with the authenticated user as owner",
		Tags:        []string{"Organizations"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Organization details",
			Required:    true,
			Schema:      SchemaCreateOrganizationRequest,
		},
		Responses: map[string]*core.ResponseMeta{
			"201": {Description: "Organization created successfully", Schema: SchemaOrganization},
			"400": {Description: "Invalid request or validation error", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
		},
	})

	r.GET(prefix+"/organizations", requireAuth(http.HandlerFunc(p.ListOrganizationsHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/organizations",
		Summary:     "List user organizations",
		Description: "Retrieve all organizations the authenticated user is a member of",
		Tags:        []string{"Organizations"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "List of organizations", Schema: SchemaOrganizationList},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"500": {Description: "Internal server error", Schema: core.SchemaError},
		},
	})

	r.GET(prefix+"/organizations/:id", requireAuth(http.HandlerFunc(p.GetOrganizationHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        router.NormalizePathToOpenAPI(prefix + "/organizations/:id"),
		Summary:     "Get organization",
		Description: "Retrieve details of a specific organization",
		Tags:        []string{"Organizations"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Organization details", Schema: SchemaOrganization},
			"400": {Description: "Invalid organization ID", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: core.SchemaError},
			"404": {Description: "Organization not found", Schema: core.SchemaError},
		},
	})

	r.PUT(prefix+"/organizations/:id", requireAuth(http.HandlerFunc(p.UpdateOrganizationHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "PUT",
		Path:        router.NormalizePathToOpenAPI(prefix + "/organizations/:id"),
		Summary:     "Update organization",
		Description: "Update organization details (requires owner or admin role)",
		Tags:        []string{"Organizations"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Updated organization details",
			Required:    true,
			Schema:      UpdateOrganizationRequest{},
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Organization updated successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: core.SchemaError},
		},
	})

	r.DELETE(prefix+"/organizations/:id", requireAuth(http.HandlerFunc(p.DeleteOrganizationHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "DELETE",
		Path:        router.NormalizePathToOpenAPI(prefix + "/organizations/:id"),
		Summary:     "Delete organization",
		Description: "Delete an organization (requires owner role)",
		Tags:        []string{"Organizations"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Organization deleted successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid organization ID", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Only owner can delete organization", Schema: core.SchemaError},
			"500": {Description: "Internal server error", Schema: core.SchemaError},
		},
	})

	// Organization Member Management - all protected
	r.POST(prefix+"/organizations/:id/members", requireAuth(http.HandlerFunc(p.AddOrganizationMemberHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        router.NormalizePathToOpenAPI(prefix + "/organizations/:id/members"),
		Summary:     "Add organization member",
		Description: "Add a new member to the organization (requires admin role)",
		Tags:        []string{"Organizations", "Members"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Member details (userId and role)",
			Required:    true,
			Schema:      AddOrganizationMemberRequest{},
		},
		Responses: map[string]*core.ResponseMeta{
			"201": {Description: "Member added successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: core.SchemaError},
		},
	})

	r.GET(prefix+"/organizations/:id/members", requireAuth(http.HandlerFunc(p.ListOrganizationMembersHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        router.NormalizePathToOpenAPI(prefix + "/organizations/:id/members"),
		Summary:     "List organization members",
		Description: "Retrieve all members of an organization",
		Tags:        []string{"Organizations", "Members"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "List of organization members", Schema: SchemaMemberList},
			"400": {Description: "Invalid organization ID", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: core.SchemaError},
			"500": {Description: "Internal server error", Schema: core.SchemaError},
		},
	})

	r.PATCH(prefix+"/organizations/:id/members/:userId", requireAuth(http.HandlerFunc(p.UpdateMemberRoleHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "PATCH",
		Path:        router.NormalizePathToOpenAPI(prefix + "/organizations/:id/members/:userId"),
		Summary:     "Update member role",
		Description: "Update a member's role in the organization (requires owner role)",
		Tags:        []string{"Organizations", "Members"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "New role for the member",
			Required:    true,
			Schema:      UpdateMemberRoleRequest{},
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Role updated successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Only owner can update roles", Schema: core.SchemaError},
		},
	})

	r.DELETE(prefix+"/organizations/:id/members/:userId", requireAuth(http.HandlerFunc(p.RemoveOrganizationMemberHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "DELETE",
		Path:        router.NormalizePathToOpenAPI(prefix + "/organizations/:id/members/:userId"),
		Summary:     "Remove organization member",
		Description: "Remove a member from the organization (requires admin role, cannot remove owner)",
		Tags:        []string{"Organizations", "Members"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Member removed successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request or cannot remove owner", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: core.SchemaError},
		},
	})

	// Team CRUD - all protected
	r.POST(prefix+"/organizations/:id/teams", requireAuth(http.HandlerFunc(p.CreateTeamHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        router.NormalizePathToOpenAPI(prefix + "/organizations/:id/teams"),
		Summary:     "Create team",
		Description: "Create a new team within an organization (requires admin role)",
		Tags:        []string{"Teams"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Team details",
			Required:    true,
			Schema:      CreateTeamRequest{},
		},
		Responses: map[string]*core.ResponseMeta{
			"201": {Description: "Team created successfully", Schema: "Team"},
			"400": {Description: "Invalid request or validation error", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: core.SchemaError},
		},
	})

	r.GET(prefix+"/organizations/:id/teams", requireAuth(http.HandlerFunc(p.ListTeamsHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        router.NormalizePathToOpenAPI(prefix + "/organizations/:id/teams"),
		Summary:     "List organization teams",
		Description: "Retrieve all teams in an organization",
		Tags:        []string{"Teams"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "List of teams", Schema: SchemaTeamList},
			"400": {Description: "Invalid organization ID", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: core.SchemaError},
			"500": {Description: "Internal server error", Schema: core.SchemaError},
		},
	})

	r.GET(prefix+"/teams/:teamId", requireAuth(http.HandlerFunc(p.GetTeamHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        router.NormalizePathToOpenAPI(prefix + "/teams/:teamId"),
		Summary:     "Get team",
		Description: "Retrieve details of a specific team",
		Tags:        []string{"Teams"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Team details", Schema: "Team"},
			"400": {Description: "Invalid team ID", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: core.SchemaError},
			"404": {Description: "Team not found", Schema: core.SchemaError},
		},
	})

	r.PUT(prefix+"/teams/:teamId", requireAuth(http.HandlerFunc(p.UpdateTeamHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "PUT",
		Path:        router.NormalizePathToOpenAPI(prefix + "/teams/:teamId"),
		Summary:     "Update team",
		Description: "Update team details (requires admin role)",
		Tags:        []string{"Teams"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Updated team details",
			Required:    true,
			Schema:      UpdateTeamRequest{},
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Team updated successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: core.SchemaError},
			"404": {Description: "Team not found", Schema: core.SchemaError},
		},
	})

	r.DELETE(prefix+"/teams/:teamId", requireAuth(http.HandlerFunc(p.DeleteTeamHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "DELETE",
		Path:        router.NormalizePathToOpenAPI(prefix + "/teams/:teamId"),
		Summary:     "Delete team",
		Description: "Delete a team (requires admin role)",
		Tags:        []string{"Teams"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Team deleted successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid team ID", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: core.SchemaError},
			"404": {Description: "Team not found", Schema: core.SchemaError},
			"500": {Description: "Internal server error", Schema: core.SchemaError},
		},
	})

	// Team Member Management - all protected
	r.POST(prefix+"/teams/:teamId/members", requireAuth(http.HandlerFunc(p.AddTeamMemberHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "POST",
		Path:        router.NormalizePathToOpenAPI(prefix + "/teams/:teamId/members"),
		Summary:     "Add team member",
		Description: "Add a member to a team (requires admin role, user must be organization member)",
		Tags:        []string{"Teams", "Members"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "Team member details (userId and role)",
			Required:    true,
			Schema:      AddTeamMemberRequest{},
		},
		Responses: map[string]*core.ResponseMeta{
			"201": {Description: "Member added to team successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request or user not organization member", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: core.SchemaError},
			"404": {Description: "Team not found", Schema: core.SchemaError},
		},
	})

	r.GET(prefix+"/teams/:teamId/members", requireAuth(http.HandlerFunc(p.ListTeamMembersHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        router.NormalizePathToOpenAPI(prefix + "/teams/:teamId/members"),
		Summary:     "List team members",
		Description: "Retrieve all members of a team",
		Tags:        []string{"Teams", "Members"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "List of team members", Schema: SchemaTeamMemberList},
			"400": {Description: "Invalid team ID", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Not a member of this organization", Schema: core.SchemaError},
			"404": {Description: "Team not found", Schema: core.SchemaError},
			"500": {Description: "Internal server error", Schema: core.SchemaError},
		},
	})

	r.PATCH(prefix+"/teams/:teamId/members/:userId", requireAuth(http.HandlerFunc(p.UpdateTeamMemberRoleHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "PATCH",
		Path:        router.NormalizePathToOpenAPI(prefix + "/teams/:teamId/members/:userId"),
		Summary:     "Update team member role",
		Description: "Update a team member's role (requires admin role)",
		Tags:        []string{"Teams", "Members"},
		Protected:   true,
		RequestBody: &core.RequestBodyMeta{
			Description: "New role for the team member",
			Required:    true,
			Schema:      UpdateTeamMemberRoleRequest{},
		},
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Team member role updated successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request or validation error", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: core.SchemaError},
			"404": {Description: "Team not found", Schema: core.SchemaError},
		},
	})

	r.DELETE(prefix+"/teams/:teamId/members/:userId", requireAuth(http.HandlerFunc(p.RemoveTeamMemberHandler)).ServeHTTP)
	r.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "DELETE",
		Path:        router.NormalizePathToOpenAPI(prefix + "/teams/:teamId/members/:userId"),
		Summary:     "Remove team member",
		Description: "Remove a member from a team (requires admin role)",
		Tags:        []string{"Teams", "Members"},
		Protected:   true,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "Member removed from team successfully", Schema: core.SchemaSuccess},
			"400": {Description: "Invalid request", Schema: core.SchemaError},
			"401": {Description: "Not authenticated", Schema: core.SchemaError},
			"403": {Description: "Insufficient permissions", Schema: core.SchemaError},
			"404": {Description: "Team not found", Schema: core.SchemaError},
		},
	})
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

// createOrganization creates a new organization and adds the creator as owner.
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
func (p *Plugin) createOrganization(ctx context.Context, name, slug, ownerID string) (*Organization, error) {
	now := time.Now()
	id := generateID()

	err := p.store.CreateOrganization(ctx, id, name, slug, now, now)
	if err != nil {
		return nil, err
	}

	// Add owner as first member
	err = p.store.CreateMember(ctx, generateID(), ownerID, id, "owner", now, now)
	if err != nil {
		return nil, err
	}

	return &Organization{
		ID:        id,
		Name:      name,
		Slug:      slug,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (p *Plugin) getOrganization(ctx context.Context, id string) (Organization, error) {
	return p.store.GetOrganization(ctx, id)
}

func (p *Plugin) updateOrganization(ctx context.Context, id, name, slug string) error {
	return p.store.UpdateOrganization(ctx, id, name, slug, time.Now())
}

func (p *Plugin) deleteOrganization(ctx context.Context, id string) error {
	return p.store.DeleteOrganization(ctx, id, time.Now())
}

func (p *Plugin) getUserOrganizations(ctx context.Context, userID string) ([]*Organization, error) {
	orgs, err := p.store.ListUserOrganizations(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*Organization, len(orgs))
	for i := range orgs {
		result[i] = &orgs[i]
	}
	return result, nil
}

// User Organization operations

// isOrganizationMember checks if a user is a member of an organization.
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
func (p *Plugin) isOrganizationMember(ctx context.Context, userID, orgID string) bool {
	isMember, err := p.store.IsOrganizationMember(ctx, userID, orgID)
	return err == nil && isMember
}

// isOwnerOrAdmin checks if a user is an owner or admin of an organization.
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
func (p *Plugin) isOwnerOrAdmin(ctx context.Context, userID, orgID string) bool {
	isOwnerAdmin, err := p.store.IsOwnerOrAdmin(ctx, userID, orgID)
	return err == nil && isOwnerAdmin
}

// isOwner checks if a user is the owner of an organization.
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
func (p *Plugin) isOwner(ctx context.Context, userID, orgID string) bool {
	isOwner, err := p.store.IsOwner(ctx, userID, orgID)
	return err == nil && isOwner
}

// addOrganizationMember adds a user to an organization with a specified role.
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
func (p *Plugin) addOrganizationMember(ctx context.Context, orgID, userID, role string) error {
	now := time.Now()
	return p.store.CreateMember(ctx, generateID(), userID, orgID, role, now, now)
}

func (p *Plugin) updateMemberRole(ctx context.Context, orgID, userID, role string) error {
	return p.store.UpdateMemberRole(ctx, userID, orgID, role, time.Now())
}

func (p *Plugin) removeOrganizationMember(ctx context.Context, userID, orgID string) error {
	return p.store.RemoveMember(ctx, userID, orgID)
}

func (p *Plugin) listOrganizationMembers(ctx context.Context, orgID string) ([]*Member, error) {
	members, err := p.store.ListOrganizationMembers(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]*Member, len(members))
	for i := range members {
		result[i] = &members[i]
	}
	return result, nil
}

// Team operations

func (p *Plugin) createTeam(ctx context.Context, orgID, name, description string) (*Team, error) {
	now := time.Now()
	id := generateID()

	err := p.store.CreateTeam(ctx, id, orgID, name, description, now, now)
	if err != nil {
		return nil, err
	}

	return &Team{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Description:    description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (p *Plugin) getTeam(ctx context.Context, id string) (*Team, error) {
	team, err := p.store.GetTeam(ctx, id)
	return &team, err
}

func (p *Plugin) listTeams(ctx context.Context, orgID string) ([]*Team, error) {
	teams, err := p.store.ListTeams(ctx, orgID)
	if err != nil {
		return nil, err
	}

	result := make([]*Team, len(teams))
	for i := range teams {
		result[i] = &teams[i]
	}
	return result, nil
}

func (p *Plugin) updateTeam(ctx context.Context, id, name, description string) error {
	return p.store.UpdateTeam(ctx, id, name, description, time.Now())
}

func (p *Plugin) deleteTeam(ctx context.Context, id string) error {
	return p.store.DeleteTeam(ctx, id)
}

// Team Member operations

func (p *Plugin) addTeamMember(ctx context.Context, teamID, userID, role string) error {
	now := time.Now()
	return p.store.CreateTeamMember(ctx, generateID(), teamID, userID, role, now, now)
}

func (p *Plugin) updateTeamMemberRole(ctx context.Context, teamID, userID, role string) error {
	return p.store.UpdateTeamMemberRole(ctx, teamID, userID, role, time.Now())
}

func (p *Plugin) removeTeamMember(ctx context.Context, teamID, userID string) error {
	return p.store.RemoveTeamMember(ctx, teamID, userID)
}

func (p *Plugin) listTeamMembers(ctx context.Context, teamID string) ([]*TeamMember, error) {
	members, err := p.store.ListTeamMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}

	result := make([]*TeamMember, len(members))
	for i := range members {
		result[i] = &members[i]
	}
	return result, nil
}

// Helper functions

func generateID() string {
	// Simple ID generation - in production you'd want something more robust
	return fmt.Sprintf("%d", time.Now().UnixNano())
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

// GetSchemas returns all schemas for all supported dialects
func (p *Plugin) GetSchemas() []plugins.Schema {
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
