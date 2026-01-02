package jwt

// Schema names for OpenAPI specification generation.
//
// These constants define the OpenAPI schema names for JWT request and response types.
// They are used in route metadata to generate accurate API documentation with typed
// request/response examples.
//
// OpenAPI Integration:
// When routes are registered with schema metadata, the OpenAPI plugin uses these
// constants to link HTTP endpoints to their typed request/response structures.
//
// Usage in Route Metadata:
//
//	route := core.Route{
//	    Path: "/getToken",
//	    Handler: handler.HandleGetToken,
//	    Metadata: map[string]any{
//	        "openapi": map[string]any{
//	            "summary": "Generate JWT tokens",
//	            "responses": map[string]any{
//	                "200": map[string]any{
//	                    "description": "Token pair generated",
//	                    "schema": jwt.SchemaTokenPair,
//	                },
//	            },
//	        },
//	    },
//	}
//
// Generated OpenAPI:
// The plugin converts these schema references into OpenAPI 3.0 schema definitions
// with JSON examples, field types, and validation rules.
const (
	// SchemaTokenRequest is the OpenAPI schema name for POST /getToken request body.
	// Currently unused as /getToken accepts no body (uses authenticated user from context).
	SchemaTokenRequest = "TokenRequest"

	// SchemaRefreshTokenRequest is the OpenAPI schema name for POST /refreshToken request.
	// Request Body:
	//   {
	//     "refresh_token": "eyJhbGc..."
	//   }
	SchemaRefreshTokenRequest = "RefreshTokenRequest"

	// SchemaTokenPair is the OpenAPI schema name for token pair responses.
	// Response Body:
	//   {
	//     "access_token": "eyJhbGc...",
	//     "access_expiry": "2024-01-01T12:15:00Z",
	//     "refresh_token": "eyJhbGc...",
	//     "refresh_expiry": "2024-01-08T12:00:00Z"
	//   }
	//
	// Used by:
	//   - POST /getToken (success response)
	//   - POST /getAccessToken (partial - access token only)
	//   - POST /refreshToken (success response)
	SchemaTokenPair = "TokenPair"
)
