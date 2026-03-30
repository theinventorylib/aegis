package types

import (
	"context"
	"time"
)

// OrganizationStore defines the interface for organization storage operations.
//
// This interface abstracts database operations for multi-tenant organization
// management, including organizations, members, teams, and team members.
//
// Thread Safety:
// Implementations must be safe for concurrent use from multiple goroutines.
//
// Transaction Considerations:
// Several operations should be atomic (e.g., CreateOrganization + CreateMember).
// Implementations should handle this appropriately.
type OrganizationStore interface {
	// ========== Organization operations ==========

	// CreateOrganization creates a new organization.
	//
	// Parameters:
	//   - ctx: Request context
	//   - id: Unique organization ID
	//   - name: Organization display name
	//   - slug: URL-friendly identifier (must be unique)
	//   - createdAt, updatedAt: Timestamps
	//
	// Returns:
	//   - error: Duplicate slug or database error
	CreateOrganization(ctx context.Context, id, name, slug string, createdAt, updatedAt time.Time) error

	// GetOrganization retrieves an organization by ID.
	GetOrganization(ctx context.Context, id string) (Organization, error)

	// GetOrganizationBySlug retrieves an organization by slug.
	// Used for URL routing like /org/acme-corp.
	GetOrganizationBySlug(ctx context.Context, slug string) (Organization, error)

	// UpdateOrganization updates organization name and/or slug.
	UpdateOrganization(ctx context.Context, id, name, slug string, updatedAt time.Time) error

	// DeleteOrganization deletes an organization.
	// Should cascade delete members, teams, and team members.
	DeleteOrganization(ctx context.Context, id string, updatedAt time.Time) error

	// ListUserOrganizations retrieves all organizations a user is a member of.
	ListUserOrganizations(ctx context.Context, userID string, offset, limit int) ([]Organization, error)

	CountUserOrganizations(ctx context.Context, userID string) (int, error)

	// ========== Member operations ==========

	// CreateMember adds a user to an organization with a role.
	//
	// Valid roles: "owner", "admin", "member"
	CreateMember(ctx context.Context, id, userID, orgID, role string, createdAt, updatedAt time.Time) error

	// GetMember retrieves a user's membership in an organization.
	GetMember(ctx context.Context, userID, orgID string) (Member, error)

	// IsOrganizationMember checks if a user is a member (any role).
	// Used by middleware for access control.
	IsOrganizationMember(ctx context.Context, userID, orgID string) (bool, error)

	// IsOwnerOrAdmin checks if a user has admin privileges.
	// Returns true if role is "owner" or "admin".
	IsOwnerOrAdmin(ctx context.Context, userID, orgID string) (bool, error)

	// IsOwner checks if a user is the organization owner.
	// Returns true only if role is "owner".
	IsOwner(ctx context.Context, userID, orgID string) (bool, error)

	// UpdateMemberRole changes a member's role.
	// Caller should verify permissions before calling.
	UpdateMemberRole(ctx context.Context, userID, orgID, role string, updatedAt time.Time) error

	// RemoveMember removes a user from an organization.
	// Should also remove from all teams in the organization.
	RemoveMember(ctx context.Context, userID, orgID string) error

	// ListOrganizationMembers retrieves all members of an organization.
	ListOrganizationMembers(ctx context.Context, orgID string, offset, limit int) ([]Member, error)

	CountOrganizationMembers(ctx context.Context, orgID string) (int, error)

	// ========== Team operations ==========

	// CreateTeam creates a new team within an organization.
	CreateTeam(ctx context.Context, id, orgID, name, description string, createdAt, updatedAt time.Time) error

	// GetTeam retrieves a team by ID.
	GetTeam(ctx context.Context, id string) (Team, error)

	// ListTeams retrieves all teams in an organization.
	ListTeams(ctx context.Context, orgID string, offset, limit int) ([]Team, error)

	CountTeams(ctx context.Context, orgID string) (int, error)

	// UpdateTeam updates team name and/or description.
	UpdateTeam(ctx context.Context, id, name, description string, updatedAt time.Time) error

	// DeleteTeam deletes a team.
	// Should cascade delete team members.
	DeleteTeam(ctx context.Context, id string) error

	// ========== Team Member operations ==========

	// CreateTeamMember adds a user to a team with a role.
	//
	// Valid roles: "lead", "member"
	// User must be an organization member.
	CreateTeamMember(ctx context.Context, id, teamID, userID, role string, createdAt, updatedAt time.Time) error

	// GetTeamMember retrieves a user's team membership.
	GetTeamMember(ctx context.Context, teamID, userID string) (TeamMember, error)

	// ListTeamMembers retrieves all members of a team.
	ListTeamMembers(ctx context.Context, teamID string, offset, limit int) ([]TeamMember, error)

	CountTeamMembers(ctx context.Context, teamID string) (int, error)

	// UpdateTeamMemberRole changes a team member's role.
	UpdateTeamMemberRole(ctx context.Context, teamID, userID, role string, updatedAt time.Time) error

	// RemoveTeamMember removes a user from a team.
	RemoveTeamMember(ctx context.Context, teamID, userID string) error
}
