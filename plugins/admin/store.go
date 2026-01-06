package admin

import (
	"context"
	"time"
)

// Store defines the interface for admin user storage operations.
//
// This interface extends the core auth.UserStore with admin-specific functionality:
//   - Role assignment and retrieval
//   - User ban management with expiry dates
//   - Platform statistics
//   - Raw database access for admin UI (flexible schema)
//
// All methods are context-aware for cancellation and timeout support.
//
// Thread Safety:
// Implementations must be safe for concurrent use.
type Store interface {
	// ========== Core User Operations ==========

	// Create creates a new user with admin fields.
	Create(ctx context.Context, user User) (User, error)

	// GetByEmail retrieves a user by email address.
	GetByEmail(ctx context.Context, email string) (User, error)

	// GetByID retrieves a user by ID.
	GetByID(ctx context.Context, id string) (User, error)

	// Update updates user fields (name, email, disabled, etc.).
	// Note: Does not update role - use AssignRole/RemoveRole instead.
	Update(ctx context.Context, user User) error

	// Delete soft-deletes a user by setting updated_at timestamp.
	Delete(ctx context.Context, id string) error

	// List retrieves paginated users.
	List(ctx context.Context, offset, limit int) ([]User, error)

	// ListUsersRaw retrieves paginated users as raw map data.
	// This supports flexible admin UIs without schema changes.
	ListUsersRaw(ctx context.Context, offset, limit int) ([]map[string]interface{}, error)

	// GetUserRaw retrieves a user as raw map data.
	GetUserRaw(ctx context.Context, userID string) (map[string]interface{}, error)

	// Count returns total user count.
	Count(ctx context.Context) (int, error)

	// ========== Role Management ==========

	// AssignRole assigns a role to a user (e.g., "admin").
	AssignRole(ctx context.Context, userID string, role string) error

	// RemoveRole removes a user's role.
	RemoveRole(ctx context.Context, userID string, role string) error

	// GetRole retrieves a user's role (empty string if no role).
	GetRole(ctx context.Context, userID string) (string, error)

	// ========== Ban Management ==========

	// BanUser bans a user with a reason and optional expiry date.
	// If expiry is nil, the ban is permanent.
	// Increments ban_counter for repeat offender tracking.
	BanUser(ctx context.Context, userID, reason string, expiry *time.Time) error

	// UnbanUser removes the ban from a user.
	UnbanUser(ctx context.Context, userID string) error

	// ========== Statistics ==========

	// GetStats retrieves platform statistics.
	GetStats(ctx context.Context) (StatsResponse, error)
}
