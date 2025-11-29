# ID Generation Guide

Complete guide to ID generation strategies in Aegis, from the default ULID approach to custom generators.

## Overview

Aegis uses **ULID (Universally Unique Lexicographically Sortable Identifier)** as the default ID generation strategy. This provides restart-safe, sortable, unique identifiers without requiring database round-trips.

---

## Default Strategy: ULID

### What is ULID?

ULID is a 26-character, URL-safe, base32-encoded identifier:

```
01ARZ3NDEKTSV4RRFFQ69G5FAV
└─┬─┘└────────┬────────────┘
  │           │
  │           └─ 80 bits of randomness
  └─ 48-bit timestamp (millisecond precision)
```

### Benefits

✅ **Restart-safe** - No collision risk when application restarts  
✅ **Distributed-safe** - Works across multiple application instances  
✅ **Sortable by time** - Database indices optimize based on creation order  
✅ **No database round-trip** - Generated in application memory  
✅ **Thread-safe** - Concurrent ID generation is safe  
✅ **Compact** - 26 characters vs UUID's 36 characters

### Example Usage

```go
import "github.com/theinventorylib/aegis/core"

// ULID is the default - no configuration needed!
id := core.GenerateID()
// Returns: "01ARZ3NDEKTSV4RRFFQ69G5FAV"
```

---

## Built-in Strategies

### UUID v4

Standard UUID format for compatibility with existing systems:

```go
import (
    "github.com/google/uuid"
    "github.com/theinventorylib/aegis/core"
)

// Option 1: Set strategy globally
core.SetIDStrategy(core.IDStrategyUUID)
id := core.GenerateID()
// Returns: "550e8400-e29b-41d4-a716-446655440000"

// Option 2: Via configuration
import "github.com/theinventorylib/aegis/config"

aegis.New(
    config.WithIDGenerator(func() string {
        return uuid.New().String()
    }),
    // ... other config
)
```

**Use when:**
- You need standard UUID format
- Integrating with existing UUID-based systems
- Compatibility is more important than sortability

### Sequential IDs (Testing Only)

⚠️ **WARNING**: Sequential IDs reset to 1 on application restart and should **ONLY** be used for testing.

```go
import "github.com/theinventorylib/aegis/core"

// For testing environments only
core.SetIDStrategy(core.IDStrategySequence)

id1 := core.GenerateID() // Returns: "1"
id2 := core.GenerateID() // Returns: "2"
id3 := core.GenerateID() // Returns: "3"
```

**Never use sequential IDs in production** - they will cause primary key collisions after restart.

---

## Custom ID Generators

Use your own ID generation library for special requirements.

### Using KSUID

```go
import (
    "github.com/segmentio/ksuid"
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
)

func main() {
    auth, _ := aegis.New(
        config.WithIDGenerator(func() string {
            return ksuid.New().String()
        }),
        // ... other config
    )
}
```

### Using nanoid

```go
import (
    gonanoid "github.com/matoous/go-nanoid/v2"
    "github.com/theinventorylib/aegis/config"
)

aegis.New(
    config.WithIDGenerator(func() string {
        id, _ := gonanoid.New()
        return id
    }),
    // ... other config
)
```

### Prefixed Sequential IDs

```go
import (
    "fmt"
    "sync/atomic"
    "github.com/theinventorylib/aegis/config"
)

var counter uint64

aegis.New(
    config.WithIDGenerator(func() string {
        n := atomic.AddUint64(&counter, 1)
        return fmt.Sprintf("user_%08d", n)
    }),
    // ... other config
)
```

### Timestamp-based IDs

```go
import (
    "fmt"
    "math/rand"
    "time"
)

func randomString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}

aegis.New(
    config.WithIDGenerator(func() string {
        return fmt.Sprintf("%d_%s", 
            time.Now().UnixNano(),
            randomString(8),
        )
    }),
    // ... other config
)
```

### Direct API Usage

Set a custom generator without using config:

```go
import (
    "github.com/oklog/ulid/v2"
    "github.com/theinventorylib/aegis/core"
)

// Set custom generator directly
core.SetCustomIDGenerator(func() string {
    return ulid.Make().String()
})

// Now all GenerateID() calls use your function
id := core.GenerateID()
```

---

## Database-Generated IDs

If you prefer database-generated IDs, modify your schema:

### PostgreSQL

```sql
CREATE TABLE auth.user (
    id BIGSERIAL PRIMARY KEY,  -- Database generates IDs
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login TIMESTAMP
);
```

### MySQL

```sql
CREATE TABLE user (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,  -- Database generates IDs
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_login TIMESTAMP
);
```

**Important**: If using database-generated IDs:
1. Modify your schema to use `SERIAL`/`AUTO_INCREMENT`
2. Remove calls to `core.GenerateID()` in your code
3. Let the database return the generated ID after INSERT

