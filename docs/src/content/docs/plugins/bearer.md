---
title: Bearer Plugin
---
## Bearer Plugin
Adds support for authenticating requests using standard Bearer tokens.
```go
import "github.com/theinventorylib/aegis/plugins/bearer"
bearerPlugin := bearer.New(bearerConfig)
aegis.Use(ctx, bearerPlugin)
```
