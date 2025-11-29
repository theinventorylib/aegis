# Aegis Plugin Priority and Ordering

## Overview

Aegis supports plugin registration with configurable priorities to ensure deterministic initialization and route mounting order. This is critical when plugins depend on each other or when certain plugins need to be initialized before others.

## Plugin Priority System

### Registration Methods

**Use (Default Priority)**:
```go
// Registers plugin with default priority (100)
err := aegis.Use(ctx, myPlugin)
```

**UseWithPriority (Custom Priority)**:
```go
// Register with explicit priority
err := aegis.UseWithPriority(ctx, myPlugin, 60)
```

### Priority Guidelines

Lower priority numbers are initialized and mounted **first**:

| Priority Range | Category | Examples |
|----------------|----------|----------|
| **0-50** | Critical infrastructure | Database plugins, core logging |
| **51-99** | High-priority auth | OAuth, JWT (password support is provided by core) |
| **100** | Default (Use method) | Standard plugins |
| **101-150** | Standard plugins | Email verification, SMS |
| **151+** | Low-priority | Admin dashboards, analytics |

## Examples

### Example 1: Basic Usage (Default Priority)

```go
// All use default priority 100
aegis.Use(ctx, emailPlugin)    // Priority: 100
aegis.Use(ctx, smsPlugin)      // Priority: 100  
aegis.Use(ctx, adminPlugin)    // Priority: 100

// Mounted in registration order since priorities are equal
```

### Example 2: Explicit Priorities

```go
// Password support is provided by core; plugins that depend on password
// (for example, Email/SMS helper flows) should be registered after core
// initialization. Core is initialized as part of aegis.New(), so register
// dependent plugins with higher priority numbers when needed.

// Example: Initialize OAuth with high priority and email/sms after
aegis.UseWithPriority(ctx, oauthPlugin, 65)
aegis.UseWithPriority(ctx, emailPlugin, 110)
aegis.UseWithPriority(ctx, smsPlugin, 110)

// Admin plugin last (just a UI)
aegis.UseWithPriority(ctx, adminPlugin, 150)

// Initialization order: oauth → email → sms → admin
```

### Example 3: Infrastructure Plugins

```go
// Logging infrastructure first
aegis.UseWithPriority(ctx, loggingPlugin, 10)

// OpenAPI documentation  
aegis.UseWithPriority(ctx, openapiPlugin, 20)

// Auth plugins
aegis.UseWithPriority(ctx, oauthPlugin, 65)

// Feature plugins
aegis.Use(ctx, emailPlugin) // Uses default 100

// Initialization order: logging → openapi → oauth → email
```

## How It Works

### Initialization Order

1. Plugins are registered with `Use` or `UseWithPriority`
2. Each plugin gets a priority (explicit or default 100)
3. During `Init`, plugin is initialized immediately
4. During `MountRoutes`, plugins mount in priority order (lowest first)

### Thread-Safety

- Plugin registration is thread-safe (protected by mutex)
- Plugins can be registered concurrently
- Order is deterministic based on priority, not registration time

### Sorting Behavior

```go
// Internally, plugins are sorted before mounting:
// 1. Lower priority numbers come first
// 2. If priorities are equal, registration order is preserved
// 3. GetPlugins() returns plugins in priority order
```

## Best Practices

### 1. Use Priorities for Dependencies

```go
// Password support is part of core. Email plugin can be registered after core
// initialization; if you need a specific ordering, use UseWithPriority.
aegis.UseWithPriority(ctx, emailPlugin, 110)
```

### 2. Group Related Plugins

```go
// Authentication plugins: 50-99
aegis.UseWithPriority(ctx, oauthPlugin, 65)
aegis.UseWithPriority(ctx, jwtPlugin, 70)

// User management plugins: 100-149
aegis.Use(ctx, emailPlugin)        // 100 (default)
aegis.Use(ctx, smsPlugin)          // 100 (default)
aegis.Use(ctx, orgPlugin)          // 100 (default)

// Admin/reporting plugins: 150+
aegis.UseWithPriority(ctx, adminPlugin, 150)
aegis.UseWithPriority(ctx, analyticsPlugin, 160)
```

### 3. Document Plugin Dependencies

In your plugin documentation, specify:
- Required priority range
- Dependencies on other plugins
- Recommended initialization order

Example:
```go
// EmailPlugin requires password support (provided by core). If you need a
// specific ordering relative to other plugins, register with UseWithPriority.
// Recommended: Register EmailPlugin with priority 110-150 when appropriate.
```

## API Reference

### Use

```go
func (a *Aegis) Use(ctx context.Context, plugin plugins.Plugin) error
```

Registers a plugin with default priority 100.

**Parameters**:
- `ctx`: Context for initialization (supports cancellation/timeout)
- `plugin`: Plugin to register

**Returns**: Error if initialization fails

### UseWithPriority

```go
func (a *Aegis) UseWithPriority(ctx context.Context, plugin plugins.Plugin, priority int) error
```

Registers a plugin with explicit priority.

**Parameters**:
- `ctx`: Context for initialization
- `plugin`: Plugin to register
- `priority`: Priority value (lower = initialized first)

**Returns**: Error if initialization fails

### GetPlugins

```go
func (a *Aegis) GetPlugins() []plugins.Plugin
```

Returns all registered plugins in priority order (lowest priority first).

**Returns**: Copy of plugins slice sorted by priority

## Migration from v1

If upgrading from a version without priority support:

```go
// Old code (v1)
aegis.Use(plugin1)
aegis.Use(plugin2)
aegis.Use(plugin3)

// New code (v2) - add context, priorities optional
aegis.Use(ctx, plugin1)                        // Default priority 100
aegis.UseWithPriority(ctx, plugin2, 60)        // High priority
aegis.UseWithPriority(ctx, plugin3, 150)       // Low priority
```

If you don't specify priorities, behavior is unchanged (all default to 100, registration order preserved).

## Debugging Plugin Order

Enable logging to see plugin registration order:

```go
import "log/slog"

aegis.New(
    config.WithLogger(slog.Default()),
    // ...
)

// Log output:
// INFO Registering plugin name=oauth version=1.0.0 priority=65
// INFO Plugin registered successfully name=oauth priority=65
// INFO Registering plugin name=email version=1.0.0 priority=110
// INFO Plugin registered successfully name=email priority=110
```

## FAQ

**Q: What happens if two plugins have the same priority?**  
A: Registration order is preserved. The plugin registered first will initialize/mount first.

**Q: Can I change a plugin's priority after registration?**  
A: No. Priority is set during registration and cannot be changed. You must re-register the plugin with a new priority.

**Q: Do I need to use priorities?**  
A: No. If your plugins don't have dependencies, you can use the default `Use` method (priority 100 for all).

**Q: What's the maximum priority value?**  
A: There's no hard limit, but we recommend staying within 0-200 for clarity.

**Q: Can I use negative priorities?**  
A: Technically yes, but not recommended. Use 0-200 range for consistency.
