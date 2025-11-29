package models

// RouteMetadata contains documentation metadata for a route.
// This is used by the OpenAPI plugin to automatically generate API documentation.
type RouteMetadata struct {
	Method      string                   // HTTP method (GET, POST, PUT, DELETE, etc.)
	Path        string                   // Route path (e.g., "/auth/logout")
	Summary     string                   // Short summary of the endpoint
	Description string                   // Detailed description
	Tags        []string                 // Tags for grouping operations
	Protected   bool                     // Whether the route requires authentication
	RequestBody *RequestBodyMeta         // Request body schema (optional)
	Responses   map[string]*ResponseMeta // Response schemas by status code
}

// RequestBodyMeta describes the request body for an endpoint.
type RequestBodyMeta struct {
	Description string      // Description of the request body
	Required    bool        // Whether the request body is required
	Schema      interface{} // Schema name as string (e.g., "CreateOrganizationRequest") or inline schema definition
}

// ResponseMeta describes a response for an endpoint.
type ResponseMeta struct {
	Description string      // Description of the response
	Schema      interface{} // Schema name as string (e.g., "Organization", "Error") or inline schema definition
}
