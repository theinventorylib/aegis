# OpenAPI Plugin

The OpenAPI plugin provides automatic OpenAPI 3.0 documentation generation for your Aegis authentication API with an integrated Scalar UI for interactive documentation.

## Overview

The openapi plugin:
- Generates OpenAPI 3.0 compliant specifications
- Documents core Aegis authentication routes automatically
- Provides an extensible API for plugins and user code to add custom endpoints
- Serves interactive API documentation via Scalar UI
- Supports custom schemas, security schemes, and tags

## Installation

```bash
go get github.com/theinventorylib/aegis/plugins/openapi
```

## Usage

### Basic Setup

```go
import (
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/plugins/openapi"
)

// Create Aegis instance
auth, err := aegis.New(
    config.WithDB(dbProvider),
    config.WithRouter(router),
)
if err != nil {
    log.Fatal(err)
}

// Register OpenAPI plugin with default configuration
openapiPlugin := openapi.New(nil) // nil uses DefaultConfig
auth.Use(context.Background(), openapiPlugin)

// Mount routes
auth.MountRoutes("/auth")

// Access documentation at:
// - http://localhost:8080/auth/docs (Scalar UI)
// - http://localhost:8080/auth/openapi.json (OpenAPI spec)
```

### Custom Configuration

```go
openapiPlugin := openapi.New(&openapi.Config{
    Title:       "My Auth API",
    Version:     "2.0.0",
    Description: "Authentication API for My Application",
    Servers: []openapi.Server{
        {
            URL:         "https://api.example.com",
            Description: "Production server",
        },
        {
            URL:         "https://staging.example.com",
            Description: "Staging server",
        },
    },
    Contact: &openapi.Contact{
        Name:  "API Support",
        Email: "support@example.com",
        URL:   "https://example.com/support",
    },
    License: &openapi.License{
        Name: "Apache 2.0",
        URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
    },
    EnableScalarUI: true,
    BasePath:       "/auth",
})
```

## Extending Documentation

### Adding Custom Endpoints

Plugins and user code can extend the OpenAPI documentation by registering custom endpoints:

```go
// Get the openapi plugin instance
openapiPlugin, ok := auth.GetPlugin("openapi")
if !ok {
    log.Fatal("openapi plugin not found")
}

// Type assert to OpenAPI plugin
oapi := openapiPlugin.(*openapi.Plugin)

// Register a custom endpoint
oapi.RegisterEndpoint("POST", "/auth/custom/endpoint", &openapi.Operation{
    Tags:        []string{"Custom"},
    Summary:     "Custom endpoint",
    Description: "A custom endpoint added by user code",
    OperationID: "customEndpoint",
    RequestBody: &openapi.RequestBody{
        Required:    true,
        Description: "Request payload",
        Content: map[string]openapi.MediaType{
            "application/json": {
                Schema: openapi.ObjectSchema("", map[string]*openapi.Schema{
                    "field1": openapi.StringSchema("First field"),
                    "field2": openapi.IntegerSchema("Second field"),
                }, []string{"field1"}),
            },
        },
    },
    Responses: map[string]*openapi.Response{
        "200": {
            Description: "Success",
            Content: map[string]openapi.MediaType{
                "application/json": {
                    Schema: openapi.RefSchema("Success"),
                },
            },
        },
    },
    Security: []openapi.SecurityRequirement{
        {"bearerAuth": []string{}},
    },
})
```

### Adding Custom Schemas

```go
// Register a reusable schema
oapi.RegisterSchema("CustomUser", openapi.ObjectSchema(
    "Extended user object",
    map[string]*openapi.Schema{
        "id":       openapi.UUIDSchema("User ID"),
        "email":    openapi.EmailSchema("User email"),
        "name":     openapi.StringSchema("User name"),
        "role":     openapi.StringSchema("User role"),
        "verified": openapi.BooleanSchema("Email verified"),
    },
    []string{"id", "email", "name"},
))

// Use the schema in an endpoint
oapi.RegisterEndpoint("GET", "/auth/users/me", &openapi.Operation{
    Tags:        []string{"Users"},
    Summary:     "Get current user",
    Description: "Returns the authenticated user's profile",
    OperationID: "getCurrentUser",
    Responses: map[string]*openapi.Response{
        "200": {
            Description: "User profile",
            Content: map[string]openapi.MediaType{
                "application/json": {
                    Schema: openapi.RefSchema("CustomUser"),
                },
            },
        },
    },
    Security: []openapi.SecurityRequirement{
        {"bearerAuth": []string{}},
    },
})
```

### Adding Custom Security Schemes

```go
// Register OAuth2 security scheme
oapi.RegisterSecurityScheme("oauth2", &openapi.SecurityScheme{
    Type:        "oauth2",
    Description: "OAuth2 authentication",
    Flows: &openapi.OAuthFlows{
        AuthorizationCode: &openapi.OAuthFlow{
            AuthorizationURL: "https://example.com/oauth/authorize",
            TokenURL:         "https://example.com/oauth/token",
            Scopes: map[string]string{
                "read":  "Read access",
                "write": "Write access",
            },
        },
    },
})
```

### Adding Tags

```go
// Add a custom tag for grouping operations
oapi.RegisterTag(openapi.Tag{
    Name:        "Users",
    Description: "User management endpoints",
})
```

## Plugin Integration

