# SMS Plugin

The SMS plugin provides phone number verification via OTP.

## Installation

```go
import "github.com/theinventorylib/aegis/plugins/sms"
```

## Usage

```go
smsPlugin := sms.New(smsConfig, nil, plugins.DialectPostgres)
aegis.Use(ctx, smsPlugin)
```

## Features

- Phone number verification
- Login via SMS OTP
