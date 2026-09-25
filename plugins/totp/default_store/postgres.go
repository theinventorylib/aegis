package defaultstore

// postgres.go — thin translator: wraps sqlcpostgres.Queries and implements querier.
// Dialect-specific types (int32 for booleans) are handled here and nowhere else.

import (
	"context"
	"database/sql"
	"log"
	"time"

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
		ID: userID, TotpSecret: secret, TotpEnabled: boolToInt[int32](enabled), UpdatedAt: pgParseTime(updatedAt),
	})
}

func (p *postgresQuerier) markSessionVerified(ctx context.Context, sessionID, userID, verifiedAt string) error {
	return p.q.MarkSessionVerified(ctx, sqlcpostgres.MarkSessionVerifiedParams{
		SessionID: sessionID, UserID: userID, VerifiedAt: pgParseTime(verifiedAt),
	})
}

func (p *postgresQuerier) isSessionVerified(ctx context.Context, sessionID, userID, since string) (bool, error) {
	return p.q.IsSessionVerified(ctx, sqlcpostgres.IsSessionVerifiedParams{
		SessionID: sessionID, UserID: userID, VerifiedAt: pgParseTime(since),
	})
}

func (p *postgresQuerier) deleteUserSessionVerifications(ctx context.Context, userID string) error {
	return p.q.DeleteUserSessionVerifications(ctx, userID)
}

func (p *postgresQuerier) deleteSessionVerification(ctx context.Context, sessionID string) error {
	return p.q.DeleteSessionVerification(ctx, sessionID)
}

var _ querier = (*postgresQuerier)(nil)

// pgParseTime converts a canonical RFC3339 string to the time.Time the
// generated postgres queries expect.
func pgParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		log.Printf("aegis/totp: ignoring malformed RFC3339 timestamp %q: %v", s, err)
		return time.Time{}
	}
	return t
}
