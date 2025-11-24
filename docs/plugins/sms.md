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

### Public Endpoints
- `POST /sms/verify`: Verify an OTP code.
- `POST /sms/login`: Login with phone number and password (if Password plugin is enabled).

### Protected Endpoints (Requires Authentication)
- `POST /sms/send`: Send an OTP code to a phone number. Requires an active session to prevent spam.
