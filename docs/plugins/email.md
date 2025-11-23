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
    DB:             dbProvider,
    Provider:       myEmailProvider, // Implements email.Provider interface
    PasswordPlugin: passwordPlugin,  // Optional: if using passwords
    
    // Optional configuration
    OTPLength:      6,
    OTPExpiration:  15 * time.Minute,
})
```

## Usage

Once registered, the plugin exposes endpoints for:

- Requesting verification emails
- Verifying OTPs
- Handling magic links
