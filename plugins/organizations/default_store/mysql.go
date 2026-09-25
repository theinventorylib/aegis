package defaultstore

import (
	"context"
	"database/sql"
	"fmt"

	sqlcmysql "github.com/theinventorylib/aegis/v2/plugins/organizations/internal/gen/mysql"
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

func (m *mysqlQuerier) createInvitation(ctx context.Context, id, organizationID, teamID, email, role, inviterID, tokenHash, status, expiresAt, createdAt, updatedAt string) error {
	return m.q.CreateInvitation(ctx, sqlcmysql.CreateInvitationParams{
		ID: id, OrganizationID: organizationID,
		TeamID: sql.NullString{String: teamID, Valid: teamID != ""},
		Email:  email, Role: role, InviterID: inviterID,
		TokenHash: tokenHash, Status: status,
		ExpiresAt: expiresAt, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) getInvitationByID(ctx context.Context, id string) (invitationRow, error) {
	inv, err := m.q.GetInvitationByID(ctx, id)
	if err != nil {
		return invitationRow{}, err
	}
	return invitationRow{
		ID: inv.ID, OrganizationID: inv.OrganizationID, TeamID: inv.TeamID,
		Email: inv.Email, Role: inv.Role, InviterID: inv.InviterID,
		TokenHash: inv.TokenHash, Status: inv.Status,
		ExpiresAt: inv.ExpiresAt, CreatedAt: inv.CreatedAt, UpdatedAt: inv.UpdatedAt,
	}, nil
}

func (m *mysqlQuerier) getInvitationByTokenHash(ctx context.Context, tokenHash string) (invitationRow, error) {
	inv, err := m.q.GetInvitationByTokenHash(ctx, tokenHash)
	if err != nil {
		return invitationRow{}, err
	}
	return invitationRow{
		ID: inv.ID, OrganizationID: inv.OrganizationID, TeamID: inv.TeamID,
		Email: inv.Email, Role: inv.Role, InviterID: inv.InviterID,
		TokenHash: inv.TokenHash, Status: inv.Status,
		ExpiresAt: inv.ExpiresAt, CreatedAt: inv.CreatedAt, UpdatedAt: inv.UpdatedAt,
	}, nil
}

func (m *mysqlQuerier) listInvitations(ctx context.Context, orgID string, teamID string, offset, limit int32) ([]invitationRow, error) {
	rows, err := m.q.ListInvitations(ctx, sqlcmysql.ListInvitationsParams{
		OrganizationID: orgID,
		Column2:        teamID,
		TeamID:         sql.NullString{String: teamID, Valid: teamID != ""},
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, err
	}
	result := make([]invitationRow, len(rows))
	for i, inv := range rows {
		result[i] = invitationRow{
			ID: inv.ID, OrganizationID: inv.OrganizationID, TeamID: inv.TeamID,
			Email: inv.Email, Role: inv.Role, InviterID: inv.InviterID,
			TokenHash: inv.TokenHash, Status: inv.Status,
			ExpiresAt: inv.ExpiresAt, CreatedAt: inv.CreatedAt, UpdatedAt: inv.UpdatedAt,
		}
	}
	return result, nil
}

func (m *mysqlQuerier) countInvitations(ctx context.Context, orgID string, teamID string) (int64, error) {
	return m.q.CountInvitations(ctx, sqlcmysql.CountInvitationsParams{
		OrganizationID: orgID,
		Column2:        teamID,
		TeamID:         sql.NullString{String: teamID, Valid: teamID != ""},
	})
}

func (m *mysqlQuerier) updateInvitationStatus(ctx context.Context, id, status, updatedAt string) error {
	return m.q.UpdateInvitationStatus(ctx, sqlcmysql.UpdateInvitationStatusParams{
		Status: status, UpdatedAt: updatedAt, ID: id,
	})
}

func (m *mysqlQuerier) deleteInvitation(ctx context.Context, id string) error {
	return m.q.DeleteInvitation(ctx, id)
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

func (x *mysqlQuerier) listMemberPermissionOverrides(ctx context.Context, orgID, userID string) ([]permissionOverrideRow, error) {
	rows, err := x.q.ListMemberPermissionOverrides(ctx, sqlcmysql.ListMemberPermissionOverridesParams{OrganizationID: orgID, UserID: userID})
	if err != nil {
		return nil, err
	}
	out := make([]permissionOverrideRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, permissionOverrideRow{
			ID: r.ID, OrganizationID: r.OrganizationID, UserID: r.UserID,
			Permission: r.Permission, Effect: r.Effect, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	return out, nil
}

func (x *mysqlQuerier) createMemberPermissionOverride(ctx context.Context, id, orgID, userID, permission, effect, createdAt, updatedAt string) error {
	return x.q.CreateMemberPermissionOverride(ctx, sqlcmysql.CreateMemberPermissionOverrideParams{
		ID: id, OrganizationID: orgID, UserID: userID, Permission: permission,
		Effect: effect, CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (x *mysqlQuerier) deleteMemberPermissionOverrides(ctx context.Context, orgID, userID string) error {
	return x.q.DeleteMemberPermissionOverrides(ctx, sqlcmysql.DeleteMemberPermissionOverridesParams{OrganizationID: orgID, UserID: userID})
}

func (x *mysqlQuerier) listOrganizationRoles(ctx context.Context, orgID string) ([]organizationRoleRow, error) {
	rows, err := x.q.ListOrganizationRoles(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]organizationRoleRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, organizationRoleRow{
			ID: r.ID, OrganizationID: r.OrganizationID, Name: r.Name,
			Permissions: r.Permissions, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	return out, nil
}

func (x *mysqlQuerier) getOrganizationRole(ctx context.Context, orgID, name string) (organizationRoleRow, error) {
	r, err := x.q.GetOrganizationRole(ctx, sqlcmysql.GetOrganizationRoleParams{OrganizationID: orgID, Name: name})
	if err != nil {
		return organizationRoleRow{}, err
	}
	return organizationRoleRow{
		ID: r.ID, OrganizationID: r.OrganizationID, Name: r.Name,
		Permissions: r.Permissions, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

func (x *mysqlQuerier) createOrganizationRole(ctx context.Context, id, orgID, name, permissions, createdAt, updatedAt string) error {
	return x.q.CreateOrganizationRole(ctx, sqlcmysql.CreateOrganizationRoleParams{
		ID: id, OrganizationID: orgID, Name: name, Permissions: permissions,
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	})
}

func (x *mysqlQuerier) updateOrganizationRole(ctx context.Context, orgID, name, permissions, updatedAt string) error {
	return x.q.UpdateOrganizationRole(ctx, sqlcmysql.UpdateOrganizationRoleParams{
		OrganizationID: orgID, Name: name, Permissions: permissions, UpdatedAt: updatedAt,
	})
}

func (x *mysqlQuerier) deleteOrganizationRole(ctx context.Context, orgID, name string) error {
	return x.q.DeleteOrganizationRole(ctx, sqlcmysql.DeleteOrganizationRoleParams{OrganizationID: orgID, Name: name})
}

func (x *mysqlQuerier) countOrganizationMembersWithRole(ctx context.Context, orgID, role string) (int64, error) {
	return x.q.CountOrganizationMembersWithRole(ctx, sqlcmysql.CountOrganizationMembersWithRoleParams{OrganizationID: orgID, Role: role})
}
