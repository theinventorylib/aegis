# Security Policy

## Production Security Recommendations

### 1. Password Hashing (Argon2id)

Aegis uses Argon2id for password hashing with secure defaults. For production deployments:

#### Recommended Parameters

```go
cfg := config.Default()
// Argon2id is configured by default with secure parameters
// Memory: 64 MB, Time: 1 iteration, Threads: 4, KeyLength: 32 bytes

// For higher security (if you have resources), increase memory and time:
cfg.WithArgon2Time(2).      // 2 iterations (default: 1)
   WithArgon2Memory(128 * 1024) // 128 MB (default: 64 MB)

aegis.New(ctx, cfg)
```

**Guidelines**:
- **Memory**: Minimum 64 MB (default). Increase to 128 MB or 256 MB if server resources allow
- **Time**: Minimum 1 iteration (default). Increase to 2-3 for extra security
- **Threads**: 4 threads (default) is optimal for most servers
- **Key Length**: 32 bytes (default) provides 256-bit security

**OWASP Recommendations** (2024):
- Memory: 64-128 MB for server-side authentication
- Time: 1-3 iterations depending on acceptable latency
- Always use Argon2id variant (not Argon2i or Argon2d)

---

### 2. Session Security

#### Session Expiry

```go
cfg := config.Default().
    WithSessionExpiry(24 * time.Hour).   // Default: 24 hours
    WithRefreshExpiry(7 * 24 * time.Hour) // Default: 7 days

aegis.New(ctx, cfg)
```

**Guidelines**:
- **Session Expiry**: 15 minutes to 24 hours depending on sensitivity
  - Banking apps: 15-30 minutes
  - E-commerce: 1-4 hours  
  - Social apps: 12-24 hours
- **Refresh Expiry**: Should be longer than session expiry (typically 7-30 days)
- **Rule**: Refresh expiry MUST be greater than session expiry

#### Cookie Security Flags

```go
cfg := config.Default().
    WithCookieSecure(true).    // Default: true (REQUIRED in production)
    WithCookieSameSite("Strict"). // Default: "Lax"
    WithCookieDomain(".example.com") // Set for subdomain sharing
// CookieHTTPOnly is always true by default (prevents XSS)

aegis.New(ctx, cfg)
```

**Cookie Flag Requirements**:
- ✅ **Secure**: MUST be `true` in production (HTTPS only)
- ✅ **HTTPOnly**: MUST be `true` (prevents JavaScript access)
- ✅ **SameSite**: 
  - `"Strict"` - Maximum protection (recommended for same-site apps)
  - `"Lax"` - Default, balances security and usability
  - `"None"` - Only if cross-site cookies required (requires Secure=true)

---

### 3. CSRF Protection

#### Web Applications

```go
cfg := config.Default().
    WithSecret([]byte("your-random-32-byte-secret"))
// CSRF protection is automatically enabled for web apps

aegis.New(ctx, cfg)
```

**Requirements**:
- Master secret MUST be cryptographically random (32+ bytes)
- CSRF secret is automatically derived from master secret
- Store in environment variables, NOT in code

#### API-Only Applications

```go
cfg := config.Default().
    WithAPIOnlyMode(true) // Skips CSRF requirement

aegis.New(ctx, cfg)
```

Only use API mode if:
- No browser-based clients
- All requests use Authorization headers (not cookies)
- No session cookies sent by browser

---

### 4. Redis Security (Optional)

If using Redis for session storage:

```go
cfg := config.Default().
    WithRedis(
        "redis-server.internal", // Use internal/private network
        6379,
        os.Getenv("REDIS_PASSWORD"), // Always use password
        0,
    )

aegis.New(ctx, cfg)
```

**Redis Security Checklist**:
- ✅ Use password authentication (`requirepass` in redis.conf)
- ✅ Bind to internal network only (not public internet)
- ✅ Use TLS/SSL for connections if over network
- ✅ Disable dangerous commands (`CONFIG`, `FLUSHALL`) in production
- ✅ Regular backups of session data

---

### 5. JWT Security (if using JWT plugin)

```go
import "github.com/theinventorylib/aegis/plugins/jwt"

jwtPlugin := jwt.New(nil) // Uses default store
aegis.Use(ctx, jwtPlugin)
```

**JWT Security Guidelines**:
- Use RS256 (RSA) for public/private key signing
- Rotate signing keys regularly (every 90 days recommended)
- Set short expiry times (15-60 minutes for access tokens)
- Use longer expiry for refresh tokens (7-30 days)
- Implement token blacklisting for logout
- Store keys securely (environment variables or key management service)

---

### 6. Database Security

#### Connection Security

```go
import "database/sql"
import _ "github.com/lib/pq" // PostgreSQL driver

// PostgreSQL with SSL
connString := "postgres://user:pass@localhost:5432/db?sslmode=require"
db, _ := sql.Open("postgres", connString)

// Configure connection pool, timeouts, etc.
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)

cfg := config.Default().WithDB(db)
aegis.New(ctx, cfg)
```

