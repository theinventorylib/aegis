package organizations

import (
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/db"
)

// Plugin implements organization and team management
type Plugin struct {
	database db.DBProvider
}

// New creates a new organizations plugin
func New(database db.DBProvider) *Plugin {
	return &Plugin{
		database: database,
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "organizations"
}

// Init initializes the organizations plugin and registers routes
func (p *Plugin) Init(a *aegis.Aegis) error {
	router := a.GetRouter()

	// Organization CRUD
	router.POST("/api/organizations", p.CreateOrganizationHandler)
	router.GET("/api/organizations", p.ListOrganizationsHandler)
	router.GET("/api/organizations/:id", p.GetOrganizationHandler)
	router.PUT("/api/organizations/:id", p.UpdateOrganizationHandler)
	router.DELETE("/api/organizations/:id", p.DeleteOrganizationHandler)

	// Organization Member Management
	router.POST("/api/organizations/:id/members", p.AddOrganizationMemberHandler)
	router.GET("/api/organizations/:id/members", p.ListOrganizationMembersHandler)
	router.PATCH("/api/organizations/:id/members/:userId", p.UpdateMemberRoleHandler)
	router.DELETE("/api/organizations/:id/members/:userId", p.RemoveOrganizationMemberHandler)

	// Team CRUD
	router.POST("/api/organizations/:id/teams", p.CreateTeamHandler)
	router.GET("/api/organizations/:id/teams", p.ListTeamsHandler)
	router.GET("/api/teams/:teamId", p.GetTeamHandler)
	router.PUT("/api/teams/:teamId", p.UpdateTeamHandler)
	router.DELETE("/api/teams/:teamId", p.DeleteTeamHandler)

	// Team Member Management
	router.POST("/api/teams/:teamId/members", p.AddTeamMemberHandler)
	router.GET("/api/teams/:teamId/members", p.ListTeamMembersHandler)
	router.PATCH("/api/teams/:teamId/members/:userId", p.UpdateTeamMemberRoleHandler)
	router.DELETE("/api/teams/:teamId/members/:userId", p.RemoveTeamMemberHandler)

	return nil
}
