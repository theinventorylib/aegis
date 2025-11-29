package openapi

// addCommonSchemas adds common reusable schemas to the spec.
// Now uses auto-generation from actual Go types to ensure schemas stay in sync.
func addCommonSchemas(spec *Spec) {
	// Error response schema - manually defined as it's a simple structure
	spec.AddSchema("Error", ObjectSchema("Error response", map[string]*Schema{
		"success": BooleanSchema("Always false for errors"),
		"error":   StringSchema("Error message"),
	}, []string{"success", "error"}))

	// Success response schema - manually defined as it's a simple structure
	spec.AddSchema("Success", ObjectSchema("Success response", map[string]*Schema{
		"success": BooleanSchema("Always true for success"),
		"message": StringSchema("Success message"),
	}, []string{"success"}))

	// Note: User and Session schemas should be auto-registered from models
	// This is done in the Init() method or by plugins
}

// addCoreRoutes adds core Aegis authentication routes to the spec.
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
							Schema: RefSchema("Session"),
						},
					},
				},
				"401": {
					Description: "Invalid or expired refresh token",
					Content: map[string]MediaType{
						"application/json": {
							Schema: RefSchema("Error"),
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
							Schema: RefSchema("Success"),
						},
					},
				},
				"401": {
					Description: "Not authenticated",
					Content: map[string]MediaType{
						"application/json": {
							Schema: RefSchema("Error"),
						},
					},
				},
			},
		},
	})
}