**Database Security Checklist**:
- ✅ Use SSL/TLS for database connections (`sslmode=require`)
- ✅ Use least-privilege database users (not root/admin)
- ✅ Store credentials in environment variables
- ✅ Regular backups
- ✅ Encrypt sensitive data at rest, if required
- ✅ Use prepared statements (Aegis does this automatically)

---

### 7. Logging and Monitoring

```go
import "log/slog"

cfg := config.Default().
    WithLogger(slog.Default())

aegis.New(ctx, cfg)
```

**Logging Best Practices**:
- ✅ Log authentication failures (for intrusion detection)
- ✅ Log administrative actions
- ✅ DO NOT log passwords, tokens, or sensitive data
- ✅ Use structured logging (JSON) for production
- ✅ Set up alerts for suspicious patterns:
  - Multiple failed login attempts
  - Unusual geographic locations
  - High-frequency API calls

---

### 8. Environment Variable Management

**Never hardcode secrets in source code**. Use environment variables:

```bash
# Required
export AEGIS_DATABASE_URL="postgres://..."
export AEGIS_CSRF_SECRET="random-32-byte-secret"

# Optional
export AEGIS_REDIS_HOST="redis.internal"
export AEGIS_REDIS_PASSWORD="redis-password"
export AEGIS_JWT_SIGNING_KEY="jwt-private-key"
```

**Secret Management Tools**:
- Docker: Use Docker secrets
- Kubernetes: Use Kubernetes secrets
- AWS: AWS Secrets Manager
- GCP: Google Secret Manager  
- Azure: Azure Key Vault
- HashiCorp Vault

---

### 9. Rate Limiting

```go
import (
    "time"
    "github.com/theinventorylib/aegis/config"
    "github.com/theinventorylib/aegis/core"
)

// Enable rate limiting with defaults (100 requests per minute per IP)
cfg := config.Default().
    WithRateLimiting()

aegis.New(ctx, cfg)

// Or with custom configuration:
cfg = config.Default().
    WithRateLimitConfig(&core.RateLimitConfig{
        RequestsPerWindow: 100,
        WindowDuration:    time.Minute,
        ByIP:              true,
    })

aegis.New(ctx, cfg)
```
```

**Rate Limiting Guidelines**:
- Login endpoints: 5-10 attempts per IP per minute
- Registration: 3-5 per IP per hour
- Password reset: 3 per email per hour
- API endpoints: 100-1000 requests per minute per user

---

### 10. Security Headers

Configure your HTTP server with security headers:

```go
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        next.ServeHTTP(w, r)
    })
})
```

---

## Security Checklist for Production

Before deploying to production, verify:

- [ ] All secrets stored in environment variables (not code)
- [ ] HTTPS enabled (TLS 1.2+)
- [ ] `CookieSecure=true`
- [ ] `CookieHTTPOnly=true`
- [ ] CSRF protection enabled (or API-only mode if appropriate)
- [ ] Database connections use SSL
- [ ] Argon2 parameters reviewed for your threat model
- [ ] Session expiry set appropriately for your use case
- [ ] Logging configured (no sensitive data logged)
- [ ] Rate limiting enabled on auth endpoints
- [ ] Security headers configured
- [ ] Redis password authentication enabled (if using Redis)
- [ ] Regular security updates applied to dependencies

---

## Vulnerability Reporting

If you discover a security vulnerability in Aegis:

1. **DO NOT** open a public GitHub issue
2. Email security concerns to: [Your security email]
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if applicable)

**Response Timeline**:
- Acknowledgment: Within 48 hours
- Status update: Within 7 days
- Fix timeline: Depends on severity (critical: 7-14 days)

---

## Security Best Practices Summary

|  Category | Requirement | Default | Production Recommendation |
|-----------|-------------|---------|--------------------------|
| HTTPS | Required | N/A | TLS 1.2+ with valid certificate |
| Cookie Secure | Required | `true` | MUST be `true` |
| Cookie HTTPOnly | Required | `true` | MUST be `true` |
| Cookie SameSite | Recommended | `"Lax"` | `"Strict"` or `"Lax"` |
| CSRF Secret | Required (web) | None | 32+ random bytes |
| Session Expiry | Required | 24h | 15min - 24h (use case dependent) |
| Refresh Expiry | Required | 7d | 7-30 days |
| Argon2 Memory | Recommended | 64 MB | 64-256 MB |
| Argon2 Time | Recommended | 1 | 1-3 iterations |
| Database SSL | Required | N/A | `sslmode=require` |
| Redis Password | Required | None | Strong password + internal network |
| Rate Limiting | Recommended | None | 5-10 login attempts/min |
| Logging | Recommended | None | Structured + monitoring |

---

## Additional Resources

- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [Argon2 RFC 9106](https://datatracker.ietf.org/doc/html/rfc9106)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
