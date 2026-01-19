# JWT API Authentication Example

This example demonstrates how to build a stateless REST API with JWT authentication using Aegis.

## Features

- Stateless JWT authentication (no server-side sessions)
- API-only mode (no cookies, no CSRF)
- Short-lived access tokens (15 minutes)
- Long-lived refresh tokens (7 days)
- API key authentication (Bearer tokens)
- Token revocation/blacklisting
- CORS enabled for cross-origin requests
- Role-based access control ready

## Prerequisites

- Go 1.21 or higher
- PostgreSQL database
- Aegis CLI tool (for migrations)

## Setup

### 1. Install Dependencies

```bash
go mod init aegis-jwt-api
go get github.com/theinventorylib/aegis
go get github.com/go-chi/chi/v5
go get github.com/go-chi/cors
go get github.com/lib/pq
```

### 2. Create Database

```bash
createdb aegis_jwt
```

### 3. Export and Run Migrations

```bash
# Install Aegis CLI
go install github.com/theinventorylib/aegis/cmd/aegis@latest

# Export migrations with JWT and Bearer plugins
aegis export --dialect postgres --plugins jwt,bearer --output ./migrations

# Run migrations
psql aegis_jwt < migrations/001_aegis_auth_schema.sql
psql aegis_jwt < migrations/002_jwt_schema.sql
psql aegis_jwt < migrations/003_bearer_schema.sql
```

### 4. Configure Application

Update the database connection in `main.go`:

```go
db, err := sql.Open("postgres", "postgres://user:password@localhost/aegis_jwt?sslmode=disable")
```

Generate a secure secret:

```bash
openssl rand -base64 32
```

### 5. Run the Application

```bash
go run main.go
```

The API will be available at http://localhost:8080

## API Usage

### 1. Register a User

```bash
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePassword123",
    "name": "Alice Johnson"
  }'
```

Response:
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "refresh_xxx",
  "user": {
    "id": "user_xxx",
    "email": "alice@example.com",
    "name": "Alice Johnson"
  }
}
```

### 2. Login

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "SecurePassword123"
  }'
```

Response:
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "refresh_yyy"
}
```

**Save the token!** You'll need it for authenticated requests.

### 3. Access Protected Endpoints

```bash
# Store token in variable
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Get user profile
curl http://localhost:8080/api/profile \
  -H "Authorization: Bearer $TOKEN"

# Get user data
curl http://localhost:8080/api/data \
  -H "Authorization: Bearer $TOKEN"

# Create data
curl -X POST http://localhost:8080/api/data \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "New Item", "description": "Test item"}'
```

### 4. Refresh Token

When the access token expires (after 15 minutes), use the refresh token:

```bash
curl -X POST http://localhost:8080/auth/jwt/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "refresh_yyy"
  }'
```

Response:
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "refresh_zzz"
}
```

### 5. Create API Key (Long-lived Token)

For server-to-server authentication, create an API key:

```bash
curl -X POST http://localhost:8080/auth/bearer/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production Server",
    "expires_in": "365d"
  }'
```

Response:
```json
{
  "success": true,
  "api_key": "ak_xxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

**Important:** Save this key securely! It won't be shown again.

Use the API key like a JWT:

```bash
curl http://localhost:8080/api/profile \
  -H "Authorization: Bearer ak_xxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

### 6. List API Keys

```bash
curl http://localhost:8080/auth/bearer/list \
  -H "Authorization: Bearer $TOKEN"
```

### 7. Revoke API Key

```bash
curl -X DELETE http://localhost:8080/auth/bearer/key_id \
  -H "Authorization: Bearer $TOKEN"
```

### 8. Revoke JWT Token

To logout or invalidate a token before expiry:

```bash
curl -X POST http://localhost:8080/auth/jwt/revoke \
  -H "Authorization: Bearer $TOKEN"
```

## JWT Token Structure

JWT tokens contain:

```json
{
  "sub": "user_01ARZ3NDEKTSV4RRFFQ69G5FAV",
  "email": "alice@example.com",
  "name": "Alice Johnson",
  "iss": "aegis-jwt-example",
  "aud": "api.example.com",
  "exp": 1640995200,
  "iat": 1640991600
}
```

Claims:
- `sub` - Subject (user ID)
- `email` - User email
- `name` - User name
- `iss` - Issuer (your application)
- `aud` - Audience (intended recipients)
- `exp` - Expiration time (Unix timestamp)
- `iat` - Issued at (Unix timestamp)

## Token Lifecycle

### Access Token Flow

```
1. User logs in
   ↓
2. Server issues JWT access token (15 min expiry)
   ↓
3. Client stores token securely
   ↓
4. Client includes token in requests (Authorization header)
   ↓
5. Server validates token signature and expiry
   ↓
6. Request processed
```

### Refresh Token Flow

```
1. Access token expires
   ↓
2. Client sends refresh token to /auth/jwt/refresh
   ↓
3. Server validates refresh token
   ↓
4. Server issues new access token + refresh token
   ↓
5. Client replaces old tokens
```

## Security Best Practices

### Token Storage

**❌ Don't:**
- Store tokens in localStorage (vulnerable to XSS)
- Include tokens in URLs
- Log tokens
- Commit tokens to version control

