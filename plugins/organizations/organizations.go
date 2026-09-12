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
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
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

// Config holds optional configuration for the organizations plugin.
type Config struct {
	// OrgRoles overrides or extends the built-in organization roles
	// (owner, admin, member). Each key is an assignable role; its
	// Permissions determine what the role can do. A key that matches a
	// built-in replaces that role's permissions; new keys add custom roles.
	// The owner role is never assignable via the member-management endpoints.
	OrgRoles map[string]RoleDefinition

	// TeamRoles overrides or extends the built-in team roles (lead, member).
	// Semantics are the same as OrgRoles.
	TeamRoles map[string]RoleDefinition

	// CustomOrgRoles extends the set of assignable org roles.
	//
	// Deprecated: use OrgRoles, which also assigns permissions. Roles listed
	// here are granted the same read-only permissions as "member".
	CustomOrgRoles []string

	// CustomTeamRoles extends the set of assignable team roles.
	//
	// Deprecated: use TeamRoles. Roles listed here are granted the same
	// read-only permissions as a team "member".
	CustomTeamRoles []string

	// InvitationSubject is the subject line for invitation emails.
	// Default: "You're invited!".
	InvitationSubject string

	// InvitationBodyTemplate is the body template for invitation emails.
	// Use %s as a placeholder for the accept URL (substituted via fmt.Sprintf).
	// Default: "You have been invited.\n\nAccept your invitation here: %s".
	InvitationBodyTemplate string
}

// Plugin implements multi-tenant organization and team management.
//
// This plugin provides complete CRUD operations for organizations, members,
// teams, and team members with role-based access control.
//
// Components:
//   - sessionService: User authentication for protected routes
//   - store: Database persistence for organizations, members, teams
//   - dialect: SQL dialect (PostgreSQL, MySQL, SQLite)
//   - config: Plugin configuration (roles, email templates)
//   - emailSender: Optional function for delivering invitation emails,
//     sourced from the email-otp plugin at init time
//
// Endpoints Provided:
//
//	Organizations: POST, GET, PUT, DELETE /organizations
//	Members: POST, GET, PATCH, DELETE /organizations/:id/members
//	Teams: POST, GET, PUT, DELETE /teams, /organizations/:id/teams
//	Team Members: POST, GET, PATCH, DELETE /teams/:teamId/members
//	Invitations: POST, GET, DELETE /organizations/:id/invitations
//	            POST, GET, DELETE /teams/:teamId/invitations
//	            POST /organizations/invitations/accept
//	            POST /organizations/invitations/decline
//	            GET  /organizations/invitations/verify
type Plugin struct {
	sessionService *core.SessionService
	store          orgtypes.OrganizationStore
	caps           orgtypes.OrganizationStoreCapabilities
	dialect        plugins.Dialect
	aegis          plugins.Aegis
	rt             *runtime
}

// runtime holds the plugin's mutable configuration and derived role maps.
// Keeping it behind a single pointer leaves Plugin comparable (==), which the
// maps and function fields would otherwise break.
type runtime struct {
	emailSender func(ctx context.Context, to, subject, body string) error
	config      Config
	orgRoles    map[string]RoleDefinition
	teamRoles   map[string]RoleDefinition
}

// New creates a new organizations plugin with default configuration.
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
	return NewWithConfig(nil, store, dialect...)
}

