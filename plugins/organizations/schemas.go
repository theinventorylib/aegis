package organizations

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// Organization Request Schemas

// CreateOrganizationRequest represents a request to create an organization.
type CreateOrganizationRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Validate validates the create organization request.
func (r CreateOrganizationRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&r.Slug, validation.Required, validation.Match(slugPattern), validation.Length(3, 50)),
	)
}

// UpdateOrganizationRequest represents a request to update an organization.
type UpdateOrganizationRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Validate validates the update organization request.
func (r UpdateOrganizationRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&r.Slug, validation.Required, validation.Match(slugPattern), validation.Length(3, 50)),
	)
}

// AddOrganizationMemberRequest represents a request to add a member to an organization.
type AddOrganizationMemberRequest struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
}

// Validate validates the add organization member request.
func (r AddOrganizationMemberRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.UserID, validation.Required),
		validation.Field(&r.Role, validation.Required, validation.In("admin", "member")),
	)
}

// UpdateMemberRoleRequest represents a request to update a member's role.
type UpdateMemberRoleRequest struct {
	Role string `json:"role"`
}

// Validate validates the update member role request.
func (r UpdateMemberRoleRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role, validation.Required, validation.In("admin", "member")),
	)
}

// Team Request Schemas

// CreateTeamRequest represents a request to create a team.
type CreateTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
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
