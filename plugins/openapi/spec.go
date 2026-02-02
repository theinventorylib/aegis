// Package openapi provides OpenAPI 3.0 specification types and generation.
package openapi

import (
	"encoding/json"
	"time"
)

// Spec represents an OpenAPI 3.0 specification document.
//
// This is the root object of an OpenAPI 3.0 document, containing all
// metadata, paths, components, and security definitions.
//
// Spec Structure:
//   - OpenAPI: Version ("3.0.3")
//   - Info: API metadata (title, version, description)
//   - Servers: API server URLs (dev, staging, production)
//   - Paths: All API endpoints and operations
//   - Components: Reusable schemas, security schemes
//   - Security: Global security requirements
//   - Tags: Endpoint categorization
type Spec struct {
	OpenAPI    string                `json:"openapi"`              // OpenAPI version (3.0.3)
	Info       Info                  `json:"info"`                 // API metadata
	Servers    []Server              `json:"servers,omitempty"`    // Server URLs
	Paths      map[string]*PathItem  `json:"paths"`                // API endpoints
	Components *Components           `json:"components,omitempty"` // Reusable components
	Security   []SecurityRequirement `json:"security,omitempty"`   // Global security
	Tags       []Tag                 `json:"tags,omitempty"`       // Endpoint tags
}

// Info provides metadata about the API.
//
// Example:
//
//	info := Info{
//	  Title:       "Aegis API",
//	  Version:     "1.0.0",
//	  Description: "Authentication API",
//	  Contact: &Contact{
//	    Name:  "API Support",
//	    Email: "support@example.com",
//	  },
//	}
type Info struct {
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
	Version        string   `json:"version"`
}

// Contact information for the API.
type Contact struct {
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
	Email string `json:"email,omitempty"`
}

// License information for the API.
type License struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// Server represents a server URL.
type Server struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"`
}

// ServerVariable represents a server URL template variable.
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty"`
	Default     string   `json:"Default"`
	Description string   `json:"description,omitempty"`
}

// PathItem describes operations available on a single path.
type PathItem struct {
	Summary     string     `json:"summary,omitempty"`
	Description string     `json:"description,omitempty"`
	Get         *Operation `json:"get,omitempty"`
	Post        *Operation `json:"post,omitempty"`
	Put         *Operation `json:"put,omitempty"`
	Delete      *Operation `json:"delete,omitempty"`
	Patch       *Operation `json:"patch,omitempty"`
	Options     *Operation `json:"options,omitempty"`
	Head        *Operation `json:"head,omitempty"`
}

// Operation describes a single API operation on a path.
//
// This represents an HTTP method (GET, POST, etc.) on a specific path.
// Each operation defines:
//   - Request requirements (parameters, body)
//   - Response definitions (status codes, schemas)
//   - Security requirements
//   - Metadata (tags, summary, description)
//
// Example:
//
//	op := &Operation{
//	  Tags:        []string{"Default"},
//	  Summary:     "Login",
//	  Description: "Authenticate user with email and password",
//	  RequestBody: &RequestBody{...},
//	  Responses:   map[string]*Response{...},
//	  Security:    []SecurityRequirement{{"cookieAuth": []string{}}},
//	}
type Operation struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]*Response  `json:"responses"`
	Security    []SecurityRequirement `json:"security,omitempty"`
	Deprecated  bool                  `json:"deprecated,omitempty"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // "query", "header", "path", "cookie"
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
	Example     any     `json:"example,omitempty"`
}

// RequestBody describes a single request body.
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
	Required    bool                 `json:"required,omitempty"`
}

// Response describes a single response from an API operation.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
	Headers     map[string]*Header   `json:"headers,omitempty"`
}

// MediaType provides schema and examples for the media type.
type MediaType struct {
	Schema  *Schema `json:"schema,omitempty"`
	Example any     `json:"example,omitempty"`
}

// Header describes a single header parameter.
type Header struct {
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Schema represents a JSON Schema object.
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Title                string             `json:"title,omitempty"`
	Description          string             `json:"description,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Enum                 []any              `json:"enum,omitempty"`
	Example              any                `json:"example,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
	AdditionalProperties any                `json:"additionalProperties,omitempty"`

	// Validation constraints
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Minimum   *float64 `json:"minimum,omitempty"`
	Maximum   *float64 `json:"maximum,omitempty"`
	MinItems  *int     `json:"minItems,omitempty"`
	MaxItems  *int     `json:"maxItems,omitempty"`
}

// Components holds reusable objects for different aspects of the OAS.
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	Responses       map[string]*Response       `json:"responses,omitempty"`
	Parameters      map[string]*Parameter      `json:"parameters,omitempty"`
	RequestBodies   map[string]*RequestBody    `json:"requestBodies,omitempty"`
	Headers         map[string]*Header         `json:"headers,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme defines a security scheme that can be used by operations.
