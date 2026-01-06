# Basic Email/Password Authentication Example

This example demonstrates how to set up basic email/password authentication with Aegis.

## Features

- User registration (signup)
- User login with email/password
- Session-based authentication with cookies
- Protected routes that require authentication
- Logout functionality
- User profile retrieval

## Prerequisites

- Go 1.21 or higher
- PostgreSQL database
- Aegis CLI tool (for migrations)

## Setup

### 1. Install Dependencies

```bash
go mod init aegis-basic-example
go get github.com/theinventorylib/aegis
go get github.com/go-chi/chi/v5
go get github.com/lib/pq
```

### 2. Create Database

```bash
createdb aegis_example
```

### 3. Export and Run Migrations

```bash
# Install Aegis CLI if not already installed
go install github.com/theinventorylib/aegis/cmd/aegis@latest

# Export migrations for PostgreSQL
aegis export --dialect postgres --output ./migrations

# Run migrations (example using psql)
psql aegis_example < migrations/001_aegis_auth_schema.sql
```

### 4. Update Database Connection

Edit `main.go` and update the database connection string:

```go
db, err := sql.Open("postgres", "postgres://your_user:your_password@localhost/aegis_example?sslmode=disable")
```

### 5. Generate a Secure Secret

In production, always use a cryptographically secure random secret:

```bash
# Generate a 32-byte base64-encoded secret
openssl rand -base64 32
```

Update `main.go` with your secret:

```go
config.WithMasterSecret([]byte("your-generated-secret-here"))
```

### 6. Run the Application

```bash
go run main.go
```

The server will start on http://localhost:8080

## Usage

### Register a New User

```bash
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePassword123",
    "name": "Alice Smith"
  }' \
  -c cookies.txt -v
```

### Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePassword123"
  }' \
  -c cookies.txt -v
```

The session cookie will be automatically saved to `cookies.txt`.

### Get Current User

```bash
curl http://localhost:8080/auth/user -b cookies.txt
```

### Access Protected Endpoint

```bash
curl http://localhost:8080/api/profile -b cookies.txt
```

### Logout

```bash
curl -X POST http://localhost:8080/auth/logout -b cookies.txt
```

## API Endpoints

### Public Endpoints

- `GET /` - Home page with documentation
- `GET /health` - Health check
- `GET /api/public-content` - Public content (personalized if authenticated)

### Authentication Endpoints (provided by Aegis)

- `POST /auth/signup` - Register a new user
- `POST /auth/login` - Login with email/password
- `POST /auth/logout` - Logout (requires authentication)
- `GET /auth/user` - Get current user (requires authentication)
- `POST /auth/session/refresh` - Refresh session token
- `GET /auth/sessions` - List all user sessions
- `DELETE /auth/sessions/:id` - Revoke a specific session

### Protected Endpoints

- `GET /api/profile` - Get user profile
- `GET /api/dashboard` - Get user dashboard
- `POST /api/settings` - Update user settings

## Authentication Flow

### Registration Flow

1. User submits email, password, and name
2. Aegis validates the input (email format, password strength)
3. Password is hashed using Argon2id
4. User and account records are created in the database
5. Session is created and cookie is set
6. User is automatically logged in

### Login Flow

1. User submits email and password
2. Aegis looks up the user by email
3. Password is verified against the stored hash
4. Session token and refresh token are generated
5. Session is stored in database and Redis (if configured)
6. Session cookie is set in the response

### Protected Route Access

1. Request includes session cookie
2. Aegis middleware extracts and validates the token
3. Session is loaded from cache/database
4. User information is added to request context
5. Handler can access user via `core.GetUser(ctx)`

## Security Features

- **Argon2id Password Hashing**: Industry-standard secure password hashing
- **CSRF Protection**: Automatic CSRF token validation (enabled by default)
- **Secure Cookies**: HttpOnly and Secure flags enabled
- **Session Expiry**: Configurable session and refresh token expiration
- **Rate Limiting**: (Configure via Aegis options if needed)

## Production Considerations

1. **Use HTTPS**: Always use HTTPS in production
2. **Secure Secrets**: Store master secret in environment variables or secret management service
3. **Database Security**: Use strong database credentials and SSL connections
4. **Enable Rate Limiting**: Protect against brute force attacks
5. **Configure CORS**: Set appropriate CORS headers for your frontend
6. **Monitoring**: Add logging and monitoring for authentication events

## Next Steps

- See example `02-oauth-auth` for social login
- See example `03-organizations` for multi-tenant support
- See example `04-api-jwt` for API-only JWT authentication