---

## Migration Guide

### From Sequential to ULID

If you're upgrading from an older version that used sequential IDs:

**Option 1: Fresh Installation**

For new deployments, ULID is already the default - no action needed.

**Option 2: Existing Data**

If you have existing data with sequential IDs:

1. **Keep existing IDs**: ULIDs can coexist with older numeric IDs in TEXT columns
2. **New IDs only**: Future records will use ULIDs
3. **No migration needed**: The TEXT column supports both formats

**Example:**
```sql
-- Existing data
id: "1", "2", "3", ...

-- New data (after upgrade)
id: "01ARZ3NDEKTSV4RRFFQ69G5FAV", "01ARZ3NDEKTSV4RRFFQ69G5FAW", ...
```

**Option 3: Continue Using Sequential (Not Recommended)**

If you must continue using sequential IDs:

```go
import "github.com/theinventorylib/aegis/core"

// At application startup, before calling aegis.New()
core.SetIDStrategy(core.IDStrategySequence)
```

⚠️ **This will cause ID collisions on restart in production!**

---

## Performance Characteristics

| Strategy | Generation Speed | Collision Risk | Sortable | Restart Safe | Size |
|----------|-----------------|----------------|----------|--------------|------|
| ULID     | ~1M/sec         | Extremely Low  | ✅ Yes   | ✅ Yes       | 26 chars |
| UUID     | ~500K/sec       | Extremely Low  | ❌ No    | ✅ Yes       | 36 chars |
| Sequential | ~10M/sec      | ⚠️ High on restart | ✅ Yes | ❌ No      | Variable |
| KSUID    | ~800K/sec       | Extremely Low  | ✅ Yes   | ✅ Yes       | 27 chars |
| nanoid   | ~2M/sec         | Extremely Low  | ❌ No    | ✅ Yes       | 21 chars |

---

## Best Practices

1. **Use ULID (default)** for most applications - it's safe, fast, and sortable
2. **Use UUID** if you need standard UUID format for compatibility
3. **Never use Sequential** in production - only for testing
4. **Use Custom** for special requirements (KSUID, nanoid, etc.)
5. **Use Database IDs** if you have strong preferences for database-managed sequences

---

## Frequently Asked Questions

### Q: Why did Aegis change from Sequential to ULID?

**A:** Sequential IDs reset on application restart, causing primary key collisions in production. ULID is safer and still provides sortability benefits.

### Q: Can I use numeric IDs instead of strings?

**A:** Yes, modify your database schema to use `BIGSERIAL` (PostgreSQL) or `BIGINT AUTO_INCREMENT` (MySQL) and remove calls to `core.GenerateID()`.

### Q: Will ULID work with my existing database?

**A:** Yes! ULID generates strings that work with TEXT/VARCHAR columns. Your existing schema using `id TEXT PRIMARY KEY` will work without changes.

### Q: What if I need shorter IDs?

**A:** Consider these alternatives:
- **nanoid**: 21 characters, customizable
- **KSUID**: 27 characters, similar to ULID
- **Custom base62**: Encode ULIDs in base62 for shorter representation

### Q: Are ULIDs safe for distributed systems?

**A:** Yes! ULIDs use cryptographic randomness and include timestamps, making collisions extremely unlikely even across multiple servers.

### Q: Can I change ID strategy after deployment?

**A:** Yes, but new IDs will use the new strategy while old IDs remain unchanged. Both can coexist in TEXT columns.

---

## Complete Example

```go
package main

import (
    "database/sql"
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
    // ULID is automatically the default - no configuration needed!
    
    sqlDB, _ := sql.Open("postgres", "postgres://user:pass@localhost/db")
    mux := http.NewServeMux()
    
    auth, err := aegis.New(
        config.WithDB(sqlDB, db.PostgreSQL),
        config.WithRouter(server.NewDefaultRouter(mux)),
        // Provide secrets via environment in production
        config.WithJWTSecret([]byte(os.Getenv("JWT_SECRET"))),
        // No WithIDGenerator needed - ULID is default!
    )
    if err != nil {
        log.Fatal(err)
    }
    
    auth.MountRoutes("/auth")
    
    // IDs will be generated as ULIDs like: "01ARZ3NDEKTSV4RRFFQ69G5FAV"
    log.Println("Server starting with ULID ID generation")
    http.ListenAndServe(":8080", mux)
}
```

---

## Further Reading

- [ULID Specification](https://github.com/ulid/spec)
- [UUID RFC 4122](https://tools.ietf.org/html/rfc4122)
- [KSUID](https://github.com/segmentio/ksuid)
- [nanoid](https://github.com/ai/nanoid)
- [Getting Started](./getting-started.md) - Basic Aegis setup
