package admin

import (
	"time"

	"github.com/theinventorylib/aegis/models"
)

// User extends the core User model with admin-specific fields.
// This prevents polluting the core User model with admin-only concerns.
type User struct {
	*models.User

	// RBAC fields
	Role string `json:"role"` // "user" or "admin"

	// Ban management fields
	Banned     bool       `json:"banned"`
	BanReason  string     `json:"banReason,omitempty"`
	BanExpiry  *time.Time `json:"banExpiry,omitempty"`  // null = permanent ban
	BanCounter int        `json:"banCounter,omitempty"` // Total number of bans
}

// ========== Request DTOs ==========

// BanUserRequest represents a request to ban a user
type BanUserRequest struct {
	UserID    string     `json:"userId"`
	Reason    string     `json:"reason"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"` // null for permanent ban
}

// UnbanUserRequest represents a request to unban a user
type UnbanUserRequest struct {
	UserID string `json:"userId"`
}

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	OwnerID string `json:"ownerId"`
}

// ========== Response DTOs ==========

// UserListResponse represents a paginated list of users with admin fields
type UserListResponse struct {
	Users      []map[string]interface{} `json:"users"`
	TotalCount int                      `json:"totalCount"`
	Offset     int                      `json:"offset"`
	Limit      int                      `json:"limit"`
}

// StatsResponse represents platform statistics
type StatsResponse struct {
	TotalUsers         int `json:"totalUsers"`
	TotalOrganizations int `json:"totalOrganizations"`
	ActiveSessions     int `json:"activeSessions"`
}

// BanResponse represents the response after banning a user
type BanResponse struct {
	Message    string     `json:"message"`
	BanCounter int        `json:"banCounter"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

// UnbanResponse represents the response after unbanning a user
type UnbanResponse struct {
	Message    string `json:"message"`
	BanCounter int    `json:"banCounter"` // Show history
}
