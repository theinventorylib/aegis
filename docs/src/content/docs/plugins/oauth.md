---
title: OAuth Plugin
---
## OAuth Plugin
Enables social login with providers like Google, GitHub, and more.
```go
import "github.com/theinventorylib/aegis/plugins/oauth"
oauthPlugin := oauth.New(cfg, nil, plugins.DialectPostgres)
aegis.Use(ctx, oauthPlugin)
```
