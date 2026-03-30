// Package defaultstore implements the SQL-backed default store for the oauth plugin.
package defaultstore

// store.go — DefaultOAuthStore, the concrete implementation of oauthtypes.Store.
//
// Dialect is selected once in NewDefaultOAuthStore; all methods delegate to the
// unexported querier interface so they are dialect-agnostic.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/theinventorylib/aegis/plugins"
	oauthtypes "github.com/theinventorylib/aegis/plugins/oauth/types"
)

// DefaultOAuthStore implements oauthtypes.Store using a SQL database.
type DefaultOAuthStore struct {
	q querier
}

// NewDefaultOAuthStore creates a DefaultOAuthStore for the given dialect.
func NewDefaultOAuthStore(db *sql.DB, dialect plugins.Dialect) (*DefaultOAuthStore, error) {
	var q querier
	switch dialect {
	case plugins.DialectPostgres:
		q = newPostgresQuerier(db)
	case plugins.DialectMySQL:
		q = newMysqlQuerier(db)
	case plugins.DialectSQLite:
		q = newSqliteQuerier(db)
	default:
		return nil, fmt.Errorf("oauth: unsupported dialect %q", dialect)
	}
	return &DefaultOAuthStore{q: q}, nil
}

// CreateConnection persists a new OAuth provider connection and returns it.
func (s *DefaultOAuthStore) CreateConnection(ctx context.Context, conn oauthtypes.Connection) (*oauthtypes.Connection, error) {
	pdata, err := json.Marshal(conn.ProviderData)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	createdAt := now
	updatedAt := now
	if !conn.CreatedAt.IsZero() {
		createdAt = conn.CreatedAt.Format(time.RFC3339)
	}
	if !conn.UpdatedAt.IsZero() {
		updatedAt = conn.UpdatedAt.Format(time.RFC3339)
	}

	r := connectionRow{
		ID:             conn.ID,
		UserID:         conn.UserID,
		Provider:       conn.Provider,
		ProviderUserID: conn.ProviderUserID,
		Email:          strToNullString(conn.Email),
		Name:           strToNullString(conn.Name),
		AvatarURL:      strToNullString(conn.AvatarURL),
		AccessToken:    conn.AccessToken,
		RefreshToken:   strToNullString(conn.RefreshToken),
		ExpiresAt:      conn.ExpiresAt.Format(time.RFC3339),
		ProviderData:   strToNullString(string(pdata)),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
	if err := s.q.createConnection(ctx, r); err != nil {
		return nil, err
	}
	return &conn, nil
}

// GetConnectionByProviderUserID retrieves an OAuth connection by provider name and the provider-assigned user ID.
func (s *DefaultOAuthStore) GetConnectionByProviderUserID(ctx context.Context, provider, providerUserID string) (*oauthtypes.Connection, error) {
	r, err := s.q.getConnectionByProviderUserID(ctx, provider, providerUserID)
	if err != nil {
		return nil, err
	}
	return buildConnection(r), nil
}

// GetConnectionsByUserID returns all OAuth connections associated with a user.
func (s *DefaultOAuthStore) GetConnectionsByUserID(ctx context.Context, userID string) ([]oauthtypes.Connection, error) {
	rows, err := s.q.getConnectionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]oauthtypes.Connection, len(rows))
	for i, r := range rows {
		result[i] = *buildConnection(r)
	}
	return result, nil
}

// UpdateConnection updates an existing OAuth provider connection.
func (s *DefaultOAuthStore) UpdateConnection(ctx context.Context, conn oauthtypes.Connection) error {
	pdata, err := json.Marshal(conn.ProviderData)
	if err != nil {
		return err
	}
	r := connectionRow{
		ID:             conn.ID,
		UserID:         conn.UserID,
		Provider:       conn.Provider,
		ProviderUserID: conn.ProviderUserID,
		Email:          strToNullString(conn.Email),
		Name:           strToNullString(conn.Name),
		AvatarURL:      strToNullString(conn.AvatarURL),
		AccessToken:    conn.AccessToken,
		RefreshToken:   strToNullString(conn.RefreshToken),
		ExpiresAt:      conn.ExpiresAt.Format(time.RFC3339),
		ProviderData:   strToNullString(string(pdata)),
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	return s.q.updateConnection(ctx, r)
}

// DeleteConnection removes the OAuth connection for the given provider and user ID.
func (s *DefaultOAuthStore) DeleteConnection(ctx context.Context, provider, userID string) error {
	return s.q.deleteConnection(ctx, provider, userID)
}

// buildConnection constructs an oauthtypes.Connection from a connectionRow.
func buildConnection(r connectionRow) *oauthtypes.Connection {
	pdStr := ""
	if r.ProviderData.Valid {
		pdStr = r.ProviderData.String
	}
	var pd map[string]any
	if pdStr != "" {
		if err := json.Unmarshal([]byte(pdStr), &pd); err != nil {
			pd = make(map[string]any)
		}
	} else {
		pd = make(map[string]any)
	}
	return &oauthtypes.Connection{
		ID:             r.ID,
		UserID:         r.UserID,
		Provider:       r.Provider,
		ProviderUserID: r.ProviderUserID,
		Email:          nullStr(r.Email),
		Name:           nullStr(r.Name),
		AvatarURL:      nullStr(r.AvatarURL),
		AccessToken:    r.AccessToken,
		RefreshToken:   nullStr(r.RefreshToken),
		ExpiresAt:      parseTime(r.ExpiresAt),
		ProviderData:   pd,
		CreatedAt:      parseTime(r.CreatedAt),
		UpdatedAt:      parseTime(r.UpdatedAt),
	}
}

// strToNullString converts s to a sql.NullString that is valid only when s is non-empty.
func strToNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// nullStr extracts the string value from ns, returning an empty string when the
// value is NULL.
func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// parseTime parses an RFC3339 timestamp string into time.Time. On parse failure
// it returns the zero value of time.Time rather than propagating the error,
// matching the convention used throughout the store for stored string timestamps.
func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s) //nolint:errcheck
	return t
}
