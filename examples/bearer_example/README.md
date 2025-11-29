# Bearer Authentication Example

This example demonstrates how to use the Bearer plugin with JWT tokens for API authentication.

## Overview

The bearer plugin enables Bearer token authentication across all Aegis routes. This example shows:
- Registering the bearer and JWT plugins
- Obtaining JWT tokens
- Using Bearer tokens for authentication
- Protected routes that accept both cookies and Bearer tokens

## Running the Example

1. **Start the database**:
```bash
# Make sure PostgreSQL is running with the aegis_dev database
createdb aegis_dev
```

2. **Run migrations**:
```bash
# Apply Aegis core migrations
# (Assuming you have migration files set up)
```

3. **Start the server**:
```bash
go run main.go
```

The server will start on `http://localhost:8080`.

## Testing Bearer Authentication

### Step 1: Create a User and Get a Session

First, you need to create a user and obtain a session token. This depends on which auth plugins you have enabled (email, password, etc.).

For this example, assume you have a session token: `<session_token>`

### Step 2: Get a JWT Token

Use your session to obtain a JWT access token:

```bash
curl -X POST http://localhost:8080/auth/jwt/token \
  -H "Cookie: aegis_session=<session_token>"
```

Response:
```json
{
  "access_token": "eyJhbGc...",
  "access_expiry": "2024-01-01T12:00:00Z",
  "refresh_token": "eyJhbGc...",
  "refresh_expiry": "2024-01-08T12:00:00Z"
}
```

### Step 3: Use Bearer Token

Now use the access token with Bearer authentication:

```bash
curl -X GET http://localhost:8080/protected \
  -H "Authorization: Bearer eyJhbGc..."
```

Response:
```json
{
  "message": "Authenticated successfully",
  "user_id": "123e4567-e89b-12d3-a456-426614174000"
}
```

## How It Works

1. **Bearer Plugin Registration**: The bearer plugin is registered, which marks that Bearer authentication is supported
2. **Core Middleware**: The core `AuthMiddleware` checks for Bearer tokens in the `Authorization` header
3. **Token Validation**: Tokens are validated via the `SessionService` (which delegates to JWT plugin for JWT tokens)
4. **User Context**: Authenticated user is injected into the request context
5. **Protected Routes**: Routes using `RequireAuth()` middleware accept both cookie and Bearer authentication

## Authentication Methods

This example supports two authentication methods:

### Cookie-based (Session)
```bash
curl -X GET http://localhost:8080/protected \
  -H "Cookie: aegis_session=<session_token>"
```

### Bearer Token
```bash
curl -X GET http://localhost:8080/protected \
  -H "Authorization: Bearer <access_token>"
```

Both methods work on all protected routes without any additional configuration.

## Code Highlights

### Registering Plugins
```go
// JWT plugin generates tokens
jwtPlugin := jwt.New(&jwt.Config{DB: dbProvider})
auth.Use(context.Background(), jwtPlugin)

// Bearer plugin enables Bearer auth
bearerPlugin := bearer.New(&bearer.Config{})
auth.Use(context.Background(), bearerPlugin)
```

### Protected Route
```go
protectedHandler := auth.RequireAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    user, _ := auth.GetUser(r.Context())
    // User is authenticated via cookie OR Bearer token
}))
```

## Security Notes

- Always use HTTPS in production when using Bearer tokens
- Bearer tokens in headers are vulnerable to interception over HTTP
- Use short-lived access tokens (default: 15 minutes)
- Implement token refresh flows for better UX
- Store tokens securely on the client side

## Next Steps

- Add password or email plugins for user registration
- Implement token refresh flow
- Add rate limiting for token endpoints
- Configure CORS for cross-origin requests
