package defaultstore

// querier.go — the single internal DB interface for the emailotp plugin.
//
// All unexported. No dialect-generated types cross this file — only standard
// Go primitives and database/sql types. The three dialect translators in
// postgres.go, mysql.go, and sqlite.go each implement this interface.

import (
	"context"
	"database/sql"
)

// emailUserRow is the canonical row returned by user lookup queries.
// All dialect-specific integer booleans are converted to bool before here.
type emailUserRow struct {
	ID, Name, CreatedAt, UpdatedAt string
	Avatar, Email                  sql.NullString
	Disabled, EmailVerified        bool
}

// querier is the one internal interface all store methods use.
// The dialect is chosen exactly once in NewDefaultEmailOTPStore; everything
// else calls through here and is dialect-agnostic.
type querier interface {
	createUser(ctx context.Context, id, name, createdAt, updatedAt string, avatar, email sql.NullString, disabled, emailVerified bool) error
	getUserByEmail(ctx context.Context, email sql.NullString) (emailUserRow, error)
	updateUserEmail(ctx context.Context, userID string, email sql.NullString, verified bool, updatedAt string) error
}

func boolToInt[T int8 | int32 | int64](b bool) T {
	if b {
		return 1
	}
	return 0
}
