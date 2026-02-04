# Go-Native Auth System with Plugins, sqlc, and Strong Typing

This document describes a **Go-idiomatic architecture** for building an authentication system with **pluggable features**, **strong typing**, **high performance**, and **multi-dialect SQL support**, inspired by systems like *Better Auth* but designed explicitly for Go.
The design prioritizes:

- Compile-time safety
- Explicit contracts
- Zero runtime schema magic
- Clear ownership of schema, types, and migrations
- Compatibility with `sqlc`

---

## 1. Goals & Constraints

### Goals

- Allow users to **own and manage their database schema**
- Allow **plugins** to define and evolve their own schema
- Support schema extensions (e.g. roles, admin features)
- Preserve **strong typing and compile-time safety**
- Maintain **high performance** (close to raw SQL)
- Feel **idiomatic to Go**

### Constraints (Go & sqlc)

- Types must be known at compile time
- sqlc runs at build time, not runtime
- Packages cannot regenerate code in dependent modules
- Schema mutation must be explicit and controlled

---

## 2. Core Design Principles (The Go Way)

1. **Contracts over configuration**
   Schema and behavior are contracts, not runtime inputs.

2. **Interfaces over injection**
   Storage is customized via interfaces, not schema blobs.

3. **Composition over mutation**
   Plugins extend core data models; they never mutate them.

4. **Compile-time safety over runtime magic**
   Prefer build-time errors to runtime surprises.

---

## 3. High-Level Architecture

```
Application
└── wires core + plugins
    ├── Auth Core
    │   ├── Stable schema
    │   ├── Stable models
    │   └── Store interfaces
    ├── Plugins
    │   ├── Plugin schema
    │   ├── Plugin models
    │   └── Plugin sqlc
    └── Database
        └── User-managed migrations
```

---

## 4. Repository & Architecture Tree

**Aegis** implementation with complete plugin ecosystem:

```
aegis/
├── aegis.go                      # Main framework entry point
├── go.mod                        # Go module definition
│
├── auth/                         # Core authentication package
│   ├── auth.go                   # Core auth logic
│   ├── models.go                 # User, Session models
│   ├── store.go                  # Store interfaces
│   ├── default_store.go          # Default sqlc implementation
│   ├── schema.go                 # Schema definitions
│   ├── migrations.go             # Migration management
│   ├── internal/
│   │   ├── sql/
│   │   │   ├── postgres/         # PostgreSQL queries
│   │   │   └── mysql/            # MySQL queries
│   │   └── gen/
│   │       └── sqlc/             # sqlc-generated code
│   └── migrations/
│       ├── postgres/             # PostgreSQL migrations
│       └── mysql/                # MySQL migrations
│
├── core/                         # Core utilities and interfaces
│   ├── auth.go                   # Authentication handlers
│   ├── session.go                # Session management
│   ├── password.go               # Argon2id password hashing
│   ├── errors.go                 # Structured error types
│   ├── validation.go             # Request validation
│   ├── ratelimit.go              # Rate limiting
│   ├── audit.go                  # Audit logging interface
│   ├── constants.go              # Named constants
│   └── ...
│
├── router/                       # HTTP routing layer
│   └── ...
│
├── plugins/                      # Plugin ecosystem (8 plugins)
│   ├── provider.go               # Plugin interface
│   ├── registry.go               # Plugin registry
│   │
│   ├── admin/                    # User management plugin
│   ├── bearer/                   # Bearer token authentication
│   ├── emailotp/                 # Email OTP verification
│   ├── jwt/                      # JWT authentication
│   ├── oauth/                    # OAuth social login
│   ├── openapi/                  # API documentation
│   ├── organizations/            # Multi-tenancy
│   └── sms/                      # SMS OTP verification
```

---

## 5. Core Auth Package

### 5.1 Core Schema (Immutable)

The core schema defines **required tables** that never change within a major version.

**Core Tables:**

```sql
-- Core Table 1: user
CREATE TABLE "user" (
    id TEXT PRIMARY KEY,
    avatar TEXT,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    disabled INTEGER NOT NULL DEFAULT 0
);

-- Core Table 2: accounts
CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,           -- 'password', 'google', 'github', etc.
    provider_account_id TEXT,         -- OAuth provider user ID
    password_hash TEXT,               -- Argon2id hash (for password provider)
    access_token TEXT,                -- OAuth access token
    refresh_token TEXT,               -- OAuth refresh token
    expires_at TEXT,                  -- Token expiry
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(provider, provider_account_id)
);

-- Core Table 3: verification
CREATE TABLE verification (
    id TEXT PRIMARY KEY,
    identifier TEXT NOT NULL,         -- Email or phone number
    token TEXT NOT NULL UNIQUE,       -- Verification code or token
    type TEXT NOT NULL,               -- 'email', 'reset', 'otp', etc.
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- Core Table 4: session
CREATE TABLE session (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,       -- Session token
    refresh_token TEXT UNIQUE,        -- Refresh token (optional)
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT
);
```

> **Rule:** Any breaking change to core schema requires a major version bump.
>
> **ID Strategy:** Uses TEXT PRIMARY KEY for flexibility (ULID default, UUID, or custom IDs)

---

### 5.2 Core Models

Core Go models mirror only the core schema:

