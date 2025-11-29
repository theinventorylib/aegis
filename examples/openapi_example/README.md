# OpenAPI Documentation Example

This example demonstrates how to use the OpenAPI plugin to generate interactive API documentation with custom endpoints.

## Overview

The openapi plugin provides:
- Automatic documentation for core Aegis routes
- Extensible API for adding custom endpoints
- Interactive Scalar UI for testing APIs
- OpenAPI 3.0 compliant JSON specification

This example shows:
- Registering the OpenAPI plugin with custom configuration
- Adding custom endpoints to the documentation
- Defining custom schemas
- Using tags to organize endpoints
- Accessing the Scalar UI

## Running the Example

1. **Start the database**:
```bash
# Make sure PostgreSQL is running with the aegis_dev database
createdb aegis_dev
```

2. **Run migrations**:
```bash
# Apply Aegis core migrations
```

3. **Start the server**:
```bash
go run main.go
```

The server will start on `http://localhost:8080`.

## Accessing Documentation

### Scalar UI (Interactive Documentation)
Open your browser and navigate to:
```
http://localhost:8080/auth/docs
```

The Scalar UI provides:
- Interactive API testing
- Code generation in multiple languages
- Search functionality
- Dark mode support
- Clean, modern interface

### OpenAPI JSON Specification
Download the raw OpenAPI 3.0 spec:
```
http://localhost:8080/auth/openapi.json
```

You can import this into tools like:
- Postman
- Insomnia
- Swagger Editor
- API testing frameworks

## Testing the API

### Public Endpoint

Test the public endpoint (no authentication required):

```bash
curl http://localhost:8080/api/public/info
```

Response:
```json
{
  "message": "Public endpoint",
  "version": "1.0.0"
}
```

### Protected Endpoint

Test the protected endpoint (requires authentication):

```bash
# With session cookie
curl http://localhost:8080/api/custom/data \
  -H "Cookie: aegis_session=<session_token>"

# With Bearer token
curl http://localhost:8080/api/custom/data \
  -H "Authorization: Bearer <access_token>"
```

Response:
```json
{
  "message": "Custom protected endpoint",
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "data": {
    "id": "123",
    "name": "Example",
    "count": 42
  }
}
```

## Code Highlights

### Registering the Plugin

```go
openapiPlugin := openapi.New(&openapi.Config{
    Title:       "Aegis Auth API Example",
    Version:     "1.0.0",
    Description: "Example API demonstrating OpenAPI plugin",
    Servers: []openapi.Server{
        {
            URL:         "http://localhost:8080",
            Description: "Development server",
        },
    },
    EnableScalarUI: true,
    BasePath:       "/auth",
})
auth.Use(context.Background(), openapiPlugin)
```

### Adding Custom Schemas

```go
oapi.RegisterSchema("CustomResponse", openapi.ObjectSchema(
    "Custom response object",
    map[string]*openapi.Schema{
        "message": openapi.StringSchema("Response message"),
        "data": openapi.ObjectSchema("", map[string]*openapi.Schema{
            "id":    openapi.UUIDSchema("Item ID"),
            "name":  openapi.StringSchema("Item name"),
            "count": openapi.IntegerSchema("Item count"),
        }, []string{"id", "name"}),
    },
    []string{"message"},
))
```

### Documenting Endpoints

```go
oapi.RegisterEndpoint("GET", "/api/custom/data", &openapi.Operation{
    Tags:        []string{"Custom"},
    Summary:     "Get custom data",
    Description: "Returns custom data for authenticated user",
    OperationID: "getCustomData",
    Security: []openapi.SecurityRequirement{
        {"cookieAuth": []string{}},
        {"bearerAuth": []string{}},
    },
    Responses: map[string]*openapi.Response{
        "200": {
            Description: "Custom data",
            Content: map[string]openapi.MediaType{
                "application/json": {
                    Schema: openapi.RefSchema("CustomResponse"),
                },
            },
        },
    },
})
```

### Adding Tags

```go
oapi.RegisterTag(openapi.Tag{
    Name:        "Custom",
    Description: "Custom endpoints for demonstration",
})
```

## What's Documented

This example automatically documents:

### Core Aegis Routes
- `POST /auth/refresh` - Refresh session
- `POST /auth/logout` - Logout
- JWT plugin routes (if registered)

### Custom Routes
- `GET /api/public/info` - Public API information
- `GET /api/custom/data` - Protected custom data

### Security Schemes
- **cookieAuth**: Session cookie authentication
- **bearerAuth**: Bearer token authentication

### Schemas
- `Error` - Error response format
- `Success` - Success response format
- `User` - User object
- `Session` - Session object
- `CustomResponse` - Custom response format

## Extending in Your Application

To add documentation for your own endpoints:

1. **Get the OpenAPI plugin instance**:
```go
openapiPlugin, ok := auth.GetPlugin("openapi")
oapi := openapiPlugin.(*openapi.Plugin)
```

2. **Register your schemas**:
```go
oapi.RegisterSchema("MySchema", openapi.ObjectSchema(...))
```

3. **Document your endpoints**:
```go
oapi.RegisterEndpoint("POST", "/my/endpoint", &openapi.Operation{...})
```

4. **Add tags for organization**:
```go
oapi.RegisterTag(openapi.Tag{Name: "MyTag", Description: "..."})
```

## Plugin Integration

Plugins can automatically extend documentation during initialization:

```go
func (p *MyPlugin) Init(ctx context.Context, aegis plugins.Aegis) error {
    if openapiPlugin, ok := aegis.GetPlugin("openapi"); ok {
        oapi := openapiPlugin.(*openapi.Plugin)
        
        // Add plugin's routes to documentation
        oapi.RegisterEndpoint("POST", "/auth/myplugin/action", &openapi.Operation{
            // ... operation definition
        })
    }
    return nil
}
```

## Best Practices

1. **Document all public endpoints**: Users should be able to discover all available APIs
2. **Use descriptive summaries**: Help users understand what each endpoint does
3. **Include examples**: Add example values to schemas
4. **Group with tags**: Organize related endpoints together
5. **Specify security**: Clearly indicate which endpoints require authentication
6. **Add descriptions**: Provide context for parameters and responses

## Troubleshooting

### Scalar UI not loading
- Check browser console for errors
- Verify CDN access (Scalar loads from jsdelivr CDN)
- Ensure `EnableScalarUI` is `true`

### Custom endpoints not appearing
- Verify `RegisterEndpoint` is called before accessing `/openapi.json`
- Check path format matches your routing
- Ensure method is uppercase (GET, POST, etc.)

## Next Steps

- Add more custom endpoints
- Integrate with other Aegis plugins (password, email, OAuth)
- Export OpenAPI spec for client code generation
- Use Scalar UI to test your API interactively
- Import spec into Postman or other API tools
