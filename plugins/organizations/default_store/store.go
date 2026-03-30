// Package defaultstore implements the SQL-backed default store for the organizations plugin.
//
// Dialect selection happens once in NewDefaultOrganizationStore; all methods are
// dialect-agnostic and delegate to the unexported querier interface.
package defaultstore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/theinventorylib/aegis/plugins"
	orgtypes "github.com/theinventorylib/aegis/plugins/organizations/types"
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

// IsOrganizationMember reports whether the user is a member of the organization.
func (s *DefaultOrganizationStore) IsOrganizationMember(ctx context.Context, userID, orgID string) (bool, error) {
	return s.q.isOrganizationMember(ctx, userID, orgID)
}

// IsOwnerOrAdmin reports whether the user has the owner or admin role in the organization.
func (s *DefaultOrganizationStore) IsOwnerOrAdmin(ctx context.Context, userID, orgID string) (bool, error) {
	return s.q.isOwnerOrAdmin(ctx, userID, orgID)
}

// IsOwner reports whether the user is the owner of the organization.
func (s *DefaultOrganizationStore) IsOwner(ctx context.Context, userID, orgID string) (bool, error) {
	return s.q.isOwner(ctx, userID, orgID)
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

// parseOrgTime parses an RFC3339 timestamp string into time.Time.
// Returns the zero value on parse failure.
func parseOrgTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
