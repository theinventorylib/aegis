package sms

import "time"

// SMSVerification represents a phone number verification record
type SMSVerification struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId,omitempty"`
	PhoneNumber string    `json:"phoneNumber"`
	Code        string    `json:"-"` // Never expose in JSON
	Purpose     string    `json:"purpose"`
	Verified    bool      `json:"verified"`
	ExpiresAt   time.Time `json:"expiresAt"`
	CreatedAt   time.Time `json:"createdAt"`
}
