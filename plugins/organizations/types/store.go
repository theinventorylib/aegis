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

	// HasOrgRole checks whether the user has any of the given org-level roles.
	// Use this instead of the deprecated IsOrganizationMember / IsOwnerOrAdmin
	// methods for new code.
	//
	// Example:
	//
	//	hasAdmin, _ := store.HasOrgRole(ctx, userID, orgID, RoleOwner, RoleAdmin)
	//	hasAny,  _ := store.HasOrgRole(ctx, userID, orgID, RoleOwner, RoleAdmin, RoleMember)
	HasOrgRole(ctx context.Context, userID, orgID string, roles ...string) (bool, error)

	// HasTeamRole checks whether the user has any of the given team-level roles.
	//
	// Example:
	//
	//	canLead, _ := store.HasTeamRole(ctx, userID, teamID, RoleTeamLead)
	HasTeamRole(ctx context.Context, userID, teamID string, roles ...string) (bool, error)

	// CanAccessTeam checks whether a user can access a team.
	// A user can access a team if they are a member of the parent organization
	// AND a member of the specific team (with any team role).
	CanAccessTeam(ctx context.Context, userID, teamID string) (bool, error)

	// Deprecated: Use HasOrgRole instead.
	IsOrganizationMember(ctx context.Context, userID, orgID string) (bool, error)

	// Deprecated: Use HasOrgRole instead.
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

	// ========== Invitation operations ==========

	// CreateInvitation stores a new invitation.
	//
	// The TokenHash field must contain the SHA-256 hash of the raw token.
	// The raw token itself is never stored — it is returned at creation time
	// and must be delivered to the invitee (the raw token is returned in the API response).
	CreateInvitation(ctx context.Context, inv Invitation) error

	// GetInvitationByID retrieves an invitation by its ID.
	GetInvitationByID(ctx context.Context, id string) (Invitation, error)

	// GetInvitationByTokenHash retrieves an invitation by its token hash.
	// Used on accept/decline where the caller provides the raw token.
	GetInvitationByTokenHash(ctx context.Context, tokenHash string) (Invitation, error)

	// ListInvitations returns a paginated list of invitations for an org,
	// optionally filtered to a specific team.
	//
	// Parameters:
	//   - orgID: Organization ID (required)
	//   - teamID: If non-empty, only return invitations for this team.
	//     Pass empty string to return all org-level invites.
	//   - offset, limit: Pagination
	ListInvitations(ctx context.Context, orgID string, teamID string, offset, limit int) ([]Invitation, error)

	// CountInvitations returns the total number of invitations matching filters.
	CountInvitations(ctx context.Context, orgID string, teamID string) (int, error)

	// UpdateInvitationStatus updates the status of an invitation.
	UpdateInvitationStatus(ctx context.Context, id, status string, updatedAt time.Time) error

	// DeleteInvitation removes an invitation by its ID.
	DeleteInvitation(ctx context.Context, id string) error
}
