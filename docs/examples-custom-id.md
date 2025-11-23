# Custom ID Generator Examples

## Example 1: Using ULID

```go
package main

import (
    "github.com/oklog/ulid/v2"
    "github.com/theinventorylib/aegis"
    "github.com/theinventorylib/aegis/config"
    "github.com/theinventorylib/aegis/core"
    "math/rand"
    "time"
)

func main() {
    // Create a ULID entropy source
    entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)
    
    // Initialize Aegis with custom ULID generator
    auth, _ := aegis.New(
        config.WithPostgres(database),
        config.WithRouter(router),
        config.WithJWTSecret(secret),
        config.WithIDGenerator(func() string {
            return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
        }),
    )
}
```

## Example 2: Using nanoid

```go
import (
    gonanoid "github.com/matoous/go-nanoid/v2"
)

auth, _ := aegis.New(
    config.WithIDGenerator(func() string {
        id, _ := gonanoid.New()
        return id
    }),
)
```

## Example 3: Prefixed Sequential IDs

```go
var counter uint64

auth, _ := aegis.New(
    config.WithIDGenerator(func() string {
        atomic.AddUint64(&counter, 1)
        return fmt.Sprintf("user_%08d", counter)
    }),
)
```

## Example 4: Timestamp-based IDs

```go
auth, _ := aegis.New(
    config.WithIDGenerator(func() string {
        return fmt.Sprintf("%d_%s", 
            time.Now().UnixNano(),
            randomString(8),
        )
    }),
)
```

## Direct API (Without Config)

```go
// Set custom generator directly
core.SetCustomIDGenerator(func() string {
    return ulid.Make().String()
})

// Now all GenerateID() calls use your function
id := core.GenerateID()
```
