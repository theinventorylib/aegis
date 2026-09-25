// Package defaultstore implements the SQL-backed default store for the organizations plugin.
//
// Dialect selection happens once in NewDefaultOrganizationStore; all methods are
// dialect-agnostic and delegate to the unexported querier interface.
package defaultstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"slices"
	"time"

	"github.com/theinventorylib/aegis/v2/plugins"
	orgtypes "github.com/theinventorylib/aegis/v2/plugins/organizations/types"
)

// DefaultOrganizationStore implements orgtypes.OrganizationStore using a SQL database.
//
// All dialect logic is confined to the querier: this struct holds only the
// single chosen querier and delegates every operation to it.
type DefaultOrganizationStore struct {
	q querier
}

// NewDefaultOrganizationStore creates a DefaultOrganizationStore for the given dialect.
func NewDefaultOrganizationStore(db *sql.DB, dialect plugins.Dialect) (*DefaultOrganizationStore, error) {
	switch dialect {
	case plugins.DialectPostgres:
		return &DefaultOrganizationStore{q: newPostgresQuerier(db)}, nil
	case plugins.DialectMySQL:
		return &DefaultOrganizationStore{q: newMysqlQuerier(db)}, nil
	case plugins.DialectSQLite:
		return &DefaultOrganizationStore{q: newSqliteQuerier(db)}, nil
	default:
		return nil, fmt.Errorf("organizations: unsupported dialect %q", dialect)
	}
}

// clampPagination normalises caller-supplied offset/limit into safe int32 values
// that every dialect translator can accept.
func clampPagination(offset, limit int) (int32, int32) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if offset > math.MaxInt32 {
		offset = math.MaxInt32
	}
	if limit > math.MaxInt32 {
		limit = math.MaxInt32
	}
	return int32(offset), int32(limit)
}

// ── Organization operations ──────────────────────────────────────────────────

// CreateOrganization creates a new organization with the given attributes.
func (s *DefaultOrganizationStore) CreateOrganization(ctx context.Context, id, name, slug string, createdAt, updatedAt time.Time) error {
	return s.q.createOrganization(ctx, id, name, slug,
		createdAt.Format(time.RFC3339),
		updatedAt.Format(time.RFC3339),
	)
}

// GetOrganization retrieves an organization by its ID.
func (s *DefaultOrganizationStore) GetOrganization(ctx context.Context, id string) (orgtypes.Organization, error) {
	o, err := s.q.getOrganization(ctx, id)
	if err != nil {
		return orgtypes.Organization{}, err
	}
	return buildOrganization(o), nil
}

// GetOrganizationBySlug retrieves an organization by its URL slug.
func (s *DefaultOrganizationStore) GetOrganizationBySlug(ctx context.Context, slug string) (orgtypes.Organization, error) {
	o, err := s.q.getOrganizationBySlug(ctx, slug)
	if err != nil {
		return orgtypes.Organization{}, err
	}
	return buildOrganization(o), nil
}

// UpdateOrganization updates the name, slug, and updatedAt timestamp for an organization.
func (s *DefaultOrganizationStore) UpdateOrganization(ctx context.Context, id, name, slug string, updatedAt time.Time) error {
	return s.q.updateOrganization(ctx, id, name, slug, updatedAt.Format(time.RFC3339))
}

// DeleteOrganization soft-deletes an organization by ID.
func (s *DefaultOrganizationStore) DeleteOrganization(ctx context.Context, id string, updatedAt time.Time) error {
	return s.q.deleteOrganization(ctx, id, updatedAt.Format(time.RFC3339))
}

// ListUserOrganizations returns a paginated list of organizations the user belongs to.
func (s *DefaultOrganizationStore) ListUserOrganizations(ctx context.Context, userID string, offset, limit int) ([]orgtypes.Organization, error) {
	off, lim := clampPagination(offset, limit)
	rows, err := s.q.listUserOrganizations(ctx, userID, off, lim)
	if err != nil {
		return nil, err
	}
	result := make([]orgtypes.Organization, len(rows))
	for i, o := range rows {
		result[i] = buildOrganization(o)
	}
	return result, nil
}

