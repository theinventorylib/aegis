---
title: JWT Plugin
---
## JWT Plugin
Handles the generation, validation, and rotation of JSON Web Tokens.
```go
import "github.com/theinventorylib/aegis/plugins/jwt"
jwtPlugin := jwt.New(jwtConfig, nil, plugins.DialectPostgres)
aegis.Use(ctx, jwtPlugin)
```
