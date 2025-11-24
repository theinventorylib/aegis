# Plugins

Aegis uses a modular plugin system to provide authentication features. This allows you to only include the code and dependencies you actually need.

## Available Plugins

| Plugin | Description |
|--------|-------------|
| [**Email**](./plugins/email.md) | Email verification via OTP or Magic Links. |
| [**SMS**](./plugins/sms.md) | Phone number verification via OTP. |
| [**OAuth**](./plugins/oauth.md) | Social login (Google, GitHub, Apple, etc.). |
| [**Password**](./plugins/password.md) | Secure password authentication with Argon2id. |
| [**JWT**](./plugins/jwt.md) | JWT token generation, validation, and rotation. |
| [**Admin**](./plugins/admin.md) | Administrative endpoints for user/org management. |
| [**Organizations**](./plugins/organizations.md) | Multi-tenant organization and team support. |

## Using Plugins

To use a plugin, you typically:

1. **Import** the plugin package.
2. **Initialize** the plugin with its configuration.
3. **Register** the plugin with your Aegis instance using `auth.Use()`.

```go
import (
    "github.com/theinventorylib/aegis/plugins/email"
)

// ...

emailPlugin := email.New(&email.Config{
    // ... config options
})

auth.Use(emailPlugin)
```

## Creating Custom Plugins

You can create your own plugins by implementing the `plugins.Plugin` interface.

```go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    
    Init(ctx context.Context, a Aegis) error
    GetMigrations() []Migration
    MountRoutes(router server.Router, prefix string)
    
    Dependencies() []Dependency
    RequiresTables() []string
    ProvidesAuthMethods() []string
}
```

See the [Custom Plugins](../examples/custom-plugin) example for more details.
