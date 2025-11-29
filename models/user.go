package models

import "time"

// User represents a user in the system.
type User struct {
	ID        string                 `json:"id"`
	Avatar    *string                `json:"avatar"`
	Name      string                 `json:"name"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
	Disabled  bool                   `json:"disabled"` // Account status
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
