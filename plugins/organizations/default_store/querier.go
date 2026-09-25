package defaultstore

// querier.go — the single internal DB interface for the organizations plugin.
//
// All unexported. No dialect-generated types cross this file — only standard
// Go primitives and database/sql types. The three dialect translators in
// postgres.go, mysql.go, and sqlite.go each implement this interface.
//
// Pagination parameters use int32 (the narrower type). SQLite translators
// widen to int64 internally.

import (
	"context"
	"database/sql"
)

// Canonical row types — dialect-neutral, owned by this package.
// All time fields are stored as RFC3339 strings and parsed in store.go.

type orgRow struct {
	ID, Name, Slug, CreatedAt, UpdatedAt string
}

type memberRow struct {
	ID, UserID, OrganizationID, Role, CreatedAt, UpdatedAt string
}

type teamRow struct {
	ID, OrganizationID, Name, CreatedAt, UpdatedAt string
	Description                                    sql.NullString
}

type teamMemberRow struct {
	ID, TeamID, UserID, Role, CreatedAt, UpdatedAt string
}

type invitationRow struct {
	ID, OrganizationID, Email, Role, InviterID, TokenHash, Status, ExpiresAt, CreatedAt, UpdatedAt string
	TeamID                                                                                         sql.NullString
}

type permissionOverrideRow struct {
	ID, OrganizationID, UserID, Permission, Effect, CreatedAt, UpdatedAt string
}

type organizationRoleRow struct {
	ID, OrganizationID, Name, Permissions, CreatedAt, UpdatedAt string
}

// listOrgRow is the reduced row returned by ListUserOrganizations
// (same fields as orgRow but kept separate for clarity).
type listOrgRow = orgRow

// querier is the one internal interface all store methods use.
// The dialect is chosen exactly once in NewDefaultOrganizationStore; everything
// else calls through here and is dialect-agnostic.
type querier interface {
	// Organization queries
	createOrganization(ctx context.Context, id, name, slug, createdAt, updatedAt string) error
	getOrganization(ctx context.Context, id string) (orgRow, error)
	getOrganizationBySlug(ctx context.Context, slug string) (orgRow, error)
	updateOrganization(ctx context.Context, id, name, slug, updatedAt string) error
	deleteOrganization(ctx context.Context, id, updatedAt string) error
	listUserOrganizations(ctx context.Context, userID string, offset, limit int32) ([]listOrgRow, error)
	countUserOrganizations(ctx context.Context, userID string) (int64, error)

	// Member queries
	createMember(ctx context.Context, id, userID, orgID, role, createdAt, updatedAt string) error
	getMember(ctx context.Context, userID, orgID string) (memberRow, error)
	isOrganizationMember(ctx context.Context, userID, orgID string) (bool, error)
	isOwnerOrAdmin(ctx context.Context, userID, orgID string) (bool, error)
	isOwner(ctx context.Context, userID, orgID string) (bool, error)
	updateMemberRole(ctx context.Context, userID, orgID, role, updatedAt string) error
	removeMember(ctx context.Context, userID, orgID string) error
	listOrganizationMembers(ctx context.Context, orgID string, offset, limit int32) ([]memberRow, error)
	countOrganizationMembers(ctx context.Context, orgID string) (int64, error)

	// Team queries
	createTeam(ctx context.Context, id, orgID, name string, description sql.NullString, createdAt, updatedAt string) error
	getTeam(ctx context.Context, id string) (teamRow, error)
	listTeams(ctx context.Context, orgID string, offset, limit int32) ([]teamRow, error)
	countTeams(ctx context.Context, orgID string) (int64, error)
	updateTeam(ctx context.Context, id, name string, description sql.NullString, updatedAt string) error
	deleteTeam(ctx context.Context, id string) error

	// Invitation queries
	createInvitation(ctx context.Context, id, organizationID, teamID, email, role, inviterID, tokenHash, status, expiresAt, createdAt, updatedAt string) error
	getInvitationByID(ctx context.Context, id string) (invitationRow, error)
	getInvitationByTokenHash(ctx context.Context, tokenHash string) (invitationRow, error)
	listInvitations(ctx context.Context, orgID string, teamID string, offset, limit int32) ([]invitationRow, error)
	countInvitations(ctx context.Context, orgID string, teamID string) (int64, error)
	updateInvitationStatus(ctx context.Context, id, status, updatedAt string) error
	deleteInvitation(ctx context.Context, id string) error

	// Team member queries
	createTeamMember(ctx context.Context, id, teamID, userID, role, createdAt, updatedAt string) error
	getTeamMember(ctx context.Context, teamID, userID string) (teamMemberRow, error)
	listTeamMembers(ctx context.Context, teamID string, offset, limit int32) ([]teamMemberRow, error)
	countTeamMembers(ctx context.Context, teamID string) (int64, error)
	updateTeamMemberRole(ctx context.Context, teamID, userID, role, updatedAt string) error
	removeTeamMember(ctx context.Context, teamID, userID string) error

	// Permission override queries
	listMemberPermissionOverrides(ctx context.Context, orgID, userID string) ([]permissionOverrideRow, error)
	createMemberPermissionOverride(ctx context.Context, id, orgID, userID, permission, effect, createdAt, updatedAt string) error
	deleteMemberPermissionOverrides(ctx context.Context, orgID, userID string) error

	// Role definition queries
	listOrganizationRoles(ctx context.Context, orgID string) ([]organizationRoleRow, error)
	getOrganizationRole(ctx context.Context, orgID, name string) (organizationRoleRow, error)
	createOrganizationRole(ctx context.Context, id, orgID, name, permissions, createdAt, updatedAt string) error
	updateOrganizationRole(ctx context.Context, orgID, name, permissions, updatedAt string) error
	deleteOrganizationRole(ctx context.Context, orgID, name string) error
	countOrganizationMembersWithRole(ctx context.Context, orgID, role string) (int64, error)
}
