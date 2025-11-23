# ID Generation Strategy Configuration

Aegis supports two ID generation strategies that you can configure based on your deployment architecture:

## Strategies

### 1. Sequential (Default) ✅
```go
// Sequence is the default - no configuration needed
// IDs will be: "1", "2", "3", "4"...
```

**Generates**: `1`, `2`, `3`, `4`...

**Best for:**
- Single-instance applications
- Simpler debugging (predictable IDs)
- Smaller storage requirements
- Human-readable IDs

**Database Support:**
- PostgreSQL: Use `BIGSERIAL` type with `DEFAULT` in schema
- MySQL: Use `BIGINT AUTO_INCREMENT` in schema

### 2. UUID (Opt-in for Distributed Systems)
```go
core.SetIDStrategy(core.IDStrategyUUID)
```

**Generates**: `550e8400-e29b-41d4-a716-446655440000`

**Best for:**
- Distributed systems
- Microservices architecture
- Multi-instance deployments
- Avoiding ID collisions across databases
- Privacy (non-sequential, harder to guess)

**Database Support:**
- PostgreSQL: Native `UUID` type
- MySQL: Store as `CHAR(36)` or `BINARY(16)`

## Configuration

### Default Behavior (Sequence)

By default, Aegis uses **sequential IDs** - no configuration needed!

```go
// Default - generates "1", "2", "3"...
id := core.GenerateID()
```

### Custom ID Generation (e.g., UUID)

To use a custom ID generation strategy (like UUIDs), use the `config.WithIDGenerator` option when initializing Aegis:

```go
package main

import (
    "github.com/google/uuid"
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
)

func main() {
    // Initialize Aegis with a custom ID generator
    auth, _ := aegis.New(
        config.WithIDGenerator(func() string {
            return uuid.NewString()
        }),
        // ... other options
    )
}
```

### Force Specific ID Type

If you need to generate a specific type of ID manually within your application code:

```go
// Always get UUID
id := core.GenerateUUID()

// Always get sequence ID
id := core.GenerateSequenceID()
```

## Database Schema Considerations

### For UUID Strategy

**PostgreSQL:**
```sql
CREATE TABLE auth.user (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- or use TEXT if you want flexibility
    -- id TEXT PRIMARY KEY,
    ...
);
```

**MySQL:**
```sql
CREATE TABLE auth.user (
    id CHAR(36) PRIMARY KEY,
    -- or for better performance:
    -- id BINARY(16) PRIMARY KEY,
    ...
);
```

### For Sequential Strategy

**PostgreSQL:**
```sql
CREATE TABLE auth.user (
    id BIGSERIAL PRIMARY KEY,
    -- Will auto-generate 1, 2, 3...
    ...
);
```

**MySQL:**
```sql
CREATE TABLE auth.user (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    ...
);
```

## Current Schema

The current Aegis core schema uses `TEXT` for maximum flexibility:
- Works with both UUID and sequential strategies
- Application generates IDs before INSERT
- Trade-off: Slightly larger storage vs flexibility

## Recommendation

**Use UUID (default)** unless you have a specific reason to use sequential IDs. UUIDs are:
- Safer for distributed systems
- No collision risk when merging databases
- More secure (can't guess next ID)
- Industry standard for modern applications