// CountUserOrganizations returns the total number of organizations the user belongs to.
func (s *DefaultOrganizationStore) CountUserOrganizations(ctx context.Context, userID string) (int, error) {
	n, err := s.q.countUserOrganizations(ctx, userID)
	return int(n), err
}

// ── Member operations ────────────────────────────────────────────────────────

// CreateMember adds a user as a member of an organization with the given role.
func (s *DefaultOrganizationStore) CreateMember(ctx context.Context, id, userID, orgID, role string, createdAt, updatedAt time.Time) error {
	return s.q.createMember(ctx, id, userID, orgID, role,
		createdAt.Format(time.RFC3339),
		updatedAt.Format(time.RFC3339),
	)
}

// GetMember retrieves the membership record for a user in an organization.
func (s *DefaultOrganizationStore) GetMember(ctx context.Context, userID, orgID string) (orgtypes.Member, error) {
	m, err := s.q.getMember(ctx, userID, orgID)
	if err != nil {
		return orgtypes.Member{}, err
	}
	return buildMember(m), nil
}

// HasOrgRole checks whether the user has any of the given org-level roles.
func (s *DefaultOrganizationStore) HasOrgRole(ctx context.Context, userID, orgID string, roles ...string) (bool, error) {
	m, err := s.q.getMember(ctx, userID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // user is not a member — not an error
		}
		return false, err
	}
	if slices.Contains(roles, m.Role) {
		return true, nil
	}
	return false, nil
}

// HasTeamRole checks whether the user has any of the given team-level roles.
func (s *DefaultOrganizationStore) HasTeamRole(ctx context.Context, userID, teamID string, roles ...string) (bool, error) {
	m, err := s.q.getTeamMember(ctx, teamID, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // user is not in the team — not an error
		}
		return false, err
	}
	if slices.Contains(roles, m.Role) {
		return true, nil
	}
	return false, nil
}

