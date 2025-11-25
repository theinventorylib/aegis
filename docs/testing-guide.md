# Testing Guide for Aegis

This guide covers testing strategies for the Aegis framework and plugins.

## Table of Contents

1. [Testing Framework](#testing-framework)
2. [Writing Tests for Plugins](#writing-tests-for-plugins)
3. [Integration Tests](#integration-tests)
4. [Surface Tests](#surface-tests)
5. [Running Tests](#running-tests)

---

## Testing Framework

Aegis provides a comprehensive testing framework in the `testing` package to make it easy to test core functionality and plugins.

### Test Helpers

#### `Setup(t *testing.T, opts ...config.Option) *TestAegis`

Creates a fully configured test Aegis instance with mock database and router.

```go
func TestMyFeature(t *testing.T) {
    testAegis := testing.Setup(t)
    defer testAegis.Cleanup()
    
    // Use testAegis for testing
}
```

#### `TestAegis` Utilities

```go
type TestAegis struct {
    *aegis.Aegis
    DB     *core.MockDB
    Router *TestRouter
    Config *config.Config
}

// Create test user
user := testAegis.CreateTestUser(t, "user-123")

// Create test session
session := testAegis.CreateTestSession(t, "user-123")

// Make HTTP requests
rec := testAegis.Request(t, "GET", "/auth/user", "")
rec := testAegis.AuthenticatedRequest(t, "GET", "/api/data", session.Token)
```

---

## Writing Tests for Plugins

### Using PluginTestSuite

The `PluginTestSuite` provides standard tests for all plugins:

```go
package myplugin_test

import (
    "testing"
    "github.com/theinventorylib/aegis/plugins/myplugin"
    testinghelpers "github.com/theinventorylib/aegis/testing"
)

func TestMyPlugin(t *testing.T) {
    plugin := myplugin.New(&myplugin.Config{})
    suite := testinghelpers.NewPluginTestSuite(t, plugin)
    defer suite.Cleanup()
    
    // Run all standard tests
    suite.RunAllTests()
}
```

### Standard Plugin Tests

The test suite automatically tests:
- ✅ **Metadata**: Name, Version, Description
- ✅ **Initialization**: Init method succeeds
- ✅ **Route Mounting**: MountRoutes doesn't panic
- ✅ **Migrations**: GetMigrations returns valid data

### Custom Plugin Tests

Add plugin-specific tests:

```go
func TestMyPluginCustomLogic(t *testing.T) {
    testAegis := testinghelpers.Setup(t)
    defer testAegis.Cleanup()
    
    plugin := myplugin.New(&myplugin.Config{
        DB: testAegis.DB,
    })
    
    ctx := context.Background()
    err := testAegis.Use(ctx, plugin)
    if err != nil {
        t.Fatal(err)
    }
    
    // Test plugin-specific functionality
    t.Run("CustomFeature", func(t *testing.T) {
        // Your custom tests here
    })
}
```

---

## Integration Tests

Integration tests verify that components work together correctly.

### Example: Complete Auth Flow

```go
func TestCompleteAuthFlow(t *testing.T) {
    testAegis := testinghelpers.Setup(t)
    defer testAegis.Cleanup()
    
    // Mount routes
    testAegis.MountRoutes("/auth")
    
    // 1. Create user
    user := testAegis.CreateTestUser(t, "user-123")
    
    // 2. Create session
    session := testAegis.CreateTestSession(t, user.ID)
    
    // 3. Make authenticated request
    rec := testAegis.AuthenticatedRequest(t, "GET", "/auth/user", session.Token)
    
    // 4. Verify response
    if rec.Code != http.StatusOK {
        t.Errorf("Expected 200, got %d", rec.Code)
    }
}
```

### Testing Multiple Plugins Together

```go
func TestMultiplePlugins(t *testing.T) {
    testAegis := testinghelpers.Setup(t)
    defer testAegis.Cleanup()
    
    ctx := context.Background()
    
    // Register plugins
    passwordPlugin := password.New(&password.Config{DB: testAegis.DB})
    emailPlugin := email.New(&email.Config{DB: testAegis.DB})
    
    testAegis.Use(ctx, passwordPlugin)
    testAegis.Use(ctx, emailPlugin)
    
    // Verify integration
    testinghelpers.AssertPluginRegistered(t, testAegis.Aegis, "password")
    testinghelpers.AssertPluginRegistered(t, testAegis.Aegis, "email")
}
```

---

## Surface Tests

Surface tests verify high-level workflows without diving into implementation details.

### Example: User Workflow

```go
func TestUserWorkflow(t *testing.T) {
    testAegis := testinghelpers.Setup(t)
    defer testAegis.Cleanup()
    
    t.Run("RegistrationToLogout", func(t *testing.T) {
        // Register
        user := testAegis.CreateTestUser(t, "new-user")
        
        // Login (create session)
        session := testAegis.CreateTestSession(t, user.ID)
        
        // Access protected resource
        rec := testAegis.AuthenticatedRequest(t, "GET", "/api/data", session.Token)
        if rec.Code == http.StatusUnauthorized {
            t.Error("Should be authorized")
        }
        
        // Logout would invalidate session
        // (test the logout logic here)
    })
}
```

### Testing Concurrency

```go
func TestConcurrentSessions(t *testing.T) {
    testAegis := testinghelpers.Setup(t)
    defer testAegis.Cleanup()
    
    const numUsers = 10
    errChan := make(chan error, numUsers)
    
    for i := 0; i < numUsers; i++ {
        go func(id int) {
            user := testAegis.CreateTestUser(t, fmt.Sprintf("user-%d", id))
            session := testAegis.CreateTestSession(t, user.ID)
            
            if session.UserID != user.ID {
                errChan <- fmt.Errorf("session mismatch")
                return
            }
            errChan <- nil
        }(i)
    }
    
    for i := 0; i < numUsers; i++ {
        if err := <-errChan; err != nil {
            t.Error(err)
        }
    }
}
```

---

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run with Race Detector

```bash
go test -race ./...
```

### Run Specific Package

```bash
go test -v ./plugins/password/...
```

### Run Integration Tests Only

```bash
go test -v ./integration_test.go
```

### Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Verbose Output

```bash
go test -v ./...
```

### Run Single Test

```bash
go test -run TestMyPlugin ./plugins/password/...
```

---

## Best Practices

### 1. Use Test Helpers

Always use `testing.Setup()` for consistent test environment:

```go
func TestSomething(t *testing.T) {
    testAegis := testing.Setup(t)
    defer testAegis.Cleanup()  // Important: cleanup resources
    
    // Your test code
}
```

### 2. Test in Parallel When Possible

```go
func TestConcurrent(t *testing.T) {
    for i := 0; i < 5; i++ {
        t.Run(fmt.Sprintf("Instance%d", i), func(t *testing.T) {
            t.Parallel()  // Run in parallel
            
            testAegis := testing.Setup(t)
            defer testAegis.Cleanup()
            
            // Test code
        })
    }
}
```

### 3. Use Subtests for Organization

```go
func TestPlugin(t *testing.T) {
    testAegis := testing.Setup(t)
    defer testAegis.Cleanup()
    
    t.Run("Initialization", func(t *testing.T) {
        // Init tests
    })
    
    t.Run("Routes", func(t *testing.T) {
        // Route tests
    })
    
    t.Run("CustomLogic", func(t *testing.T) {
        // Logic tests
    })
}
```

### 4. Always Test with Race Detector

```bash
go test -race ./...
```

This catches concurrency bugs early.

### 5. Use Assertion Helpers

```go
// Check plugin is registered
testinghelpers.AssertPluginRegistered(t, aegis, "myplugin")

// Check plugin is NOT registered
testinghelpers.AssertPluginNotRegistered(t, aegis, "myplugin")
```

---

## Example: Complete Plugin Test

```go
package myplugin_test

import (
    "context"
    "testing"
    
    "github.com/theinventorylib/aegis/plugins/myplugin"
    testinghelpers "github.com/theinventorylib/aegis/testing"
)

func TestMyPlugin(t *testing.T) {
    // Setup
    testAegis := testinghelpers.Setup(t)
    defer testAegis.Cleanup()
    
    plugin := myplugin.New(&myplugin.Config{
        DB: testAegis.DB,
    })
    
    // Test suite - standard tests
    suite := testinghelpers.NewPluginTestSuite(t, plugin)
    suite.RunAllTests()
    
    // Custom tests
    t.Run("CustomFeature", func(t *testing.T) {
        ctx := context.Background()
        err := testAegis.Use(ctx, plugin)
        if err != nil {
            t.Fatal(err)
        }
        
      // Test your custom plugin logic
        testinghelpers.AssertPluginRegistered(t, testAegis.Aegis, "myplugin")
    })
}
```

---

## Mocking

### MockPlugin

Use `MockPlugin` for testing plugin interactions:

```go
mockPlugin := testinghelpers.NewMockPlugin("test")
mockPlugin.InitFunc = func(ctx context.Context, a aegis.Aegis) error {
    // Custom init logic for testing
    return nil
}

testAegis.Use(context.Background(), mockPlugin)
```

### MockDB

The `MockDB` is automatically provided by `testing.Setup()`:

```go
testAegis := testing.Setup(t)

// Access mock DB
testAegis.DB.Users["user-123"] = &models.User{ID: "user-123"}
```

---

## Troubleshooting

### Tests Hang

**Cause**: Context not cancelled or deadlock.

**Solution**: Always use `defer cancel()` with contexts and check for race conditions with `-race`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
```

### Data Races Detected

**Cause**: Concurrent access to shared state without protection.

**Solution**: Use mutexes or atomic operations:

```go
var mu sync.Mutex
mu.Lock()
sharedState = newValue
mu.Unlock()
```

### Tests Fail Intermittently

**Cause**: Race condition or timing issue.

**Solution**: Run with `-race` flag and fix detected races. Avoid time-based assertions.

---

## Related Documentation

- [API Reference](./api-reference.md)
- [Concurrency Best Practices](./concurrency-best-practices.md)
- [Plugin Development Guide](./plugin-development.md)
