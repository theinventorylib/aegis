# ID Generation Example

This example demonstrates the different ID generation strategies available in Aegis.

## Overview

Aegis supports multiple ID generation strategies:
- **ULID** (default) - Sortable, time-based, restart-safe
- **UUID** - Standard UUID v4 format
- **Sequential** - Simple counter (testing only)
- **Custom** - Your own ID generation function

## Running the Example

```bash
cd examples/id-generation
go run main.go
```

## Expected Output

```
=== Aegis ID Generation Examples ===
1. ULID (Default Strategy)
   Benefits: Sortable, time-based, restart-safe
   ID 1: 01ARZ3NDEKTSV4RRFFQ69G5FAV
   ID 2: 01ARZ3NDEKTSV4RRFFQ69G5FAW
   ID 3: 01ARZ3NDEKTSV4RRFFQ69G5FAX
   ID 4: 01ARZ3NDEKTSV4RRFFQ69G5FAY
   ID 5: 01ARZ3NDEKTSV4RRFFQ69G5FAZ

2. UUID Strategy
   Benefits: Standard format, widely compatible
   ID 1: 550e8400-e29b-41d4-a716-446655440000
   ID 2: 6ba7b810-9dad-11d1-80b4-00c04fd430c8
   ID 3: 6ba7b811-9dad-11d1-80b4-00c04fd430c8

4. Custom Strategy
   Benefits: Use any ID library you prefer
   ID 1: user_a3f5b2c1
   ID 2: user_d7e9f4a2
   ID 3: user_b8c6d3e5

=== Summary ===
✅ ULID is the default (restart-safe, sortable)
✅ UUID available for compatibility
⚠️  Sequence only for testing
✅ Custom generator support

See docs/id-generation.md for complete guide
```

## Strategy Comparison

| Strategy | Sortable | Restart Safe | Size | Use Case |
|----------|----------|--------------|------|----------|
| ULID (default) | ✅ | ✅ | 26 chars | Production (recommended) |
| UUID | ❌ | ✅ | 36 chars | Standard UUID compatibility |
| Sequential | ✅ | ❌ | Variable | Testing only |
| Custom | Depends | Depends | Variable | Special requirements |

## Code Highlights

### ULID (Default)

```go
import "github.com/theinventorylib/aegis/core"

// No configuration needed - ULID is default
id := core.GenerateID()
// Returns: "01ARZ3NDEKTSV4RRFFQ69G5FAV"
```

### UUID

```go
import "github.com/theinventorylib/aegis/core"

core.SetIDStrategy(core.IDStrategyUUID)
id := core.GenerateID()
// Returns: "550e8400-e29b-41d4-a716-446655440000"
```

### Custom Generator

```go
import (
    "fmt"
    "github.com/google/uuid"
    "github.com/theinventorylib/aegis/core"
)

core.SetCustomIDGenerator(func() string {
    return fmt.Sprintf("user_%s", uuid.New().String()[:8])
})

id := core.GenerateID()
// Returns: "user_a3f5b2c1"
```

## Using in Your Application

### Via Configuration

```go
import (
    "github.com/google/uuid"
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
)

auth, _ := aegis.New(
    config.WithIDGenerator(func() string {
        return uuid.New().String()
    }),
    // ... other config
)
```

### Via Direct API

```go
import "github.com/theinventorylib/aegis/core"

// Set before initializing Aegis
core.SetIDStrategy(core.IDStrategyULID)  // or UUID
// or
core.SetCustomIDGenerator(yourFunction)
```

## Best Practices

1. **Use ULID (default)** - Restart-safe and sortable
2. **Avoid Sequential** - Only for testing, not production
3. **Set strategy early** - Before calling `aegis.New()`
4. **Be consistent** - Don't change strategies mid-deployment

## Learn More

- [ID Generation Guide](../../docs/id-generation.md) - Complete documentation
- [Getting Started](../../docs/getting-started.md) - Basic Aegis setup
- [Configuration](../../docs/getting-started.md#basic-configuration) - Configuration options