// CanAccessTeam checks whether a user can access a team.
//
// Access requires being a member of the parent organization AND a member of the
// team, with any role in each — custom roles are not second-class here.
func (s *DefaultOrganizationStore) CanAccessTeam(ctx context.Context, userID, teamID string) (bool, error) {
	t, err := s.q.getTeam(ctx, teamID)
	if err != nil {
		return false, err
	}
	if _, err := s.q.getMember(ctx, userID, t.OrganizationID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if _, err := s.q.getTeamMember(ctx, teamID, userID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsOrganizationMember reports whether the user is a member of the organization.
//
// Deprecated: Use HasOrgRole instead. Returns true for any role.
func (s *DefaultOrganizationStore) IsOrganizationMember(ctx context.Context, userID, orgID string) (bool, error) {
	_, err := s.q.getMember(ctx, userID, orgID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsOwnerOrAdmin reports whether the user has the owner or admin role in the organization.
//
// Deprecated: Use HasOrgRole instead.
func (s *DefaultOrganizationStore) IsOwnerOrAdmin(ctx context.Context, userID, orgID string) (bool, error) {
	return s.HasOrgRole(ctx, userID, orgID, orgtypes.RoleOwner, orgtypes.RoleAdmin)
}

// IsOwner reports whether the user is the owner of the organization.
func (s *DefaultOrganizationStore) IsOwner(ctx context.Context, userID, orgID string) (bool, error) {
	return s.HasOrgRole(ctx, userID, orgID, orgtypes.RoleOwner)
}

// UpdateMemberRole updates the role of a member within an organization.
func (s *DefaultOrganizationStore) UpdateMemberRole(ctx context.Context, userID, orgID, role string, updatedAt time.Time) error {
	return s.q.updateMemberRole(ctx, userID, orgID, role, updatedAt.Format(time.RFC3339))
}

// RemoveMember removes a user from an organization.
func (s *DefaultOrganizationStore) RemoveMember(ctx context.Context, userID, orgID string) error {
	return s.q.removeMember(ctx, userID, orgID)
}

// ListOrganizationMembers returns a paginated list of members in an organization.
func (s *DefaultOrganizationStore) ListOrganizationMembers(ctx context.Context, orgID string, offset, limit int) ([]orgtypes.Member, error) {
	off, lim := clampPagination(offset, limit)
	rows, err := s.q.listOrganizationMembers(ctx, orgID, off, lim)
	if err != nil {
		return nil, err
	}
	result := make([]orgtypes.Member, len(rows))
	for i, m := range rows {
		result[i] = buildMember(m)
	}
	return result, nil
}

// CountOrganizationMembers returns the total number of members in an organization.
func (s *DefaultOrganizationStore) CountOrganizationMembers(ctx context.Context, orgID string) (int, error) {
	n, err := s.q.countOrganizationMembers(ctx, orgID)
	return int(n), err
}

// ── Team operations ──────────────────────────────────────────────────────────

// CreateTeam creates a new team within an organization.
func (s *DefaultOrganizationStore) CreateTeam(ctx context.Context, id, orgID, name, description string, createdAt, updatedAt time.Time) error {
	return s.q.createTeam(ctx, id, orgID, name,
		sql.NullString{String: description, Valid: description != ""},
		createdAt.Format(time.RFC3339),
		updatedAt.Format(time.RFC3339),
	)
}

// GetTeam retrieves a team by its ID.
func (s *DefaultOrganizationStore) GetTeam(ctx context.Context, id string) (orgtypes.Team, error) {
	t, err := s.q.getTeam(ctx, id)
	if err != nil {
		return orgtypes.Team{}, err
	}
	return buildTeam(t), nil
}

// ListTeams returns a paginated list of teams within an organization.
func (s *DefaultOrganizationStore) ListTeams(ctx context.Context, orgID string, offset, limit int) ([]orgtypes.Team, error) {
	off, lim := clampPagination(offset, limit)
	rows, err := s.q.listTeams(ctx, orgID, off, lim)
	if err != nil {
		return nil, err
	}
	result := make([]orgtypes.Team, len(rows))
	for i, t := range rows {
		result[i] = buildTeam(t)
	}
	return result, nil
}

// CountTeams returns the total number of teams within an organization.
func (s *DefaultOrganizationStore) CountTeams(ctx context.Context, orgID string) (int, error) {
	n, err := s.q.countTeams(ctx, orgID)
	return int(n), err
}

// UpdateTeam updates the name, description, and updatedAt timestamp for a team.
func (s *DefaultOrganizationStore) UpdateTeam(ctx context.Context, id, name, description string, updatedAt time.Time) error {
	return s.q.updateTeam(ctx, id, name,
		sql.NullString{String: description, Valid: description != ""},
		updatedAt.Format(time.RFC3339),
	)
}

// DeleteTeam removes a team by its ID.
func (s *DefaultOrganizationStore) DeleteTeam(ctx context.Context, id string) error {
	return s.q.deleteTeam(ctx, id)
}

// ── Invitation operations ───────────────────────────────────────────────────

// CreateInvitation stores a new invitation.
func (s *DefaultOrganizationStore) CreateInvitation(ctx context.Context, inv orgtypes.Invitation) error {
	teamID := sql.NullString{Valid: inv.TeamID != nil && *inv.TeamID != ""}
	if teamID.Valid {
		teamID.String = *inv.TeamID
	}
	return s.q.createInvitation(ctx,
		inv.ID,
		inv.OrganizationID,
		teamID.String,
		inv.Email,
		inv.Role,
		inv.InviterID,
		inv.TokenHash,
		inv.Status,
		inv.ExpiresAt.Format(time.RFC3339),
		inv.CreatedAt.Format(time.RFC3339),
		inv.UpdatedAt.Format(time.RFC3339),
	)
}

// GetInvitationByID retrieves an invitation by ID.
func (s *DefaultOrganizationStore) GetInvitationByID(ctx context.Context, id string) (orgtypes.Invitation, error) {
	inv, err := s.q.getInvitationByID(ctx, id)
	if err != nil {
		return orgtypes.Invitation{}, err
	}
	return buildInvitation(inv), nil
}

// GetInvitationByTokenHash retrieves an invitation by token hash.
func (s *DefaultOrganizationStore) GetInvitationByTokenHash(ctx context.Context, tokenHash string) (orgtypes.Invitation, error) {
	inv, err := s.q.getInvitationByTokenHash(ctx, tokenHash)
	if err != nil {
		return orgtypes.Invitation{}, err
	}
	return buildInvitation(inv), nil
}

// ListInvitations returns a paginated list of invitations for an org.
func (s *DefaultOrganizationStore) ListInvitations(ctx context.Context, orgID string, teamID string, offset, limit int) ([]orgtypes.Invitation, error) {
	off, lim := clampPagination(offset, limit)
	rows, err := s.q.listInvitations(ctx, orgID, teamID, off, lim)
	if err != nil {
		return nil, err
	}
	result := make([]orgtypes.Invitation, len(rows))
	for i, inv := range rows {
		result[i] = buildInvitation(inv)
	}
	return result, nil
}

// CountInvitations returns the total number of invitations matching filters.
func (s *DefaultOrganizationStore) CountInvitations(ctx context.Context, orgID string, teamID string) (int, error) {
	n, err := s.q.countInvitations(ctx, orgID, teamID)
	return int(n), err
}

// UpdateInvitationStatus updates the status of an invitation.
func (s *DefaultOrganizationStore) UpdateInvitationStatus(ctx context.Context, id, status string, updatedAt time.Time) error {
	return s.q.updateInvitationStatus(ctx, id, status, updatedAt.Format(time.RFC3339))
}

// DeleteInvitation deletes an invitation by ID.
func (s *DefaultOrganizationStore) DeleteInvitation(ctx context.Context, id string) error {
	return s.q.deleteInvitation(ctx, id)
}

// ── Team member operations ───────────────────────────────────────────────────

// CreateTeamMember adds a user as a member of a team with the given role.
func (s *DefaultOrganizationStore) CreateTeamMember(ctx context.Context, id, teamID, userID, role string, createdAt, updatedAt time.Time) error {
	return s.q.createTeamMember(ctx, id, teamID, userID, role,
		createdAt.Format(time.RFC3339),
		updatedAt.Format(time.RFC3339),
	)
}

// GetTeamMember retrieves the membership record for a user in a team.
func (s *DefaultOrganizationStore) GetTeamMember(ctx context.Context, teamID, userID string) (orgtypes.TeamMember, error) {
	m, err := s.q.getTeamMember(ctx, teamID, userID)
	if err != nil {
		return orgtypes.TeamMember{}, err
	}
	return buildTeamMember(m), nil
}

// ListTeamMembers returns a paginated list of members in a team.
func (s *DefaultOrganizationStore) ListTeamMembers(ctx context.Context, teamID string, offset, limit int) ([]orgtypes.TeamMember, error) {
	off, lim := clampPagination(offset, limit)
	rows, err := s.q.listTeamMembers(ctx, teamID, off, lim)
	if err != nil {
		return nil, err
	}
	result := make([]orgtypes.TeamMember, len(rows))
	for i, m := range rows {
		result[i] = buildTeamMember(m)
	}
	return result, nil
}

// CountTeamMembers returns the total number of members in a team.
func (s *DefaultOrganizationStore) CountTeamMembers(ctx context.Context, teamID string) (int, error) {
	n, err := s.q.countTeamMembers(ctx, teamID)
	return int(n), err
}

// UpdateTeamMemberRole updates the role of a member within a team.
func (s *DefaultOrganizationStore) UpdateTeamMemberRole(ctx context.Context, teamID, userID, role string, updatedAt time.Time) error {
	return s.q.updateTeamMemberRole(ctx, teamID, userID, role, updatedAt.Format(time.RFC3339))
}

// RemoveTeamMember removes a user from a team.
func (s *DefaultOrganizationStore) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	return s.q.removeTeamMember(ctx, teamID, userID)
}

// ── Builders ─────────────────────────────────────────────────────────────────

// buildOrganization converts an orgRow returned by the querier into the public
// orgtypes.Organization domain model.
func buildOrganization(o orgRow) orgtypes.Organization {
	return orgtypes.Organization{
		ID:        o.ID,
		Name:      o.Name,
		Slug:      o.Slug,
		CreatedAt: parseOrgTime(o.CreatedAt),
		UpdatedAt: parseOrgTime(o.UpdatedAt),
	}
}

// buildMember converts a memberRow returned by the querier into the public
// orgtypes.Member domain model.
func buildMember(m memberRow) orgtypes.Member {
	return orgtypes.Member{
		ID:             m.ID,
		UserID:         m.UserID,
		OrganizationID: m.OrganizationID,
		Role:           m.Role,
		CreatedAt:      parseOrgTime(m.CreatedAt),
		UpdatedAt:      parseOrgTime(m.UpdatedAt),
	}
}

// buildTeam converts a teamRow returned by the querier into the public
// orgtypes.Team domain model. A NULL description column is treated as an empty string.
func buildTeam(t teamRow) orgtypes.Team {
	desc := ""
	if t.Description.Valid {
		desc = t.Description.String
	}
	return orgtypes.Team{
		ID:             t.ID,
		OrganizationID: t.OrganizationID,
		Name:           t.Name,
		Description:    desc,
		CreatedAt:      parseOrgTime(t.CreatedAt),
		UpdatedAt:      parseOrgTime(t.UpdatedAt),
	}
}

// buildTeamMember converts a teamMemberRow returned by the querier into the public
// orgtypes.TeamMember domain model.
func buildTeamMember(m teamMemberRow) orgtypes.TeamMember {
	return orgtypes.TeamMember{
		ID:        m.ID,
		TeamID:    m.TeamID,
		UserID:    m.UserID,
		Role:      m.Role,
		CreatedAt: parseOrgTime(m.CreatedAt),
		UpdatedAt: parseOrgTime(m.UpdatedAt),
	}
}

// buildInvitation converts an invitationRow returned by the querier into the public
// orgtypes.Invitation domain model. A NULL team_id column is treated as an
// unset (nil) pointer.
func buildInvitation(inv invitationRow) orgtypes.Invitation {
	result := orgtypes.Invitation{
		ID:             inv.ID,
		OrganizationID: inv.OrganizationID,
		Email:          inv.Email,
		Role:           inv.Role,
		InviterID:      inv.InviterID,
		TokenHash:      inv.TokenHash,
		Status:         inv.Status,
		ExpiresAt:      parseOrgTime(inv.ExpiresAt),
		CreatedAt:      parseOrgTime(inv.CreatedAt),
		UpdatedAt:      parseOrgTime(inv.UpdatedAt),
	}
	if inv.TeamID.Valid {
		result.TeamID = &inv.TeamID.String
	}
	return result
}

// parseOrgTime parses an RFC3339 timestamp string into time.Time. An empty
// string (nullable column) maps to the zero time; a malformed value is logged
// and also maps to the zero time.
func parseOrgTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		log.Printf("aegis/organizations: ignoring malformed RFC3339 timestamp %q: %v", s, err)
		return time.Time{}
	}
	return t
}

// ListMemberPermissionOverrides returns the member's permission overrides.
func (s *DefaultOrganizationStore) ListMemberPermissionOverrides(ctx context.Context, orgID, userID string) ([]orgtypes.MemberPermissionOverride, error) {
	rows, err := s.q.listMemberPermissionOverrides(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]orgtypes.MemberPermissionOverride, 0, len(rows))
	for _, r := range rows {
		out = append(out, orgtypes.MemberPermissionOverride{
			ID:             r.ID,
			OrganizationID: r.OrganizationID,
			UserID:         r.UserID,
			Permission:     r.Permission,
			Effect:         orgtypes.PermissionEffect(r.Effect),
			CreatedAt:      parseOrgTime(r.CreatedAt),
			UpdatedAt:      parseOrgTime(r.UpdatedAt),
		})
	}
	return out, nil
}

// CreateMemberPermissionOverride stores one override.
func (s *DefaultOrganizationStore) CreateMemberPermissionOverride(ctx context.Context, o orgtypes.MemberPermissionOverride) error {
	return s.q.createMemberPermissionOverride(ctx, o.ID, o.OrganizationID, o.UserID, o.Permission, string(o.Effect),
		o.CreatedAt.UTC().Format(time.RFC3339), o.UpdatedAt.UTC().Format(time.RFC3339))
}

// DeleteMemberPermissionOverrides removes every override for a member.
func (s *DefaultOrganizationStore) DeleteMemberPermissionOverrides(ctx context.Context, orgID, userID string) error {
	return s.q.deleteMemberPermissionOverrides(ctx, orgID, userID)
}

// ListOrganizationRoles returns the organization's persisted custom roles.
func (s *DefaultOrganizationStore) ListOrganizationRoles(ctx context.Context, orgID string) ([]orgtypes.OrganizationRole, error) {
	rows, err := s.q.listOrganizationRoles(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]orgtypes.OrganizationRole, 0, len(rows))
	for _, r := range rows {
		out = append(out, buildOrganizationRole(r))
	}
	return out, nil
}

