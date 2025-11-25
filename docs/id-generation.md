# ID Generation Guide

## Overview

Aegis uses **ULID (Universally Unique Lexicographically Sortable Identifier)** as the default ID generation strategy. This provides several advantages over traditional approaches while giving you the flexibility to use alternative strategies if needed.

## Default Strategy: ULID

### What is ULID?

ULID is a 26-character, URL-safe, base32-encoded identifier that is:
- **Sortable**: IDs are lexicographically sortable by creation time
- **Unique**: Extremely low collision probability (128-bit randomness)
- **Compact**: 26 characters vs UUID's 36 characters
- **Time-based**: First 48 bits encode timestamp (millisecond precision)
- **Safe across restarts**: Unlike sequence-based IDs, ULIDs work correctly even after application restarts

### Example ULID

```
01ARZ3NDEKTSV4RRFFQ69G5FAV
```

### Benefits

✅ **Works after restarts** - No collision risk when your application restarts  
✅ **Distributed systems** - Safe to use across multiple application instances  
✅ **Sortable by time** - Database indices can optimize based on creation order  
✅ **No database round-trip** - Generated in application memory  
✅ **Thread-safe** - Concurrent ID generation is safe  

## Alternative Strategies

You can override the default ULID strategy using the configuration system:

### UUID v4

Standard UUID format if you need compatibility with existing systems:

```go
import (
    "github.com/google/uuid"
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
    "github.com/theinventorylib/aegis/core"
)

// Option 1: Override at the core level
core.SetIDStrategy(core.IDStrategyUUID)

// Option 2: Set via config
aegis.New(
    config.WithIDGenerator(func() string {
        return uuid.New().String()
    }),
    // ... other config
)
```

### Sequential IDs (Testing Only)

⚠️ **WARNING**: Sequential IDs reset to 1 on application restart and should **ONLY** be used for testing.

```go
// For testing environments only
core.SetIDStrategy(core.IDStrategySequence)
```

**Never use sequential IDs in production** - they will cause primary key collisions after restart.

### Custom ID Generator

Use your own ID generation library (KSUID, nanoid, etc.):

```go
import "github.com/segmentio/ksuid"

aegis.New(
    config.WithIDGenerator(func() string {
        return ksuid.New().String()
    }),
    // ... other config
)
```

## Direct Strategy Control

For advanced use cases, you can control the ID strategy directly:

```go
import "github.com/theinventorylib/aegis/core"

// Set strategy before initializing Aegis
core.SetIDStrategy(core.IDStrategyULID)    // Default
core.SetIDStrategy(core.IDStrategyUUID)     // UUID v4
core.SetIDStrategy(core.IDStrategySequence) // Sequential (testing only)

// Or use custom generator
core.SetCustomIDGenerator(func() string {
    return "my-custom-id-" + someLib.Generate()
})
```

## Database-Generated IDs

If you prefer database-generated IDs (SERIAL, AUTO_INCREMENT), modify your schema:

### PostgreSQL

```sql
CREATE TABLE auth.user (
    id BIGSERIAL PRIMARY KEY,  -- Database generates IDs
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'
);
```

### MySQL

```sql
CREATE TABLE user (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,  -- Database generates IDs
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    metadata JSON
);
```

**Important**: If using database-generated IDs:
1. Modify your schema to use `SERIAL`/`AUTO_INCREMENT`
2. Remove calls to `core.GenerateID()` in your code
3. Let the database return the generated ID after INSERT

## Migration from Sequential to ULID

If you're upgrading from an older version that used sequential IDs:

### Option 1: Fresh Installation

For new deployments, ULID is already the default - no action needed.

### Option 2: Existing Data

If you have existing data with sequential IDs:

1. **Keep existing IDs**: ULIDs can coexist with older numeric IDs in TEXT columns
2. **New IDs only**: Future records will use ULIDs
3. **No migration needed**: The TEXT column supports both formats

### Option 3: Continue Using Sequence (Not Recommended)

If you must continue using sequential IDs:

```go
// At application startup, before calling aegis.New()
import "github.com/theinventorylib/aegis/core"

core.SetIDStrategy(core.IDStrategySequence)
```

⚠️ **This will cause ID collisions on restart in production!**

## Performance Characteristics

| Strategy | Generation Speed | Collision Risk | Sortable | Restart Safe | Size |
|----------|-----------------|----------------|----------|--------------|------|
| ULID     | ~1M/sec         | Extremely Low  | ✅ Yes   | ✅ Yes       | 26 chars |
| UUID     | ~500K/sec       | Extremely Low  | ❌ No    | ✅ Yes       | 36 chars |
| Sequence | ~10M/sec        | ⚠️ High on restart | ✅ Yes | ❌ No      | Variable |

## Best Practices

1. **Use ULID (default)** for most applications - it's safe, fast, and sortable
2. **Use UUID** if you need standard UUID format for compatibility
3. **Never use Sequence** in production - only for testing
4. **Use Custom** for special requirements (KSUID, nanoid, etc.)
5. **Use Database IDs** if you have strong preferences for database-managed sequences

## Frequently Asked Questions

### Q: Why did Aegis change from Sequence to ULID?

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

## Example: Complete Setup

```go
package main

import (
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
    "github.com/theinventorylib/aegis/server"
)

func main() {
    // ULID is automatically the default - no configuration needed!
    
    a, err := aegis.New(
        config.WithPostgres("postgres://user:pass@localhost/db"),
        config.WithRouter(server.NewDefaultRouter(http.NewServeMux())),
        config.WithCSRFSecret([]byte("your-secret-key-here")),
    )
    if err != nil {
        panic(err)
    }
    
    // IDs will be generated as ULIDs like: "01ARZ3NDEKTSV4RRFFQ69G5FAV"
    // No additional configuration required!
}
```

## Further Reading

- [ULID Specification](https://github.com/ulid/spec)
- [UUID RFC 4122](https://tools.ietf.org/html/rfc4122)
- [Aegis Configuration Guide](./configuration.md)
