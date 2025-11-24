# Database Setup Guide

Aegis is designed to be **database-agnostic**, working with any SQL database that has a Go driver. Users bring their own database connection using Go's standard `database/sql` package.

## Quick Start

### Basic Setup

```go
import (
    "database/sql"
    _ "github.com/lib/pq"  // Your choice of driver
    
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
    "github.com/theinventorylib/aegis/db"
)

func main() {
    // 1. Create a database connection (standard Go)
    sqlDB, err := sql.Open("postgres", connString)
    if err != nil {
        log.Fatal(err)
    }
    defer sqlDB.Close()
    
    // 2. Initialize Aegis with your connection
    auth, err := aegis.New(
        config.WithDB(sqlDB, db.PostgreSQL),
        config.WithJWTSecret([]byte("your-secret-key")),
        // ... other options
    )
}
```

## Supported Databases

Aegis supports any SQL database through dialects. Built-in dialects:

### PostgreSQL

**Recommended drivers:**
- `github.com/lib/pq` - Pure Go, well-tested
- `github.com/jackc/pgx/v5/stdlib` - Feature-rich, better performance

**Connection string format:**
```
postgres://username:password@localhost:5432/database?sslmode=disable
```

**Example:**
```go
import _ "github.com/lib/pq"

sqlDB, _ := sql.Open("postgres", 
    "postgres://user:pass@localhost:5432/aegis_db?sslmode=disable")

auth, _ := aegis.New(
    config.WithDB(sqlDB, db.PostgreSQL),
    // ... options
)
```

**Convenience helper** (uses lib/pq):
```go
auth, _ := aegis.New(
    config.WithPostgres("postgres://user:pass@localhost:5432/aegis_db"),
    // ... options
)
```

---

### MySQL

**Recommended driver:**
- `github.com/go-sql-driver/mysql` - Official MySQL driver

**Connection string format:**
```
username:password@tcp(localhost:3306)/database?parseTime=true
```

> **Important:** Always include `?parseTime=true` for MySQL to properly handle timestamps.

**Example:**
```go
import _ "github.com/go-sql-driver/mysql"

sqlDB, _ := sql.Open("mysql",
    "user:pass@tcp(127.0.0.1:3306)/aegis_db?parseTime=true")

auth, _ := aegis.New(
    config.WithDB(sqlDB, db.MySQL),
    // ... options
)
```

**Convenience helper:**
```go
auth, _ := aegis.New(
    config.WithMySQL("user:pass@tcp(127.0.0.1:3306)/aegis_db?parseTime=true"),
    // ... options
)
```

---

### SQLite

**Recommended driver:**
- `github.com/mattn/go-sqlite3` - CGo-based (requires C compiler)

**Use case:** Perfect for development, testing, and embedded applications

**Example (in-memory):**
```go
import _ "github.com/mattn/go-sqlite3"

sqlDB, _ := sql.Open("sqlite3", ":memory:")

auth, _ := aegis.New(
    config.WithDB(sqlDB, db.SQLite),
    // ... options
)
```

**Example (file-based):**
```go
sqlDB, _ := sql.Open("sqlite3", "./aegis.db")
```

## Configuration Options

### Option 1: WithDB (Recommended)

Use this when you manage your own database connection:

```go
config.WithDB(sqlDB *sql.DB, dialect db.Dialect)
```

**Benefits:**
- Full control over connection pooling
- Use any driver you prefer
- Share connections with other parts of your app

**Example:**
```go
import _ "github.com/jackc/pgx/v5/stdlib"

// Configure your own connection pool
sqlDB, _ := sql.Open("pgx", connString)
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(5)
sqlDB.SetConnMaxLifetime(5 * time.Minute)

auth, _ := aegis.New(
    config.WithDB(sqlDB, db.PostgreSQL),
    // ... other options
)
```

### Option 2: Convenience Helpers

For quick setup, use built-in helpers:

