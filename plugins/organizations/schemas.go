package organizations

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// slugPattern defines the allowed format for organization slugs.
// Slugs must be lowercase alphanumeric with hyphens, suitable for URLs.
// Examples: "acme-corp", "tech-startup", "my-org-123"
var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// ========== Organization Request Schemas ==========

// CreateOrganizationRequest represents a request to create an organization.
//
// Validation Rules:
//   - name: Required, 1-100 characters (organization display name)
//   - slug: Required, 3-50 characters, lowercase alphanumeric + hyphens only
//
// Example:
//
//	{
//	  "name": "Acme Corporation",
//	  "slug": "acme-corp"
//	}
type CreateOrganizationRequest struct {
	Name string `json:"name"` // Organization display name
	Slug string `json:"slug"` // URL-friendly identifier (must be unique)
}

// Validate validates the create organization request.
//
// Returns:
//   - error: Validation error if name or slug is invalid
func (r CreateOrganizationRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&r.Slug, validation.Required, validation.Match(slugPattern), validation.Length(3, 50)),
	)
}

// UpdateOrganizationRequest represents a request to update an organization.
//
// Validation Rules:
//   - name: Required, 1-100 characters
//   - slug: Required, 3-50 characters, lowercase alphanumeric + hyphens only
//
// Note: Both fields must be provided even if only updating one.
// The handler will apply the new values.
type UpdateOrganizationRequest struct {
	Name string `json:"name"` // Updated organization name
	Slug string `json:"slug"` // Updated URL-friendly identifier
}

// Validate validates the update organization request.
func (r UpdateOrganizationRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&r.Slug, validation.Required, validation.Match(slugPattern), validation.Length(3, 50)),
	)
}

// AddOrganizationMemberRequest represents a request to add a member to an organization.
//
// Validation Rules:
//   - userId: Required (must be a valid user ID in the system)
//   - role: Required, must be "admin" or "member" ("owner" cannot be assigned this way)
//
// Example:
//
//	{
//	  "userId": "user_xyz789",
//	  "role": "admin"
//	}
//
// Security Note:
// The "owner" role cannot be assigned via this endpoint to prevent privilege escalation.
// Ownership is assigned during organization creation or via explicit transfer (if implemented).
type AddOrganizationMemberRequest struct {
	UserID string `json:"userId"` // User ID to add
	Role   string `json:"role"`   // Member role ("admin" or "member")
}

// Validate validates the add organization member request.
func (r AddOrganizationMemberRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserID, validation.Required),
		validation.Field(&r.Role, validation.Required, validation.In("admin", "member")),
	)
}

// UpdateMemberRoleRequest represents a request to update a member's role.
//
// Validation Rules:
//   - role: Required, must be "admin" or "member"
//
// Example:
//
//	{
//	  "role": "admin"
//	}
//
// Security Note:
// Cannot update to "owner" role via this endpoint. Ownership transfer requires
// a separate flow with additional safeguards.
type UpdateMemberRoleRequest struct {
	Role string `json:"role"` // New role ("admin" or "member")
}

// Validate validates the update member role request.
func (r UpdateMemberRoleRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role, validation.Required, validation.In("admin", "member")),
	)
}

// ========== Team Request Schemas ==========

// CreateTeamRequest represents a request to create a team within an organization.
//
// Validation Rules:
//   - name: Required, 1-100 characters (team display name)
//   - description: Optional, max 500 characters (team purpose)
//
// Example:
//
//	{
//	  "name": "Engineering",
//	  "description": "Software development team"
//	}
type CreateTeamRequest struct {
	Name        string `json:"name"`        // Team display name
	Description string `json:"description"` // Team purpose/description
}

// Validate validates the create team request.
func (r CreateTeamRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&r.Description, validation.Length(0, 500)),
	)
}

// UpdateTeamRequest represents a request to update a team.
type UpdateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Validate validates the update team request.
func (r UpdateTeamRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&r.Description, validation.Length(0, 500)),
	)
}

// AddTeamMemberRequest represents a request to add a member to a team.
type AddTeamMemberRequest struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

// Validate validates the add team member request.
func (r AddTeamMemberRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserID, validation.Required),
		validation.Field(&r.Role, validation.Required, validation.In("lead", "member")),
	)
}

// UpdateTeamMemberRoleRequest represents a request to update a team member's role.
type UpdateTeamMemberRoleRequest struct {
	Role string `json:"role"`
}

// Validate validates the update team member role request.
func (r UpdateTeamMemberRoleRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role, validation.Required, validation.In("lead", "member")),
	)
}
