# Email Plugin

The Email plugin provides email verification capabilities using OTP (One-Time Password) or Magic Links.

## Installation

```bash
go get github.com/theinventorylib/aegis/plugins/email
```

## Configuration

```go
import "github.com/theinventorylib/aegis/plugins/email"

emailPlugin := email.New(&email.Config{
    DB:        dbProvider,
    Provider:  myEmailProvider, // Implements email.Provider interface
    
    // Optional configuration
    OTPLength:     6,
    OTPExpiration: 15 * time.Minute,
})

// Note: Password support is provided by core AuthService. The email plugin
// exposes helper methods that call into core when creating users with passwords.
```

## Usage

Once registered, the plugin exposes endpoints for:

- Requesting verification emails
- Verifying OTPs
- Handling magic links
