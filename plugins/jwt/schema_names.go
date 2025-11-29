package jwt

// Schema names for OpenAPI documentation.
// These constants ensure type-safe schema references in route metadata.
const (
	// Request schemas
	SchemaTokenRequest        = "TokenRequest"
	SchemaRefreshTokenRequest = "RefreshTokenRequest"

	// Response schemas
	SchemaTokenPair = "TokenPair"
)
