# Advanced Database Setup

This guide covers advanced database configuration, production setup, migrations, and troubleshooting for Aegis.

> **Note**: For basic database setup, see [Getting Started](./getting-started.md).

## Advanced Database Configuration

### Connection Pooling

Aegis uses your `*sql.DB` connection as-is. Configure pooling before passing to Aegis:

```go
import (
    "database/sql"
    "time"
)

sqlDB, _ := sql.Open("postgres", connString)

// Production-recommended settings
sqlDB.SetMaxOpenConns(25)                // Max concurrent connections
sqlDB.SetMaxIdleConns(5)                 // Keep idle connections ready
sqlDB.SetConnMaxLifetime(5 * time.Minute) // Close old connections
sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Close idle connections

// Test the connection
if err := sqlDB.Ping(); err != nil {
    log.Fatal(err)
}

auth, _ := aegis.New(config.WithDB(sqlDB, db.PostgreSQL))
// For tests, prefer environment provided secrets
auth, _ := aegis.New(
    config.WithDB(sqlDB, db.SQLite),
    config.WithJWTSecret([]byte(os.Getenv("JWT_SECRET"))),
    config.WithAPIOnlyMode(true),
)
```

**Tuning guidelines:**

| Setting | Development | Production | High-Traffic |
|---------|-------------|------------|--------------|
| `MaxOpenConns` | 10 | 25 | 50-100 |
| `MaxIdleConns` | 2 | 5 | 10-25 |
| `ConnMaxLifetime` | 10 min | 5 min | 3 min |
| `ConnMaxIdleTime` | 15 min | 10 min | 5 min |

**Considerations:**
- Too many connections → database overload
- Too few connections → request queuing
- Monitor with `db.Stats()` in production

### Multiple Database Support

Aegis works with any `database/sql` driver through dialects.

#### PostgreSQL Drivers

**lib/pq** (Pure Go):
```go
import _ "github.com/lib/pq"

sqlDB, _ := sql.Open("postgres", connString)
auth, _ := aegis.New(config.WithDB(sqlDB, db.PostgreSQL))
```

**pgx** (Better performance):
```go
import _ "github.com/jackc/pgx/v5/stdlib"

sqlDB, _ := sql.Open("pgx", connString)
auth, _ := aegis.New(config.WithDB(sqlDB, db.PostgreSQL))
```

**Driver comparison:**

| Feature | lib/pq | pgx |
|---------|--------|-----|
| Pure Go | ✅ | ✅ |
| Performance | Good | Excellent |
| Features | Standard | Advanced |
| Maintenance | Stable | Active |
| Binary size | Smaller | Larger |

**Recommendation**: Use `lib/pq` for simplicity, `pgx` for performance.

#### MySQL Drivers

**go-sql-driver/mysql**:
```go
import _ "github.com/go-sql-driver/mysql"

connString := "user:pass@tcp(localhost:3306)/db?parseTime=true"
sqlDB, _ := sql.Open("mysql", connString)
auth, _ := aegis.New(config.WithDB(sqlDB, db.MySQL))
```

> **Critical**: Always include `?parseTime=true` for MySQL

#### SQLite

**mattn/go-sqlite3** (CGo-based):
```go
import _ "github.com/mattn/go-sqlite3"

// In-memory (testing)
sqlDB, _ := sql.Open("sqlite3", ":memory:")

// File-based
sqlDB, _ := sql.Open("sqlite3", "./aegis.db")

auth, _ := aegis.New(config.WithDB(sqlDB, db.SQLite))
```

**Use cases:**
- ✅ Unit tests (in-memory)
- ✅ Development
- ✅ Embedded applications
- ❌ Production (use PostgreSQL or MySQL)

### Dialect Differences

Aegis handles SQL dialect differences automatically:

| Dialect | Placeholder | ID Retrieval | Notes |
|---------|-------------|--------------|-------|
| PostgreSQL | `$1, $2, $3` | `RETURNING id` | Best for complex queries |
| MySQL | `?, ?, ?` | `LAST_INSERT_ID()` | Most widely used |
| SQLite | `?, ?, ?` | `LAST_INSERT_ID()` | Great for testing |

The dialect determines:
- Query placeholder style
- ID retrieval after inserts
- SQL function compatibility

---

## Production Setup

### Connection Limits

