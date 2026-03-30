// Package types defines the domain models and types used by the organizations plugin.
package types

import "time"

// Organization represents a workspace, company, or tenant in a multi-tenant system.
//
// Organizations are the top-level container for resources in a multi-tenant application.
// Each organization has members with roles and can contain teams for hierarchical access.
//
// Database Table: organization
// Unique Constraint: slug (for URL-friendly access like /org/acme-corp)
//
// Example:
//
//	{
//	  "id": "org_abc123",
//	  "name": "Acme Corporation",
//	  "slug": "acme-corp",
//	  "createdAt": "2024-01-01T00:00:00Z",
//	  "updatedAt": "2024-01-01T00:00:00Z"
//	}
type Organization struct {
	ID        string    `json:"id"`        // Unique organization identifier
	Name      string    `json:"name"`      // Display name (e.g., "Acme Corporation")
	Slug      string    `json:"slug"`      // URL-friendly identifier (e.g., "acme-corp")
	CreatedAt time.Time `json:"createdAt"` // When the organization was created
	UpdatedAt time.Time `json:"updatedAt"` // Last update timestamp
}

// Member represents a user's membership in an organization with a role.
//
// Members link users to organizations with role-based permissions. The role
// determines what actions the user can perform within the organization.
//
// Database Table: members
// Unique Constraint: (user_id, organization_id)
// Foreign Keys: user_id → auth.users.id, organization_id → organization.id
//
// Roles:
//   - "owner": Full control, can delete organization and manage all settings
//   - "admin": Can invite/remove members, create teams, but cannot delete org
//   - "member": Read access to organization resources, cannot manage
//
// Example:
//
//	{
//	  "id": "mem_xyz789",
//	  "userId": "user_123",
//	  "organizationId": "org_abc123",
//	  "role": "admin",
//	  "createdAt": "2024-01-01T00:00:00Z",
//	  "updatedAt": "2024-01-01T00:00:00Z"
//	}
type Member struct {
	ID             string    `json:"id"`             // Unique membership identifier
	UserID         string    `json:"userId"`         // User ID (foreign key)
	OrganizationID string    `json:"organizationId"` // Organization ID (foreign key)
	Role           string    `json:"role"`           // Member role ("owner", "admin", "member")
	CreatedAt      time.Time `json:"createdAt"`      // When the user joined the organization
	UpdatedAt      time.Time `json:"updatedAt"`      // Last role update timestamp
}

// Team represents a group within an organization.
//
// Teams provide hierarchical organization within a tenant, allowing for
// department-level or project-level access control. Teams belong to a single
// organization and can have members with team-specific roles.
//
// Database Table: team
// Foreign Key: organization_id → organization.id
//
// Use Cases:
//   - Department teams ("Engineering", "Sales", "Marketing")
//   - Project teams ("Product Launch", "Q4 Initiative")
//   - Access control groups ("Beta Testers", "Premium Features")
//
// Example:
//
//	{
//	  "id": "team_def456",
//	  "organizationId": "org_abc123",
//	  "name": "Engineering",
//	  "description": "Software development team",
//	  "createdAt": "2024-01-01T00:00:00Z",
//	  "updatedAt": "2024-01-01T00:00:00Z"
//	}
type Team struct {
	ID             string    `json:"id"`             // Unique team identifier
	OrganizationID string    `json:"organizationId"` // Parent organization ID
	Name           string    `json:"name"`           // Team display name
	Description    string    `json:"description"`    // Team purpose/description
	CreatedAt      time.Time `json:"createdAt"`      // When the team was created
	UpdatedAt      time.Time `json:"updatedAt"`      // Last update timestamp
}

// TeamMember represents a user's membership in a team with a role.
//
// Team members must also be members of the parent organization. Teams provide
// an additional layer of access control within an organization.
//
// Database Table: team_member
// Unique Constraint: (team_id, user_id)
// Foreign Keys: team_id → team.id, user_id → auth.users.id
//
// Roles:
//   - "lead": Can manage team members and settings
//   - "member": Participate in team activities
//
// Example:
//
//	{
//	  "id": "tmem_ghi789",
//	  "teamId": "team_def456",
//	  "userId": "user_123",
//	  "role": "lead",
//	  "createdAt": "2024-01-01T00:00:00Z",
//	  "updatedAt": "2024-01-01T00:00:00Z"
//	}
type TeamMember struct {
	ID        string    `json:"id"`        // Unique team membership identifier
	TeamID    string    `json:"teamId"`    // Team ID (foreign key)
	UserID    string    `json:"userId"`    // User ID (foreign key)
	Role      string    `json:"role"`      // Team role ("lead", "member")
	CreatedAt time.Time `json:"createdAt"` // When the user joined the team
	UpdatedAt time.Time `json:"updatedAt"` // Last role update timestamp
}
