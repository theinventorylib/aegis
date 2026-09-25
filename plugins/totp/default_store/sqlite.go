package defaultstore

// sqlite.go — thin translator: wraps sqlcsqlite.Queries and implements querier.
// Dialect-specific types (int64 for booleans) are handled here and nowhere else.

import (
	"context"
	"database/sql"

	sqlcsqlite "github.com/theinventorylib/aegis/v2/plugins/totp/internal/gen/sqlite"
)

type sqliteQuerier struct{ q *sqlcsqlite.Queries }

func newSQLiteQuerier(db *sql.DB) *sqliteQuerier {
	return &sqliteQuerier{q: sqlcsqlite.New(db)}
}

func (s *sqliteQuerier) getCredential(ctx context.Context, userID string) (totpCredentialRow, error) {
	row, err := s.q.GetCredential(ctx, userID)
	if err != nil {
		return totpCredentialRow{}, err
	}
	return totpCredentialRow{Secret: row.TotpSecret, Enabled: row.TotpEnabled != 0}, nil
}

func (s *sqliteQuerier) updateCredential(ctx context.Context, userID string, secret sql.NullString, enabled bool, updatedAt string) error {
	return s.q.UpdateCredential(ctx, sqlcsqlite.UpdateCredentialParams{
		ID: userID, TotpSecret: secret, TotpEnabled: boolToInt[int64](enabled), UpdatedAt: updatedAt,
	})
}

func (s *sqliteQuerier) markSessionVerified(ctx context.Context, sessionID, userID, verifiedAt string) error {
	return s.q.MarkSessionVerified(ctx, sqlcsqlite.MarkSessionVerifiedParams{
		SessionID: sessionID, UserID: userID, VerifiedAt: verifiedAt,
	})
}

func (s *sqliteQuerier) isSessionVerified(ctx context.Context, sessionID, userID, since string) (bool, error) {
	return s.q.IsSessionVerified(ctx, sqlcsqlite.IsSessionVerifiedParams{
		SessionID: sessionID, UserID: userID, VerifiedAt: since,
	})
}

func (s *sqliteQuerier) deleteUserSessionVerifications(ctx context.Context, userID string) error {
	return s.q.DeleteUserSessionVerifications(ctx, userID)
}

func (s *sqliteQuerier) deleteSessionVerification(ctx context.Context, sessionID string) error {
	return s.q.DeleteSessionVerification(ctx, sessionID)
}

var _ querier = (*sqliteQuerier)(nil)
