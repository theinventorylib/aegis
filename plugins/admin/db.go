package admin

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
)

// DB handles database operations for the admin plugin
type DB struct {
	provider db.Provider
}

// NewDB creates a new admin database handler
func NewDB(provider db.Provider) *DB {
	return &DB{provider: provider}
}

// GetUserByID retrieves a user with admin-specific fields populated
func (d *DB) GetUserByID(ctx context.Context, userID string) (*User, error) {
	var adminUser User
	var coreUser models.User

	// Query with extended fields
	query := `
		SELECT id, created_at, updated_at, disabled,
		       COALESCE(role, 'user') as role,
		       COALESCE(banned, false) as banned,
		       COALESCE(ban_reason, '') as ban_reason,
		       ban_expiry,
		       COALESCE(ban_counter, 0) as ban_counter
		FROM auth.user
		WHERE id = $1
	`

	err := d.provider.QueryRow(ctx, query, userID).Scan(
		&coreUser.ID,
		&coreUser.CreatedAt,
		&coreUser.UpdatedAt,
		&coreUser.Disabled,
		&adminUser.Role,
		&adminUser.Banned,
		&adminUser.BanReason,
		&adminUser.BanExpiry,
		&adminUser.BanCounter,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	adminUser.User = &coreUser
	return &adminUser, nil
}

// UpdateUser updates a user including admin-specific fields
func (d *DB) UpdateUser(ctx context.Context, user *User) error {
	// Update both core and admin fields
	query := `
		UPDATE auth.user
		SET updated_at = NOW(),
		    disabled = $2,
		    role = $3,
		    banned = $4,
		    ban_reason = $5,
		    ban_expiry = $6,
		    ban_counter = $7
		WHERE id = $1
	`

	_, err := d.provider.Exec(ctx, query,
		user.ID,
		user.Disabled,
		user.Role,
		user.Banned,
		user.BanReason,
		user.BanExpiry,
		user.BanCounter,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// ListUsers retrieves users with admin fields populated
func (d *DB) ListUsers(ctx context.Context, offset, limit int) ([]*User, error) {
	query := `
		SELECT id, created_at, updated_at, disabled,
		       COALESCE(role, 'user') as role,
		       COALESCE(banned, false) as banned,
		       COALESCE(ban_reason, '') as ban_reason,
		       ban_expiry,
		       COALESCE(ban_counter, 0) as ban_counter
		FROM auth.user
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := d.provider.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var adminUser User
		var coreUser models.User

		err := rows.Scan(
			&coreUser.ID,
			&coreUser.CreatedAt,
			&coreUser.UpdatedAt,
			&coreUser.Disabled,
			&adminUser.Role,
			&adminUser.Banned,
			&adminUser.BanReason,
			&adminUser.BanExpiry,
			&adminUser.BanCounter,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		adminUser.User = &coreUser
		users = append(users, &adminUser)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return users, nil
}

// ListUsersRaw retrieves users as a map for flexible display (includes email)
func (d *DB) ListUsersRaw(ctx context.Context, offset, limit int) ([]map[string]interface{}, error) {
	rows, err := d.provider.Query(ctx, `
		SELECT id, created_at, updated_at, 
		       COALESCE(email, '') as email, 
		       COALESCE(role, 'user') as role,
		       disabled
		FROM auth.user
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, email, role string
		var createdAt, updatedAt interface{}
		var disabled bool

		if err := rows.Scan(&id, &createdAt, &updatedAt, &email, &role, &disabled); err != nil {
			return nil, err
		}

		users = append(users, map[string]interface{}{
			"id":        id,
			"createdAt": createdAt,
			"updatedAt": updatedAt,
			"email":     email,
			"role":      role,
			"disabled":  disabled,
		})
	}
	return users, nil
}

// GetUserRaw retrieves a single user as a map (includes email)
func (d *DB) GetUserRaw(ctx context.Context, userID string) (map[string]interface{}, error) {
	var id, email, role string
	var createdAt, updatedAt interface{}
	var disabled bool

	err := d.provider.QueryRow(ctx, `
		SELECT id, created_at, updated_at, 
		       COALESCE(email, '') as email, 
		       COALESCE(role, 'user') as role,
		       disabled
		FROM auth.user
		WHERE id = $1
	`, userID).Scan(&id, &createdAt, &updatedAt, &email, &role, &disabled)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":        id,
		"createdAt": createdAt,
		"updatedAt": updatedAt,
		"email":     email,
		"role":      role,
		"disabled":  disabled,
	}, nil
}

// BanUser bans a user with reason and optional expiry
func (d *DB) BanUser(ctx context.Context, userID, reason string, expiry interface{}) error {
	query := `
		UPDATE auth.user
		SET banned = true,
		    ban_reason = $2,
		    ban_expiry = $3,
		    ban_counter = ban_counter + 1,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := d.provider.Exec(ctx, query, userID, reason, expiry)
	if err != nil {
		return fmt.Errorf("failed to ban user: %w", err)
	}

	return nil
}

// UnbanUser removes ban from a user
func (d *DB) UnbanUser(ctx context.Context, userID string) error {
	query := `
		UPDATE auth.user
		SET banned = false,
		    ban_reason = NULL,
		    ban_expiry = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := d.provider.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to unban user: %w", err)
	}

	return nil
}

