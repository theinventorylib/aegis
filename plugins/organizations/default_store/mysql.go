package defaultstore

import (
	"context"
	"database/sql"
	"fmt"

	sqlcmysql "github.com/theinventorylib/aegis/plugins/organizations/internal/gen/mysql"
)

// mysqlQuerier adapts sqlcmysql.Queries to the querier interface.
// MySQL uses int32 for Limit/Offset pagination.
type mysqlQuerier struct {
	q *sqlcmysql.Queries
}

func newMysqlQuerier(db *sql.DB) *mysqlQuerier {
	return &mysqlQuerier{q: sqlcmysql.New(db)}
}

func (m *mysqlQuerier) createOrganization(ctx context.Context, id, name, slug, createdAt, updatedAt string) error {
	return m.q.CreateOrganization(ctx, sqlcmysql.CreateOrganizationParams{
		ID: id, Name: name, Slug: slug, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) getOrganization(ctx context.Context, id string) (orgRow, error) {
	o, err := m.q.GetOrganization(ctx, id)
	if err != nil {
		return orgRow{}, err
	}
	return orgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}, nil
}

func (m *mysqlQuerier) getOrganizationBySlug(ctx context.Context, slug string) (orgRow, error) {
	o, err := m.q.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return orgRow{}, err
	}
	return orgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}, nil
}

