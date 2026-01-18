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

// addCoreRoutes adds core Aegis authentication routes to the spec.
//
// This function documents the core authentication endpoints that are
// always available in Aegis:
//   - POST /refresh - Refresh session with refresh token
//   - POST /logout  - Invalidate current session
//
// These routes are added during plugin initialization and serve as
// baseline documentation. Plugin routes are added dynamically via
// route metadata collection.
//
// Parameters:
//   - spec: OpenAPI spec to add routes to
//   - basePath: Base path for routes (e.g., "/auth")
func addCoreRoutes(spec *Spec, basePath string) {
	// Session refresh endpoint
	spec.AddPath(basePath+"/refresh", &PathItem{
		Post: &Operation{
			Tags:        []string{"Session"},
			Summary:     "Refresh session",
			Description: "Refresh an existing session to extend its lifetime",
			OperationID: "refreshSession",
			RequestBody: &RequestBody{
				Required:    true,
				Description: "Refresh token",
				Content: map[string]MediaType{
					"application/json": {
						Schema: ObjectSchema("", map[string]*Schema{
							"refresh_token": StringSchema("Refresh token"),
						}, []string{"refresh_token"}),
					},
				},
			},
			Responses: map[string]*Response{
				"200": {
					Description: "Session refreshed successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: RefSchema(core.SchemaSession),
						},
					},
				},
				"401": {
					Description: "Invalid or expired refresh token",
					Content: map[string]MediaType{
						"application/json": {
							Schema: RefSchema(core.SchemaError),
						},
					},
				},
			},
		},
	})

	// Logout endpoint
	spec.AddPath(basePath+"/logout", &PathItem{
		Post: &Operation{
			Tags:        []string{"Session"},
			Summary:     "Logout",
			Description: "Invalidate the current session",
			OperationID: "logout",
			Security: []SecurityRequirement{
				{"cookieAuth": []string{}},
				{"bearerAuth": []string{}},
			},
			Responses: map[string]*Response{
				"200": {
					Description: "Logged out successfully",
					Content: map[string]MediaType{
						"application/json": {
							Schema: RefSchema(core.SchemaSuccess),
						},
					},
				},
				"401": {
					Description: "Not authenticated",
					Content: map[string]MediaType{
						"application/json": {
							Schema: RefSchema(core.SchemaError),
						},
					},
				},
			},
		},
	})
}