**PostgreSQL:**
```sql
-- Check current connections
SELECT count(*) FROM pg_stat_activity;

-- Check max connections
SHOW max_connections;

-- Increase if needed (postgresql.conf)
max_connections = 100
```

**MySQL:**
```sql
-- Check current connections
SHOW STATUS LIKE 'Threads_connected';

-- Check max connections
SHOW VARIABLES LIKE 'max_connections';

-- Increase if needed
SET GLOBAL max_connections = 200;
```

**Application side:**
```go
// Never exceed database max_connections
// Rule of thumb: max_connections / number_of_app_instances
sqlDB.SetMaxOpenConns(25)
```

### Timeouts and Retries

```go
import (
    "context"
    "database/sql"
    "time"
)

// Set connection timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

sqlDB, err := sql.Open("postgres", connString)
if err != nil {
    return err
}

// Test connection with timeout
if err := sqlDB.PingContext(ctx); err != nil {
    return fmt.Errorf("database ping failed: %w", err)
}

// Set query timeouts in your code
ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

rows, err := sqlDB.QueryContext(ctx, "SELECT * FROM auth.user")
```

### Health Checks

Implement database health checks for monitoring:

```go
func healthCheck(db *sql.DB) error {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("database unhealthy: %w", err)
    }
    
    return nil
}

// HTTP health endpoint
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    if err := healthCheck(sqlDB); err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("OK"))
})
```

### Monitoring Connections

```go
import "time"

// Monitor connection pool
ticker := time.NewTicker(30 * time.Second)
go func() {
    for range ticker.C {
        stats := sqlDB.Stats()
        log.Printf("DB Stats: Open=%d InUse=%d Idle=%d WaitCount=%d",
            stats.OpenConnections,
            stats.InUse,
            stats.Idle,
            stats.WaitCount,
        )
        
        // Alert if wait count is high
        if stats.WaitCount > 100 {
            log.Warn("High database wait count - consider increasing MaxOpenConns")
        }
    }
}()
```

---

## Database Migrations

### Using the CLI

Export migrations for your database and plugins:

```bash
# Export core + all plugins
aegis export --format goose --output ./migrations

# Export specific plugins
aegis export --format sql --plugins password,email --output ./migrations

# Export for golang-migrate
aegis export --format golang-migrate --output ./db/migrations
```

See [CLI Reference](./cli.md) for details.

### Integration with Goose

**Install Goose:**
```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

**Export and run:**
```bash
# Export migrations
aegis export --format goose --output ./migrations

# Run migrations
goose -dir ./migrations postgres "$DATABASE_URL" up

# Rollback
goose -dir ./migrations postgres "$DATABASE_URL" down

# Check status
goose -dir ./migrations postgres "$DATABASE_URL" status
```

**In code:**
```go
import "github.com/pressly/goose/v3"

func runMigrations(db *sql.DB) error {
    if err := goose.Up(db, "./migrations"); err != nil {
        return err
    }
    return nil
}
```

### Integration with golang-migrate

**Install golang-migrate:**
```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

**Export and run:**
```bash
# Export migrations
aegis export --format golang-migrate --output ./migrations

# Run migrations
migrate -path ./migrations -database "$DATABASE_URL" up

# Rollback
migrate -path ./migrations -database "$DATABASE_URL" down 1

# Force version (if stuck)
migrate -path ./migrations -database "$DATABASE_URL" force 1
```

### Custom Migration Workflows

**Raw SQL:**
```bash
# Export as SQL
aegis export --format sql --output ./migrations

# Run with psql
for f in migrations/*.sql; do
    psql $DATABASE_URL < $f
done

# Or with mysql
for f in migrations/*.sql; do
    mysql -h localhost -u user -p database < $f
done
```

**Version control:**
```bash
# Commit migrations to git
git add migrations/
git commit -m "Add Aegis migrations"

# Deploy to production
scp migrations/* production:/app/migrations/
ssh production "cd /app && goose -dir migrations postgres \$DATABASE_URL up"
```

---

## Testing Strategies

### SQLite for Unit Tests

Use in-memory SQLite for fast, isolated tests:

