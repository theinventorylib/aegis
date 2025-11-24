# Password Plugin

The Password plugin provides secure password-based authentication using Argon2id hashing.

## Installation

```bash
go get github.com/theinventorylib/aegis/plugins/password
```

## Configuration

```go
import "github.com/theinventorylib/aegis/plugins/password"

passwordPlugin := password.New(&password.Config{
    DB:     dbProvider,
    UserDB: dbProvider,
    
    // Optional configuration
    MinLength: 8,
})
```

## Usage

### Public Endpoints
- `POST /password/register`: Register a new user (if enabled).
- `POST /password/login`: Login with email/phone and password.
- `POST /password/reset`: Request a password reset.

### Protected Endpoints (Requires Authentication)
- `POST /password/change`: Change the current user's password.