func (m *mysqlQuerier) updateOrganization(ctx context.Context, id, name, slug, updatedAt string) error {
	return m.q.UpdateOrganization(ctx, sqlcmysql.UpdateOrganizationParams{
		ID: id, Name: name, Slug: slug, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) deleteOrganization(ctx context.Context, id, updatedAt string) error {
	return m.q.DeleteOrganization(ctx, sqlcmysql.DeleteOrganizationParams{ID: id, UpdatedAt: updatedAt})
}

func (m *mysqlQuerier) listUserOrganizations(ctx context.Context, userID string, offset, limit int32) ([]listOrgRow, error) {
	rows, err := m.q.ListUserOrganizations(ctx, sqlcmysql.ListUserOrganizationsParams{
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

func (m *mysqlQuerier) countUserOrganizations(ctx context.Context, userID string) (int64, error) {
	return m.q.CountUserOrganizations(ctx, userID)
}

func (m *mysqlQuerier) createMember(ctx context.Context, id, userID, orgID, role, createdAt, updatedAt string) error {
	return m.q.CreateMember(ctx, sqlcmysql.CreateMemberParams{
		ID: id, UserID: userID, OrganizationID: orgID, Role: role, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) getMember(ctx context.Context, userID, orgID string) (memberRow, error) {
	r, err := m.q.GetMember(ctx, sqlcmysql.GetMemberParams{UserID: userID, OrganizationID: orgID})
	if err != nil {
		return memberRow{}, err
	}
	return memberRow{ID: r.ID, UserID: r.UserID, OrganizationID: r.OrganizationID, Role: r.Role, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}

func (m *mysqlQuerier) isOrganizationMember(ctx context.Context, userID, orgID string) (bool, error) {
	return m.q.IsOrganizationMember(ctx, sqlcmysql.IsOrganizationMemberParams{UserID: userID, OrganizationID: orgID})
}

func (m *mysqlQuerier) isOwnerOrAdmin(ctx context.Context, userID, orgID string) (bool, error) {
	return m.q.IsOwnerOrAdmin(ctx, sqlcmysql.IsOwnerOrAdminParams{UserID: userID, OrganizationID: orgID})
}

func (m *mysqlQuerier) isOwner(ctx context.Context, userID, orgID string) (bool, error) {
	return m.q.IsOwner(ctx, sqlcmysql.IsOwnerParams{UserID: userID, OrganizationID: orgID})
}

func (m *mysqlQuerier) updateMemberRole(ctx context.Context, userID, orgID, role, updatedAt string) error {
	return m.q.UpdateMemberRole(ctx, sqlcmysql.UpdateMemberRoleParams{
		UserID: userID, OrganizationID: orgID, Role: role, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) removeMember(ctx context.Context, userID, orgID string) error {
	return m.q.RemoveMember(ctx, sqlcmysql.RemoveMemberParams{UserID: userID, OrganizationID: orgID})
}

func (m *mysqlQuerier) listOrganizationMembers(ctx context.Context, orgID string, offset, limit int32) ([]memberRow, error) {
	rows, err := m.q.ListOrganizationMembers(ctx, sqlcmysql.ListOrganizationMembersParams{
		OrganizationID: orgID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]memberRow, len(rows))
	for i, r := range rows {
		result[i] = memberRow{ID: r.ID, UserID: r.UserID, OrganizationID: r.OrganizationID, Role: r.Role, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	}
	return result, nil
}

func (m *mysqlQuerier) countOrganizationMembers(ctx context.Context, orgID string) (int64, error) {
	return m.q.CountOrganizationMembers(ctx, orgID)
}

func (m *mysqlQuerier) createTeam(ctx context.Context, id, orgID, name string, description sql.NullString, createdAt, updatedAt string) error {
	return m.q.CreateTeam(ctx, sqlcmysql.CreateTeamParams{
		ID: id, OrganizationID: orgID, Name: name, Description: description, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) getTeam(ctx context.Context, id string) (teamRow, error) {
	t, err := m.q.GetTeam(ctx, id)
	if err != nil {
		return teamRow{}, err
	}
	return teamRow{ID: t.ID, OrganizationID: t.OrganizationID, Name: t.Name, Description: t.Description, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}, nil
}

func (m *mysqlQuerier) listTeams(ctx context.Context, orgID string, offset, limit int32) ([]teamRow, error) {
	rows, err := m.q.ListTeams(ctx, sqlcmysql.ListTeamsParams{
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

func (m *mysqlQuerier) countTeams(ctx context.Context, orgID string) (int64, error) {
	return m.q.CountTeams(ctx, orgID)
}

func (m *mysqlQuerier) updateTeam(ctx context.Context, id, name string, description sql.NullString, updatedAt string) error {
	return m.q.UpdateTeam(ctx, sqlcmysql.UpdateTeamParams{
		ID: id, Name: name, Description: description, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) deleteTeam(ctx context.Context, id string) error {
	return m.q.DeleteTeam(ctx, id)
}

func (m *mysqlQuerier) createTeamMember(ctx context.Context, id, teamID, userID, role, createdAt, updatedAt string) error {
	return m.q.CreateTeamMember(ctx, sqlcmysql.CreateTeamMemberParams{
		ID: id, TeamID: teamID, UserID: userID, Role: role, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) getTeamMember(ctx context.Context, teamID, userID string) (teamMemberRow, error) {
	r, err := m.q.GetTeamMember(ctx, sqlcmysql.GetTeamMemberParams{TeamID: teamID, UserID: userID})
	if err != nil {
		return teamMemberRow{}, err
	}
	return teamMemberRow{ID: r.ID, TeamID: r.TeamID, UserID: r.UserID, Role: r.Role, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}

func (m *mysqlQuerier) listTeamMembers(ctx context.Context, teamID string, offset, limit int32) ([]teamMemberRow, error) {
	rows, err := m.q.ListTeamMembers(ctx, sqlcmysql.ListTeamMembersParams{
		TeamID: teamID, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]teamMemberRow, len(rows))
	for i, r := range rows {
		result[i] = teamMemberRow{ID: r.ID, TeamID: r.TeamID, UserID: r.UserID, Role: r.Role, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	}
	return result, nil
}

func (m *mysqlQuerier) countTeamMembers(ctx context.Context, teamID string) (int64, error) {
	return m.q.CountTeamMembers(ctx, teamID)
}

func (m *mysqlQuerier) updateTeamMemberRole(ctx context.Context, teamID, userID, role, updatedAt string) error {
	return m.q.UpdateTeamMemberRole(ctx, sqlcmysql.UpdateTeamMemberRoleParams{
		TeamID: teamID, UserID: userID, Role: role, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) removeTeamMember(ctx context.Context, teamID, userID string) error {
	return m.q.RemoveTeamMember(ctx, sqlcmysql.RemoveTeamMemberParams{TeamID: teamID, UserID: userID})
}

// compile-time check
var _ querier = (*mysqlQuerier)(nil)

// suppress unused import warning when all methods are generated
var _ = fmt.Sprintf
