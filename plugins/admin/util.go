package admin

import (
	"context"
	"time"
)

// CheckBanExpiry checks if a user's ban has expired
// Returns true if the ban has expired and user should be unbanned
func CheckBanExpiry(_ context.Context, user *User) bool {
	if !user.Banned {
		return false
	}

	// If no expiry set, it's a permanent ban
	if user.BanExpiry == nil {
		return false
	}

	// Check if ban has expired
	if time.Now().After(*user.BanExpiry) {
		// Ban has expired, should be unbanned
		return true
	}

	return false
}

// AutoUnbanExpiredUsers finds and unbans all users whose bans have expired
// This should be called periodically by a background job
func AutoUnbanExpiredUsers(ctx context.Context, db interface {
	Query(ctx context.Context, query string, args ...interface{}) (interface{}, error)
	Exec(ctx context.Context, query string, args ...interface{}) (interface{}, error)
}) error {
	// Update all users whose ban has expired
	_, err := db.Exec(ctx, `
		UPDATE auth.user 
		SET banned = false, 
		    ban_reason = NULL, 
		    ban_expiry = NULL
		WHERE banned = true 
		  AND ban_expiry IS NOT NULL 
		  AND ban_expiry <= NOW()
	`)
	return err
}