```go
config.WithPostgres(connString string)  // Uses lib/pq
config.WithMySQL(connString string)     // Uses go-sql-driver/mysql
```

These helpers:
- Create the `*sql.DB` connection for you
- Test the connection with `Ping()`
- Return errors during initialization

**Example:**
```go
auth, err := aegis.New(
    config.WithPostgres("postgres://user:pass@localhost/db"),
    // ... other options
)
// err will contain connection errors if any
```

## Connection Pooling

Aegis uses your `*sql.DB` connection as-is. Configure pooling before passing to Aegis:

```go
sqlDB, _ := sql.Open("postgres", connString)

// Recommended production settings
sqlDB.SetMaxOpenConns(25)              // Max concurrent connections
sqlDB.SetMaxIdleConns(5)               // Keep idle connections ready
sqlDB.SetConnMaxLifetime(5*time.Minute) // Close old connections
sqlDB.SetConnMaxIdleTime(10*time.Minute) // Close idle connections

// Test the connection
if err := sqlDB.Ping(); err != nil {
    log.Fatal(err)
}

auth, _ := aegis.New(config.WithDB(sqlDB, db.PostgreSQL))
```

## Migration from Old API

If you're using the old Aegis API with driver-specific functions:

### Before:
```go
database, err := db.NewPostgresProvider(connString)
auth, err := aegis.New(config.WithPostgres(database), ...)
```

### After (Option A - Simple):
```go
auth, err := aegis.New(
    config.WithPostgres(connString),  // Now accepts string!
    // ... other options
)
```

### After (Option B - Full Control):
```go
sqlDB, err := sql.Open("postgres", connString)
// ... configure pooling ...
auth, err := aegis.New(
    config.WithDB(sqlDB, db.PostgreSQL),
    // ... other options
)
```

## Dialects Explained

Aegis uses dialects to handle SQL syntax differences:

| Dialect | Placeholder | ID Generation | Notes |
|---------|-------------|---------------|-------|
| `db.PostgreSQL` | `$1, $2` | `RETURNING` | Best for complex queries |
| `db.MySQL` | `?, ?` | `LAST_INSERT_ID()` | Most widely used |
| `db.SQLite` | `?, ?` | `LAST_INSERT_ID()` | Great for testing |

The dialect determines:
- Query placeholder style
- ID retrieval method after inserts
- SQL function compatibility

## Testing with SQLite

SQLite is perfect for unit tests:

```go
func TestUserCreation(t *testing.T) {
    // In-memory database - fast and isolated
    sqlDB, _ := sql.Open("sqlite3", ":memory:")
    defer sqlDB.Close()
    
    // Run migrations
    // ... create tables ...
    
    auth, _ := aegis.New(
        config.WithDB(sqlDB, db.SQLite),
        config.WithJWTSecret([]byte("test-secret")),
        config.WithAPIOnlyMode(true),
    )
    
    // Test your code
    // ...
}
```

## Common Issues

### Issue: "could not import database driver"

**Solution:** Make sure you import the driver as a blank import:
```go
import _ "github.com/lib/pq"
```

### Issue: "no rows in result set"

**Solution:** Check your dialect matches your database. PostgreSQL uses `$1` placeholders while MySQL/SQLite use `?`.

### Issue: "connection refused"

**Solution:** 
1. Verify database is running
2. Check connection string
3. Test with `sqlDB.Ping()`

### Issue: "too many connections"

**Solution:** Configure connection pooling:
```go
sqlDB.SetMaxOpenConns(25)  // Adjust based on your needs
```

## Examples

See the `examples/` directory:
- `examples/basic/` - Simple PostgreSQL setup
- `examples/multiple-drivers/` - Compare different drivers
- `examples/plugins_demo/` - Full-featured application

## Next Steps

- [Schema Setup](./schema.md) - Create required database tables
- [Plugin System](./plugins.md) - Add email, SMS, OAuth support
- [Deployment](./deployment.md) - Production best practices
