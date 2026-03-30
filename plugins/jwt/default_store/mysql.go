package defaultstore

// mysql.go — thin translator: wraps sqlcmysql.Queries and implements querier.
// Dialect-specific types (time.Time for CreatedAt, sql.NullTime for ExpiresAt)
// are handled here and nowhere else.

import (
	"context"
	"database/sql"
	"time"

	sqlcmysql "github.com/theinventorylib/aegis/plugins/jwt/internal/gen/mysql"
)

type mysqlQuerier struct{ q *sqlcmysql.Queries }

func newMySQLQuerier(db *sql.DB) *mysqlQuerier {
	return &mysqlQuerier{q: sqlcmysql.New(db)}
}

func (m *mysqlQuerier) getCurrentJWK(ctx context.Context, algorithm string, use sql.NullString) (string, error) {
	return m.q.GetCurrentJWK(ctx, sqlcmysql.GetCurrentJWKParams{
		Algorithm: algorithm,
		Use:       use,
	})
}

func (m *mysqlQuerier) storeJWK(ctx context.Context, kid, keyData, algorithm string, use sql.NullString, createdAt string, expiresAt sql.NullString) error {
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
	return m.q.StoreJWK(ctx, sqlcmysql.StoreJWKParams{
		Kid:       kid,
		KeyData:   keyData,
		Algorithm: algorithm,
		Use:       use,
		CreatedAt: ct,
		ExpiresAt: ea,
	})
}

func (m *mysqlQuerier) deleteExpiredJWKS(ctx context.Context) error {
	return m.q.DeleteExpiredJWKS(ctx)
}

func (m *mysqlQuerier) getAllCurrentJWKS(ctx context.Context) ([]jwkRow, error) {
	rows, err := m.q.GetAllCurrentJWKS(ctx)
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
