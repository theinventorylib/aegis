# Password Authentication (Core Feature)

> [!IMPORTANT]
> Password authentication is **core functionality** in Aegis, not a plugin. It's built directly into the `AuthService` and available by default.

## Overview

Password-based authentication is implemented in the core `AuthService` using industry-standard Argon2id hashing. Unlike other authentication methods which are provided as plugins (Email, SMS, OAuth), password functionality is always available in any Aegis instance.

## Key Points

- **No plugin required**: Password functionality is built into core
- **Secure by default**: Uses Argon2id hashing algorithm
- **Always available**: No need to import or register any plugin

## Creating Users with Passwords

### From Application Code

Use `AuthService.CreateUserWithPassword` to create a user with a password:

```go
authService := auth.GetAuthService()
user, err := authService.CreateUserWithPassword(context.Background(), "s3cur3P@ssw0rd")
if err != nil {
    log.Fatal(err)
}
```

### Via Email Plugin

The Email plugin provides a convenience method that combines email and password:

```go
import "github.com/theinventorylib/aegis/plugins/email"

emailPlugin := email.New(&email.Config{})
user, err := emailPlugin.CreateUserWithEmailAndPassword(
    context.Background(),
    "user@example.com",
    "s3cur3P@ssw0rd",
)
```

### Via SMS Plugin

Similarly, the SMS plugin provides a convenience method:

```go
import "github.com/theinventorylib/aegis/plugins/sms"

smsPlugin := sms.New(&sms.Config{})
user, err := smsPlugin.CreateUserWithPhoneAndPassword(
    context.Background(),
    "+1234567890",
    "s3cur3P@ssw0rd",
)
```

## Verifying Passwords

Use `AuthService.VerifyPassword` to check if a password is correct:

```go
authService := auth.GetAuthService()
valid, err := authService.VerifyPassword(context.Background(), user.ID, "providedPassword")
if err != nil {
    log.Fatal(err)
}

if valid {
    // Password is correct, create session
    session, err := sessionService.CreateSession(context.Background(), user.ID)
}
```

## Migration from Old Plugin API

If you previously used the now-deprecated `plugins/password` package, update your code:

**Old (deprecated):**
```go
import "github.com/theinventorylib/aegis/plugins/password"

passwordPlugin := password.New(&password.Config{})
auth.Use(ctx, passwordPlugin)
```

**New (recommended):**
```go
// No import or plugin registration needed
// Password functionality is built into AuthService
authService := auth.GetAuthService()
user, err := authService.CreateUserWithPassword(ctx, password)
```

## Security Features

- **Argon2id Algorithm**: Memory-hard function resistant to GPU attacks
- **Automatic Salting**: Each password gets a unique salt
- **Secure Defaults**: Pre-configured with recommended parameters
- **Constant-Time Comparison**: Prevents timing attacks

## See Also

- [Core Concepts](./core-concepts.md) - Understanding Aegis architecture
- [Email Plugin](./plugins/email.md) - Email verification and authentication
- [SMS Plugin](./plugins/sms.md) - Phone number verification
- [Getting Started](./getting-started.md) - Basic Aegis setup

