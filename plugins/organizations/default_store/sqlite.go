package defaultstore

import (
	"context"
	"database/sql"
	"fmt"

	sqlcsqlite "github.com/theinventorylib/aegis/plugins/organizations/internal/gen/sqlite"
)

// sqliteQuerier adapts sqlcsqlite.Queries to the querier interface.
// SQLite uses int64 for Limit/Offset, so each list method widens the int32 params.
type sqliteQuerier struct {
	q *sqlcsqlite.Queries
}

func newSqliteQuerier(db *sql.DB) *sqliteQuerier {
	return &sqliteQuerier{q: sqlcsqlite.New(db)}
}

func (s *sqliteQuerier) createOrganization(ctx context.Context, id, name, slug, createdAt, updatedAt string) error {
	return s.q.CreateOrganization(ctx, sqlcsqlite.CreateOrganizationParams{
		ID: id, Name: name, Slug: slug, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) getOrganization(ctx context.Context, id string) (orgRow, error) {
	o, err := s.q.GetOrganization(ctx, id)
	if err != nil {
		return orgRow{}, err
	}
	return orgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}, nil
}

func (s *sqliteQuerier) getOrganizationBySlug(ctx context.Context, slug string) (orgRow, error) {
	o, err := s.q.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return orgRow{}, err
	}
	return orgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}, nil
}

