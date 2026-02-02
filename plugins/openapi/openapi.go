// Package openapi provides automatic OpenAPI 3.0 specification generation for Aegis.
//
// This plugin generates interactive API documentation with:
//   - OpenAPI 3.0.3 specification (JSON)
//   - Scalar documentation UI (interactive browser interface)
//   - Automatic schema generation from Go structs
//   - Security scheme definitions (cookie, bearer)
//   - Route metadata collection and transformation
//
// Documentation Features:
//   - Auto-generates schemas from Go types using reflection
//   - Parses validation tags for schema constraints
//   - Supports request/response body documentation
//   - Security requirements (protected vs public routes)
//   - Tag-based organization
//
// Route Structure:
//   - GET /openapi.json - OpenAPI specification (JSON)
//   - GET /docs         - Scalar documentation UI (if enabled)
//
// Usage:
//
//	cfg := &openapi.Config{
//	  Title:          "My API",
//	  Version:        "1.0.0",
//	  EnableScalarUI: true,
//	}
//	plugin := openapi.New(cfg)
//	aegis.RegisterPlugin(plugin)
package openapi

import (
	"context"
	"reflect"
	"sync"

	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/router"
)

// Plugin provides automatic OpenAPI 3.0 documentation generation.
//
// This plugin integrates with the Aegis routing system to collect route metadata
// and generate comprehensive API documentation with interactive UI.
//
// Features:
//   - Real-time spec generation from route metadata
//   - Thread-safe spec updates
//   - Scalar UI integration for interactive documentation
//   - Multiple security scheme support
//   - Schema validation from Go struct tags
type Plugin struct {
	// spec holds the OpenAPI 3.0 specification
	spec *Spec
	// config holds plugin configuration
	config *Config
	// mu protects spec for thread-safe updates during spec regeneration
	mu sync.RWMutex
}

// Config holds OpenAPI plugin configuration.
//
// Example:
//
//	cfg := &openapi.Config{
//	  Title:          "Aegis Authentication API",
//	  Version:        "1.0.0",
//	  Description:    "Complete authentication API",
//	  EnableScalarUI: true,
//	  BasePath:       "/auth",
//	  Servers: []openapi.Server{
//	    {URL: "https://api.example.com", Description: "Production"},
//	  },
//	}
type Config struct {
	// Title for the API documentation
	Title string
	// Version of the API
	Version string
	// Description of the API
	Description string
	// Servers to include in the spec (for multi-environment APIs)
	Servers []Server
	// Contact information for API maintainers
	Contact *Contact
	// License information for the API
	License *License
	// EnableScalarUI enables the Scalar documentation UI at /docs
	EnableScalarUI bool
	// BasePath for the API (e.g., "/auth", "/api/v1")
	BasePath string
}

// DefaultConfig returns default OpenAPI configuration.
//
// Default Settings:
//   - Title: "Aegis Authentication API"
//   - Version: "1.0.0"
//   - ScalarUI: Enabled
//   - BasePath: "/auth"
//   - Server: http://localhost:8080 (development)
//   - License: MIT
//
// Returns:
//   - *Config: Default configuration ready for customization
func DefaultConfig() *Config {
	return &Config{
		Title:          "Aegis Authentication API",
		Version:        "1.0.0",
		Description:    "API documentation for Aegis authentication framework",
		EnableScalarUI: true,
		BasePath:       "/auth",
		Servers: []Server{
			{
				URL:         "http://localhost:8080",
				Description: "Development server",
			},
		},
		License: &License{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
	}
}

// New creates a new OpenAPI plugin with automatic schema generation.
//
// Initialization:
//  1. Create base OpenAPI 3.0.3 spec
//  2. Configure servers, contact, license
//  3. Add security schemes (cookie, bearer)
//  4. Add default tags (default, Session)
//  5. Add common schemas (Error, Success)
//
// Parameters:
//   - cfg: Plugin configuration (uses defaults if nil)
//
// Returns:
//   - *Plugin: Configured plugin ready for initialization
//
// Example:
//
//	plugin := openapi.New(&openapi.Config{
//	  Title:   "My API",
//	  Version: "2.0.0",
//	})
func New(cfg *Config) *Plugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create base spec
	spec := NewSpec(cfg.Title, cfg.Version, cfg.Description)
	spec.Servers = cfg.Servers

	if cfg.Contact != nil {
		spec.Info.Contact = cfg.Contact
	}
	if cfg.License != nil {
		spec.Info.License = cfg.License
	}

	// Add default security schemes
	spec.AddSecurityScheme("cookieAuth", &SecurityScheme{
		Type:        "apiKey",
		In:          "cookie",
		Name:        "aegis_session",
		Description: "Session cookie authentication",
	})

	spec.AddSecurityScheme("bearerAuth", &SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "Bearer token authentication (JWT or session token)",
	})

	// Add default tags
	spec.AddTag(Tag{Name: "Default", Description: "Core authentication endpoints"})
	spec.AddTag(Tag{Name: "Session", Description: "Session management endpoints"})

	// Add common schemas
	addCommonSchemas(spec)

	return &Plugin{
		spec:   spec,
		config: cfg,
	}
}

