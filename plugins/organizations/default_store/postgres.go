package defaultstore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sqlcpostgres "github.com/theinventorylib/aegis/v2/plugins/organizations/internal/gen/postgres"
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
		ID: id, Name: name, Slug: slug, CreatedAt: pgParseTime(createdAt), UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) getOrganization(ctx context.Context, id string) (orgRow, error) {
	o, err := p.q.GetOrganization(ctx, id)
	if err != nil {
		return orgRow{}, err
	}
	return orgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: pgFormatTime(o.CreatedAt), UpdatedAt: pgFormatTime(o.UpdatedAt)}, nil
}

func (p *postgresQuerier) getOrganizationBySlug(ctx context.Context, slug string) (orgRow, error) {
	o, err := p.q.GetOrganizationBySlug(ctx, slug)
	if err != nil {
		return orgRow{}, err
	}
	return orgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: pgFormatTime(o.CreatedAt), UpdatedAt: pgFormatTime(o.UpdatedAt)}, nil
}

func (p *postgresQuerier) updateOrganization(ctx context.Context, id, name, slug, updatedAt string) error {
	return p.q.UpdateOrganization(ctx, sqlcpostgres.UpdateOrganizationParams{
		ID: id, Name: name, Slug: slug, UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) deleteOrganization(ctx context.Context, id, updatedAt string) error {
	return p.q.DeleteOrganization(ctx, sqlcpostgres.DeleteOrganizationParams{ID: id, UpdatedAt: pgParseTime(updatedAt)})
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
		result[i] = listOrgRow{ID: o.ID, Name: o.Name, Slug: o.Slug, CreatedAt: pgFormatTime(o.CreatedAt), UpdatedAt: pgFormatTime(o.UpdatedAt)}
	}
	return result, nil
}

func (p *postgresQuerier) countUserOrganizations(ctx context.Context, userID string) (int64, error) {
	return p.q.CountUserOrganizations(ctx, userID)
}

func (p *postgresQuerier) createMember(ctx context.Context, id, userID, orgID, role, createdAt, updatedAt string) error {
	return p.q.CreateMember(ctx, sqlcpostgres.CreateMemberParams{
		ID: id, UserID: userID, OrganizationID: orgID, Role: role, CreatedAt: pgParseTime(createdAt), UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) getMember(ctx context.Context, userID, orgID string) (memberRow, error) {
	m, err := p.q.GetMember(ctx, sqlcpostgres.GetMemberParams{UserID: userID, OrganizationID: orgID})
	if err != nil {
		return memberRow{}, err
	}
	return memberRow{ID: m.ID, UserID: m.UserID, OrganizationID: m.OrganizationID, Role: m.Role, CreatedAt: pgFormatTime(m.CreatedAt), UpdatedAt: pgFormatTime(m.UpdatedAt)}, nil
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
		UserID: userID, OrganizationID: orgID, Role: role, UpdatedAt: pgParseTime(updatedAt),
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
		result[i] = memberRow{ID: m.ID, UserID: m.UserID, OrganizationID: m.OrganizationID, Role: m.Role, CreatedAt: pgFormatTime(m.CreatedAt), UpdatedAt: pgFormatTime(m.UpdatedAt)}
	}
	return result, nil
}

func (p *postgresQuerier) countOrganizationMembers(ctx context.Context, orgID string) (int64, error) {
	return p.q.CountOrganizationMembers(ctx, orgID)
}

func (p *postgresQuerier) createTeam(ctx context.Context, id, orgID, name string, description sql.NullString, createdAt, updatedAt string) error {
	return p.q.CreateTeam(ctx, sqlcpostgres.CreateTeamParams{
		ID: id, OrganizationID: orgID, Name: name, Description: description, CreatedAt: pgParseTime(createdAt), UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) getTeam(ctx context.Context, id string) (teamRow, error) {
	t, err := p.q.GetTeam(ctx, id)
	if err != nil {
		return teamRow{}, err
	}
	return teamRow{ID: t.ID, OrganizationID: t.OrganizationID, Name: t.Name, Description: t.Description, CreatedAt: pgFormatTime(t.CreatedAt), UpdatedAt: pgFormatTime(t.UpdatedAt)}, nil
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
		result[i] = teamRow{ID: t.ID, OrganizationID: t.OrganizationID, Name: t.Name, Description: t.Description, CreatedAt: pgFormatTime(t.CreatedAt), UpdatedAt: pgFormatTime(t.UpdatedAt)}
	}
	return result, nil
}

func (p *postgresQuerier) countTeams(ctx context.Context, orgID string) (int64, error) {
	return p.q.CountTeams(ctx, orgID)
}

func (p *postgresQuerier) updateTeam(ctx context.Context, id, name string, description sql.NullString, updatedAt string) error {
	return p.q.UpdateTeam(ctx, sqlcpostgres.UpdateTeamParams{
		ID: id, Name: name, Description: description, UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) deleteTeam(ctx context.Context, id string) error {
	return p.q.DeleteTeam(ctx, id)
}

func (p *postgresQuerier) createInvitation(ctx context.Context, id, organizationID, teamID, email, role, inviterID, tokenHash, status, expiresAt, createdAt, updatedAt string) error {
	return p.q.CreateInvitation(ctx, sqlcpostgres.CreateInvitationParams{
		ID: id, OrganizationID: organizationID,
		TeamID: sql.NullString{String: teamID, Valid: teamID != ""},
		Email:  email, Role: role, InviterID: inviterID,
		TokenHash: tokenHash, Status: status,
		ExpiresAt: pgParseTime(expiresAt), CreatedAt: pgParseTime(createdAt), UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) getInvitationByID(ctx context.Context, id string) (invitationRow, error) {
	inv, err := p.q.GetInvitationByID(ctx, id)
	if err != nil {
		return invitationRow{}, err
	}
	return invitationRow{
		ID: inv.ID, OrganizationID: inv.OrganizationID, TeamID: inv.TeamID,
		Email: inv.Email, Role: inv.Role, InviterID: inv.InviterID,
		TokenHash: inv.TokenHash, Status: inv.Status,
		ExpiresAt: pgFormatTime(inv.ExpiresAt), CreatedAt: pgFormatTime(inv.CreatedAt), UpdatedAt: pgFormatTime(inv.UpdatedAt),
	}, nil
}

func (p *postgresQuerier) getInvitationByTokenHash(ctx context.Context, tokenHash string) (invitationRow, error) {
	inv, err := p.q.GetInvitationByTokenHash(ctx, tokenHash)
	if err != nil {
		return invitationRow{}, err
	}
	return invitationRow{
		ID: inv.ID, OrganizationID: inv.OrganizationID, TeamID: inv.TeamID,
		Email: inv.Email, Role: inv.Role, InviterID: inv.InviterID,
		TokenHash: inv.TokenHash, Status: inv.Status,
		ExpiresAt: pgFormatTime(inv.ExpiresAt), CreatedAt: pgFormatTime(inv.CreatedAt), UpdatedAt: pgFormatTime(inv.UpdatedAt),
	}, nil
}

func (p *postgresQuerier) listInvitations(ctx context.Context, orgID string, teamID string, offset, limit int32) ([]invitationRow, error) {
	rows, err := p.q.ListInvitations(ctx, sqlcpostgres.ListInvitationsParams{
		OrganizationID: orgID, Column2: teamID, Limit: limit, Offset: offset,
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
			ExpiresAt: pgFormatTime(inv.ExpiresAt), CreatedAt: pgFormatTime(inv.CreatedAt), UpdatedAt: pgFormatTime(inv.UpdatedAt),
		}
	}
	return result, nil
}

func (p *postgresQuerier) countInvitations(ctx context.Context, orgID string, teamID string) (int64, error) {
	return p.q.CountInvitations(ctx, sqlcpostgres.CountInvitationsParams{
		OrganizationID: orgID, Column2: teamID,
	})
}

func (p *postgresQuerier) updateInvitationStatus(ctx context.Context, id, status, updatedAt string) error {
	return p.q.UpdateInvitationStatus(ctx, sqlcpostgres.UpdateInvitationStatusParams{
		ID: id, Status: status, UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) deleteInvitation(ctx context.Context, id string) error {
	return p.q.DeleteInvitation(ctx, id)
}

func (p *postgresQuerier) createTeamMember(ctx context.Context, id, teamID, userID, role, createdAt, updatedAt string) error {
	return p.q.CreateTeamMember(ctx, sqlcpostgres.CreateTeamMemberParams{
		ID: id, TeamID: teamID, UserID: userID, Role: role, CreatedAt: pgParseTime(createdAt), UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) getTeamMember(ctx context.Context, teamID, userID string) (teamMemberRow, error) {
	m, err := p.q.GetTeamMember(ctx, sqlcpostgres.GetTeamMemberParams{TeamID: teamID, UserID: userID})
	if err != nil {
		return teamMemberRow{}, err
	}
	return teamMemberRow{ID: m.ID, TeamID: m.TeamID, UserID: m.UserID, Role: m.Role, CreatedAt: pgFormatTime(m.CreatedAt), UpdatedAt: pgFormatTime(m.UpdatedAt)}, nil
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
		result[i] = teamMemberRow{ID: m.ID, TeamID: m.TeamID, UserID: m.UserID, Role: m.Role, CreatedAt: pgFormatTime(m.CreatedAt), UpdatedAt: pgFormatTime(m.UpdatedAt)}
	}
	return result, nil
}

func (p *postgresQuerier) countTeamMembers(ctx context.Context, teamID string) (int64, error) {
	return p.q.CountTeamMembers(ctx, teamID)
}

func (p *postgresQuerier) updateTeamMemberRole(ctx context.Context, teamID, userID, role, updatedAt string) error {
	return p.q.UpdateTeamMemberRole(ctx, sqlcpostgres.UpdateTeamMemberRoleParams{
		TeamID: teamID, UserID: userID, Role: role, UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) removeTeamMember(ctx context.Context, teamID, userID string) error {
	return p.q.RemoveTeamMember(ctx, sqlcpostgres.RemoveTeamMemberParams{TeamID: teamID, UserID: userID})
}

// compile-time check
var _ querier = (*postgresQuerier)(nil)

// suppress unused import warning when all methods are generated
var _ = fmt.Sprintf

func (x *postgresQuerier) listMemberPermissionOverrides(ctx context.Context, orgID, userID string) ([]permissionOverrideRow, error) {
	rows, err := x.q.ListMemberPermissionOverrides(ctx, sqlcpostgres.ListMemberPermissionOverridesParams{OrganizationID: orgID, UserID: userID})
	if err != nil {
		return nil, err
	}
	out := make([]permissionOverrideRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, permissionOverrideRow{
			ID: r.ID, OrganizationID: r.OrganizationID, UserID: r.UserID,
			Permission: r.Permission, Effect: r.Effect, CreatedAt: pgFormatTime(r.CreatedAt), UpdatedAt: pgFormatTime(r.UpdatedAt),
		})
	}
	return out, nil
}

func (x *postgresQuerier) createMemberPermissionOverride(ctx context.Context, id, orgID, userID, permission, effect, createdAt, updatedAt string) error {
	return x.q.CreateMemberPermissionOverride(ctx, sqlcpostgres.CreateMemberPermissionOverrideParams{
		ID: id, OrganizationID: orgID, UserID: userID, Permission: permission,
		Effect: effect, CreatedAt: pgParseTime(createdAt), UpdatedAt: pgParseTime(updatedAt),
	})
}

func (x *postgresQuerier) deleteMemberPermissionOverrides(ctx context.Context, orgID, userID string) error {
	return x.q.DeleteMemberPermissionOverrides(ctx, sqlcpostgres.DeleteMemberPermissionOverridesParams{OrganizationID: orgID, UserID: userID})
}

func (x *postgresQuerier) listOrganizationRoles(ctx context.Context, orgID string) ([]organizationRoleRow, error) {
	rows, err := x.q.ListOrganizationRoles(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]organizationRoleRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, organizationRoleRow{
			ID: r.ID, OrganizationID: r.OrganizationID, Name: r.Name,
			Permissions: r.Permissions, CreatedAt: pgFormatTime(r.CreatedAt), UpdatedAt: pgFormatTime(r.UpdatedAt),
		})
	}
	return out, nil
}

func (x *postgresQuerier) getOrganizationRole(ctx context.Context, orgID, name string) (organizationRoleRow, error) {
	r, err := x.q.GetOrganizationRole(ctx, sqlcpostgres.GetOrganizationRoleParams{OrganizationID: orgID, Name: name})
	if err != nil {
		return organizationRoleRow{}, err
	}
	return organizationRoleRow{
		ID: r.ID, OrganizationID: r.OrganizationID, Name: r.Name,
		Permissions: r.Permissions, CreatedAt: pgFormatTime(r.CreatedAt), UpdatedAt: pgFormatTime(r.UpdatedAt),
	}, nil
}

func (x *postgresQuerier) createOrganizationRole(ctx context.Context, id, orgID, name, permissions, createdAt, updatedAt string) error {
	return x.q.CreateOrganizationRole(ctx, sqlcpostgres.CreateOrganizationRoleParams{
		ID: id, OrganizationID: orgID, Name: name, Permissions: permissions,
		CreatedAt: pgParseTime(createdAt), UpdatedAt: pgParseTime(updatedAt),
	})
}

func (x *postgresQuerier) updateOrganizationRole(ctx context.Context, orgID, name, permissions, updatedAt string) error {
	return x.q.UpdateOrganizationRole(ctx, sqlcpostgres.UpdateOrganizationRoleParams{
		OrganizationID: orgID, Name: name, Permissions: permissions, UpdatedAt: pgParseTime(updatedAt),
	})
}

func (x *postgresQuerier) deleteOrganizationRole(ctx context.Context, orgID, name string) error {
	return x.q.DeleteOrganizationRole(ctx, sqlcpostgres.DeleteOrganizationRoleParams{OrganizationID: orgID, Name: name})
}

func (x *postgresQuerier) countOrganizationMembersWithRole(ctx context.Context, orgID, role string) (int64, error) {
	return x.q.CountOrganizationMembersWithRole(ctx, sqlcpostgres.CountOrganizationMembersWithRoleParams{OrganizationID: orgID, Role: role})
}

// pgParseTime converts a canonical RFC3339 string to the time.Time the
// generated postgres queries expect.
func pgParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// pgFormatTime converts a time.Time back to the canonical RFC3339 string.
func pgFormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// pgParseNullTime converts a nullable canonical string to sql.NullTime.
func pgParseNullTime(ns sql.NullString) sql.NullTime {
	if !ns.Valid {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: pgParseTime(ns.String), Valid: true}
}

// pgFormatNullTime converts sql.NullTime back to a nullable canonical string.
func pgFormatNullTime(nt sql.NullTime) sql.NullString {
	if !nt.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: pgFormatTime(nt.Time), Valid: true}
}
