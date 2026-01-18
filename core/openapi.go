package core

// Core schema names for OpenAPI documentation.
//
// These constants provide type-safe references to OpenAPI schema names used
// throughout the Aegis framework. They are used by:
//   - RouteMetadata: Annotating HTTP handlers with request/response schemas
//   - OpenAPI plugin: Generating OpenAPI 3.0 specifications
//   - API documentation: Auto-generating API docs from code
//
// Benefits of using constants:
//   - Type safety: Compile-time checking (no typos in schema names)
//   - Refactoring: Easy to rename schemas across the codebase
//   - Discoverability: IDE autocomplete shows available schemas
//
// Naming convention:
//   - Singular for entities: SchemaUser, not SchemaUsers
//   - Descriptive suffixes: SchemaUserList for lists, SchemaLoginRequest for requests
//
// Example usage in RouteMetadata:
//
//	metadata := &core.RouteMetadata{
//		RequestBody: &core.RequestBodyMeta{
//			Schema: core.SchemaLoginRequest,
//		},
//		Responses: map[string]*core.ResponseMeta{
//			"200": {Schema: core.SchemaUser},
//			"401": {Schema: core.SchemaError},
//		},
//	}
const (
	// Core entity schemas
	SchemaUser            = "User"
	SchemaEnrichedUser    = "EnrichedUser"
	SchemaSession         = "Session"
	SchemaSessionWithUser = "SessionWithUser"

	// Common response schemas
	SchemaError   = "Error"
	SchemaSuccess = "Success"

	// Request schemas
	SchemaRefreshTokenRequest = "RefreshTokenRequest"
	SchemaLoginRequest        = "LoginRequest"
	SchemaRegisterRequest     = "RegisterRequest"

	// List schemas
	SchemaSessionList = "SessionList"
)
