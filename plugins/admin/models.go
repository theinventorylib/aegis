package admin

import (
	"time"

	"github.com/theinventorylib/aegis/auth"
)

// User represents a user with admin-specific extensions.
//
// This model extends auth.User with role-based access control and ban management.
//
// Admin Extensions:
//   - Role: User role (e.g., "admin") for authorization checks
//   - Banned: Whether user is currently banned
//   - BanReason: Admin-provided reason for ban
//   - BanExpiry: Ban expiration date (nil for permanent bans)
//   - BanCounter: Number of times user has been banned (for repeat offender tracking)
//
// Database Mapping:
// These fields are stored in additional columns on the 'user' table:
//   - role (VARCHAR)
//   - banned (BOOLEAN)
//   - ban_reason (TEXT)
//   - ban_expiry (TIMESTAMP, nullable)
//   - ban_counter (INTEGER, default 0)
//
// Example:
//
//	user := admin.User{
//	  User: auth.User{ID: "user_123", Email: "admin@example.com"},
//	  Role: "admin",
//	  Banned: false,
//	}
type User struct {
	auth.User
	Role string `json:"role"` // User role for RBAC (e.g., "admin")

	// Ban management fields
	Banned     bool       `json:"banned"`               // Current ban status
	BanReason  string     `json:"banReason,omitempty"`  // Reason for ban
	BanExpiry  *time.Time `json:"banExpiry,omitempty"`  // Ban expiration (nil = permanent)
	BanCounter int        `json:"banCounter,omitempty"` // Number of bans (repeat offender tracking)
}

// ========== Request DTOs ==========

// BanUserRequest represents a request to ban a user.
//
// Validation:
//   - reason: Required, provides context for ban decision
//   - expiresAt: Optional, nil for permanent ban
//
// Example (temporary ban):
//
//	{
//	  "reason": "Spam",
//	  "expiresAt": "2024-12-31T23:59:59Z"
//	}
//
// Example (permanent ban):
//
//	{
//	  "reason": "Terms of service violation",
//	  "expiresAt": null
//	}
type BanUserRequest struct {
	Reason    string     `json:"reason"`              // Ban reason (required)
	ExpiresAt *time.Time `json:"expiresAt,omitempty"` // Ban expiration (nil = permanent)
}

// ========== Response DTOs ==========

// UserListResponse represents a paginated list of users.
//
// Example Response:
//
//	{
//	  "users": [{"id": "user_1", "email": "user1@example.com", "role": "admin"}],
//	  "totalCount": 100,
//	  "offset": 0,
//	  "limit": 20
//	}
type UserListResponse struct {
	Users      []User `json:"users"`      // Users in current page
	TotalCount int    `json:"totalCount"` // Total number of users
	Offset     int    `json:"offset"`     // Current offset (for pagination)
	Limit      int    `json:"limit"`      // Page size
}

// StatsResponse represents platform statistics.
//
// Example Response:
//
//	{
//	  "totalUsers": 1234
//	}
type StatsResponse struct {
	TotalUsers int `json:"totalUsers"` // Total registered users
}