Plugins can automatically extend the OpenAPI documentation during initialization:

```go
// In your plugin's Init method
func (p *MyPlugin) Init(ctx context.Context, aegis plugins.Aegis) error {
    // Get OpenAPI plugin if registered
    if openapiPlugin, ok := aegis.GetPlugin("openapi"); ok {
        oapi := openapiPlugin.(*openapi.Plugin)
        
        // Add plugin's routes to documentation
        oapi.RegisterTag(openapi.Tag{
            Name:        "MyPlugin",
            Description: "My plugin endpoints",
        })
        
        oapi.RegisterEndpoint("POST", "/auth/myplugin/action", &openapi.Operation{
            Tags:        []string{"MyPlugin"},
            Summary:     "Plugin action",
            Description: "Performs a plugin-specific action",
            // ... rest of operation definition
        })
    }
    
    return nil
}
```

## Default Documentation

The OpenAPI plugin automatically documents the following core Aegis routes:

### Session Management
- `POST /auth/refresh` - Refresh session
- `POST /auth/logout` - Logout

### Security Schemes

Two security schemes are pre-configured:

1. **cookieAuth**: Session cookie authentication
   - Type: API Key
   - Location: Cookie
   - Name: `aegis_session`

2. **bearerAuth**: Bearer token authentication
   - Type: HTTP
   - Scheme: Bearer
   - Format: JWT

### Common Schemas

Pre-defined reusable schemas:
- `Error` - Error response format
- `Success` - Success response format
- `User` - User object
- `Session` - Session object

## Scalar UI Features

The integrated Scalar UI provides:
- **Interactive API testing**: Test endpoints directly from the browser
- **Modern design**: Clean, responsive interface
- **Code generation**: Generate client code in multiple languages
- **Search**: Quickly find endpoints and schemas
- **Dark mode**: Automatic dark mode support

Access the Scalar UI at `/auth/docs` (or your configured base path + `/docs`).

## Configuration Reference

### Config Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Title` | string | "Aegis Authentication API" | API title |
| `Version` | string | "1.0.0" | API version |
| `Description` | string | "API documentation for Aegis..." | API description |
| `Servers` | []Server | localhost:8080 | Server URLs |
| `Contact` | *Contact | nil | Contact information |
| `License` | *License | MIT | License information |
| `EnableScalarUI` | bool | true | Enable Scalar UI |
| `BasePath` | string | "/auth" | Base path for routes |

## API Reference

### Plugin Methods

#### RegisterEndpoint
```go
func (p *Plugin) RegisterEndpoint(method, path string, operation *Operation)
```
Adds a custom endpoint to the OpenAPI spec.

#### RegisterSchema
```go
func (p *Plugin) RegisterSchema(name string, schema *Schema)
```
Adds a reusable schema component.

#### RegisterSecurityScheme
```go
func (p *Plugin) RegisterSecurityScheme(name string, scheme *SecurityScheme)
```
Adds a custom security scheme.

#### RegisterTag
```go
func (p *Plugin) RegisterTag(tag Tag)
```
Adds a tag for grouping operations.

#### GetSpec
```go
func (p *Plugin) GetSpec() *Spec
```
Returns a copy of the current OpenAPI spec.

### Helper Functions

The package provides helper functions for creating schemas:

- `StringSchema(description string) *Schema`
- `IntegerSchema(description string) *Schema`
- `BooleanSchema(description string) *Schema`
- `ObjectSchema(description string, properties map[string]*Schema, required []string) *Schema`
- `ArraySchema(description string, items *Schema) *Schema`
- `RefSchema(ref string) *Schema`
- `DateTimeSchema(description string) *Schema`
- `EmailSchema(description string) *Schema`
- `UUIDSchema(description string) *Schema`

## Best Practices

1. **Document all public endpoints**: Use `RegisterEndpoint` for all user-facing routes
2. **Use schema references**: Define reusable schemas with `RegisterSchema` and reference them with `RefSchema`
3. **Add descriptions**: Provide clear descriptions for operations, parameters, and schemas
4. **Tag operations**: Group related operations with tags for better organization
5. **Include examples**: Add example values to schemas for better documentation
6. **Security requirements**: Specify security requirements for protected endpoints

## Troubleshooting

### Scalar UI not loading

**Problem**: `/docs` endpoint returns 404 or blank page.

**Solutions**:
1. Verify `EnableScalarUI` is set to `true` in config
2. Check that routes are mounted correctly
3. Ensure CDN access (Scalar loads from CDN)

### Custom endpoints not appearing

**Problem**: Endpoints registered via `RegisterEndpoint` don't appear in docs.

**Solutions**:
1. Ensure `RegisterEndpoint` is called before accessing `/openapi.json`
2. Verify the path format matches your base path
3. Check for typos in method names (must be uppercase: GET, POST, etc.)

### CORS errors

**Problem**: Browser console shows CORS errors when accessing OpenAPI spec.

**Solutions**:
The `/openapi.json` endpoint includes `Access-Control-Allow-Origin: *` header. If still experiencing issues, check your reverse proxy or server configuration.

## Examples

See the [openapi example](../../examples/openapi_example) for a complete working example with custom endpoint registration.

## Related Plugins

- [Bearer Plugin](./bearer.md) - Bearer token authentication
- [JWT Plugin](./jwt.md) - JWT token generation
- [Admin Plugin](./admin.md) - Administrative endpoints
