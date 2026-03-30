package defaultstore

// querier.go — the single internal DB interface for the oauth plugin.
//
// All unexported. No dialect-generated types cross this file — only standard
// Go primitives and database/sql types. The three dialect translators in
// postgres.go, mysql.go, and sqlite.go each implement this interface.
//
// Since all three dialects use identical field types (string + sql.NullString,
// no boolean or time differences), the querier and dialect translators are
// nearly identical.

import (
	"context"
	"database/sql"
)

// connectionRow is the canonical row returned by connection queries.
// All fields from the database are normalized to strings and sql.NullString.
type connectionRow struct {
	ID             string
	UserID         string
	Provider       string
	ProviderUserID string
	AccessToken    string
	ExpiresAt      string
	CreatedAt      string
	UpdatedAt      string
	Email          sql.NullString
	Name           sql.NullString
	AvatarURL      sql.NullString
	RefreshToken   sql.NullString
	ProviderData   sql.NullString
}

// querier is the one internal interface all store methods use.
// The dialect is chosen exactly once in NewDefaultOAuthStore; everything
// else calls through here and is dialect-agnostic.
type querier interface {
	createConnection(ctx context.Context, r connectionRow) error
	getConnectionByProviderUserID(ctx context.Context, provider, providerUserID string) (connectionRow, error)
	getConnectionsByUserID(ctx context.Context, userID string) ([]connectionRow, error)
	updateConnection(ctx context.Context, r connectionRow) error
	deleteConnection(ctx context.Context, provider, userID string) error
}
