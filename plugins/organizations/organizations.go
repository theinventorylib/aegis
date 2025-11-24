package organizations

import (
	"context"
	"net/http"

	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
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
	router.GET(prefix+"/organizations", requireAuth(http.HandlerFunc(p.ListOrganizationsHandler)).ServeHTTP)
	router.GET(prefix+"/organizations/:id", requireAuth(http.HandlerFunc(p.GetOrganizationHandler)).ServeHTTP)
	router.PUT(prefix+"/organizations/:id", requireAuth(http.HandlerFunc(p.UpdateOrganizationHandler)).ServeHTTP)
	router.DELETE(prefix+"/organizations/:id", requireAuth(http.HandlerFunc(p.DeleteOrganizationHandler)).ServeHTTP)

	// Organization Member Management - all protected
	router.POST(prefix+"/organizations/:id/members", requireAuth(http.HandlerFunc(p.AddOrganizationMemberHandler)).ServeHTTP)
	router.GET(prefix+"/organizations/:id/members", requireAuth(http.HandlerFunc(p.ListOrganizationMembersHandler)).ServeHTTP)
	router.PATCH(prefix+"/organizations/:id/members/:userId", requireAuth(http.HandlerFunc(p.UpdateMemberRoleHandler)).ServeHTTP)
	router.DELETE(prefix+"/organizations/:id/members/:userId", requireAuth(http.HandlerFunc(p.RemoveOrganizationMemberHandler)).ServeHTTP)

	// Team CRUD - all protected
	router.POST(prefix+"/organizations/:id/teams", requireAuth(http.HandlerFunc(p.CreateTeamHandler)).ServeHTTP)
	router.GET(prefix+"/organizations/:id/teams", requireAuth(http.HandlerFunc(p.ListTeamsHandler)).ServeHTTP)
	router.GET(prefix+"/teams/:teamId", requireAuth(http.HandlerFunc(p.GetTeamHandler)).ServeHTTP)
	router.PUT(prefix+"/teams/:teamId", requireAuth(http.HandlerFunc(p.UpdateTeamHandler)).ServeHTTP)
	router.DELETE(prefix+"/teams/:teamId", requireAuth(http.HandlerFunc(p.DeleteTeamHandler)).ServeHTTP)

	// Team Member Management - all protected
	router.POST(prefix+"/teams/:teamId/members", requireAuth(http.HandlerFunc(p.AddTeamMemberHandler)).ServeHTTP)
	router.GET(prefix+"/teams/:teamId/members", requireAuth(http.HandlerFunc(p.ListTeamMembersHandler)).ServeHTTP)
	router.PATCH(prefix+"/teams/:teamId/members/:userId", requireAuth(http.HandlerFunc(p.UpdateTeamMemberRoleHandler)).ServeHTTP)
	router.DELETE(prefix+"/teams/:teamId/members/:userId", requireAuth(http.HandlerFunc(p.RemoveTeamMemberHandler)).ServeHTTP)
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