```go
// User represents an authenticated user identity
type User struct {
    ID            string
    Name          string
    Email         string
    EmailVerified bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// Account represents an authentication method for a user
type Account struct {
    ID                string
    UserID            string
    Provider          string  // 'password', 'google', 'github', etc.
    ProviderAccountID string  // OAuth provider user ID
    PasswordHash      string  // Argon2id hash (for password provider)
    CreatedAt         time.Time
    UpdatedAt         time.Time
}

// Session represents an active user session
type Session struct {
    ID           string
    UserID       string
    Token        string
    RefreshToken string
    ExpiresAt    time.Time
    CreatedAt    time.Time
}

// Verification represents an email/phone verification code
type Verification struct {
    ID         string
    Identifier string  // Email or phone number
    Code       string  // Verification code
    Purpose    string  // 'email_verification', 'password_reset', etc.
    ExpiresAt  time.Time
    CreatedAt  time.Time
}
```

These models are **never extended directly by plugins**. Plugins use composition and separate tables.

---

### 5.3 Store Interfaces

Core logic depends exclusively on interfaces:

```go
// UserStore manages user persistence
type UserStore interface {
    Create(ctx context.Context, user User) (User, error)
    GetByEmail(ctx context.Context, email string) (User, error)
    GetByID(ctx context.Context, id string) (User, error)
    Update(ctx context.Context, user User) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, limit, offset int) ([]User, error)
    Count(ctx context.Context) (int64, error)
}
// ... (other interfaces)
```

This decouples auth logic from any specific database or ORM.

---

## 6. sqlc as an Implementation Detail

The core package provides a **reference implementation** using sqlc.

- sqlc runs only inside the package that owns the schema
- Generated code is internal
- Users do not regenerate core sqlc output

**Benefits:**

- Strong typing
- High performance
- Compile-time query validation

---

## 7. Plugin System

Plugins are first-class, isolated extensions.

### 7.1 Plugin Capabilities & Rules

Plugins **can**:

- Add tables
- Add additive columns
- Define joins against core tables
- Define plugin-specific models

Plugins **cannot**:

- Modify core Go types
- Remove or rename core columns
- Change core semantics

### 7.2 Plugin Schema Ownership

Each plugin owns its schema completely and independently.

**Example: Organizations Plugin Schema**

```sql
-- Organization table
CREATE TABLE organization (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    disabled INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Member (memberships)
CREATE TABLE members (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(user_id, organization_id)
);
```

### 7.3 Plugin Models & Types

Plugin models are unique to the plugin.

**Organizations Plugin Models:**

```go
type Organization struct {
    ID        string
    Name      string
    Slug      string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## 8. Handling Schema Extensions & Modifications

### Preferred: Separate Tables

- No conflicts
- Clear ownership
- Safe evolution

### Allowed: Additive Columns

- Only `ADD COLUMN`
- Core code ignores the column
- Plugin owns all access

### Projection Pattern

Plugins define composite views via joins, producing plugin-owned types.

---

## 9. Store Composition & Dependency Flow

**Framework Initialization:**

```go
package main

import (
    "context"
    "database/sql"
    
    "github.com/go-chi/chi/v5"
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
    "github.com/theinventorylib/aegis/plugins/oauth"
    "github.com/theinventorylib/aegis/plugins/jwt"
    "github.com/theinventorylib/aegis/plugins/organizations"
)

func main() {
    // 1. Setup database
    db, _ := sql.Open("postgres", "postgres://localhost/myapp?sslmode=disable")
    
    // 2. Setup router
    r := chi.NewRouter()
    
    // 3. Create Aegis instance
    cfg := config.Default().
        WithDB(db).
        WithRouter(r).
        WithSecret([]byte("your-32-byte-secret-key-here!!!!")).
        WithRedis("localhost", 6379, "", 0).
        WithRateLimiting()
    
    a, _ := aegis.New(context.Background(), cfg)
    
    // 4. Register plugins (priority-based initialization)
    orgPlugin := organizations.New(nil, plugins.DialectPostgres)
    a.UseWithPriority(ctx, orgPlugin, 10)  // Lower priority = earlier init
    
    oauthPlugin := oauth.New(oauthConfig, nil, plugins.DialectPostgres)
    a.Use(ctx, oauthPlugin)  // Default priority = 100
    
    jwtPlugin := jwt.New(jwtConfig, nil, plugins.DialectPostgres)
    a.Use(ctx, jwtPlugin)
    
    // 5. Mount all routes
    a.MountRoutes("/auth")
    
    // 6. Start server
    http.ListenAndServe(":8080", r)
}
```

**Dependency Flow:**

- Core exposes `*sql.DB` for advanced plugin usage
- Plugins depend on core, never the reverse
- Plugin initialization validates schema requirements
- Plugins access core services via `Aegis` interface

---

## 10. Migrations Strategy

| Layer  | Owns Migrations     |
|--------|---------------------|
| Core   | Auth package        |
| Plugin | Plugin package      |
| App    | Executes migrations |

The application controls migration order and tooling.

---

## 11. Dialect Support

- sqlc per dialect (Postgres, MySQL, etc.)
- Build-time selection
- No runtime SQL translation

---

## 12. CLI & Tooling

**Aegis CLI** (`cmd/aegis`) provides migration export tooling:

```bash
# Install CLI
go install github.com/theinventorylib/aegis/cmd/aegis@latest

# Export migrations for all plugins
aegis export --all --format goose --output ./migrations
```

**Features:**

- Exports core auth schema
- Collects plugin migrations automatically
- Generates README documentation
- Validates schema requirements
- Multi-dialect support (PostgreSQL, MySQL)

---

## 13. Why This Architecture Works

- Aligns with Go's type system
- Avoids runtime schema mutation
- Keeps sqlc effective
- Scales to many plugins
- Keeps ownership and responsibilities clear
