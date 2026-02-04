# JWT Plugin

The JWT plugin handles the generation, validation, and rotation of JSON Web Tokens.

## Installation

```go
import "github.com/theinventorylib/aegis/plugins/jwt"
```

## Usage

```go
jwtPlugin := jwt.New(jwtConfig, nil, plugins.DialectPostgres)
aegis.Use(ctx, jwtPlugin)
```

## Features

- Automatic key rotation
- JWKS endpoint
- Custom claims support