// Name returns the plugin identifier.
func (p *Plugin) Name() string {
	return "openapi"
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "OpenAPI 3.0 documentation generation with Scalar UI"
}

// Init initializes the OpenAPI plugin.
func (p *Plugin) Init(_ context.Context, _ plugins.Aegis) error {
	// Auto-register core model schemas from actual Go types
	p.RegisterSchemaFromType(core.SchemaUser, auth.User{})
	p.RegisterSchemaFromType(core.SchemaEnrichedUser, core.EnrichedUser{})
	p.RegisterSchemaFromType(core.SchemaSession, auth.Session{})
	p.RegisterSchemaFromType(core.SchemaSessionWithUser, core.SessionWithUser{})
	p.RegisterSchemaFromType(core.SchemaLoginRequest, core.LoginRequest{})
	p.RegisterSchemaFromType(core.SchemaRegisterRequest, core.RegisterRequest{})

	return nil
}

// GetMigrations returns the plugin migrations.
func (p *Plugin) GetMigrations() []plugins.Migration {
	// No migrations needed - stateless plugin
	return []plugins.Migration{}
}

// MountRoutes registers HTTP routes for the plugin.
func (p *Plugin) MountRoutes(router router.Router, prefix string) {
	handler := NewHandler(p, router)

	// Serve OpenAPI spec as JSON
	router.GET(prefix+"/openapi.json", handler.ServeSpec)
	router.RegisterRouteMetadata(core.RouteMetadata{
		Method:      "GET",
		Path:        prefix + "/openapi.json",
		Summary:     "OpenAPI spec",
		Description: "Get the OpenAPI specification JSON",
		Tags:        []string{"OpenAPI"},
		Protected:   false,
		Responses: map[string]*core.ResponseMeta{
			"200": {Description: "OpenAPI JSON"},
		},
	})

	// Serve Scalar UI if enabled
	if p.config.EnableScalarUI {
		router.GET(prefix+"/docs", handler.ServeScalarUI)
		router.RegisterRouteMetadata(core.RouteMetadata{
			Method:      "GET",
			Path:        prefix + "/docs",
			Summary:     "API docs UI",
			Description: "Interactive API documentation UI",
			Tags:        []string{"OpenAPI"},
			Protected:   false,
			Responses: map[string]*core.ResponseMeta{
				"200": {Description: "HTML docs UI"},
			},
		})
	}
}

