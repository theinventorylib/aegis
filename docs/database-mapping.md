# Database Field Mapping

## Overview

Aegis uses manual SQL queries without an ORM, which allows for explicit control over database operations but requires careful attention to field name mappings between Go models and SQL schemas.

## Field Naming Conventions

### General Pattern
- **Go Models**: Use `PascalCase` for struct fields
- **SQL Columns**: Use `snake_case` for column names
- **JSON Tags**: Use `camelCase` for API serialization

### Core Models

#### User Model

**Go Model** (`models/user.go`):
```go
type User struct {
    ID        string                 `json:"id"`
    CreatedAt time.Time              `json:"createdAt"`
    UpdatedAt time.Time              `json:"updatedAt"`
    Disabled  bool                   `json:"disabled"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
```

**SQL Schema** (`auth.user`):
```sql
CREATE TABLE auth.user (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    metadata JSONB DEFAULT '{}'
);
```

**Field Mapping**:
| Go Field | SQL Column | Notes |
|----------|------------|-------|
| `ID` | `id` | TEXT primary key, app-generated (ULID by default) |
| `CreatedAt` | `created_at` | Automatic timestamp |
| `UpdatedAt` | `updated_at` | Updated via trigger |
| `Disabled` | `disabled` | Account status flag |
| `Metadata` | `metadata` | JSONB for extensibility |

---

#### Account Model

**Go Model** (`models/account.go`):
```go
type Account struct {
    ID                string    `json:"id"`
    UserID            string    `json:"userId"`
    Provider          string    `json:"provider"`
    ProviderAccountID *string   `json:"providerAccountId,omitempty"`
    Password          *string   `json:"-"` // Never expose in JSON
    // ...
}
```

**SQL Schema** (`auth.accounts`):
```sql
CREATE TABLE auth.accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_account_id TEXT,
    password_hash TEXT,  -- NOTE: Different name from Go field
    -- ...
);
```

**Field Mapping**:
| Go Field | SQL Column | Notes |
|----------|------------|-------|
| `Password` | `password_hash` | **Name differs** - Go uses `Password`, SQL uses `password_hash` |
| `AccessToken` | `access_token` | OAuth access token (not exposed in JSON) |
| `RefreshToken` | `refresh_token` | OAuth refresh token (not exposed in JSON) |

> **Important**: The `Password` field in Go maps to `password_hash` in SQL. This is intentional:
> - SQL name is more explicit about storing a hash, not plain text
> - Go field is simpler for code readability
> - Manual queries must use the correct SQL column name

---

#### Session Model

**Go Model** (`models/models.go`):
```go
type Session struct {
    ID           string    `json:"id"`
    UserID       string    `json:"userId"`
    Token        string    `json:"token"`
    RefreshToken string    `json:"refreshToken,omitempty"`
    ExpiresAt    time.Time `json:"expiresAt"`
    CreatedAt    time.Time `json:"createdAt"`
    IPAddress    string    `json:"ipAddress,omitempty"`
    UserAgent    string    `json:"userAgent,omitempty"`
    Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
```

**SQL Schema** (`auth.session`):
```sql
CREATE TABLE auth.session (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    metadata JSONB DEFAULT '{}'
);
```

**Field Mapping**: All fields follow standard snake_case conversion.

---

## Query Examples

### Correct Field Mapping

```go
// CORRECT: Using password_hash in SQL
row := db.QueryRow(`
    SELECT id, user_id, provider, password_hash
    FROM auth.accounts
    WHERE user_id = $1 AND provider = 'password'
`, userID)

var account models.Account
err := row.Scan(&account.ID, &account.UserID, &account.Provider, &account.Password)
```

### Incorrect Field Mapping

```go
// INCORRECT: Using 'password' instead of 'password_hash'
row := db.QueryRow(`
    SELECT id, user_id, provider, password
    FROM auth.accounts
    WHERE user_id = $1
`, userID) // This will fail - column 'password' does not exist
```

---

## Plugin Tables

Plugins follow the same naming conventions:

### Email Plugin (`plugins_email.verifications`)
- Go: `Code`, `Token`, `Verified`
- SQL: `code`, `token`, `verified`

### SMS Plugin (`plugins_sms.verifications`)
- Go: `PhoneNumber`, `Code`, `ExpiresAt`
- SQL: `phone_number`, `code`, `expires_at`

### OAuth Plugin (`plugins_oauth.connections`)
- Go: `ProviderUserID`, `AccessToken`, `RefreshToken`
- SQL: `provider_user_id`, `access_token`, `refresh_token`

---

## Best Practices

1. **Always use SQL column names in queries**
   - Wrong: `SELECT Password FROM accounts`
   - Right: `SELECT password_hash FROM accounts`

2. **Use snake_case for all SQL identifiers**
   - Tables: `auth.user`, `auth.accounts`
   - Columns: `user_id`, `created_at`, `password_hash`

3. **Document field name differences**
   - If Go and SQL names differ significantly, add comments
   - Example: `Password` → `password_hash`

4. **Never expose sensitive fields in JSON**
   - Use `json:"-"` tag for passwords, tokens, secrets
   - Example: ``Password *string `json:"-"` ``

5. **Use JSONB metadata for extensibility**
   - Avoid adding many nullable columns
   - Store plugin-specific data in `metadata` JSONB field
   - Allows schema evolution without migrations

---

## Common Pitfalls

### ❌ Wrong: Assuming Go field names match SQL
```go
db.QueryRow("SELECT ID, CreatedAt FROM auth.user WHERE ID = $1", id)
// SQL is case-sensitive for quoted identifiers
```

### ✅ Correct: Use snake_case SQL column names
```go
db.QueryRow("SELECT id, created_at FROM auth.user WHERE id = $1", id)
```

### ❌ Wrong: Exposing password hashes in JSON
```go
type Account struct {
    Password string `json:"password"` // NEVER DO THIS
}
```

### ✅ Correct: Hide sensitive data from JSON
```go
type Account struct {
    Password *string `json:"-"` // Hidden from JSON serialization
}
```

---

## Migration Considerations

When adding new fields:

1. **Add to SQL schema first**
   ```sql
   ALTER TABLE auth.user ADD COLUMN last_login TIMESTAMP;
   ```

2. **Add to Go model**
   ```go
   type User struct {
       // ...
       LastLogin *time.Time `json:"lastLogin,omitempty"`
   }
   ```

3. **Update all queries**
   - Add to SELECT statements
   - Add to INSERT statements
   - Add to UPDATE statements

4. **Use nullable types for optional fields**
   - `*string` instead of `string`
   - `*time.Time` instead of `time.Time`
   - Allows distinction between "not set" and "set to default value"

---

## Summary

- **No ORM**: All queries use manual SQL
- **Naming**: Go uses PascalCase, SQL uses snake_case
- **Mapping**: Most fields auto-convert, except special cases like `Password` → `password_hash`
- **Extensibility**: Use JSONB `metadata` columns for plugin-specific data
- **Security**: Never expose passwords, tokens, or secrets in JSON tags