// GetOrganizationRole retrieves a persisted role by name.
func (s *DefaultOrganizationStore) GetOrganizationRole(ctx context.Context, orgID, name string) (orgtypes.OrganizationRole, error) {
	r, err := s.q.getOrganizationRole(ctx, orgID, name)
	if err != nil {
		return orgtypes.OrganizationRole{}, err
	}
	return buildOrganizationRole(r), nil
}

// CreateOrganizationRole stores a new custom role.
func (s *DefaultOrganizationStore) CreateOrganizationRole(ctx context.Context, role orgtypes.OrganizationRole) error {
	perms, err := marshalPermissions(role.Permissions)
	if err != nil {
		return err
	}
	return s.q.createOrganizationRole(ctx, role.ID, role.OrganizationID, role.Name, perms,
		role.CreatedAt.UTC().Format(time.RFC3339), role.UpdatedAt.UTC().Format(time.RFC3339))
}

// UpdateOrganizationRole replaces a role's permissions.
func (s *DefaultOrganizationStore) UpdateOrganizationRole(ctx context.Context, role orgtypes.OrganizationRole) error {
	perms, err := marshalPermissions(role.Permissions)
	if err != nil {
		return err
	}
	return s.q.updateOrganizationRole(ctx, role.OrganizationID, role.Name, perms, role.UpdatedAt.UTC().Format(time.RFC3339))
}

