# JWT Plugin

The JWT plugin handles JSON Web Token generation, validation, and rotation. It is a core component of the Aegis authentication system.

## Installation

```bash
go get github.com/theinventorylib/aegis/plugins/jwt
```

## Configuration

```go
import "github.com/theinventorylib/aegis/plugins/jwt"

jwtPlugin := jwt.New(&jwt.Config{
    DB:             dbProvider,
    SessionService: sessionService,
})
```

## Endpoints

### Public
- `GET /.well-known/jwks.json`: Standard JWKS endpoint for public key discovery (RFC 8414).
- `GET /jwt/jwks`: Convenience alias for JWKS.
- `POST /jwt/refreshToken`: Refresh an access token using a valid refresh token.

### Protected (Requires Authentication)
- `POST /jwt/token`: Generate a new token pair (requires active session).
- `POST /jwt/getAccessToken`: Get a new access token (requires active session).
- `POST /jwt/logout`: Revoke the current token/session.
