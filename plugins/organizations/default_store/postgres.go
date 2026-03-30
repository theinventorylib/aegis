package defaultstore

import (
	"context"
	"database/sql"
	"fmt"

	sqlcpostgres "github.com/theinventorylib/aegis/plugins/organizations/internal/gen/postgres"
)

// postgresQuerier adapts sqlcpostgres.Queries to the querier interface.
// Postgres uses int32 for Limit/Offset pagination.
type postgresQuerier struct {
	q *sqlcpostgres.Queries
}

func newPostgresQuerier(db *sql.DB) *postgresQuerier {
	return &postgresQuerier{q: sqlcpostgres.New(db)}
}

func (p *postgresQuerier) createOrganization(ctx context.Context, id, name, slug, createdAt, updatedAt string) error {
	return p.q.CreateOrganization(ctx, sqlcpostgres.CreateOrganizationParams{
		ID: id, Name: name, Slug: slug, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (p *postgresQuerier) getOrganization(ctx context.Context, id string) (orgRow, error) {
	o, err := p.q.GetOrganization(ctx, id)
	if err != nil {
		return orgRow{}, err
	}
	return orgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}, nil
}

func (p *postgresQuerier) getOrganizationBySlug(ctx context.Context, slug string) (orgRow, error) {
	o, err := p.q.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return orgRow{}, err
	}
	return orgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}, nil
}

func (p *postgresQuerier) updateOrganization(ctx context.Context, id, name, slug, updatedAt string) error {
	return p.q.UpdateOrganization(ctx, sqlcpostgres.UpdateOrganizationParams{
		ID: id, Name: name, Slug: slug, UpdatedAt: updatedAt,
	})
}

func (p *postgresQuerier) deleteOrganization(ctx context.Context, id, updatedAt string) error {
	return p.q.DeleteOrganization(ctx, sqlcpostgres.DeleteOrganizationParams{ID: id, UpdatedAt: updatedAt})
}

func (p *postgresQuerier) listUserOrganizations(ctx context.Context, userID string, offset, limit int32) ([]listOrgRow, error) {
	rows, err := p.q.ListUserOrganizations(ctx, sqlcpostgres.ListUserOrganizationsParams{
		UserID: userID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]listOrgRow, len(rows))
	for i, o := range rows {
		result[i] = listOrgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}
	}
	return result, nil
}

func (p *postgresQuerier) countUserOrganizations(ctx context.Context, userID string) (int64, error) {
	return p.q.CountUserOrganizations(ctx, userID)
}

