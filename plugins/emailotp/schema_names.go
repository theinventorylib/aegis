package emailotp

// Schema names for OpenAPI specification generation.
//
// These constants define the OpenAPI schema names for emailotp request/response types.
// They are used in route metadata to generate accurate API documentation with typed
// request/response examples.
const (
	// Request schemas
	SchemaLoginWithEmailRequest    = "LoginWithEmailRequest"
	SchemaRegisterWithEmailRequest = "RegisterWithEmailRequest"
	SchemaSendOTPRequest           = "SendOTPRequest"
	SchemaVerifyOTPRequest         = "VerifyOTPRequest"
)