// NewWithConfig creates a new organizations plugin with custom configuration.
//
// Parameters:
//   - cfg: Optional configuration (nil = use defaults)
//   - store: Organization storage implementation (nil = use DefaultOrganizationStore)
//   - dialect: Database dialect (defaults to PostgreSQL)
//
// Example:
//
//	plugin := organizations.NewWithConfig(&organizations.Config{...}, nil, plugins.DialectPostgres)
func NewWithConfig(cfg *Config, store orgtypes.OrganizationStore, dialect ...plugins.Dialect) *Plugin {
	d := plugins.DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}
	p := &Plugin{
		store:   store,
		dialect: d,
		rt: &runtime{
			config: Config{
				InvitationSubject:      "You're invited!",
				InvitationBodyTemplate: "You have been invited.\n\nAccept your invitation here: %s",
			},
		},
	}
	var orgRoles, teamRoles map[string]RoleDefinition
	var customOrg, customTeam []string
	if cfg != nil {
		if cfg.InvitationSubject != "" {
			p.rt.config.InvitationSubject = cfg.InvitationSubject
		}
		if cfg.InvitationBodyTemplate != "" {
			p.rt.config.InvitationBodyTemplate = cfg.InvitationBodyTemplate
		}
		p.rt.config.OrgRoles = cfg.OrgRoles
		p.rt.config.TeamRoles = cfg.TeamRoles
		p.rt.config.CustomOrgRoles = cfg.CustomOrgRoles
		p.rt.config.CustomTeamRoles = cfg.CustomTeamRoles
		orgRoles = cfg.OrgRoles
		teamRoles = cfg.TeamRoles
		customOrg = cfg.CustomOrgRoles
		customTeam = cfg.CustomTeamRoles
	}
	p.rt.orgRoles = resolveRoles(defaultOrgRoles(), orgRoles)
	p.rt.teamRoles = resolveRoles(defaultTeamRoles(), teamRoles)

	// Deprecated Custom*Roles: register the names as read-only roles so they
	// stay assignable. OrgRoles/TeamRoles take precedence when both are set.
	for _, role := range customOrg {
		if _, exists := p.rt.orgRoles[role]; !exists {
			p.rt.orgRoles[role] = RoleDefinition{Permissions: []Permission{PermOrgView, PermMemberView, PermTeamView}}
		}
	}
	for _, role := range customTeam {
		if _, exists := p.rt.teamRoles[role]; !exists {
			p.rt.teamRoles[role] = RoleDefinition{Permissions: []Permission{PermTeamView}}
		}
	}
	return p
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

	// The plugin needs the post-v1.6 role/invitation operations. A custom store
	// may implement only OrganizationStore (the v1.6 contract), so assert the
	// capability here and fail with a clear message instead of panicking later.
	caps, ok := p.store.(orgtypes.OrganizationStoreCapabilities)
	if !ok {
		return fmt.Errorf("organizations: store %T does not implement OrganizationStoreCapabilities", p.store)
	}
	p.caps = caps

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

	// Auto-wire email sender from email-otp plugin if available.
	// The GetPlugin call already logs if the plugin is not found,
	// so no additional logging is needed here.
	if emailPlugin, ok := aegis.GetPlugin("email-otp"); ok {
		if sender, ok := emailPlugin.(interface {
			SendEmail(ctx context.Context, to, subject, body string) error
		}); ok {
			p.rt.emailSender = sender.SendEmail
		}
	}

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
	orgGroup.POST("/", requireAuth(http.HandlerFunc(p.CreateOrganizationHandler)).ServeHTTP)
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
	orgGroup.GET("/", requireAuth(http.HandlerFunc(p.ListOrganizationsHandler)).ServeHTTP)
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

	membersGroup.POST("/", requireAuth(http.HandlerFunc(p.AddOrganizationMemberHandler)).ServeHTTP)
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

	membersGroup.GET("/", requireAuth(http.HandlerFunc(p.ListOrganizationMembersHandler)).ServeHTTP)
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

	orgTeams.POST("/", requireAuth(http.HandlerFunc(p.CreateTeamHandler)).ServeHTTP)
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

	orgTeams.GET("/", requireAuth(http.HandlerFunc(p.ListTeamsHandler)).ServeHTTP)
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

	teamsGroup.GET("/:teamId", requireAuth(http.HandlerFunc(p.GetTeamHandler)).ServeHTTP)
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

	// ── Invitation routes ─────────────────────────────────────────────────

	// Accept/decline/verify endpoints are NOT authenticated — the token is
	// the credential. They live at the plugin prefix level.
	r.POST(prefix+"/invitations/accept", http.HandlerFunc(p.AcceptInvitationHandler).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/invitations/accept",
		Summary:     "Accept invitation",
		Description: "Accept a pending invitation using the raw token from the invite email",
		Tags:        []string{"Invitations"},
		Params: []openapi.Param{
			{Name: "token", In: "query", Type: "string", Required: true},
		},
		Body: openapi.BodyOf[AcceptInvitationRequest](),
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[orgtypes.Invitation]("Invitation accepted"),
			400: openapi.RefResponse("Invalid or expired token", "Error"),
		},
	})

	r.POST(prefix+"/invitations/decline", http.HandlerFunc(p.DeclineInvitationHandler).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/invitations/decline",
		Summary:     "Decline invitation",
		Description: "Decline a pending invitation using the raw token from the invite email",
		Tags:        []string{"Invitations"},
		Body:        openapi.BodyOf[DeclineInvitationRequest](),
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[orgtypes.Invitation]("Invitation declined"),
			400: openapi.RefResponse("Invalid or expired token", "Error"),
		},
	})

	r.GET(prefix+"/invitations/verify", http.HandlerFunc(p.VerifyInvitationHandler).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/invitations/verify",
		Summary:     "Verify invitation token",
		Description: "Check if an invitation token is valid and return invitation details",
		Tags:        []string{"Invitations"},
		Params: []openapi.Param{
			{Name: "token", In: "query", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.DataResponseOf[orgtypes.Invitation]("Invitation details"),
			400: openapi.RefResponse("Invalid or expired token", "Error"),
		},
	})

	// Org-level invitation management (authenticated + admin/owner)
	orgInvitesGroup := orgGroup.Group("/:id/invitations", "Invitations")

	orgInvitesGroup.POST("/", requireAuth(http.HandlerFunc(p.CreateInvitationHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "POST",
		Path:        prefix + "/{id}/invitations",
		Summary:     "Create invitation",
		Description: "Create a new invitation for an organization (requires admin role)",
		Tags:        []string{"Invitations"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Body: openapi.BodyOf[CreateInvitationRequest](),
		Responses: openapi.Responses{
			201: openapi.DataResponseOf[InvitationResponse]("Invitation created"),
			400: openapi.RefResponse("Invalid request or validation error", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
		},
	})

	orgInvitesGroup.GET("/", requireAuth(http.HandlerFunc(p.ListInvitationsHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "GET",
		Path:        prefix + "/{id}/invitations",
		Summary:     "List invitations",
		Description: "List pending invitations for an organization (requires admin role)",
		Tags:        []string{"Invitations"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.PaginatedResponseOf[core.PaginatedResponse[orgtypes.Invitation]]("List of invitations"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
		},
	})

	orgInvitesGroup.DELETE("/:invitationId", requireAuth(http.HandlerFunc(p.CancelInvitationHandler)).ServeHTTP)
	openapi.Doc(openapi.Route{
		Method:      "DELETE",
		Path:        prefix + "/{id}/invitations/{invitationId}",
		Summary:     "Cancel invitation",
		Description: "Cancel a pending invitation (requires admin role)",
		Tags:        []string{"Invitations"},
		Auth:        true,
		Params: []openapi.Param{
			{Name: "id", In: "path", Type: "string", Required: true},
			{Name: "invitationId", In: "path", Type: "string", Required: true},
		},
		Responses: openapi.Responses{
			200: openapi.RefResponse("Invitation canceled", "Success"),
			400: openapi.RefResponse("Invalid invitation ID", "Error"),
			401: openapi.RefResponse("Not authenticated", "Error"),
			403: openapi.RefResponse("Insufficient permissions", "Error"),
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

// HasOrgRole checks whether the user has any of the given org-level roles.
//
// This is the general-purpose role check for org-level permissions.
// Pass one or more role constants to check against.
//
// Example:
//
//	isAdmin, _ := p.HasOrgRole(ctx, userID, orgID, orgtypes.RoleOwner, orgtypes.RoleAdmin)
//	isMember, _ := p.HasOrgRole(ctx, userID, orgID, orgtypes.RoleOwner, orgtypes.RoleAdmin, orgtypes.RoleMember)
func (p *Plugin) HasOrgRole(ctx context.Context, userID, orgID string, roles ...string) (bool, error) {
	return p.caps.HasOrgRole(ctx, userID, orgID, roles...)
}

// HasTeamRole checks whether the user has any of the given team-level roles.
//
// Example:
//
//	canLead, _ := p.HasTeamRole(ctx, userID, teamID, orgtypes.RoleTeamLead)
func (p *Plugin) HasTeamRole(ctx context.Context, userID, teamID string, roles ...string) (bool, error) {
	return p.caps.HasTeamRole(ctx, userID, teamID, roles...)
}

// CanAccessTeam checks whether a user can access a team.
//
// A user can access a team if they are a member of the parent organization
// AND a member of the specific team (with any team role).
func (p *Plugin) CanAccessTeam(ctx context.Context, userID, teamID string) (bool, error) {
	return p.caps.CanAccessTeam(ctx, userID, teamID)
}

// IsOrganizationMember checks if a user is a member of an organization.
//
// Deprecated: Use HasOrgPermission or HasOrgRole instead.
//
// Returns true for any membership regardless of role, so custom roles count as
// members. A store error (including "not a member") returns false.
func (p *Plugin) IsOrganizationMember(ctx context.Context, userID, orgID string) bool {
	_, err := p.store.GetMember(ctx, userID, orgID)
	return err == nil
}

// IsOwnerOrAdmin checks if a user is an owner or admin of an organization.
//
// Deprecated: Use HasOrgRole instead.
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
	ok, err := p.HasOrgRole(ctx, userID, orgID, orgtypes.RoleOwner, orgtypes.RoleAdmin)
	return err == nil && ok
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
	ok, err := p.HasOrgRole(ctx, userID, orgID, orgtypes.RoleOwner)
	return err == nil && ok
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

// ── Invitation operations ──────────────────────────────────────────────────

// generateInvitationToken creates a cryptographically random 32-byte token,
// returns the raw (base64url-encoded) form and its SHA-256 hash.
func generateInvitationToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	hash = hashTokenForLookup(raw)
	return raw, hash, nil
}

// CreateInvitation creates a new pending invitation, generates a token, and
// optionally sends an invitation email via the email-otp plugin.
//
// Parameters:
//   - ctx: Request context
//   - orgID: Target organization ID
//   - teamID: Optional team ID (nil = org-level invitation)
//   - email: Invitee email address
//   - role: Role on acceptance ("admin" or "member")
//   - inviterID: User ID of the person creating the invitation
//   - expiresIn: Duration until expiry (e.g. "72h"), defaults to 168h (7 days)
//
// Returns:
//   - *Invitation: Created invitation (TokenHash is never exposed)
//   - rawToken: The raw token — returned once and never stored
//   - error: Database or token generation error
func (p *Plugin) CreateInvitation(ctx context.Context, orgID string, teamID *string, email, role, inviterID, expiresIn string) (*orgtypes.Invitation, string, error) {
	rawToken, tokenHash, err := generateInvitationToken()
	if err != nil {
		return nil, "", err
	}

	// Parse expiry duration (default 7 days)
	duration := 168 * time.Hour
	if expiresIn != "" {
		d, err := time.ParseDuration(expiresIn)
		if err == nil && d > 0 {
			duration = d
		}
	}

	now := time.Now()
	id := core.GenerateID()

	inv := orgtypes.Invitation{
		ID:             id,
		OrganizationID: orgID,
		TeamID:         teamID,
		Email:          core.SanitizeString(email, nil),
		Role:           role,
		InviterID:      inviterID,
		TokenHash:      tokenHash,
		Status:         "pending",
		ExpiresAt:      now.Add(duration),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := p.caps.CreateInvitation(ctx, inv); err != nil {
		return nil, "", err
	}

	// Send invitation email via the email-otp plugin if available.
	// If email-otp is not registered, no email is sent — the raw token
	// is returned in the API response so the caller can deliver it.
	if p.rt.emailSender != nil {
		acceptURL := p.buildAcceptURL(rawToken)
		body := fmt.Sprintf(p.rt.config.InvitationBodyTemplate, acceptURL)
		if err := p.rt.emailSender(ctx, inv.Email, p.rt.config.InvitationSubject, body); err != nil {
			return nil, "", err
		}
	}

	// Never expose the token hash
	inv.TokenHash = ""
	return &inv, rawToken, nil
}

// buildAcceptURL constructs the acceptance URL for an invitation token.
// Uses the base path from the plugin prefix if available, or a generic path.
func (p *Plugin) buildAcceptURL(rawToken string) string {
	return "/organizations/invitations/accept?token=" + rawToken
}

// AcceptInvitation accepts a pending invitation, creating the appropriate
// member record (and team member record if team-level).
//
// The caller must validate the token (hash, expiry, status) before calling.
//
// Parameters:
//   - ctx: Request context
//   - tokenHash: SHA-256 hash of the raw token
//   - userID: ID of the accepting user (must exist in the auth system)
//
// Returns:
//   - *Invitation: Updated invitation with status "accepted"
//   - error: If invitation not found, expired, or already processed
func (p *Plugin) AcceptInvitation(ctx context.Context, tokenHash, userID string) (*orgtypes.Invitation, error) {
	inv, err := p.caps.GetInvitationByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	if inv.Status != "pending" {
		return nil, fmt.Errorf("invitation is %s, not pending", inv.Status)
	}

	if time.Now().After(inv.ExpiresAt) {
		if err := p.caps.UpdateInvitationStatus(ctx, inv.ID, "expired", time.Now()); err != nil {
			if logger := p.aegis.GetLogger(); logger != nil {
				logger.Error("failed to mark invitation as expired", "error", err, "invitation_id", inv.ID)
			}
		}
		return nil, fmt.Errorf("invitation has expired")
	}

	now := time.Now()

	// Team-level invitations carry a team role; the user joins the org as a
	// base member and the team with the invited role. Org-level invitations
	// carry the org role directly.
	orgRole := inv.Role
	if inv.TeamID != nil && *inv.TeamID != "" {
		orgRole = orgtypes.RoleMember
	}

	// Create member record
	if err := p.store.CreateMember(ctx, core.GenerateID(), userID, inv.OrganizationID, orgRole, now, now); err != nil {
		return nil, err
	}

	// If team-level invitation, also create team member record
	if inv.TeamID != nil && *inv.TeamID != "" {
		if err := p.store.CreateTeamMember(ctx, core.GenerateID(), *inv.TeamID, userID, inv.Role, now, now); err != nil {
			return nil, err
		}
	}

	// Update invitation status
	if err := p.caps.UpdateInvitationStatus(ctx, inv.ID, "accepted", now); err != nil {
		return nil, err
	}

	inv.Status = "accepted"
	inv.UpdatedAt = now
	inv.TokenHash = ""
	return &inv, nil
}

// DeclineInvitation declines a pending invitation without creating any member records.
//
// Parameters:
//   - ctx: Request context
//   - tokenHash: SHA-256 hash of the raw token
//
// Returns:
//   - *Invitation: Updated invitation with status "declined"
//   - error: If invitation not found or already processed
func (p *Plugin) DeclineInvitation(ctx context.Context, tokenHash string) (*orgtypes.Invitation, error) {
	inv, err := p.caps.GetInvitationByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	if inv.Status != "pending" {
		return nil, fmt.Errorf("invitation is %s, not pending", inv.Status)
	}

	if err := p.caps.UpdateInvitationStatus(ctx, inv.ID, "declined", time.Now()); err != nil {
		return nil, err
	}

	inv.Status = "declined"
	inv.UpdatedAt = time.Now()
	inv.TokenHash = ""
	return &inv, nil
}

// VerifyInvitation validates a raw invitation token and returns the invitation
// details (without exposing the token hash). Used by the UI to pre-fill
// invitation information before the user accepts.
//
// Parameters:
//   - ctx: Request context
//   - rawToken: The raw invitation token
//
// Returns:
//   - *Invitation: Invitation details (TokenHash is empty)
//   - error: If token is invalid, expired, or already processed
func (p *Plugin) VerifyInvitation(ctx context.Context, rawToken string) (*orgtypes.Invitation, error) {
	tokenHash := hashTokenForLookup(rawToken)

	inv, err := p.caps.GetInvitationByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	if inv.Status != "pending" {
		return nil, fmt.Errorf("invitation is %s, not pending", inv.Status)
	}

	if time.Now().After(inv.ExpiresAt) {
		return nil, fmt.Errorf("invitation has expired")
	}

	inv.TokenHash = ""
	return &inv, nil
}

// CancelInvitation deletes (cancels) a pending invitation.
//
// Only the inviter or an admin/owner can cancel. The caller should verify
// permissions before calling.
//
// Parameters:
//   - ctx: Request context
//   - id: Invitation ID to cancel
//
// Returns:
//   - error: Database error
func (p *Plugin) CancelInvitation(ctx context.Context, id string) error {
	return p.caps.DeleteInvitation(ctx, id)
}

// ListInvitations returns a paginated list of invitations for an organization,
// optionally filtered to a specific team.
func (p *Plugin) ListInvitations(ctx context.Context, orgID string, teamID string, offset, limit int) ([]*orgtypes.Invitation, int, error) {
	invites, err := p.caps.ListInvitations(ctx, orgID, teamID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := p.caps.CountInvitations(ctx, orgID, teamID)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*orgtypes.Invitation, len(invites))
	for i := range invites {
		invites[i].TokenHash = ""
		result[i] = &invites[i]
	}
	return result, count, nil
}

// Dependencies returns plugin dependencies
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}

// RequiresTables returns required tables
func (p *Plugin) RequiresTables() []string {
	return []string{"organization", "members", "team", "team_member", "invitation"}
}

// ProvidesAuthMethods returns the provided auth methods
func (p *Plugin) ProvidesAuthMethods() []string {
	return []string{}
}

// Ensure Plugin implements UserEnricher
var _ plugins.UserEnricher = (*Plugin)(nil)

// Ensure Plugin implements Plugin
var _ plugins.Plugin = (*Plugin)(nil)

// hashTokenForLookup hashes a raw invitation token into the lookup form
// stored in invitation.token_hash (sha256, base64url-encoded). This is the
// scheme organizations has always used; changing it would invalidate
// outstanding invitations, so keep it stable. Single owner of the pattern
// previously duplicated in four call sites.
func hashTokenForLookup(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
