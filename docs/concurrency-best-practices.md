# Concurrency Best Practices for Aegis

## Overview

Aegis is designed to be thread-safe and support concurrent operations. This guide covers best practices for concurrent usage of Aegis in production applications.

## Thread-Safe Operations

### 1. Plugin Registration

**Thread-Safe**: ✅ Yes

```go
// Safe to register plugins concurrently from multiple goroutines
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        plugin := mypackage.NewPlugin(id)
        err := aegis.Use(ctx, plugin)
        if err != nil {
            log.Printf("Failed to register plugin %d: %v", id, err)
        }
    }(i)
}
wg.Wait()
```

**How It Works**:
- Plugin registration uses `sync.RWMutex` write lock  
- Concurrent registrations are serialized internally
- Safe to call from multiple goroutines

---

### 2. Plugin Access

**Thread-Safe**: ✅ Yes

```go
// Safe to access plugins concurrently
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        // GetPlugin uses read lock - very fast
        plugin, ok := aegis.GetPlugin("myplugin")
        if ok {
            // Use plugin
        }
    }()
}
wg.Wait()
```

**How It Works**:
- Plugin access uses `sync.RWMutex` read lock
- Multiple readers can access simultaneously
- No blocking between concurrent reads

---

### 3. Request Handling

**Thread-Safe**: ✅ Yes

```go
// Aegis middleware is thread-safe
// Each request runs in its own goroutine - no shared state
r.Use(aegis.AuthMiddleware())

r.Get("/api/data", func(w http.ResponseWriter, r *http.Request) {
    // This runs concurrently for each request - completely safe
    user, err := aegis.GetUser(r.Context())
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    // Process request...
})
```

**How It Works**:
- Each HTTP request has its own goroutine and context
- No shared state between requests
- Session data is safely loaded from database/Redis per request

---

## Context Management

### 1. Use Context for Cancellation

**Best Practice**: Always use context for plugin initialization to support timeouts and cancellation.

```go
// Good: With timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := aegis.Use(ctx, myPlugin)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Error("Plugin initialization timed out")
    }
    return err
}

// Avoid: No timeout
err := aegis.Use(context.Background(), myPlugin) // Could hang forever
```

---

### 2. Context Propagation in Plugins

**Plugin authors**: Respect context cancellation in `Init` method.

```go
func (p *MyPlugin) Init(ctx context.Context, a plugins.Aegis) error {
    // Good: Check context before long operations
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Perform initialization...
    if err := p.loadConfig(ctx); err != nil {
        return err
    }
    
    // Check again after each step
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // More initialization...
    return nil
}
```

---

## Database Concurrency

### 1. Connection Pooling

**Best Practice**: Configure database connection pool for your concurrency needs.

```go
// Configure connection pool BEFORE passing to Aegis
sqlDB, _ := sql.Open("postgres", connString)

// Set limits based on your workload
sqlDB.SetMaxOpenConns(25)    // Maximum number of open connections
sqlDB.SetMaxIdleConns(5)      // Maximum idle connections
sqlDB.SetConnMaxLifetime(5 * time.Minute)
sqlDB.SetConnMaxIdleTime(10 * time.Minute)

// Now pass to Aegis
aegis.New(
    config.WithDB(sqlDB, db.PostgreSQL),
    // ...
)
```

**Guidelines**:
- `MaxOpenConns`: 10-100 depending on server capacity
- `MaxIdleConns`: 25-50% of MaxOpenConns
- Set connection lifetimes to prevent stale connections

---

### 2. Transaction Safety

**Database transactions are NOT thread-safe**. Each transaction must be used by a single goroutine.

```go
// Good: Transaction in single goroutine
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback()

// All operations in same goroutine
if err := tx.Exec(...); err != nil {
    return err
}

return tx.Commit()

// Bad: Sharing transaction across goroutines
tx, _ := db.Begin()
go func() {
    tx.Exec(...) // UNSAFE! Don't do this
}()
```

---

## Redis Concurrency

### 1. Redis Client Thread-Safety

**Redis clients are thread-safe** - safe to share across goroutines.

```go
// Single Redis client shared across all requests - this is correct
aegis.New(
    config.WithRedis("localhost", 6379, "password", 0),
   // ...
)

// Redis client is reused safely for all sessions
```

---

### 2. Session Locking

**Session updates** use optimistic locking (CAS - Compare-And-Swap) in Redis.

```go
// Multiple concurrent requests updating same session
// Aegis handles this internally with retry logic
// You don't need to do anything special
```

---

## Race Detector

### Running Tests with Race Detector

**Always run tests with race detector** during development:

```bash
go test -race ./...
```

**In CI/CD**:
```yaml
# .github/workflows/ci.yml
- name: Run tests
  run: go test -v -race -coverprofile=coverage.txt ./...
```

