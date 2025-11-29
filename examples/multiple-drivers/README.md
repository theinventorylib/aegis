# Multiple Database Drivers Example

This example demonstrates Aegis's database-agnostic design by showing how to use different SQL drivers with the same codebase.

## Overview

Aegis works with any `database/sql` driver through its dialect system. This example shows:
- PostgreSQL with `lib/pq` driver
- PostgreSQL with `pgx` driver (commented)
- MySQL (commented)
- SQLite (commented)

## Database-Agnostic Design

Aegis uses Go's standard `database/sql` package, making it compatible with any SQL database driver:

```go
// Same code, different drivers!
sqlDB, _ := sql.Open("postgres", connString)  // or "pgx", "mysql", "sqlite3"
auth, _ := aegis.New(
    config.WithDB(sqlDB, db.PostgreSQL),  // or db.MySQL, db.SQLite
)
```

## Running the Example

### Prerequisites

1. **PostgreSQL** (for default example):
```bash
# Install PostgreSQL
brew install postgresql  # macOS
sudo apt-get install postgresql  # Ubuntu

# Create database
createdb aegis_dev
```

2. **Import the driver**:
```bash
go get github.com/lib/pq
```

### Run

```bash
cd examples/multiple-drivers
go run main.go
```

## Expected Output

```
Aegis Database-Agnostic Example
================================
Example 1: PostgreSQL with lib/pq driver
-----------------------------------------
✓ Successfully initialized Aegis with PostgreSQL (lib/pq)
✓ Database provider: *db.SQLProvider
```

## Driver Comparison

### PostgreSQL Drivers

#### lib/pq (Pure Go)

**Pros:**
- ✅ Pure Go (no CGo)
- ✅ Well-tested and stable
- ✅ Smaller binary size
- ✅ Easy to cross-compile

**Cons:**
- ⚠️ Slower than pgx
- ⚠️ Fewer advanced features

**Usage:**
```go
import _ "github.com/lib/pq"

sqlDB, _ := sql.Open("postgres", connString)
auth, _ := aegis.New(config.WithDB(sqlDB, db.PostgreSQL))
```

#### pgx (High Performance)

**Pros:**
- ✅ Better performance
- ✅ More PostgreSQL features
- ✅ Active development
- ✅ Connection pooling built-in

**Cons:**
- ⚠️ Larger binary size
- ⚠️ More complex API

**Usage:**
```go
import _ "github.com/jackc/pgx/v5/stdlib"

sqlDB, _ := sql.Open("pgx", connString)
auth, _ := aegis.New(config.WithDB(sqlDB, db.PostgreSQL))
```

### MySQL Driver

**go-sql-driver/mysql** (Official)

**Usage:**
```go
import _ "github.com/go-sql-driver/mysql"

connString := "user:pass@tcp(127.0.0.1:3306)/aegis_db?parseTime=true"
sqlDB, _ := sql.Open("mysql", connString)
auth, _ := aegis.New(config.WithDB(sqlDB, db.MySQL))
```

> **Important**: Always include `?parseTime=true` for MySQL

### SQLite Driver

**mattn/go-sqlite3** (CGo-based)

**Usage:**
```go
import _ "github.com/mattn/go-sqlite3"

// In-memory (perfect for testing)
sqlDB, _ := sql.Open("sqlite3", ":memory:")

// File-based
sqlDB, _ := sql.Open("sqlite3", "./aegis.db")

auth, _ := aegis.New(config.WithDB(sqlDB, db.SQLite))
```

## Trying Different Drivers

### 1. PostgreSQL with pgx

Uncomment in `main.go`:
```go
import _ "github.com/jackc/pgx/v5/stdlib"

func pgxExample() {
    // ... implementation
}
```

Then:
```bash
go get github.com/jackc/pgx/v5/stdlib
go run main.go
```

### 2. MySQL

Uncomment in `main.go`:
```go
import _ "github.com/go-sql-driver/mysql"

func mysqlExample() {
    // ... implementation
}
```

Setup MySQL:
```bash
# Create database
mysql -u root -p -e "CREATE DATABASE aegis_db"

# Update connection string in main.go
connString := "root:password@tcp(127.0.0.1:3306)/aegis_db?parseTime=true"
```

Then:
```bash
go get github.com/go-sql-driver/mysql
go run main.go
```

### 3. SQLite

Uncomment in `main.go`:
```go
import _ "github.com/mattn/go-sqlite3"

func sqliteExample() {
    // ... implementation
}
```

Then:
```bash
go get github.com/mattn/go-sqlite3
go run main.go
```

## Connection String Formats

### PostgreSQL
```
postgres://username:password@localhost:5432/database?sslmode=disable
```

### MySQL
```
username:password@tcp(localhost:3306)/database?parseTime=true
```

### SQLite
```
:memory:           # In-memory
./aegis.db         # File-based
/path/to/db.sqlite # Absolute path
```

## Dialect Differences

Aegis handles SQL dialect differences automatically:

| Feature | PostgreSQL | MySQL | SQLite |
|---------|------------|-------|--------|
| Placeholder | `$1, $2` | `?, ?` | `?, ?` |
| ID Retrieval | `RETURNING id` | `LAST_INSERT_ID()` | `LAST_INSERT_ID()` |
| Schema | Supported | Not used | Not used |
| JSON | `JSONB` | `JSON` | `TEXT` |

## Production Recommendations

**PostgreSQL** (Recommended):
- ✅ Best for production
- ✅ Full feature support
- ✅ ACID compliant
- ✅ Excellent performance
- **Driver**: `pgx` for performance, `lib/pq` for simplicity

**MySQL**:
- ✅ Widely supported
- ✅ Good performance
- ✅ Easy to deploy
- **Driver**: `go-sql-driver/mysql`

**SQLite**:
- ✅ Perfect for testing
- ✅ Great for development
- ✅ Embedded applications
- ❌ Not recommended for production web apps
- **Driver**: `mattn/go-sqlite3`

## Learn More

- [Database Setup Guide](../../docs/database-setup.md) - Advanced database configuration
- [Getting Started](../../docs/getting-started.md) - Basic Aegis setup
- [Core Concepts](../../docs/core-concepts.md) - Database provider architecture
