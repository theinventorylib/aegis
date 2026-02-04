# Email Plugin

The Email plugin provides email-based verification and authentication, including OTP login and Magic Links.

## Installation

```go
import "github.com/theinventorylib/aegis/plugins/emailotp"
```

## Usage

```go
// Configure email provider (e.g., SMTP or transparent)
emailConfig := emailotp.Config{
    // ...
}

emailPlugin := emailotp.New(emailConfig, nil, plugins.DialectPostgres)
aegis.Use(ctx, emailPlugin)
```

## Features

- Email verification (OTP or Link)
- Password reset
- Login via email OTP
