// Package openapi provides automatic OpenAPI 3.0 specification generation for Aegis.
//
// This plugin generates interactive API documentation with:
//   - OpenAPI 3.0.3 specification (JSON)
//   - Scalar documentation UI (interactive browser interface)
//   - Automatic schema generation from Go structs
//   - Security scheme definitions (cookie, bearer)
//   - Global pending queue for decoupled route registration
//
// Documentation Features:
//   - Auto-generates schemas from Go types using reflection
//   - Parses validation tags for schema constraints
//   - Supports request/response body documentation
//   - Security requirements (protected vs public routes)
//   - Tag-based organization
//   - Automatic operationId derivation from Method+Path
//   - Forces required: true on all path parameters
//
// Usage:
//
// Any plugin or user code calls openapi.Doc() to register a route:
//
//	openapi.Doc(openapi.Route{
//	    Method:  "GET",
//	    Path:    "/api/users/{id}",
//	    Summary: "Get user by ID",
//	    Tags:    []string{"Users"},
//	    Auth:    true,
//	    Params: []openapi.Param{
//	        {Name: "id", In: "path", Type: "string", Required: true},
//	    },
//	    Responses: openapi.Responses{
//	        200: openapi.ResponseOf[UserResponse]("User retrieved"),
//	    },
//	})
//
// The OpenAPI plugin is always registered last in a.Use():
//
//	a.Use(ctx, openapi.New(&openapi.Config{
//	    Title:      "My App API",
//	    Version:    "1.0.0",
//	    EnableDocs: true,
//	    DocsPath:   "/docs",
//	    SpecPath:   "/openapi.json",
//	}))
package openapi

import (
	"context"
	"fmt"
	"sync"

	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/router"
)

// Plugin provides automatic OpenAPI 3.0 documentation generation.
//
// This plugin integrates with the Aegis routing system to collect route
// documentation registered via the global Doc() function and generate
// comprehensive API documentation with interactive UI.
//
// Features:
//   - Decoupled registration via global Doc() function
//   - Pending queue that buffers docs until plugin initializes
//   - Thread-safe spec updates
//   - Scalar UI integration for interactive documentation
//   - Multiple security scheme support
//   - Schema validation from Go struct tags
//   - Automatic operationId derivation
type Plugin struct {
	// spec holds the OpenAPI 3.0 specification
	spec *Spec
	// config holds plugin configuration
	config *Config
	// mu protects spec for thread-safe updates
	mu sync.RWMutex
}

// Config holds OpenAPI plugin configuration.
//
// Example:
//
//	cfg := &openapi.Config{
//	    Title:      "My App API",
//	    Version:    "1.0.0",
//	    Description: "Authentication and application API",
//	    EnableDocs: true,
//	    DocsPath:   "/docs",
//	    SpecPath:   "/openapi.json",
//	    Servers: []openapi.Server{
//	        {URL: "https://api.example.com", Description: "Production"},
//	    },
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
	// EnableDocs enables the Scalar documentation UI
	EnableDocs bool
	// DocsPath is the path for the documentation UI (e.g., "/docs").
	// The plugin no longer inserts its own name into the path.
	// If empty, defaults to "/docs".
	DocsPath string
	// SpecPath is the path for the OpenAPI spec JSON (e.g., "/openapi.json").
	// If empty, defaults to "/openapi.json".
	SpecPath string
}