type SecurityScheme struct {
	Type             string      `json:"type"` // "apiKey", "http", "oauth2", "openIdConnect"
	Description      string      `json:"description,omitempty"`
	Name             string      `json:"name,omitempty"`             // For apiKey
	In               string      `json:"in,omitempty"`               // For apiKey: "query", "header", "cookie"
	Scheme           string      `json:"scheme,omitempty"`           // For http: "bearer", "basic"
	BearerFormat     string      `json:"bearerFormat,omitempty"`     // For http bearer
	Flows            *OAuthFlows `json:"flows,omitempty"`            // For oauth2
	OpenIDConnectURL string      `json:"openIdConnectUrl,omitempty"` // For openIdConnect
}

// OAuthFlows allows configuration of the supported OAuth Flows.
type OAuthFlows struct {
	Implicit          *OAuthFlow `json:"implicit,omitempty"`
	Password          *OAuthFlow `json:"password,omitempty"`
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty"`
}

// OAuthFlow configuration details.
type OAuthFlow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes"`
}

// SecurityRequirement lists required security schemes to execute an operation.
type SecurityRequirement map[string][]string

// Tag adds metadata to a single tag.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// NewSpec creates a new OpenAPI 3.0 specification with default values.
func NewSpec(title, version, description string) *Spec {
	return &Spec{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:       title,
			Version:     version,
			Description: description,
		},
		Paths: make(map[string]*PathItem),
		Components: &Components{
			Schemas:         make(map[string]*Schema),
			Responses:       make(map[string]*Response),
			Parameters:      make(map[string]*Parameter),
			SecuritySchemes: make(map[string]*SecurityScheme),
		},
		Tags: []Tag{},
	}
}

// AddPath adds or updates a path in the specification.
func (s *Spec) AddPath(path string, item *PathItem) {
	if s.Paths == nil {
		s.Paths = make(map[string]*PathItem)
	}
	s.Paths[path] = item
}

// AddSchema adds a reusable schema component.
func (s *Spec) AddSchema(name string, schema *Schema) {
	if s.Components == nil {
		s.Components = &Components{
			Schemas: make(map[string]*Schema),
		}
	}
	if s.Components.Schemas == nil {
		s.Components.Schemas = make(map[string]*Schema)
	}
	s.Components.Schemas[name] = schema
}

// AddSecurityScheme adds a security scheme component.
func (s *Spec) AddSecurityScheme(name string, scheme *SecurityScheme) {
	if s.Components == nil {
		s.Components = &Components{
			SecuritySchemes: make(map[string]*SecurityScheme),
		}
	}
	if s.Components.SecuritySchemes == nil {
		s.Components.SecuritySchemes = make(map[string]*SecurityScheme)
	}
	s.Components.SecuritySchemes[name] = scheme
}

// AddTag adds a tag to the specification.
func (s *Spec) AddTag(tag Tag) {
	s.Tags = append(s.Tags, tag)
}

// ToJSON converts the spec to JSON bytes.
func (s *Spec) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// Helper functions for common schemas

// StringSchema creates a string schema.
func StringSchema(description string) *Schema {
	return &Schema{
		Type:        "string",
		Description: description,
	}
}

// IntegerSchema creates an integer schema.
func IntegerSchema(description string) *Schema {
	return &Schema{
		Type:        "integer",
		Description: description,
	}
}

// BooleanSchema creates a boolean schema.
func BooleanSchema(description string) *Schema {
	return &Schema{
		Type:        "boolean",
		Description: description,
	}
}

// ObjectSchema creates an object schema.
func ObjectSchema(description string, properties map[string]*Schema, required []string) *Schema {
	return &Schema{
		Type:        "object",
		Description: description,
		Properties:  properties,
		Required:    required,
	}
}

// ArraySchema creates an array schema.
func ArraySchema(description string, items *Schema) *Schema {
	return &Schema{
		Type:        "array",
		Description: description,
		Items:       items,
	}
}

// RefSchema creates a reference to a schema component.
func RefSchema(ref string) *Schema {
	return &Schema{
		Ref: "#/components/schemas/" + ref,
	}
}

// DateTimeSchema creates a date-time string schema.
func DateTimeSchema(description string) *Schema {
	return &Schema{
		Type:        "string",
		Format:      "date-time",
		Description: description,
		Example:     time.Now().Format(time.RFC3339),
	}
}

// EmailSchema creates an email string schema.
func EmailSchema(description string) *Schema {
	return &Schema{
		Type:        "string",
		Format:      "email",
		Description: description,
		Example:     "user@example.com",
	}
}

// UUIDSchema creates a UUID string schema.
func UUIDSchema(description string) *Schema {
	return &Schema{
		Type:        "string",
		Format:      "uuid",
		Description: description,
		Example:     "123e4567-e89b-12d3-a456-426614174000",
	}
}
