---
title: Email Plugin
---
## Email Plugin
Provides email-based verification and authentication, including OTP login and Magic Links.
```go
import "github.com/theinventorylib/aegis/plugins/emailotp"
emailPlugin := emailotp.New(emailConfig, nil, plugins.DialectPostgres)
aegis.Use(ctx, emailPlugin)
```
