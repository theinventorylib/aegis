package defaultstore

// querier.go — the single internal DB interface for the JWT plugin.
//
// All unexported. No dialect-generated types cross this file — only standard
// Go primitives and database/sql types. The three dialect translators in
// postgres.go, mysql.go, and sqlite.go each implement this interface.

import (
	"context"
	"database/sql"
)

// jwkRow is the canonical row type returned by getAllCurrentJWKS.
// All fields use dialect-neutral types (string for times, NullString for
// optional values). Dialect translators convert from their DB-specific types
// before returning this struct.
type jwkRow struct {
	Kid, KeyData, Algorithm string
	Use                     sql.NullString
	CreatedAt               string         // RFC3339
	ExpiresAt               sql.NullString // RFC3339 if valid, else not valid
}

// querier is the one internal interface all store methods use.
// The dialect is chosen exactly once in NewDefaultJWTStore; everything
// else calls through here and is dialect-agnostic.
type querier interface {
	getCurrentJWK(ctx context.Context, algorithm string, use sql.NullString) (string, error)
	storeJWK(ctx context.Context, kid, keyData, algorithm string, use sql.NullString, createdAt string, expiresAt sql.NullString) error
	deleteExpiredJWKS(ctx context.Context) error
	getAllCurrentJWKS(ctx context.Context) ([]jwkRow, error)
}
