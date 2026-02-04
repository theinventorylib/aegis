# Aegis Plugins

Aegis uses a modular plugin architecture. You can opt-in to the features you need by registering the corresponding plugins.

## Available Plugins

- [**Admin**](./admin) - User management and administrative endpoints.
- [**Bearer**](./bearer) - Authenticate requests using Bearer tokens.
- [**Email**](./email) - Email verification via OTP or simple link mechanisms.
- [**JWT**](./jwt) - JSON Web Token generation, validation, and rotation.
- [**OAuth**](./oauth) - Social login support (Google, GitHub, generic providers).
- [**OpenAPI**](./openapi) - Serve interactive API documentation (Swagger/Scalar).
- [**Organizations**](./organizations) - Multi-tenancy, teams, and member roles.
- [**SMS**](./sms) - Phone number verification via SMS OTP.

## Using Plugins

Plugins are registered during Aegis initialization:

```go
import (
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/plugins/oauth"
)

// ... setup ...

// Initialize plugin
oauthPlugin := oauth.New(oauthConfig, nil, plugins.DialectPostgres)

// Register with Aegis
a.Use(ctx, oauthPlugin)
```
