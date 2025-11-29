# Plugins

Aegis uses a modular plugin system to provide authentication features. This allows you to only include the code and dependencies you actually need.

## Available Plugins

> [!NOTE]
> Password authentication is **core functionality**, not a plugin. See [Password Authentication](../password.md) for details.

| Plugin | Description |
|--------|-------------|
| [**Email**](./plugins/email.md) | Email verification via OTP or Magic Links. |
| [**SMS**](./plugins/sms.md) | Phone number verification via OTP. |
| [**OAuth**](./plugins/oauth.md) | Social login (Google, GitHub, Apple, etc.). |
| [**JWT**](./plugins/jwt.md) | JWT token generation, validation, and rotation. |
| [**Bearer**](./plugins/bearer.md) | Bearer token authentication support (marker plugin). |
| [**OpenAPI**](./plugins/openapi.md) | OpenAPI 3.0 documentation with Scalar UI. |
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

---

## Plugin Interface Abstraction

### Using the `plugins.Aegis` Interface

During plugin initialization (`Init` method), plugins receive an `Aegis` instance. However, this is passed as a **minimal interface**, not the full implementation:

```go
// plugins/provider.go
type Aegis interface {
    GetSessionService() *core.SessionService
}
```

**Why?** To prevent plugins from accessing internal Aegis state and maintain clean separation of concerns.

---

### ✅ Correct Usage

Plugins should **only use methods defined in the `plugins.Aegis` interface**:

```go
func (p *MyPlugin) Init(ctx context.Context, a plugins.Aegis) error {
    // CORRECT: Using the interface method
    sessionService := a.GetSessionService()
    
    // Use the session service as needed
    p.sessionService = sessionService
    
    return nil
}
```

---

### ❌ Incorrect Usage

**DO NOT** type-assert to access internal methods:

```go
func (p *MyPlugin) Init(ctx context.Context, a plugins.Aegis) error {
    // WRONG: Type assertion breaks abstraction
    aegisImpl := a.(*aegis.Aegis)
    
    // WRONG: Accessing internal methods not in interface
    db := aegisImpl.GetDB()
    router := aegisImpl.GetRouter()
    
    // This creates tight coupling and breaks encapsulation
    return nil
}
```

**Why is this wrong?**
- Breaks the abstraction barrier
- Creates tight coupling between plugins and core
- Makes plugins fragile to internal changes
- Violates interface-based design principles

---

### Accessing Database and Router

If your plugin needs database or router access, they should be **passed during plugin construction**, not accessed from the Aegis instance:

```go
// CORRECT: Pass dependencies during construction
type MyPlugin struct {
    db     db.Provider
    router server.Router
}

func New(database db.Provider) *MyPlugin {
    return &MyPlugin{
        db: database,
    }
}

func (p *MyPlugin) Init(ctx context.Context, a plugins.Aegis) error {
    // Plugin already has what it needs from constructor
    // No need to extract from Aegis instance
    return nil
}

func (p *MyPlugin) MountRoutes(router server.Router, prefix string) {
    // Router is provided to MountRoutes
    router.Post(prefix+"/my-endpoint", p.handleRequest)
}
```

---

### Plugin Discovery After Registration

If a plugin needs to interact with another plugin, use the plugin registry **after registration**:

```go
// In your application code (not in plugin Init):
auth, _ := aegis.New(...)

// Register plugins
auth.Use(ctx, emailPlugin)
auth.Use(ctx, adminPlugin)

// Now plugins can discover each other via auth.GetPlugin()
if ap, ok := auth.GetPlugin("admin"); ok {
    adminPlugin := ap.(*admin.Plugin)
    // Use the admin plugin
}
```

**During `Init`**, plugins can discover previously registered plugins:

```go
func (p *MyPlugin) Init(ctx context.Context, a plugins.Aegis) error {
    // Type assert carefully only if you NEED another plugin
    if aegisImpl, ok := a.(interface{ GetPlugin(string) (plugins.Plugin, bool) }); ok {
        if emailPluginInterface, found := aegisImpl.GetPlugin("email"); found {
            if ep, ok := emailPluginInterface.(*email.Plugin); ok {
                p.emailPlugin = ep
                // Now you can use email plugin functionality
            }
        }
    }
    
    return nil
}
```

> **Note**: This pattern is acceptable when plugins genuinely depend on each other, but should be used sparingly. Document such dependencies in `RequiresTables()` or `Dependencies()`.

---

### Best Practices

1. **Respect the interface** - Only use methods defined in `plugins.Aegis`
2. **Pass dependencies explicitly** - Don't extract them from Aegis instance
3. **Document dependencies** - Use `Dependencies()` and `RequiresTables()` methods
4. **Avoid type assertions when possible** - Keep plugins loosely coupled
5. **Design for testability** - Interfaces make mocking easier

---

### Summary

| Pattern | Status | Reason |
|---------|--------|--------|
| Using `a.GetSessionService()` | ✅ Correct | Part of interface |
| Type asserting to `*aegis.Aegis` | ❌ Wrong | Breaks abstraction |
| Passing DB in constructor | ✅ Correct | Explicit dependency |
| Extracting DB from Aegis | ❌ Wrong | Tight coupling |
| Discovering plugins via `GetPlugin()` | ⚠️ Use sparingly | Okay for genuine dependencies |
