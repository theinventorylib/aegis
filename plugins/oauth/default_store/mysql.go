package defaultstore

// mysql.go — wraps sqlcmysql.Queries and implements querier.
// All fields are string or sql.NullString — no type conversion needed.

import (
	"context"
	"database/sql"

	sqlcmysql "github.com/theinventorylib/aegis/plugins/oauth/internal/gen/mysql"
)

type mysqlQuerier struct{ q *sqlcmysql.Queries }

func newMysqlQuerier(db *sql.DB) *mysqlQuerier {
	return &mysqlQuerier{q: sqlcmysql.New(db)}
}

func (m *mysqlQuerier) createConnection(ctx context.Context, r connectionRow) error {
	return m.q.CreateConnection(ctx, sqlcmysql.CreateConnectionParams{
		ID:             r.ID,
		UserID:         r.UserID,
		Provider:       r.Provider,
		ProviderUserID: r.ProviderUserID,
		Email:          r.Email,
		Name:           r.Name,
		AvatarUrl:      r.AvatarURL,
		AccessToken:    r.AccessToken,
		RefreshToken:   r.RefreshToken,
		ExpiresAt:      r.ExpiresAt,
		ProviderData:   r.ProviderData,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	})
}

func (m *mysqlQuerier) getConnectionByProviderUserID(ctx context.Context, provider, providerUserID string) (connectionRow, error) {
	c, err := m.q.GetConnectionByProviderUserID(ctx, sqlcmysql.GetConnectionByProviderUserIDParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		return connectionRow{}, err
	}
	return connectionRow{
		ID: c.ID, UserID: c.UserID, Provider: c.Provider, ProviderUserID: c.ProviderUserID,
		Email: c.Email, Name: c.Name, AvatarURL: c.AvatarUrl,
		AccessToken: c.AccessToken, RefreshToken: c.RefreshToken,
		ExpiresAt: c.ExpiresAt, ProviderData: c.ProviderData,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}, nil
}

func (m *mysqlQuerier) getConnectionsByUserID(ctx context.Context, userID string) ([]connectionRow, error) {
	rows, err := m.q.GetConnectionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]connectionRow, len(rows))
	for i, c := range rows {
		result[i] = connectionRow{
			ID: c.ID, UserID: c.UserID, Provider: c.Provider, ProviderUserID: c.ProviderUserID,
			Email: c.Email, Name: c.Name, AvatarURL: c.AvatarUrl,
			AccessToken: c.AccessToken, RefreshToken: c.RefreshToken,
			ExpiresAt: c.ExpiresAt, ProviderData: c.ProviderData,
			CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
		}
	}
	return result, nil
}

func (m *mysqlQuerier) updateConnection(ctx context.Context, r connectionRow) error {
	return m.q.UpdateConnection(ctx, sqlcmysql.UpdateConnectionParams{
		ID:             r.ID,
		UserID:         r.UserID,
		Provider:       r.Provider,
		ProviderUserID: r.ProviderUserID,
		Email:          r.Email,
		Name:           r.Name,
		AvatarUrl:      r.AvatarURL,
		AccessToken:    r.AccessToken,
		RefreshToken:   r.RefreshToken,
		ExpiresAt:      r.ExpiresAt,
		ProviderData:   r.ProviderData,
		UpdatedAt:      r.UpdatedAt,
	})
}

func (m *mysqlQuerier) deleteConnection(ctx context.Context, provider, userID string) error {
	return m.q.DeleteConnection(ctx, sqlcmysql.DeleteConnectionParams{
		Provider: provider,
		UserID:   userID,
	})
}