---

## Common Pitfalls

### ❌ Pitfall 1: Sharing Mutable State in Plugins

```go
// Bad: Mutable state without protection
type MyPlugin struct {
    counter int // UNSAFE if modified concurrently
}

func (p *MyPlugin) HandleRequest(w http.ResponseWriter, r *http.Request) {
    p.counter++ // Race condition!
}
```

**Solution**: Use atomic operations or mutexes:

```go
// Good: Thread-safe counter
type MyPlugin struct {
    counter atomic.Int64
}

func (p *MyPlugin) HandleRequest(w http.ResponseWriter, r *http.Request) {
    p.counter.Add(1) // Safe!
}

// Or with mutex:
type MyPlugin struct {
    mu      sync.Mutex
    counter int
}

func (p *MyPlugin) HandleRequest(w http.ResponseWriter, r *http.Request) {
    p.mu.Lock()
    p.counter++
    p.mu.Unlock()
}
```

---

### ❌ Pitfall 2: Not Respecting Context Cancellation

```go
// Bad: Ignoring context
func (p *MyPlugin) Init(ctx context.Context, a plugins.Aegis) error {
    time.Sleep(10 * time.Second) // Blocked even if context cancelled
    return nil
}
```

**Solution**: Check context or use context-aware operations:

```go
// Good: Respecting context
func (p *MyPlugin) Init(ctx context.Context, a plugins.Aegis) error {
    select {
    case <-time.After(10 * time.Second):
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

---

### ❌ Pitfall 3: Modifying Returned Slices

```go
// Bad: Modifying returned slice
plugins := aegis.GetPlugins()
plugins[0] = nil  // Could affect internal state in some implementations
```

**Solution**: GetPlugins returns a copy, but still avoid mutation:

```go
// Good: Treat as read-only or make your own copy
plugins := aegis.GetPlugins()
for _, p := range plugins {
    log.Println(p.Name()) // Read-only access
}
```

---

## Performance Tips

### 1. Minimize Lock Contention

**Read locks are cheap** - use `GetPlugin` liberally:

```go
// This is fine - read locks don't block each other
for i := 0; i < 1000; i++ {
    plugin, _ := aegis.GetPlugin("myplugin")
    plugin.DoSomething()
}
```

**Write locks are expensive** - batch plugin registrations:

```go
// Good: Register all plugins once at startup
aegis.Use(ctx, plugin1)
aegis.Use(ctx, plugin2)
aegis.Use(ctx, plugin3)

// Bad: Repeated registration at runtime
// Don't register/unregister plugins during request handling
```

---

### 2. Connection Pool Tuning

Monitor your connection pool usage:

```go
stats := sqlDB.Stats()
log.Printf("Open connections: %d", stats.OpenConnections)
log.Printf("In use: %d", stats.InUse)
log.Printf("Idle: %d", stats.Idle)
log.Printf("Wait count: %d", stats.WaitCount) // High = need more connections
log.Printf("Wait duration: %v", stats.WaitDuration)
```

**Adjust based on metrics**:
- High `WaitCount` → Increase `MaxOpenConns`
- Many `Idle` connections → Decrease `MaxIdleConns`

---

## Testing for Race Conditions

### 1. Concurrent Request Tests

```go
func TestConcurrentAuth(t *testing.T) {
    aegis := setupTestAegis(t)
    
    const numRequests = 100
    var wg sync.WaitGroup
    wg.Add(numRequests)
    
    for i := 0; i < numRequests; i++ {
        go func() {
            defer wg.Done()
            // Simulate request
            ctx := context.Background()
            _, _ = aegis.GetUser(ctx)
        }()
    }
    
    wg.Wait()
}
```

### 2. Run with Race Detector

```bash
go test -race -run TestConcurrentAuth
```

---

## Summary Checklist

✅ **DO**:
- Use race detector in development and CI
- Configure database connection pools  
- Use context with timeouts for plugin initialization
- Protect mutable state in plugins with mutexes/atomics
- Use read-only access to returned data
- Test concurrent scenarios

❌ **DON'T**:
- Share database transactions across goroutines
- Mutate returned slices from GetPlugins
- Ignore context cancellation in plugin Init
- Register/unregister plugins during request handling
- Create new database connections per request

---

## Related Documentation

- [Plugin Priorities](./plugin-priorities.md) - Plugin ordering and dependencies
- [SECURITY.md](../SECURITY.md) - Security best practices
- [Getting Started](./getting-started.md) - Configuration options

---

## Go Concurrency Resources

- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Share Memory By Communicating](https://go.dev/blog/codelab-share)
- [Race Detector](https://go.dev/doc/articles/race_detector)
