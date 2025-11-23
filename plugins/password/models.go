package password

import "time"

// PasswordAccount represents a password-based authentication account
// Stored in auth.accounts table with provider = "password"
type PasswordAccount struct {
	ID           string    `json:"id"`
	UserID       string    `json:"userId"`
	Provider     string    `json:"provider"` // Always "password"
	PasswordHash string    `json:"-"`        // Never expose in JSON
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