// DeleteOrganizationRole removes a custom role.
func (s *DefaultOrganizationStore) DeleteOrganizationRole(ctx context.Context, orgID, name string) error {
	return s.q.deleteOrganizationRole(ctx, orgID, name)
}

// CountOrganizationMembersWithRole counts members holding a role.
func (s *DefaultOrganizationStore) CountOrganizationMembersWithRole(ctx context.Context, orgID, role string) (int, error) {
	n, err := s.q.countOrganizationMembersWithRole(ctx, orgID, role)
	return int(n), err
}

// buildOrganizationRole converts a canonical row into the public model.
func buildOrganizationRole(r organizationRoleRow) orgtypes.OrganizationRole {
	var perms []string
	if err := json.Unmarshal([]byte(r.Permissions), &perms); err != nil {
		perms = []string{}
	}
	if perms == nil {
		perms = []string{}
	}
	return orgtypes.OrganizationRole{
		ID:             r.ID,
		OrganizationID: r.OrganizationID,
		Name:           r.Name,
		Permissions:    perms,
		CreatedAt:      parseOrgTime(r.CreatedAt),
		UpdatedAt:      parseOrgTime(r.UpdatedAt),
	}
}

func marshalPermissions(perms []string) (string, error) {
	if perms == nil {
		perms = []string{}
	}
	b, err := json.Marshal(perms)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
