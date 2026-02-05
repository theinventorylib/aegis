---
title: OpenAPI Plugin
---
## OpenAPI Plugin
Generates interactive API documentation (Scalar UI).
```go
import "github.com/theinventorylib/aegis/plugins/openapi"
openapiPlugin := openapi.New(openapi.Config{Title: "My API", Path: "/docs"})
aegis.Use(ctx, openapiPlugin)
```