**✅ Do:**
- Use httpOnly cookies for web apps (if not pure API)
- Use secure storage on mobile (Keychain, KeyStore)
- Use environment variables for API keys
- Implement token rotation

### Token Configuration

```go
jwt.New(
    jwt.WithExpiry(15*time.Minute),         // Short-lived for security
    jwt.WithRefreshExpiry(7*24*time.Hour),  // Balance security and UX
    jwt.WithIssuer("your-app-name"),        // Identify your app
    jwt.WithAudience("your-api-domain"),    // Prevent token reuse
)
```

### Production Checklist

- [ ] Use HTTPS only
- [ ] Set short access token expiry (5-15 minutes)
- [ ] Implement token refresh logic in client
- [ ] Enable rate limiting
- [ ] Log authentication events
- [ ] Monitor for suspicious activity
- [ ] Rotate secrets regularly
- [ ] Implement CORS properly (whitelist origins)
- [ ] Add request ID tracking
- [ ] Use strong password requirements

## Client Implementation Examples

### JavaScript/TypeScript

```typescript
class APIClient {
  private accessToken: string | null = null;
  private refreshToken: string | null = null;
  
  async login(email: string, password: string) {
    const res = await fetch('http://localhost:8080/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password })
    });
    
    const data = await res.json();
    this.accessToken = data.token;
    this.refreshToken = data.refresh_token;
    
    // Store in secure storage (NOT localStorage)
    return data;
  }
  
  async request(url: string, options: RequestInit = {}) {
    // Add auth header
    options.headers = {
      ...options.headers,
      'Authorization': `Bearer ${this.accessToken}`
    };
    
    let res = await fetch(url, options);
    
    // If unauthorized, try refreshing token
    if (res.status === 401) {
      await this.refreshAccessToken();
      
      // Retry original request
      options.headers = {
        ...options.headers,
        'Authorization': `Bearer ${this.accessToken}`
      };
      res = await fetch(url, options);
    }
    
    return res;
  }
  
  async refreshAccessToken() {
    const res = await fetch('http://localhost:8080/auth/jwt/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: this.refreshToken })
    });
    
    const data = await res.json();
    this.accessToken = data.token;
    this.refreshToken = data.refresh_token;
  }
}
```

### Go Client

```go
type APIClient struct {
    BaseURL      string
    AccessToken  string
    RefreshToken string
    HTTPClient   *http.Client
}

func (c *APIClient) Login(email, password string) error {
    body := map[string]string{
        "email":    email,
        "password": password,
    }
    
    resp, err := c.post("/auth/login", body)
    if err != nil {
        return err
    }
    
    c.AccessToken = resp["token"].(string)
    c.RefreshToken = resp["refresh_token"].(string)
    return nil
}

func (c *APIClient) Request(method, path string, body any) (*http.Response, error) {
    // Create request with auth header
    req, _ := http.NewRequest(method, c.BaseURL+path, nil)
    req.Header.Set("Authorization", "Bearer "+c.AccessToken)
    
    resp, err := c.HTTPClient.Do(req)
    
    // Refresh token if expired
    if resp.StatusCode == 401 {
        c.RefreshAccessToken()
        req.Header.Set("Authorization", "Bearer "+c.AccessToken)
        resp, err = c.HTTPClient.Do(req)
    }
    
    return resp, err
}
```

### Python Client

```python
import requests

class APIClient:
    def __init__(self, base_url):
        self.base_url = base_url
        self.access_token = None
        self.refresh_token = None
    
    def login(self, email, password):
        resp = requests.post(f"{self.base_url}/auth/login", json={
            "email": email,
            "password": password
        })
        data = resp.json()
        self.access_token = data["token"]
        self.refresh_token = data["refresh_token"]
        return data
    
    def request(self, method, path, **kwargs):
        headers = kwargs.get("headers", {})
        headers["Authorization"] = f"Bearer {self.access_token}"
        kwargs["headers"] = headers
        
        resp = requests.request(method, f"{self.base_url}{path}", **kwargs)
        
        if resp.status_code == 401:
            self.refresh_access_token()
            headers["Authorization"] = f"Bearer {self.access_token}"
            resp = requests.request(method, f"{self.base_url}{path}", **kwargs)
        
        return resp
    
    def refresh_access_token(self):
        resp = requests.post(f"{self.base_url}/auth/jwt/refresh", json={
            "refresh_token": self.refresh_token
        })
        data = resp.json()
        self.access_token = data["token"]
        self.refresh_token = data["refresh_token"]
```

## Troubleshooting

### "Invalid or expired token"

- Check that token hasn't expired (access tokens expire in 15 minutes)
- Verify token is included in `Authorization: Bearer {token}` header
- Try refreshing the token

### "Unauthorized" on protected endpoints

- Ensure you're sending the Authorization header
- Verify token format: `Authorization: Bearer eyJ...`
- Check that user exists and token is valid

### CORS errors

- Verify CORS is enabled in the server
- Check that origin is allowed
- Ensure credentials mode is correct (should be false for API-only)

## Next Steps

- Combine with Organizations plugin for multi-tenant APIs
- Add rate limiting per user/API key
- Implement role-based access control (RBAC)
- Add webhook authentication with JWT
- Set up monitoring and alerting
- Implement API versioning
