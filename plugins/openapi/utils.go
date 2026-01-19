package openapi

import (
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
//
// These schemas are referenced by core routes and plugins.
//
// Note: User and Session schemas are auto-registered from Go types
// during Init() to ensure they stay in sync with model definitions.
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

	// Note: User and Session schemas should be auto-registered from models
	// This is done in the Init() method or by plugins
}