// UpdateSpec updates the OpenAPI spec with route metadata.
func (p *Plugin) UpdateSpec(metadata []core.RouteMetadata) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, meta := range metadata {
		// Skip OpenAPI's own routes if they are tagged as "OpenAPI"
		skip := false
		for _, tag := range meta.Tags {
			if tag == "OpenAPI" {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Create operation
		op := &Operation{
			Summary:     meta.Summary,
			Description: meta.Description,
			Tags:        meta.Tags,
			Responses:   make(map[string]*Response),
		}

		if meta.Protected {
			op.Security = []SecurityRequirement{
				{"cookieAuth": []string{}},
				{"bearerAuth": []string{}},
			}
		}

		// Handle Request Body
		if meta.RequestBody != nil {
			schema := p.resolveSchema(meta.RequestBody.Schema)
			op.RequestBody = &RequestBody{
				Description: meta.RequestBody.Description,
				Required:    meta.RequestBody.Required,
				Content: map[string]MediaType{
					"application/json": {
						Schema: schema,
					},
				},
			}
		}

		// Handle Responses
		for status, respMeta := range meta.Responses {
			schema := p.resolveSchema(respMeta.Schema)
			op.Responses[status] = &Response{
				Description: respMeta.Description,
				Content: map[string]MediaType{
					"application/json": {
						Schema: schema,
					},
				},
			}
		}

		// Add path to spec
		p.addPathOperation(meta.Path, meta.Method, op)
	}
}

// resolveSchema resolves a schema reference or definition.
// If v is a string, it returns a reference to that schema name.
// If v is a struct/type, it generates the schema, registers it, and returns a reference.
func (p *Plugin) resolveSchema(v any) *Schema {
	if v == nil {
		return nil
	}

	// If string, assume it's a reference name
	if name, ok := v.(string); ok {
		return RefSchema(name)
	}

	// If it's a struct/type, generate schema using reflection
	schema := GenerateSchema(v)

	// Get type name for registration
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Struct {
		name := t.Name()
		if name != "" {
			p.spec.AddSchema(name, schema)
			return RefSchema(name)
		}
	}

	return schema
}

func (p *Plugin) addPathOperation(path, method string, op *Operation) {
	pathItem := p.spec.Paths[path]
	if pathItem == nil {
		pathItem = &PathItem{}
		p.spec.Paths[path] = pathItem
	}

	switch method {
	case "GET":
		pathItem.Get = op
	case "POST":
		pathItem.Post = op
	case "PUT":
		pathItem.Put = op
	case "DELETE":
		pathItem.Delete = op
	case "PATCH":
		pathItem.Patch = op
	case "OPTIONS":
		pathItem.Options = op
	case "HEAD":
		pathItem.Head = op
	}
}

// RequiresTables returns required database tables.
func (p *Plugin) RequiresTables() []string {
	// No database tables required
	return []string{}
}

// ProvidesAuthMethods returns authentication methods provided.
func (p *Plugin) ProvidesAuthMethods() []string {
	// Doesn't provide auth methods, only documentation
	return []string{}
}

// GetSchemas returns all schemas for all supported dialects
func (p *Plugin) GetSchemas() []plugins.Schema {
	// OpenAPI plugin doesn't have its own schema
	return []plugins.Schema{}
}

// Dependencies returns plugin dependencies.
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}

// RegisterEndpoint adds a custom endpoint to the OpenAPI spec.
// This allows other plugins and user code to extend the documentation.
func (p *Plugin) RegisterEndpoint(method, path string, operation *Operation) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pathItem := p.spec.Paths[path]
	if pathItem == nil {
		pathItem = &PathItem{}
		p.spec.Paths[path] = pathItem
	}

	switch method {
	case "GET":
		pathItem.Get = operation
	case "POST":
		pathItem.Post = operation
	case "PUT":
		pathItem.Put = operation
	case "DELETE":
		pathItem.Delete = operation
	case "PATCH":
		pathItem.Patch = operation
	case "OPTIONS":
		pathItem.Options = operation
	case "HEAD":
		pathItem.Head = operation
	}
}

// RegisterSchema adds a reusable schema component.
func (p *Plugin) RegisterSchema(name string, schema *Schema) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spec.AddSchema(name, schema)
}

// RegisterSecurityScheme adds a custom security scheme.
func (p *Plugin) RegisterSecurityScheme(name string, scheme *SecurityScheme) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spec.AddSecurityScheme(name, scheme)
}

// RegisterTag adds a tag for grouping operations.
func (p *Plugin) RegisterTag(tag Tag) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.spec.AddTag(tag)
}

// GetSpec returns a copy of the current OpenAPI spec.
func (p *Plugin) GetSpec() *Spec {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.spec
}

// RegisterSchemaFromType automatically generates and registers an OpenAPI schema from a Go type.
// This eliminates the need for manual schema definitions and ensures schemas stay in sync with Go structs.
//
// Example usage:
//
//	p.RegisterSchemaFromType("User", core.User{})
//	p.RegisterSchemaFromType("CreateOrganizationRequest", organizations.CreateOrganizationRequest{})
func (p *Plugin) RegisterSchemaFromType(name string, example any) {
	schema := GenerateSchema(example)
	p.spec.AddSchema(name, schema)
}
