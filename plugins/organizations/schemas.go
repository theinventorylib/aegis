package organizations

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	orgtypes "github.com/theinventorylib/aegis/plugins/organizations/types"
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
		validation.Field(&r.Role, validation.Required, validation.In(orgtypes.RoleAdmin, orgtypes.RoleMember)),
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
		validation.Field(&r.Role, validation.Required, validation.In(orgtypes.RoleAdmin, orgtypes.RoleMember)),
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
		validation.Field(&r.Role, validation.Required, validation.In(orgtypes.RoleTeamLead, orgtypes.RoleMember)),
	)
}

// UpdateTeamMemberRoleRequest represents a request to update a team member's role.
type UpdateTeamMemberRoleRequest struct {
	Role string `json:"role"`
}

// Validate validates the update team member role request.
func (r UpdateTeamMemberRoleRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Role, validation.Required, validation.In(orgtypes.RoleTeamLead, orgtypes.RoleMember)),
	)
}

// ========== Invitation Request Schemas ==========

// CreateInvitationRequest represents a request to invite a user to an organization or team.
//
// Validation Rules:
//   - email: Required, valid email format
//   - role: Required, must be "admin" or "member"
//   - teamId: Optional (omit for org-level invitation)
//   - expiresIn: Optional duration string (e.g. "72h"), defaults to 7 days
//
// Example Org-Level:
//
//	{
//	  "email": "newuser@example.com",
//	  "role": "member"
//	}
//
// Example Team-Level:
//
//	{
//	  "email": "dev@example.com",
//	  "role": "member",
//	  "teamId": "team_def456"
//	}
type CreateInvitationRequest struct {
	Email     string  `json:"email"`               // Invitee email address
	Role      string  `json:"role"`                // Role on acceptance ("admin" or "member")
	TeamID    *string `json:"teamId,omitempty"`    // Optional team ID (omit for org-level)
	ExpiresIn string  `json:"expiresIn,omitempty"` // Optional duration (e.g. "72h"), default 168h
}

// Validate validates the create invitation request.
func (r CreateInvitationRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Email, validation.Required, validation.Length(1, 255)),
		validation.Field(&r.Role, validation.Required, validation.In(orgtypes.RoleAdmin, orgtypes.RoleMember)),
		validation.Field(&r.TeamID, validation.When(r.TeamID != nil && *r.TeamID != "", validation.Length(1, 255))),
	)
}

// AcceptInvitationRequest represents a request to accept an invitation.
//
// Validation Rules:
//   - token: Required
type AcceptInvitationRequest struct {
	Token string `json:"token"` // Raw invitation token from invite email
}

// Validate validates the accept invitation request.
func (r AcceptInvitationRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Token, validation.Required),
	)
}

// DeclineInvitationRequest represents a request to decline an invitation.
//
// Validation Rules:
//   - token: Required
type DeclineInvitationRequest struct {
	Token string `json:"token"` // Raw invitation token from invite email
}

// Validate validates the decline invitation request.
func (r DeclineInvitationRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Token, validation.Required),
	)
}

// InvitationResponse is the response returned when creating an invitation.
// It includes the raw token which must be delivered to the invitee.
type InvitationResponse struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organizationId"`
	TeamID         *string `json:"teamId,omitempty"`
	Email          string  `json:"email"`
	Role           string  `json:"role"`
	Token          string  `json:"token"` // Raw token — only returned at creation
	Status         string  `json:"status"`
	ExpiresAt      string  `json:"expiresAt"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
}

// ── Plugin-level validation (uses the resolved role registry) ────────────

// orgMemberRoles returns the assignable org-level roles (all configured roles
// except owner, which cannot be assigned through the member endpoints).
func (p *Plugin) orgMemberRoles() []any {
	roles := make([]any, 0, len(p.orgRoles))
	for role := range p.orgRoles {
		if role == orgtypes.RoleOwner {
			continue
		}
		roles = append(roles, role)
	}
	return roles
}

// teamMemberRoles returns the assignable team-level roles.
func (p *Plugin) teamMemberRoles() []any {
	roles := make([]any, 0, len(p.teamRoles))
	for role := range p.teamRoles {
		roles = append(roles, role)
	}
	return roles
}

// ValidateAddMember validates an AddOrganizationMemberRequest against the
// configured org roles.
func (p *Plugin) ValidateAddMember(req AddOrganizationMemberRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.UserID, validation.Required),
		validation.Field(&req.Role, validation.Required, validation.In(p.orgMemberRoles()...)),
	)
}

// ValidateUpdateMemberRole validates an UpdateMemberRoleRequest against the
// configured org roles.
func (p *Plugin) ValidateUpdateMemberRole(req UpdateMemberRoleRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Role, validation.Required, validation.In(p.orgMemberRoles()...)),
	)
}

// ValidateAddTeamMember validates an AddTeamMemberRequest against the
// configured team roles.
func (p *Plugin) ValidateAddTeamMember(req AddTeamMemberRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.UserID, validation.Required),
		validation.Field(&req.Role, validation.Required, validation.In(p.teamMemberRoles()...)),
	)
}

// ValidateUpdateTeamMemberRole validates an UpdateTeamMemberRoleRequest against
// the configured team roles.
func (p *Plugin) ValidateUpdateTeamMemberRole(req UpdateTeamMemberRoleRequest) error {
	return validation.ValidateStruct(&req,
		validation.Field(&req.Role, validation.Required, validation.In(p.teamMemberRoles()...)),
	)
}

// ValidateCreateInvitation validates a CreateInvitationRequest. The role is
// validated against team roles when the invitation targets a team, and against
// org roles otherwise.
func (p *Plugin) ValidateCreateInvitation(req CreateInvitationRequest) error {
	roles := p.orgMemberRoles()
	if req.TeamID != nil && *req.TeamID != "" {
		roles = p.teamMemberRoles()
	}
	return validation.ValidateStruct(&req,
		validation.Field(&req.Email, validation.Required, validation.Length(1, 255)),
		validation.Field(&req.Role, validation.Required, validation.In(roles...)),
		validation.Field(&req.TeamID, validation.When(req.TeamID != nil && *req.TeamID != "", validation.Length(1, 255))),
	)
}
