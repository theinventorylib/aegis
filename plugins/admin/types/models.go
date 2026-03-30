// Package types defines the domain models and request/response types used by the admin plugin.
package types

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
//	user := types.User{
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

// UpdateRoleRequest represents a request to update a user's role.
//
// Validation:
//   - role: Required, the new role to assign to the user (e.g., "admin", "user")
//
// Example:
//
//	{
//	  "role": "admin"
//	}
type UpdateRoleRequest struct {
	Role string `json:"role"` // New role to assign (required)
}

// ========== Response DTOs ==========

// Note: list endpoints should use core.PaginatedResponse[T] for pagination metadata.

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
