package defaultstore

// querier.go — the single internal DB interface for the totp plugin.
//
// All unexported. No dialect-generated types cross this file — only standard
// Go primitives and database/sql types. The three dialect translators in
// postgres.go, mysql.go, and sqlite.go each implement this interface.

import (
	"context"
	"database/sql"
)

// totpCredentialRow is the dialect-neutral credential row.
type totpCredentialRow struct {
	Secret  string
	Enabled bool
}

// querier is the one internal interface all store methods use.
// The dialect is chosen exactly once in NewDefaultTOTPStore; everything else
// calls through here and is dialect-agnostic.
type querier interface {
	getCredential(ctx context.Context, userID string) (totpCredentialRow, error)
	updateCredential(ctx context.Context, userID string, secret sql.NullString, enabled bool, updatedAt string) error
	markSessionVerified(ctx context.Context, sessionID, userID, verifiedAt string) error
	isSessionVerified(ctx context.Context, sessionID, userID, since string) (bool, error)
	deleteUserSessionVerifications(ctx context.Context, userID string) error
	deleteSessionVerification(ctx context.Context, sessionID string) error
}