func (s *sqliteQuerier) updateOrganization(ctx context.Context, id, name, slug, updatedAt string) error {
	return s.q.UpdateOrganization(ctx, sqlcsqlite.UpdateOrganizationParams{
		ID: id, Name: name, Slug: slug, UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) deleteOrganization(ctx context.Context, id, updatedAt string) error {
	return s.q.DeleteOrganization(ctx, sqlcsqlite.DeleteOrganizationParams{ID: id, UpdatedAt: updatedAt})
}

func (s *sqliteQuerier) listUserOrganizations(ctx context.Context, userID string, offset, limit int32) ([]listOrgRow, error) {
	rows, err := s.q.ListUserOrganizations(ctx, sqlcsqlite.ListUserOrganizationsParams{
		UserID: userID, Limit: int64(limit), Offset: int64(offset),
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

func (s *sqliteQuerier) countUserOrganizations(ctx context.Context, userID string) (int64, error) {
	return s.q.CountUserOrganizations(ctx, userID)
}

func (s *sqliteQuerier) createMember(ctx context.Context, id, userID, orgID, role, createdAt, updatedAt string) error {
	return s.q.CreateMember(ctx, sqlcsqlite.CreateMemberParams{
		ID: id, UserID: userID, OrganizationID: orgID, Role: role, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) getMember(ctx context.Context, userID, orgID string) (memberRow, error) {
	r, err := s.q.GetMember(ctx, sqlcsqlite.GetMemberParams{UserID: userID, OrganizationID: orgID})
	if err != nil {
		return memberRow{}, err
	}
	return memberRow{ID: r.ID, UserID: r.UserID, OrganizationID: r.OrganizationID, Role: r.Role, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}

func (s *sqliteQuerier) isOrganizationMember(ctx context.Context, userID, orgID string) (bool, error) {
	return s.q.IsOrganizationMember(ctx, sqlcsqlite.IsOrganizationMemberParams{UserID: userID, OrganizationID: orgID})
}

func (s *sqliteQuerier) isOwnerOrAdmin(ctx context.Context, userID, orgID string) (bool, error) {
	return s.q.IsOwnerOrAdmin(ctx, sqlcsqlite.IsOwnerOrAdminParams{UserID: userID, OrganizationID: orgID})
}

func (s *sqliteQuerier) isOwner(ctx context.Context, userID, orgID string) (bool, error) {
	return s.q.IsOwner(ctx, sqlcsqlite.IsOwnerParams{UserID: userID, OrganizationID: orgID})
}

func (s *sqliteQuerier) updateMemberRole(ctx context.Context, userID, orgID, role, updatedAt string) error {
	return s.q.UpdateMemberRole(ctx, sqlcsqlite.UpdateMemberRoleParams{
		UserID: userID, OrganizationID: orgID, Role: role, UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) removeMember(ctx context.Context, userID, orgID string) error {
	return s.q.RemoveMember(ctx, sqlcsqlite.RemoveMemberParams{UserID: userID, OrganizationID: orgID})
}

func (s *sqliteQuerier) listOrganizationMembers(ctx context.Context, orgID string, offset, limit int32) ([]memberRow, error) {
	rows, err := s.q.ListOrganizationMembers(ctx, sqlcsqlite.ListOrganizationMembersParams{
		OrganizationID: orgID, Limit: int64(limit), Offset: int64(offset),
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

func (s *sqliteQuerier) countOrganizationMembers(ctx context.Context, orgID string) (int64, error) {
	return s.q.CountOrganizationMembers(ctx, orgID)
}

func (s *sqliteQuerier) createTeam(ctx context.Context, id, orgID, name string, description sql.NullString, createdAt, updatedAt string) error {
	return s.q.CreateTeam(ctx, sqlcsqlite.CreateTeamParams{
		ID: id, OrganizationID: orgID, Name: name, Description: description, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) getTeam(ctx context.Context, id string) (teamRow, error) {
	t, err := s.q.GetTeam(ctx, id)
	if err != nil {
		return teamRow{}, err
	}
	return teamRow{ID: t.ID, OrganizationID: t.OrganizationID, Name: t.Name, Description: t.Description, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}, nil
}

func (s *sqliteQuerier) listTeams(ctx context.Context, orgID string, offset, limit int32) ([]teamRow, error) {
	rows, err := s.q.ListTeams(ctx, sqlcsqlite.ListTeamsParams{
		OrganizationID: orgID, Limit: int64(limit), Offset: int64(offset),
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

func (s *sqliteQuerier) countTeams(ctx context.Context, orgID string) (int64, error) {
	return s.q.CountTeams(ctx, orgID)
}

func (s *sqliteQuerier) updateTeam(ctx context.Context, id, name string, description sql.NullString, updatedAt string) error {
	return s.q.UpdateTeam(ctx, sqlcsqlite.UpdateTeamParams{
		ID: id, Name: name, Description: description, UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) deleteTeam(ctx context.Context, id string) error {
	return s.q.DeleteTeam(ctx, id)
}

func (s *sqliteQuerier) createTeamMember(ctx context.Context, id, teamID, userID, role, createdAt, updatedAt string) error {
	return s.q.CreateTeamMember(ctx, sqlcsqlite.CreateTeamMemberParams{
		ID: id, TeamID: teamID, UserID: userID, Role: role, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) getTeamMember(ctx context.Context, teamID, userID string) (teamMemberRow, error) {
	r, err := s.q.GetTeamMember(ctx, sqlcsqlite.GetTeamMemberParams{TeamID: teamID, UserID: userID})
	if err != nil {
		return teamMemberRow{}, err
	}
	return teamMemberRow{ID: r.ID, TeamID: r.TeamID, UserID: r.UserID, Role: r.Role, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}

func (s *sqliteQuerier) listTeamMembers(ctx context.Context, teamID string, offset, limit int32) ([]teamMemberRow, error) {
	rows, err := s.q.ListTeamMembers(ctx, sqlcsqlite.ListTeamMembersParams{
		TeamID: teamID, Limit: int64(limit), Offset: int64(offset),
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

func (s *sqliteQuerier) countTeamMembers(ctx context.Context, teamID string) (int64, error) {
	return s.q.CountTeamMembers(ctx, teamID)
}

func (s *sqliteQuerier) updateTeamMemberRole(ctx context.Context, teamID, userID, role, updatedAt string) error {
	return s.q.UpdateTeamMemberRole(ctx, sqlcsqlite.UpdateTeamMemberRoleParams{
		TeamID: teamID, UserID: userID, Role: role, UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) removeTeamMember(ctx context.Context, teamID, userID string) error {
	return s.q.RemoveTeamMember(ctx, sqlcsqlite.RemoveTeamMemberParams{TeamID: teamID, UserID: userID})
}

// compile-time check
var _ querier = (*sqliteQuerier)(nil)

// suppress unused import warning when all methods are generated
var _ = fmt.Sprintf
