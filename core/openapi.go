package core

// Core schema names for OpenAPI documentation.
//
// These constants provide type-safe references to OpenAPI schema names used
// throughout the Aegis framework. They are used by:
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
const (
	// Common response schemas
	SchemaError   = "Error"
	SchemaSuccess = "Success"
)
