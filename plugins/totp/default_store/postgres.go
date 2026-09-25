package defaultstore

// postgres.go — thin translator: wraps sqlcpostgres.Queries and implements querier.
// Dialect-specific types (int32 for booleans) are handled here and nowhere else.

import (
	"context"
	"database/sql"
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
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// pgFormatTime converts a time.Time back to the canonical RFC3339 string.
func pgFormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// pgParseNullTime converts a nullable canonical string to sql.NullTime.
func pgParseNullTime(ns sql.NullString) sql.NullTime {
	if !ns.Valid {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: pgParseTime(ns.String), Valid: true}
}

// pgFormatNullTime converts sql.NullTime back to a nullable canonical string.
func pgFormatNullTime(nt sql.NullTime) sql.NullString {
	if !nt.Valid {
		return sql.NullString{}
	}
	return sql.NullString{String: pgFormatTime(nt.Time), Valid: true}
}
