package defaultstore

// postgres.go — thin translator: wraps sqlcpostgres.Queries and implements querier.
// Dialect-specific types (time.Time for CreatedAt, sql.NullTime for ExpiresAt)
// are handled here and nowhere else.

import (
	"context"
	"database/sql"
	"time"

	sqlcpostgres "github.com/theinventorylib/aegis/plugins/jwt/internal/gen/postgres"
)

type postgresQuerier struct{ q *sqlcpostgres.Queries }

func newPostgresQuerier(db *sql.DB) *postgresQuerier {
	return &postgresQuerier{q: sqlcpostgres.New(db)}
}

func (p *postgresQuerier) getCurrentJWK(ctx context.Context, algorithm string, use sql.NullString) (string, error) {
	return p.q.GetCurrentJWK(ctx, sqlcpostgres.GetCurrentJWKParams{
		Algorithm: algorithm,
		Use:       use,
	})
}

func (p *postgresQuerier) storeJWK(ctx context.Context, kid, keyData, algorithm string, use sql.NullString, createdAt string, expiresAt sql.NullString) error {
	ct, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return err
	}
	var ea sql.NullTime
	if expiresAt.Valid {
		t, err := time.Parse(time.RFC3339, expiresAt.String)
		if err != nil {
			return err
		}
		ea = sql.NullTime{Time: t, Valid: true}
	}
	return p.q.StoreJWK(ctx, sqlcpostgres.StoreJWKParams{
		Kid:       kid,
		KeyData:   keyData,
		Algorithm: algorithm,
		Use:       use,
		CreatedAt: ct,
		ExpiresAt: ea,
	})
}

func (p *postgresQuerier) deleteExpiredJWKS(ctx context.Context) error {
	return p.q.DeleteExpiredJWKS(ctx)
}

func (p *postgresQuerier) getAllCurrentJWKS(ctx context.Context) ([]jwkRow, error) {
	rows, err := p.q.GetAllCurrentJWKS(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]jwkRow, len(rows))
	for i, r := range rows {
		var ea sql.NullString
		if r.ExpiresAt.Valid {
			ea = sql.NullString{String: r.ExpiresAt.Time.Format(time.RFC3339), Valid: true}
		}
		out[i] = jwkRow{
			Kid:       r.Kid,
			KeyData:   r.KeyData,
			Algorithm: r.Algorithm,
			Use:       r.Use,
			CreatedAt: r.CreatedAt.Format(time.RFC3339),
			ExpiresAt: ea,
		}
	}
	return out, nil
}
