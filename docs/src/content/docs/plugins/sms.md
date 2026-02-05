---
title: SMS Plugin
---
## SMS Plugin
Provides phone number verification via OTP.
```go
import "github.com/theinventorylib/aegis/plugins/sms"
smsPlugin := sms.New(smsConfig, nil, plugins.DialectPostgres)
aegis.Use(ctx, smsPlugin)
```
