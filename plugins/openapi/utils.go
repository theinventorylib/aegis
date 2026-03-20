package openapi

import (
	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/core"
)

// ========== SCHEMA UTILITY FUNCTIONS ==========
//
// These functions provide a fluent API for building OpenAPI schemas.
// They are used internally by the spec generator and can be used by
// plugins to manually define custom schemas.

// addCommonSchemas adds common reusable schemas to the spec.
//
// This function adds standard response schemas:
//   - Error: {"success": false, "error": "message"}
//   - Success: {"success": true, "message": "message"}
//   - Session: auth.Session model
//   - User: auth.User model
//   - RefreshTokenRequest: {"refreshToken": "..."}
//
// These schemas are referenced by core routes and plugins.
func addCommonSchemas(spec *Spec) {
	// Error response schema - manually defined as it's a simple structure
	spec.AddSchema(core.SchemaError, ObjectSchema("Error response", map[string]*Schema{
		"success": BooleanSchema("Always false for errors"),
		"error":   StringSchema("Error message"),
	}, []string{"success", "error"}))

	// Success response schema - manually defined as it's a simple structure
	spec.AddSchema(core.SchemaSuccess, ObjectSchema("Success response", map[string]*Schema{
		"success": BooleanSchema("Always true for success"),
		"message": StringSchema("Success message"),
	}, []string{"success"}))

	// Session schema - auto-generated from auth.Session model
	spec.AddSchema("Session", GenerateSchema(auth.Session{}))

	// User schema - auto-generated from auth.User model
	spec.AddSchema("User", GenerateSchema(auth.User{}))

	// RefreshTokenRequest schema - used by core session refresh and JWT refresh
	spec.AddSchema("RefreshTokenRequest", ObjectSchema("Refresh token request", map[string]*Schema{
		"refreshToken": StringSchema("The refresh token to use"),
	}, []string{"refreshToken"}))
}
