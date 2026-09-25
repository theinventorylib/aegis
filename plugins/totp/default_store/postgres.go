package defaultstore

// postgres.go — thin translator: wraps sqlcpostgres.Queries and implements querier.
// Dialect-specific types (int32 for booleans) are handled here and nowhere else.

import (
	"context"
	"database/sql"

	sqlcpostgres "github.com/theinventorylib/aegis/v2/plugins/totp/internal/gen/postgres"
)

type postgresQuerier struct{ q *sqlcpostgres.Queries }

func newPostgresQuerier(db *sql.DB) *postgresQuerier {
	return &postgresQuerier{q: sqlcpostgres.New(db)}
}

func (p *postgresQuerier) getCredential(ctx context.Context, userID string) (totpCredentialRow, error) {
	row, err := p.q.GetCredential(ctx, userID)
	if err != nil {
		return totpCredentialRow{}, err
	}
	return totpCredentialRow{Secret: row.TotpSecret, Enabled: row.TotpEnabled != 0}, nil
}

func (p *postgresQuerier) updateCredential(ctx context.Context, userID string, secret sql.NullString, enabled bool, updatedAt string) error {
	return p.q.UpdateCredential(ctx, sqlcpostgres.UpdateCredentialParams{
		ID: userID, TotpSecret: secret, TotpEnabled: boolToInt[int32](enabled), UpdatedAt: updatedAt,
	})
}

func (p *postgresQuerier) markSessionVerified(ctx context.Context, sessionID, userID, verifiedAt string) error {
	return p.q.MarkSessionVerified(ctx, sqlcpostgres.MarkSessionVerifiedParams{
		SessionID: sessionID, UserID: userID, VerifiedAt: verifiedAt,
	})
}

func (p *postgresQuerier) isSessionVerified(ctx context.Context, sessionID, userID, since string) (bool, error) {
	return p.q.IsSessionVerified(ctx, sqlcpostgres.IsSessionVerifiedParams{
		SessionID: sessionID, UserID: userID, VerifiedAt: since,
	})
}

func (p *postgresQuerier) deleteUserSessionVerifications(ctx context.Context, userID string) error {
	return p.q.DeleteUserSessionVerifications(ctx, userID)
}

func (p *postgresQuerier) deleteSessionVerification(ctx context.Context, sessionID string) error {
	return p.q.DeleteSessionVerification(ctx, sessionID)
}

var _ querier = (*postgresQuerier)(nil)
