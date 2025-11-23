# SMS Plugin

The SMS plugin provides phone number verification via OTP.

## Installation

```bash
go get github.com/theinventorylib/aegis/plugins/sms
```

## Configuration

```go
import "github.com/theinventorylib/aegis/plugins/sms"

smsPlugin := sms.New(&sms.Config{
    DB:       dbProvider,
    Provider: mySMSProvider, // Implements sms.Provider interface
    
    // Optional configuration
    OTPLength:      6,
    OTPExpiration:  5 * time.Minute,
})
```

## Usage

Exposes endpoints for:

- Sending SMS OTP
- Verifying SMS OTP