func (p *postgresQuerier) createMember(ctx context.Context, id, userID, orgID, role, createdAt, updatedAt string) error {
	return p.q.CreateMember(ctx, sqlcpostgres.CreateMemberParams{
		ID: id, UserID: userID, OrganizationID: orgID, Role: role, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (p *postgresQuerier) getMember(ctx context.Context, userID, orgID string) (memberRow, error) {
	m, err := p.q.GetMember(ctx, sqlcpostgres.GetMemberParams{UserID: userID, OrganizationID: orgID})
	if err != nil {
		return memberRow{}, err
	}
	return memberRow{ID: m.ID, UserID: m.UserID, OrganizationID: m.OrganizationID, Role: m.Role, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}, nil
}

func (p *postgresQuerier) isOrganizationMember(ctx context.Context, userID, orgID string) (bool, error) {
	return p.q.IsOrganizationMember(ctx, sqlcpostgres.IsOrganizationMemberParams{UserID: userID, OrganizationID: orgID})
}

func (p *postgresQuerier) isOwnerOrAdmin(ctx context.Context, userID, orgID string) (bool, error) {
	return p.q.IsOwnerOrAdmin(ctx, sqlcpostgres.IsOwnerOrAdminParams{UserID: userID, OrganizationID: orgID})
}

func (p *postgresQuerier) isOwner(ctx context.Context, userID, orgID string) (bool, error) {
	return p.q.IsOwner(ctx, sqlcpostgres.IsOwnerParams{UserID: userID, OrganizationID: orgID})
}

func (p *postgresQuerier) updateMemberRole(ctx context.Context, userID, orgID, role, updatedAt string) error {
	return p.q.UpdateMemberRole(ctx, sqlcpostgres.UpdateMemberRoleParams{
		UserID: userID, OrganizationID: orgID, Role: role, UpdatedAt: updatedAt,
	})
}

func (p *postgresQuerier) removeMember(ctx context.Context, userID, orgID string) error {
	return p.q.RemoveMember(ctx, sqlcpostgres.RemoveMemberParams{UserID: userID, OrganizationID: orgID})
}

func (p *postgresQuerier) listOrganizationMembers(ctx context.Context, orgID string, offset, limit int32) ([]memberRow, error) {
	rows, err := p.q.ListOrganizationMembers(ctx, sqlcpostgres.ListOrganizationMembersParams{
		OrganizationID: orgID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]memberRow, len(rows))
	for i, m := range rows {
		result[i] = memberRow{ID: m.ID, UserID: m.UserID, OrganizationID: m.OrganizationID, Role: m.Role, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
	}
	return result, nil
}

func (p *postgresQuerier) countOrganizationMembers(ctx context.Context, orgID string) (int64, error) {
	return p.q.CountOrganizationMembers(ctx, orgID)
}

func (p *postgresQuerier) createTeam(ctx context.Context, id, orgID, name string, description sql.NullString, createdAt, updatedAt string) error {
	return p.q.CreateTeam(ctx, sqlcpostgres.CreateTeamParams{
		ID: id, OrganizationID: orgID, Name: name, Description: description, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (p *postgresQuerier) getTeam(ctx context.Context, id string) (teamRow, error) {
	t, err := p.q.GetTeam(ctx, id)
	if err != nil {
		return teamRow{}, err
	}
	return teamRow{ID: t.ID, OrganizationID: t.OrganizationID, Name: t.Name, Description: t.Description, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}, nil
}

func (p *postgresQuerier) listTeams(ctx context.Context, orgID string, offset, limit int32) ([]teamRow, error) {
	rows, err := p.q.ListTeams(ctx, sqlcpostgres.ListTeamsParams{
		OrganizationID: orgID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]teamRow, len(rows))
	for i, t := range rows {
		result[i] = teamRow{ID: t.ID, OrganizationID: t.OrganizationID, Name: t.Name, Description: t.Description, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
	}
	return result, nil
}

func (p *postgresQuerier) countTeams(ctx context.Context, orgID string) (int64, error) {
	return p.q.CountTeams(ctx, orgID)
}

func (p *postgresQuerier) updateTeam(ctx context.Context, id, name string, description sql.NullString, updatedAt string) error {
	return p.q.UpdateTeam(ctx, sqlcpostgres.UpdateTeamParams{
		ID: id, Name: name, Description: description, UpdatedAt: updatedAt,
	})
}

func (p *postgresQuerier) deleteTeam(ctx context.Context, id string) error {
	return p.q.DeleteTeam(ctx, id)
}

func (p *postgresQuerier) createTeamMember(ctx context.Context, id, teamID, userID, role, createdAt, updatedAt string) error {
	return p.q.CreateTeamMember(ctx, sqlcpostgres.CreateTeamMemberParams{
		ID: id, TeamID: teamID, UserID: userID, Role: role, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (p *postgresQuerier) getTeamMember(ctx context.Context, teamID, userID string) (teamMemberRow, error) {
	m, err := p.q.GetTeamMember(ctx, sqlcpostgres.GetTeamMemberParams{TeamID: teamID, UserID: userID})
	if err != nil {
		return teamMemberRow{}, err
	}
	return teamMemberRow{ID: m.ID, TeamID: m.TeamID, UserID: m.UserID, Role: m.Role, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}, nil
}

func (p *postgresQuerier) listTeamMembers(ctx context.Context, teamID string, offset, limit int32) ([]teamMemberRow, error) {
	rows, err := p.q.ListTeamMembers(ctx, sqlcpostgres.ListTeamMembersParams{
		TeamID: teamID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]teamMemberRow, len(rows))
	for i, m := range rows {
		result[i] = teamMemberRow{ID: m.ID, TeamID: m.TeamID, UserID: m.UserID, Role: m.Role, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
	}
	return result, nil
}

func (p *postgresQuerier) countTeamMembers(ctx context.Context, teamID string) (int64, error) {
	return p.q.CountTeamMembers(ctx, teamID)
}

func (p *postgresQuerier) updateTeamMemberRole(ctx context.Context, teamID, userID, role, updatedAt string) error {
	return p.q.UpdateTeamMemberRole(ctx, sqlcpostgres.UpdateTeamMemberRoleParams{
		TeamID: teamID, UserID: userID, Role: role, UpdatedAt: updatedAt,
	})
}

func (p *postgresQuerier) removeTeamMember(ctx context.Context, teamID, userID string) error {
	return p.q.RemoveTeamMember(ctx, sqlcpostgres.RemoveTeamMemberParams{TeamID: teamID, UserID: userID})
}

// compile-time check
var _ querier = (*postgresQuerier)(nil)

// suppress unused import warning when all methods are generated
var _ = fmt.Sprintf
