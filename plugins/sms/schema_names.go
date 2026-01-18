package sms

// Schema names for OpenAPI specification generation.
// These constants define the OpenAPI schema names for SMS request/response types.
// They are used in route metadata to generate accurate API documentation with typed
// request/response examples.
const (
	// Request schemas
	SchemaLoginWithPhoneRequest    = "LoginWithPhoneRequest"
	SchemaRegisterWithPhoneRequest = "RegisterWithPhoneRequest"
	SchemaSendOTPRequest           = "SendOTPRequest"
	SchemaVerifyOTPRequest         = "VerifyOTPRequest"
)
