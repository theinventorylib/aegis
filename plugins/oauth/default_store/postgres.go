package defaultstore

// postgres.go — wraps sqlcpostgres.Queries and implements querier.
// All fields are string or sql.NullString — no type conversion needed.

import (
	"context"
	"database/sql"

	sqlcpostgres "github.com/theinventorylib/aegis/plugins/oauth/internal/gen/postgres"
)

type postgresQuerier struct{ q *sqlcpostgres.Queries }

func newPostgresQuerier(db *sql.DB) *postgresQuerier {
	return &postgresQuerier{q: sqlcpostgres.New(db)}
}

func (p *postgresQuerier) createConnection(ctx context.Context, r connectionRow) error {
	return p.q.CreateConnection(ctx, sqlcpostgres.CreateConnectionParams{
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

func (p *postgresQuerier) getConnectionByProviderUserID(ctx context.Context, provider, providerUserID string) (connectionRow, error) {
	c, err := p.q.GetConnectionByProviderUserID(ctx, sqlcpostgres.GetConnectionByProviderUserIDParams{
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

func (p *postgresQuerier) getConnectionsByUserID(ctx context.Context, userID string) ([]connectionRow, error) {
	rows, err := p.q.GetConnectionsByUserID(ctx, userID)
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

func (p *postgresQuerier) updateConnection(ctx context.Context, r connectionRow) error {
	return p.q.UpdateConnection(ctx, sqlcpostgres.UpdateConnectionParams{
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

func (p *postgresQuerier) deleteConnection(ctx context.Context, provider, userID string) error {
	return p.q.DeleteConnection(ctx, sqlcpostgres.DeleteConnectionParams{
		Provider: provider,
		UserID:   userID,
	})
}
