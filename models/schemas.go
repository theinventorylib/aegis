package models

// Core schema names for OpenAPI documentation.
// These constants ensure type-safe schema references across the codebase.
const (
	// Core entity schemas
	SchemaUser    = "User"
	SchemaSession = "Session"

	// Common response schemas
	SchemaError   = "Error"
	SchemaSuccess = "Success"

	// Request schemas
	SchemaRefreshTokenRequest = "RefreshTokenRequest"

	// List schemas
	SchemaSessionList = "SessionList"
)
