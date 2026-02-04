# OpenAPI Plugin

The OpenAPI plugin (formerly Swagger) generates interactive API documentation for your Aegis endpoints.

## Installation

```go
import "github.com/theinventorylib/aegis/plugins/openapi"
```

## Usage

```go
// Mounts Scalar UI at /docs
openapiPlugin := openapi.New(openapi.Config{
    Title: "My App API",
    Path:  "/docs",
})
aegis.Use(ctx, openapiPlugin)
```

This plugin automatically discovers routes from core and other registered plugins.