// DefaultConfig returns default OpenAPI configuration.
//
// Default Settings:
//   - Title: "Aegis Authentication API"
//   - Version: "1.0.0"
//   - EnableDocs: true
//   - Server: http://localhost:8080 (development)
//   - License: MIT
//   - DocsPath: "/docs"
//   - SpecPath: "/openapi.json"
//
// Returns:
//   - *Config: Default configuration ready for customization
func DefaultConfig() *Config {
	return &Config{
		Title:       "Aegis Authentication API",
		Version:     "1.0.0",
		Description: "API documentation for Aegis authentication framework",
		EnableDocs:  true,
		DocsPath:    "/docs",
		SpecPath:    "/openapi.json",
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
//  4. Add common schemas (Error, Success)
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
//	    Title:      "My API",
//	    Version:    "2.0.0",
//	    EnableDocs: true,
//	    DocsPath:   "/docs",
//	    SpecPath:   "/openapi.json",
//	})
func New(cfg *Config) *Plugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Apply defaults for paths
	if cfg.DocsPath == "" {
		cfg.DocsPath = "/docs"
	}
	if cfg.SpecPath == "" {
		cfg.SpecPath = "/openapi.json"
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
	return "2.0.0"
}

// Description returns a human-readable description.
func (p *Plugin) Description() string {
	return "OpenAPI 3.0 documentation generation with Scalar UI"
}

// Init initializes the OpenAPI plugin and drains all pending route
// registrations from the global queue.
//
// By the time this is called, all other plugins and user code have
// already called Doc() to register their routes.
func (p *Plugin) Init(_ context.Context, _ plugins.Aegis) error {
	// Drain pending queue — this sets us as the active plugin
	// and processes all buffered Doc() calls
	drainPending(p)
	return nil
}

// GetMigrations returns the plugin migrations.
func (p *Plugin) GetMigrations() []plugins.Migration {
	// No migrations needed - stateless plugin
	return []plugins.Migration{}
}

// MountRoutes registers HTTP routes for the plugin.
func (p *Plugin) MountRoutes(r router.Router, prefix string) {
	handler := NewHandler(p)

	// Serve OpenAPI spec at the configured SpecPath
	specPath := prefix + p.config.SpecPath
	r.GET(specPath, handler.ServeSpec)

	// Serve Scalar UI at the configured DocsPath if enabled
	if p.config.EnableDocs {
		docsPath := prefix + p.config.DocsPath
		r.GET(docsPath, handler.ServeScalarUI)
	}
}

// register converts a Route into OpenAPI spec entries.
// This is called either immediately (if plugin is active) or
// during drainPending.
func (p *Plugin) register(r Route) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Normalize path parameters from :param to {param} style
	r.Path = normalizePathParams(r.Path)

	// Derive operationId if not set
	operationID := r.OperationID
	if operationID == "" {
		operationID = deriveOperationID(r.Method, r.Path)
	}

	// Build operation
	op := &Operation{
		OperationID: operationID,
		Summary:     r.Summary,
		Description: r.Description,
		Tags:        r.Tags,
		Deprecated:  r.Deprecated,
		Responses:   make(map[string]*Response),
	}

	// Security
	if r.Auth {
		op.Security = []SecurityRequirement{
			{"cookieAuth": []string{}},
			{"bearerAuth": []string{}},
		}
	}

	// Parameters
	for _, param := range r.Params {
		// Force required: true for path parameters (OpenAPI 3.0 spec requirement)
		required := param.Required
		if param.In == "path" {
			required = true
		}

		schema := &Schema{Type: param.Type, Format: param.Format}
		if len(param.Enum) > 0 {
			enumVals := make([]any, len(param.Enum))
			for i, v := range param.Enum {
				enumVals[i] = v
			}
			schema.Enum = enumVals
		}

		op.Parameters = append(op.Parameters, Parameter{
			Name:        param.Name,
			In:          param.In,
			Description: param.Description,
			Required:    required,
			Schema:      schema,
		})
	}

	// Request Body
	if r.Body != nil && r.Body.schema != nil {
		bodySchema := r.Body.schema
		// Register the schema in components and use a $ref
		if r.Body.name != "" {
			// If bodySchema is already a $ref, don't add it as a new component.
			// Otherwise we'll create self-referencing components like:
			//   components.schemas.RefreshTokenRequest.$ref -> RefreshTokenRequest
			// which breaks codegen.
			if bodySchema.Ref == "" {
				p.spec.AddSchema(r.Body.name, bodySchema)
			}
			bodySchema = RefSchema(r.Body.name)
		}
		op.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {
					Schema: bodySchema,
				},
			},
		}
	}

	// Responses
	for status, respDef := range r.Responses {
		statusStr := fmt.Sprintf("%d", status)
		resp := &Response{
			Description: respDef.Description,
		}

		if respDef.schema != nil {
			schema := respDef.schema
			// Register the schema in components and use a $ref if it has a name
			if respDef.name != "" && respDef.schema.Ref == "" {
				p.spec.AddSchema(respDef.name, schema)
				schema = RefSchema(respDef.name)
			}
			resp.Content = map[string]MediaType{
				"application/json": {
					Schema: schema,
				},
			}
		}

		op.Responses[statusStr] = resp
	}

	// Ensure at least a default response exists
	if len(op.Responses) == 0 {
		op.Responses["200"] = &Response{Description: "Successful operation"}
	}

	// Add path to spec
	p.addPathOperation(r.Path, r.Method, op)

	// Auto-add tags to spec if not present
	for _, tagName := range r.Tags {
		found := false
		for _, existing := range p.spec.Tags {
			if existing.Name == tagName {
				found = true
				break
			}
		}
		if !found {
			p.spec.AddTag(Tag{Name: tagName})
		}
	}
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

// GetSchemas returns all schemas for all supported dialects.
func (p *Plugin) GetSchemas() []plugins.Schema {
	// OpenAPI plugin doesn't have its own database schema
	return []plugins.Schema{}
}

// Dependencies returns plugin dependencies.
func (p *Plugin) Dependencies() []plugins.Dependency {
	return []plugins.Dependency{}
}

// GetSpec returns a copy of the current OpenAPI spec.
func (p *Plugin) GetSpec() *Spec {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.spec
}

// RegisterSchemaFromType automatically generates and registers an OpenAPI schema
// from a Go type. This can be used by plugins or user code to register schemas
// that are referenced in routes.
func (p *Plugin) RegisterSchemaFromType(name string, example any) {
	p.mu.Lock()
	defer p.mu.Unlock()

	schema := GenerateSchema(example)
	p.spec.AddSchema(name, schema)
}

// Ensure Plugin implements plugins.Plugin
var _ plugins.Plugin = (*Plugin)(nil)
