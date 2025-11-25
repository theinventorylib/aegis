# Aegis API Reference

Complete API documentation for the Aegis authentication framework.

## Core API

### Creating an Aegis Instance

#### `New(opts ...config.Option) (*Aegis, error)`

Creates and initializes a new Aegis authentication framework instance.

**Parameters**:
- `opts`: Variable number of configuration options

**Returns**:
- `*Aegis`: Configured Aegis instance
- `error`: Configuration or initialization error

**Example**:
```go
auth, err := aegis.New(
    config.WithDB(sqlDB, db.PostgreSQL),
    config.WithRouter(router),
    config.WithCSRFSecret([]byte("your-secret")),
)
```

**See Also**: [Configuration Options](#configuration-options)

---

## Plugin Management

### `Use(ctx context.Context, plugin plugins.Plugin) error`

Registers a plugin with default priority (100). This is the canonical method for plugin registration.

**Parameters**:
- `ctx`: Context for initialization (supports cancellation/timeout)
- `plugin`: Plugin instance to register

**Returns**:
- `error`: Initialization error if plugin.Init() fails

**Example**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := aegis.Use(ctx, emailPlugin)
if err != nil {
    log.Fatal(err)
}
```

**Thread-Safety**: ✅ Safe to call concurrently

---

### `UseWithPriority(ctx context.Context, plugin plugins.Plugin, priority int) error`

Registers a plugin with explicit priority for deterministic initialization order.

**Parameters**:
- `ctx`: Context for initialization
- `plugin`: Plugin instance to register
- `priority`: Priority value (lower = initialized first)

**Returns**:
- `error`: Initialization error

**Priority Guidelines**:
- 0-50: Critical infrastructure
- 51-99: High-priority auth plugins
- 100: Default (Use method)
- 101-150: Standard plugins
- 151+: Low-priority plugins

**Example**:
```go
// Password plugin first (high priority)
aegis.UseWithPriority(ctx, passwordPlugin, 60)

// Email plugin later (default priority)
aegis.Use(ctx, emailPlugin) // priority: 100
```

**Thread-Safety**: ✅ Safe to call concurrently

---

### `RegisterPlugin(plugin plugins.Plugin) error`

**Deprecated**: Use `Use(context.Background(), plugin)` instead.

Registers a plugin using context.Background(). Provided for backward compatibility.

---

### `GetPlugin(name string) (plugins.Plugin, bool)`

Retrieves a registered plugin by name.

**Parameters**:
- `name`: Plugin name (from `plugin.Name()`)

**Returns**:
- `plugins.Plugin`: The plugin instance
- `bool`: `true` if found, `false` if not found

**Example**:
```go
plugin, ok := aegis.GetPlugin("email")
if !ok {
    log.Fatal("Email plugin not registered")
}

emailPlugin := plugin.(*email.Plugin)
```

**Thread-Safety**: ✅ Safe to call concurrently

---

### `GetPlugins() []plugins.Plugin`

Returns all registered plugins in priority order (lowest priority first).

**Returns**:
- `[]plugins.Plugin`: Copy of plugins slice sorted by priority

**Example**:
```go
plugins := aegis.GetPlugins()
for _, p := range plugins {
    log.Printf("Plugin: %s (v%s)", p.Name(), p.Version())
}
```

**Thread-Safety**: ✅ Safe to call concurrently

---

## Route Management

### `MountRoutes(prefix string)`

Mounts all authentication routes (core + plugins) to the router.

**Parameters**:
- `prefix`: URL prefix for all auth routes (e.g., `"/auth"`)

**Example**:
```go
// Mounts:
//   /auth/logout
//   /auth/user
//   /auth/session/validate
//   etc.
aegis.MountRoutes("/auth")
```

**Mounted Routes** (Core):
- `POST /logout` - Logout current user
- `GET /user` - Get current user
- `GET /session/validate` - Validate current session
- `GET /sessions` - List user's sessions
- `DELETE /sessions/:id` - Revoke specific session
- `DELETE /sessions` - Revoke all sessions
- `POST /session/refresh` - Refresh session with refresh token

**Plugin Routes**: Plugins mount their own routes in priority order.

---

## Middleware

### `AuthMiddleware() func(http.Handler) http.Handler`

Returns middleware that validates sessions and injects user into request context.

**Returns**:
- Middleware function

**Usage**:
```go
// Apply to specific routes
r.Group(func(r chi.Router) {
    r.Use(aegis.AuthMiddleware())
    r.Get("/api/data", handler) // User available in context
})
```

**Behavior**:
- Validates session token from cookie or Authorization header
- Injects authenticated user into `context`
- Does NOT reject unauthenticated requests (use `RequireAuth` for that)

---

### `RequireAuth() func(http.Handler) http.Handler`

Returns middleware that requires authentication (rejects unauthenticated requests).

**Returns**:
- Middleware function

**Usage**:
```go
// Apply to protected routes
protectedRouter.Use(aegis.RequireAuth())
```

**Behavior**:
- Returns 401 Unauthorized if no valid session
- Continues to next handler if authenticated

---

## Context Helpers

### `GetUser(ctx context.Context) (*models.User, error)`

Extracts authenticated user from request context.

**Parameters**:
- `ctx`: Request context (from `r.Context()`)

**Returns**:
- `*models.User`: Authenticated user
- `error`: Error if not authenticated

**Example**:
```go
func handler(w http.ResponseWriter, r *http.Request) {
    user, err := aegis.GetUser(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    fmt.Fprintf(w, "Hello, %s!", user.ID)
}
```

---

### `Authenticated(ctx context.Context) bool`

Checks if request context has an authenticated user.

**Parameters**:
- `ctx`: Request context

**Returns**:
- `bool`: `true` if authenticated, `false` otherwise

**Example**:
```go
if !aegis.Authenticated(r.Context()) {
    http.Redirect(w, r, "/login", http.StatusFound)
    return
}
```

---

## Accessors

### `GetDB() db.Provider`

Returns the database provider instance.

**Returns**:
- `db.Provider`: Database provider

---

### `GetRouter() server.Router`

Returns the router instance.

**Returns**:
- `server.Router`: Router instance

---

### `GetConfig() *config.Config`

Returns the configuration instance.

**Returns**:
- `*config.Config`: Configuration

---

### `GetSessionService() *core.SessionService`

Returns the session service instance.

**Returns**:
- `*core.SessionService`: Session service

---

## Configuration Options

### Database

#### `WithDB(sqlDB interface{}, dialect db.Dialect) Option`

Sets database provider from `*sql.DB` connection.

**Parameters**:
- `sqlDB`: `*sql.DB` instance or `db.Provider`
- `dialect`: SQL dialect (`db.PostgreSQL`, `db.MySQL`, `db.SQLite`)

**Example**:
```go
sqlDB, _ := sql.Open("postgres", connString)
config.WithDB(sqlDB, db.PostgreSQL)
```

---

#### `WithPostgres(connString string) Option`

Convenience helper for PostgreSQL with lib/pq driver.

**Parameters**:
- `connString`: PostgreSQL connection string

**Example**:
```go
config.WithPostgres("postgres://user:pass@localhost/db?sslmode=require")
```

---

#### `WithMySQL(connString string) Option`

Convenience helper for MySQL with go-sql-driver/mysql.

**Parameters**:
- `connString`: MySQL connection string

**Example**:
```go
config.WithMySQL("user:pass@tcp(localhost:3306)/db?parseTime=true")
```

---

### Router

#### `WithRouter(router server.Router) Option`

Sets the HTTP router.

**Parameters**:
- `router`: Router instance (`*server.ChiRouter` or `*server.DefaultRouter`)

**Example**:
```go
config.WithRouter(server.NewChiRouter(chi.NewRouter()))
```

---

### Security

#### `WithCSRFSecret(secret []byte) Option`

Sets CSRF protection secret (required for web apps).

**Parameters**:
- `secret`: Random bytes (32+ bytes recommended)

**Example**:
```go
config.WithCSRFSecret([]byte(os.Getenv("CSRF_SECRET")))
```

---

#### `WithSessionExpiry(duration time.Duration) Option`

Sets session token expiry duration.

**Parameters**:
- `duration`: Expiry duration

**Default**: 24 hours

**Example**:
```go
config.WithSessionExpiry(1 * time.Hour)
```

---

#### `WithRefreshExpiry(duration time.Duration) Option`

Sets refresh token expiry duration.

**Parameters**:
- `duration`: Expiry duration (must be > session expiry)

**Default**: 7 days

**Example**:
```go
config.WithRefreshExpiry(30 * 24 * time.Hour) // 30 days
```

---

### Cookies

#### `WithCookieDomain(domain string) Option`

Sets cookie domain for subdomain sharing.

**Parameters**:
- `domain`: Domain string (e.g., ".example.com")

**Default**: "" (current domain only)

---

#### `WithCookieSecure(secure bool) Option`

Sets whether cookies require HTTPS.

**Parameters**:
- `secure`: `true` for HTTPS only, `false` for HTTP (dev only)

**Default**: `true` (**must be true in production**)

---

#### `WithCookieSameSite(sameSite string) Option`

Sets SameSite cookie attribute.

**Parameters**:
- `sameSite`: `"Strict"`, `"Lax"`, or `"None"`

**Default**: `"Lax"`

---

### Observability

#### `WithLogger(logger Logger) Option`

Sets optional logger for lifecycle events.

**Parameters**:
- `logger`: Logger instance (implements `config.Logger` interface)

**Example**:
```go
import "log/slog"
config.WithLogger(slog.Default())
```

---

### Misc

#### `WithAPIOnlyMode(enabled bool) Option`

Enables API-only mode (skips CSRF requirement).

**Parameters**:
- `enabled`: `true` for API mode, `false` for web mode

**Default**: `false`

**Example**:
```go
config.WithAPIOnlyMode(true) // For REST APIs
```

---

#### `WithRedis(host string, port int, password string, db int) Option`

Enables Redis session storage.

**Parameters**:
- `host`: Redis server hostname
- `port`: Redis server port
- `password`: Redis password
- `db`: Redis database number (0-15)

**Example**:
```go
config.WithRedis("localhost", 6379, os.Getenv("REDIS_PASSWORD"), 0)
```

---

#### `WithIDGenerator(generator func() string) Option`

Sets custom ID generation function.

**Parameters**:
- `generator`: Function that returns string IDs

**Example**:
```go
import "github.com/oklog/ulid/v2"
config.WithIDGenerator(func() string {
    return ulid.Make().String()
})
```

---

## Plugin Interface

### Required Methods

```go
type Plugin interface {
    // Identity
    Name() string
    Version() string
    Description() string
    
    // Lifecycle
    Init(ctx context.Context, a Aegis) error
    GetMigrations() []Migration
    
    // Routing
    MountRoutes(router server.Router, prefix string)
    
    // Metadata
    Dependencies() []Dependency
    RequiresTables() []string
    ProvidesAuthMethods() []string
}
```

**See**: [Plugin Development Guide](./plugin-development.md)

---

## Models

### User

```go
type User struct {
    ID        string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## Thread-Safety

All public Aegis methods are thread-safe:
- ✅ `Use` / `UseWithPriority`
- ✅ `GetPlugin` / `GetPlugins`  
- ✅ `AuthMiddleware` / `RequireAuth`
- ✅ `GetUser` / `Authenticated`

**See**: [Concurrency Best Practices](./concurrency-best-practices.md)

---

## Related Documentation

- [SECURITY.md](../SECURITY.md) - Production security recommendations
- [Plugin Priorities](./plugin-priorities.md) - Plugin ordering
- [Concurrency Best Practices](./concurrency-best-practices.md) - Thread-safety guide
- [Configuration Guide](./configuration.md) - Configuration details
