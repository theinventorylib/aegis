package sms

// Provider is the interface for SMS providers
type Provider interface {
	SendSMS(to, message string) error
	SendOTP(to, code string) error
}
