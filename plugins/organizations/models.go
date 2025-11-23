package organizations

import (
	"time"
)

// Organization represents a workspace/company
type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Team represents a team within an organization
type Team struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organizationId"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// UserOrganization represents a user's membership in an organization
type UserOrganization struct {
	ID             string    `json:"id"`
	UserID         string    `json:"userId"`
	OrganizationID string    `json:"organizationId"`
	Role           string    `json:"role"` // "owner", "admin", "member"
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// TeamMember represents a user's membership in a team
type TeamMember struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"teamId"`
	UserID    string    `json:"userId"`
	Role      string    `json:"role"` // "lead", "member"
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
