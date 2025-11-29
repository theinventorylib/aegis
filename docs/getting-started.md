# Getting Started

This comprehensive guide will help you get Aegis up and running in your Go application, from installation through your first working authentication system.

## Prerequisites

- **Go**: Version 1.21 or higher
- **Database**: PostgreSQL, MySQL, or SQLite
- **Basic Go knowledge**: Familiarity with Go modules and `database/sql`

## Installation

### 1. Install the Library

Add Aegis to your project:

```bash
go get github.com/theinventorylib/aegis
```

### 2. Install the CLI Tool (Recommended)

The CLI tool exports database migrations:

```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

Verify installation:

```bash
aegis version
```

---

## Database Setup

Aegis does **not** auto-migrate your database. You control when and how migrations run.

### Step 1: Export Migrations

Use the CLI to export migration files:

```bash
# Export core schema only
aegis export --format sql --output ./migrations/aegis

# Or export with plugins
aegis export --format goose --plugins password,email --output ./migrations
```

### Step 2: Run Migrations

Apply migrations using your preferred tool:

**PostgreSQL (psql):**
```bash
psql $DATABASE_URL -f ./migrations/aegis/001_aegis_core.sql
```

**Goose:**
```bash
goose -dir ./migrations postgres "$DATABASE_URL" up
```

**golang-migrate:**
```bash
migrate -path ./migrations -database "$DATABASE_URL" up
```

### Database Quick Reference

#### PostgreSQL

**Connection string:**
```
postgres://username:password@localhost:5432/database?sslmode=disable
```

**Setup:**
```bash
createdb aegis_db
psql aegis_db < ./migrations/001_aegis_core.sql
```

**Recommended driver:**
```bash
go get github.com/lib/pq
```

#### MySQL

**Connection string:**
```
username:password@tcp(localhost:3306)/database?parseTime=true
```

> **Important**: Always include `?parseTime=true` for MySQL

**Setup:**
```bash
mysql -u root -p -e "CREATE DATABASE aegis_db"
mysql -u root -p aegis_db < ./migrations/001_aegis_core.sql
```

**Recommended driver:**
```bash
go get github.com/go-sql-driver/mysql
```

#### SQLite

**Connection string:**
```
:memory:           # In-memory (testing)
./aegis.db         # File-based
```

**Recommended driver:**
```bash
go get github.com/mattn/go-sqlite3
```

---

## Basic Configuration

Aegis uses functional options for type-safe configuration.

### Required Options

#### 1. Database Connection

```go
import (
    "database/sql"
    _ "github.com/lib/pq"  // Import your driver
    
    "github.com/theinventorylib/aegis/config"
    "github.com/theinventorylib/aegis/db"
)

// Option A: Bring your own connection
sqlDB, _ := sql.Open("postgres", connString)
config.WithDB(sqlDB, db.PostgreSQL)

// Option B: Use convenience helper (PostgreSQL)
config.WithPostgres("postgres://user:pass@localhost/db")

// Option C: Use convenience helper (MySQL)
config.WithMySQL("user:pass@tcp(localhost:3306)/db?parseTime=true")
```

**Supported dialects:**
- `db.PostgreSQL` - PostgreSQL (lib/pq or pgx)
- `db.MySQL` - MySQL/MariaDB
- `db.SQLite` - SQLite

#### 2. Router

```go
import (
    "net/http"
    "github.com/theinventorylib/aegis/server"
)

// Default router (wraps http.ServeMux)
mux := http.NewServeMux()
router := server.NewDefaultRouter(mux)
config.WithRouter(router)

// Or use Chi router
import "github.com/go-chi/chi/v5"
chiRouter := chi.NewRouter()
config.WithRouter(server.NewChiRouter(chiRouter))
```

#### 3. JWT Secret

```go
// Use a secure random key (32+ bytes recommended)
config.WithJWTSecret([]byte("your-very-secure-secret-key-here"))

// In production, load from environment
import "os"
config.WithJWTSecret([]byte(os.Getenv("JWT_SECRET")))
```

### Common Options

#### CSRF Protection

```go
// Required for web applications (cookie-based auth)
config.WithCSRFSecret([]byte("your-csrf-secret-key"))

// Skip CSRF for API-only applications
config.WithAPIOnlyMode(true)
```

#### Session Configuration

```go
import "time"

// Session token expiry (default: 24 hours)
config.WithSessionExpiry(1 * time.Hour)

// Refresh token expiry (default: 7 days)
config.WithRefreshExpiry(30 * 24 * time.Hour)
```

#### Cookie Settings

```go
// Cookie domain (for subdomain sharing)
config.WithCookieDomain(".example.com")

// HTTPS-only cookies (MUST be true in production)
config.WithCookieSecure(true)

// SameSite attribute
config.WithCookieSameSite("Lax")  // or "Strict", "None"
```

#### ID Generation

```go
import (
    "github.com/google/uuid"
    "github.com/theinventorylib/aegis/core"
)

// Default: ULID (recommended - sortable, restart-safe)
// No configuration needed!

// Alternative: UUID
core.SetIDStrategy(core.IDStrategyUUID)