// SetUserRole updates a user's role
func (d *DB) SetUserRole(ctx context.Context, userID, role string) error {
	query := `
		UPDATE auth.user
		SET role = $2,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := d.provider.Exec(ctx, query, userID, role)
	if err != nil {
		return fmt.Errorf("failed to set user role: %w", err)
	}

	return nil
}

// GetUserRole retrieves a specific user's role
func (d *DB) GetUserRole(ctx context.Context, userID string) (string, error) {
	var role string
	query := `SELECT COALESCE(role, 'user') FROM auth.user WHERE id = $1`

	err := d.provider.QueryRow(ctx, query, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "user", nil // Default to 'user' if not found
		}
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	return role, nil
}

// HasUserRole checks if a user has a specific role
func (d *DB) HasUserRole(ctx context.Context, userID, role string) (bool, error) {
	userRole, err := d.GetUserRole(ctx, userID)
	if err != nil {
		return false, err
	}
	return userRole == role, nil
}

// DeleteUser deletes a user (cascades to sessions, etc.)
func (d *DB) DeleteUser(ctx context.Context, userID string) error {
	return d.provider.DeleteUser(ctx, userID)
}

// DeleteUserSessions deletes all sessions for a user
func (d *DB) DeleteUserSessions(ctx context.Context, userID string) error {
	return d.provider.DeleteUserSessions(ctx, userID)
}

// ========== ORGANIZATION OPERATIONS ==========

// CreateOrganization creates a new organization
func (d *DB) CreateOrganization(ctx context.Context, id, name, slug, ownerID string) (map[string]interface{}, error) {
	// Transaction to create org and add owner
	tx, err := d.provider.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert org
	_, err = tx.Exec(ctx, `
		INSERT INTO auth.organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`, id, name, slug)
	if err != nil {
		return nil, err
	}

	// Add owner
	memberID := "uorg_" + ownerID + "_" + id
	_, err = tx.Exec(ctx, `
		INSERT INTO auth.user_organizations (id, user_id, organization_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, 'owner', NOW(), NOW())
	`, memberID, ownerID, id)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":   id,
		"name": name,
		"slug": slug,
	}, nil
}

// ListOrganizations lists all organizations
func (d *DB) ListOrganizations(ctx context.Context, offset, limit int) ([]interface{}, error) {
	rows, err := d.provider.Query(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM auth.organizations
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []interface{}
	for rows.Next() {
		var id, name, slug string
		var createdAt, updatedAt interface{}

		if err := rows.Scan(&id, &name, &slug, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		orgs = append(orgs, map[string]interface{}{
			"id":         id,
			"name":       name,
			"slug":       slug,
			"created_at": createdAt,
			"updated_at": updatedAt,
		})
	}
	return orgs, nil
}

// GetOrganization gets a specific organization
func (d *DB) GetOrganization(ctx context.Context, orgID string) (interface{}, error) {
	var id, name, slug string
	var createdAt, updatedAt interface{}

	err := d.provider.QueryRow(ctx, `
		SELECT id, name, slug, created_at, updated_at
		FROM auth.organizations
		WHERE id = $1
	`, orgID).Scan(&id, &name, &slug, &createdAt, &updatedAt)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":         id,
		"name":       name,
		"slug":       slug,
		"created_at": createdAt,
		"updated_at": updatedAt,
	}, nil
}

// BanOrganization bans an organization
func (d *DB) BanOrganization(ctx context.Context, orgID string) error {
	_, err := d.provider.Exec(ctx, `
		UPDATE auth.organizations SET disabled = true WHERE id = $1
	`, orgID)
	return err
}

// DeleteOrganization deletes an organization
func (d *DB) DeleteOrganization(ctx context.Context, orgID string) error {
	_, err := d.provider.Exec(ctx, "DELETE FROM auth.organizations WHERE id = $1", orgID)
	return err
}

// GetStats returns platform statistics
func (d *DB) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// Count total users
	totalUsers, err := d.provider.CountUsers(ctx)
	if err != nil {
		return nil, err
	}

	// Count organizations
	var totalOrgs int
	err = d.provider.QueryRow(ctx, "SELECT COUNT(*) FROM auth.organizations").Scan(&totalOrgs)
	if err != nil {
		// Fallback if table doesn't exist
		totalOrgs = 0
	}

	return map[string]interface{}{
		"totalUsers":         totalUsers,
		"totalOrganizations": totalOrgs,
		"activeSessions":     0, // Would require additional DB query
	}, nil
}
