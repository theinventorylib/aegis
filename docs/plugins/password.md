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

Exposes endpoints for:

- Registration (with email/password)
- Login
- Password Reset
