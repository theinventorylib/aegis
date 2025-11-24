package core

import (
	"context"
	"errors"

	"github.com/theinventorylib/aegis/models"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	userContextKey contextKey = "aegis_user"
)

// WithUser adds a user to the context
func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUser extracts the user from the context
func GetUser(ctx context.Context) (*models.User, error) {
	user, ok := ctx.Value(userContextKey).(*models.User)
	if !ok || user == nil {
		return nil, errors.New("user not found in context")
	}
	return user, nil
}

// Authenticated checks if the context has an authenticated user
func Authenticated(ctx context.Context) bool {
	user, _ := GetUser(ctx)
	return user != nil
}
