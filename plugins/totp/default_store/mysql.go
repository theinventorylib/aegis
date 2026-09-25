package defaultstore

// mysql.go — thin translator: wraps sqlcmysql.Queries and implements querier.
// Dialect-specific types (int8 for booleans) are handled here and nowhere else.

import (
	"context"
	"database/sql"

	sqlcmysql "github.com/theinventorylib/aegis/v2/plugins/totp/internal/gen/mysql"
)

type mysqlQuerier struct{ q *sqlcmysql.Queries }

func newMySQLQuerier(db *sql.DB) *mysqlQuerier {
	return &mysqlQuerier{q: sqlcmysql.New(db)}
}

func (m *mysqlQuerier) getCredential(ctx context.Context, userID string) (totpCredentialRow, error) {
	row, err := m.q.GetCredential(ctx, userID)
	if err != nil {
		return totpCredentialRow{}, err
	}
	return totpCredentialRow{Secret: row.TotpSecret, Enabled: row.TotpEnabled != 0}, nil
}

func (m *mysqlQuerier) updateCredential(ctx context.Context, userID string, secret sql.NullString, enabled bool, updatedAt string) error {
	return m.q.UpdateCredential(ctx, sqlcmysql.UpdateCredentialParams{
		ID: userID, TotpSecret: secret, TotpEnabled: boolToInt[int8](enabled), UpdatedAt: updatedAt,
	})
}

func (m *mysqlQuerier) markSessionVerified(ctx context.Context, sessionID, userID, verifiedAt string) error {
	return m.q.MarkSessionVerified(ctx, sqlcmysql.MarkSessionVerifiedParams{
		SessionID: sessionID, UserID: userID, VerifiedAt: verifiedAt,
	})
}

func (m *mysqlQuerier) isSessionVerified(ctx context.Context, sessionID, userID, since string) (bool, error) {
	return m.q.IsSessionVerified(ctx, sqlcmysql.IsSessionVerifiedParams{
		SessionID: sessionID, UserID: userID, VerifiedAt: since,
	})
}

func (m *mysqlQuerier) deleteUserSessionVerifications(ctx context.Context, userID string) error {
	return m.q.DeleteUserSessionVerifications(ctx, userID)
}

func (m *mysqlQuerier) deleteSessionVerification(ctx context.Context, sessionID string) error {
	return m.q.DeleteSessionVerification(ctx, sessionID)
}

var _ querier = (*mysqlQuerier)(nil)
