package oauth

// Schema names for OpenAPI specification generation.
//
// These constants define the OpenAPI schema names for OAuth request types.
// They are used in route metadata to generate accurate API documentation.
const (
	// SchemaLinkAccountRequest is the OpenAPI schema name for account linking.
	// Request Body:
	//   {
	//     "provider": "google"
	//   }
	SchemaLinkAccountRequest = "LinkAccountRequest"
)