```go
func TestUserCreation(t *testing.T) {
    // Create in-memory database
    sqlDB, err := sql.Open("sqlite3", ":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer sqlDB.Close()
    
    // Run migrations (you'll need to export SQLite-compatible migrations)
    // ... create tables ...
    
    // Initialize Aegis
    auth, err := aegis.New(
        config.WithDB(sqlDB, db.SQLite),
        // For test environments prefer env-configured secrets
        config.WithJWTSecret([]byte(os.Getenv("JWT_SECRET"))),
        config.WithAPIOnlyMode(true),
    )
    if err != nil {
        t.Fatal(err)
    }
    
    // Test your code
    // ...
}
```

**Benefits:**
- ✅ Fast (in-memory)
- ✅ Isolated (each test gets fresh DB)
- ✅ No external dependencies
- ✅ Parallel test execution

### Test Database Setup

For integration tests with PostgreSQL/MySQL:

```go
func setupTestDB(t *testing.T) *sql.DB {
    // Use test database
    connString := os.Getenv("TEST_DATABASE_URL")
    if connString == "" {
        t.Skip("TEST_DATABASE_URL not set")
    }
    
    sqlDB, err := sql.Open("postgres", connString)
    if err != nil {
        t.Fatal(err)
    }
    
    // Run migrations
    // ...
    
    // Cleanup after test
    t.Cleanup(func() {
        // Truncate tables
        sqlDB.Exec("TRUNCATE auth.user, auth.accounts, auth.session CASCADE")
        sqlDB.Close()
    })
    
    return sqlDB
}
```

**Best practices:**
- Use separate test database
- Truncate tables between tests
- Use transactions and rollback
- Parallel-safe test isolation

---

## Troubleshooting

### Connection Issues

**Error: "connection refused"**

**Causes:**
- Database not running
- Wrong host/port
- Firewall blocking connection

**Solutions:**
```bash
# Check if database is running
# PostgreSQL
pg_isready -h localhost -p 5432

# MySQL
mysqladmin ping -h localhost

# Test connection manually
psql "postgres://user:pass@localhost/db"
mysql -h localhost -u user -p
```

**Error: "too many connections"**

**Causes:**
- `MaxOpenConns` too high
- Connection leak
- Too many app instances

**Solutions:**
```go
// Reduce max connections
sqlDB.SetMaxOpenConns(10)

// Check for leaks
defer rows.Close() // Always close rows!

// Monitor connections
stats := sqlDB.Stats()
log.Printf("Open: %d, InUse: %d", stats.OpenConnections, stats.InUse)
```

### Query Issues

**Error: "no rows in result set"**

**Causes:**
- Migrations not run
- Wrong dialect
- Data doesn't exist

**Solutions:**
```bash
# Verify tables exist
psql $DATABASE_URL -c "\dt auth.*"

# Check dialect matches database
# PostgreSQL uses $1, MySQL/SQLite use ?
```

**Error: "syntax error near $1"**

**Cause:** Wrong dialect (using PostgreSQL dialect with MySQL/SQLite)

**Solution:**
```go
// Make sure dialect matches database
config.WithDB(sqlDB, db.MySQL) // Not db.PostgreSQL!
```

### Performance Issues

**Slow queries:**

```sql
-- PostgreSQL: Enable query logging
ALTER DATABASE aegis_db SET log_statement = 'all';

-- Check slow queries
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
ORDER BY mean_time DESC
LIMIT 10;
```

**Solutions:**
- Add indexes on frequently queried columns
- Increase connection pool size
- Use connection pooling (PgBouncer for PostgreSQL)
- Optimize queries

**Connection pool exhaustion:**

```go
// Monitor wait count
stats := sqlDB.Stats()
if stats.WaitCount > 0 {
    log.Printf("Connections waiting: %d", stats.WaitCount)
    // Increase MaxOpenConns or optimize query time
}
```

---

## Examples

See the `examples/` directory:
- [`examples/basic/`](../../examples/basic) - Simple PostgreSQL setup
- [`examples/multiple-drivers/`](../../examples/multiple-drivers) - Compare different drivers
- [`examples/plugins_demo/`](../../examples/plugins_demo) - Full-featured application

---

## Next Steps

- [Getting Started](./getting-started.md) - Basic database setup
- [CLI Reference](./cli.md) - Export migrations
- [Core Concepts](./core-concepts.md) - Understand the schema
- [Testing Guide](./testing-guide.md) - Test your authentication