// Custom generator
config.WithIDGenerator(func() string {
    return uuid.New().String()
})
```

See [ID Generation Guide](./id-generation.md) for details.

---

## Your First Aegis App

Here's a complete working example:

```go
package main

import (
    "database/sql"
    "fmt"
    "log"
    "net/http"
    "os"

    _ "github.com/lib/pq"

    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
    "github.com/theinventorylib/aegis/db"
    "github.com/theinventorylib/aegis/server"
)

func main() {
    // 1. Connect to database
    connString := os.Getenv("DATABASE_URL")
    if connString == "" {
        connString = "postgres://user:pass@localhost:5432/aegis_db?sslmode=disable"
    }
    
    sqlDB, err := sql.Open("postgres", connString)
    if err != nil {
        log.Fatal("Database connection failed:", err)
    }
    defer sqlDB.Close()
    
    // Test connection
    if err := sqlDB.Ping(); err != nil {
        log.Fatal("Database ping failed:", err)
    }

    // 2. Create router
    mux := http.NewServeMux()
    router := server.NewDefaultRouter(mux)

    // 3. Initialize Aegis
    auth, err := aegis.New(
        config.WithDB(sqlDB, db.PostgreSQL),
        config.WithRouter(router),
        config.WithJWTSecret([]byte(os.Getenv("JWT_SECRET"))),
        config.WithCSRFSecret([]byte(os.Getenv("CSRF_SECRET"))),
    )
    if err != nil {
        log.Fatal("Aegis initialization failed:", err)
    }

    // 4. Mount authentication routes
    // This registers routes like /auth/logout, /auth/user, etc.
    auth.MountRoutes("/auth")

    // 5. Add your application routes
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Welcome to Aegis!")
    })
    
    // Protected route example
    protectedHandler := auth.RequireAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, _ := auth.GetUser(r.Context())
        fmt.Fprintf(w, "Hello, user %s!", user.ID)
    }))
    mux.Handle("/protected", protectedHandler)

    // 6. Start server
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    fmt.Printf("Server starting on :%s\n", port)
    fmt.Println("Auth routes mounted at /auth")
    log.Fatal(http.ListenAndServe(":"+port, mux))
}
```

### Testing Your Setup

```bash
# Start your server
go run main.go

# Test the welcome endpoint
curl http://localhost:8080/

# Test a protected endpoint (should return 401)
curl http://localhost:8080/protected
```

---

## Adding Plugins

Aegis functionality is extended through plugins. Note: password-based authentication is now implemented in core rather than as a separate plugin. You have two options when working with password authentication:

- Use the core `AuthService` directly to create users and verify passwords. Example:

```go
// Create a user with a password using the core auth service
authService := auth.GetAuthService()
user, err := authService.CreateUserWithPassword(context.Background(), "s3cur3P@ssw0rd")
if err != nil {
    log.Fatal("failed to create user with password:", err)
}
_ = user
```

- Or use the Email/SMS plugins which provide convenience helpers that call into core. For example, `CreateUserWithEmailAndPassword` or `CreateUserWithPhoneAndPassword` will create the user and a password account atomically from your application code.

See [Plugins Guide](./plugins.md) for other available plugins and how to register them.

---

## Environment Variables

For production, use environment variables:

```bash
# .env file
DATABASE_URL=postgres://user:pass@localhost:5432/aegis_prod?sslmode=require
JWT_SECRET=your-production-jwt-secret-min-32-bytes
CSRF_SECRET=your-production-csrf-secret-min-32-bytes
PORT=8080
```

Load with a package like `godotenv`:

```go
import "github.com/joho/godotenv"

func main() {
    godotenv.Load()
    // ... rest of setup
}
```

---

## Next Steps

### Learn Core Concepts
- [Core Concepts](./core-concepts.md) - Understand Aegis architecture
- [API Reference](./api-reference.md) - Complete API documentation

### Add Authentication Methods
    - [Email Plugin](./plugins/email.md) - Email verification and magic links
- [SMS Plugin](./plugins/sms.md) - Phone verification
- [OAuth Plugin](./plugins/oauth.md) - Social login (Google, GitHub, etc.)

### Advanced Topics
- [Database Setup](./database-setup.md) - Advanced database configuration
- [ID Generation](./id-generation.md) - Customize ID generation
- [Testing Guide](./testing-guide.md) - Test your authentication

### Production
- [Security Best Practices](../SECURITY.md) - Production security
- [Concurrency Guide](./concurrency-best-practices.md) - Thread-safety

---

## Troubleshooting

### "no rows in result set"
- Verify migrations were applied: `psql $DATABASE_URL -c "\dt auth.*"`
- Check dialect matches your database

### "connection refused"
- Verify database is running
- Test connection: `psql $DATABASE_URL` or `mysql -h localhost -u user -p`
- Check connection string format

### "CSRF token missing"
- For web apps, ensure `WithCSRFSecret()` is set
- For APIs, use `WithAPIOnlyMode(true)`

### Import errors
- Ensure database driver is imported: `import _ "github.com/lib/pq"`
- Run `go mod tidy`

For more help, see [Database Setup Guide](./database-setup.md) or [open an issue](https://github.com/theinventorylib/aegis/issues).
