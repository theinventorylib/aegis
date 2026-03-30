package defaultstore

// sqlite.go — thin translator: wraps sqlcsqlite.Queries and implements querier.
// SQLite stores timestamps as RFC3339 strings, so no time conversion is needed.

import (
	"context"
	"database/sql"

	sqlcsqlite "github.com/theinventorylib/aegis/plugins/jwt/internal/gen/sqlite"
)

type sqliteQuerier struct{ q *sqlcsqlite.Queries }

func newSQLiteQuerier(db *sql.DB) *sqliteQuerier {
	return &sqliteQuerier{q: sqlcsqlite.New(db)}
}

func (s *sqliteQuerier) getCurrentJWK(ctx context.Context, algorithm string, use sql.NullString) (string, error) {
	return s.q.GetCurrentJWK(ctx, sqlcsqlite.GetCurrentJWKParams{
		Algorithm: algorithm,
		Use:       use,
	})
}

func (s *sqliteQuerier) storeJWK(ctx context.Context, kid, keyData, algorithm string, use sql.NullString, createdAt string, expiresAt sql.NullString) error {
	return s.q.StoreJWK(ctx, sqlcsqlite.StoreJWKParams{
		Kid:       kid,
		KeyData:   keyData,
		Algorithm: algorithm,
		Use:       use,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	})
}

func (s *sqliteQuerier) deleteExpiredJWKS(ctx context.Context) error {
	return s.q.DeleteExpiredJWKS(ctx)
}

func (s *sqliteQuerier) getAllCurrentJWKS(ctx context.Context) ([]jwkRow, error) {
	rows, err := s.q.GetAllCurrentJWKS(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]jwkRow, len(rows))
	for i, r := range rows {
		out[i] = jwkRow{
			Kid:       r.Kid,
			KeyData:   r.KeyData,
			Algorithm: r.Algorithm,
			Use:       r.Use,
			CreatedAt: r.CreatedAt,
			ExpiresAt: r.ExpiresAt,
		}
	}
	return out, nil
}
