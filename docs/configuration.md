# Configuration

Aegis is designed to be configured via code using the functional options pattern. This provides type safety and flexibility.

## Core Configuration

When initializing Aegis with `aegis.New()`, you can pass several options:

### `config.WithDB(db *sql.DB, driver db.DriverType)`

**Required**. Sets the database connection and driver type.

```go
auth, _ := aegis.New(
    config.WithDB(sqlDB, db.PostgreSQL),
)
```

Supported drivers:
- `db.PostgreSQL`
- `db.MySQL`
- `db.SQLite`

### `config.WithJWTSecret(secret []byte)`

**Required**. Sets the secret key used for signing JWTs.

```go
auth, _ := aegis.New(
    // ...
    config.WithJWTSecret([]byte("your-secret-key")),
)
```

### `config.WithRouter(router server.Router)`

**Required**. Sets the router adapter.

```go
// Create a default router wrapping http.ServeMux
mux := http.NewServeMux()
router := server.NewDefaultRouter(mux)

auth, _ := aegis.New(
    config.WithRouter(router),
    // ...
)
```

### `config.WithIDGenerator(gen func() string)`

**Optional**. Sets a custom ID generator function. By default, Aegis uses **sequential IDs** (e.g., "1", "2", "3").

```go
auth, _ := aegis.New(
    // ...
    config.WithIDGenerator(func() string {
        return "custom-id-" + uuid.NewString()
    }),
)
```

## Plugin Configuration

Each plugin has its own configuration struct. See the [Plugins](./plugins.md) documentation for details on configuring specific plugins.
