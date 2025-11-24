// Package oauth provides OAuth database operations.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB provides database operations for OAuth plugin
type DB struct {
	pool *pgxpool.Pool
}

// NewDB creates a new OAuth plugin database instance
func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{pool: pool}
}

// SaveConnection creates or updates an OAuth connection
func (db *DB) SaveConnection(ctx context.Context, conn *Connection) error {
	providerDataJSON, err := json.Marshal(conn.ProviderData)
	if err != nil {
		return fmt.Errorf("failed to marshal provider data: %w", err)
	}

	_, err = db.pool.Exec(ctx, `
		INSERT INTO plugins_oauth.connections 
		    (id, user_id, provider, provider_user_id, email, name, avatar_url, 
		     access_token, refresh_token, expires_at, provider_data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (provider, provider_user_id) DO UPDATE
		SET 
		    access_token = $8,
		    refresh_token = $9,
		    expires_at = $10,
		    email = $5,
		    name = $6,
		    avatar_url = $7,
		    provider_data = $11,
		    updated_at = $13
	`, conn.ID, conn.UserID, conn.Provider, conn.ProviderUserID, conn.Email, conn.Name,
		conn.AvatarURL, conn.AccessToken, conn.RefreshToken, conn.ExpiresAt,
		providerDataJSON, conn.CreatedAt, conn.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save OAuth connection: %w", err)
	}
	return nil
}

// getConnectionByFields is a helper to query OAuth connections by different field combinations.
func (db *DB) getConnectionByFields(ctx context.Context, whereClause string, args ...interface{}) (*Connection, error) {
	conn := &Connection{}
	var providerDataJSON []byte

	query := `
		SELECT id, user_id, provider, provider_user_id, email, name, avatar_url, 
		       access_token, refresh_token, expires_at, provider_data, created_at, updated_at
		FROM plugins_oauth.connections
		WHERE ` + whereClause

	err := db.pool.QueryRow(ctx, query, args...).Scan(
		&conn.ID, &conn.UserID, &conn.Provider, &conn.ProviderUserID, &conn.Email, &conn.Name,
		&conn.AvatarURL, &conn.AccessToken, &conn.RefreshToken, &conn.ExpiresAt,
		&providerDataJSON, &conn.CreatedAt, &conn.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("OAuth connection not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth connection: %w", err)
	}

	if err := json.Unmarshal(providerDataJSON, &conn.ProviderData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider data: %w", err)
	}

	return conn, nil
}

// GetConnection retrieves an OAuth connection by provider and user ID.
func (db *DB) GetConnection(ctx context.Context, provider, userID string) (*Connection, error) {
	return db.getConnectionByFields(ctx, "provider = $1 AND user_id = $2", provider, userID)
}

// GetConnectionByProviderUserID retrieves connection by provider and provider user ID.
func (db *DB) GetConnectionByProviderUserID(ctx context.Context, provider, providerUserID string) (*Connection, error) {
	return db.getConnectionByFields(ctx, "provider = $1 AND provider_user_id = $2", provider, providerUserID)
}

// GetUserConnections retrieves all OAuth connections for a user
func (db *DB) GetUserConnections(ctx context.Context, userID string) ([]*Connection, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, user_id, provider, provider_user_id, email, name, avatar_url, 
		       access_token, refresh_token, expires_at, provider_data, created_at, updated_at
		FROM plugins_oauth.connections
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query connections: %w", err)
	}
	defer rows.Close()

	var connections []*Connection
	for rows.Next() {
		conn := &Connection{}
		var providerDataJSON []byte

		err := rows.Scan(
			&conn.ID, &conn.UserID, &conn.Provider, &conn.ProviderUserID, &conn.Email, &conn.Name,
			&conn.AvatarURL, &conn.AccessToken, &conn.RefreshToken, &conn.ExpiresAt,
			&providerDataJSON, &conn.CreatedAt, &conn.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connection: %w", err)
		}

		if err := json.Unmarshal(providerDataJSON, &conn.ProviderData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider data: %w", err)
		}

		connections = append(connections, conn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connections: %w", err)
	}

	return connections, nil
}

// DeleteConnection deletes an OAuth connection
func (db *DB) DeleteConnection(ctx context.Context, provider, userID string) error {
	_, err := db.pool.Exec(ctx, `
		DELETE FROM plugins_oauth.connections
		WHERE provider = $1 AND user_id = $2
	`, provider, userID)

	if err != nil {
		return fmt.Errorf("failed to delete OAuth connection: %w", err)
	}
	return nil
}
