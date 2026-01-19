# Aegis Examples

This directory contains practical examples demonstrating how to use Aegis for different authentication scenarios.

## 📚 Available Examples

### [01-basic-auth](./01-basic-auth) - Email/Password Authentication
**Perfect for:** Traditional web applications

- User registration and login with email/password
- Session-based authentication with cookies
- Protected routes with middleware
- User profile management
- CSRF protection

**Key Technologies:** Chi router, PostgreSQL, cookies

---

### [02-oauth-auth](./02-oauth-auth) - Social Login (OAuth)
**Perfect for:** Applications with Google/GitHub login

- OAuth 2.0 integration (Google, GitHub)
- Automatic account linking
- Fallback to email/password authentication
- Interactive web interface
- Session management

**Key Technologies:** Goth (OAuth), Gorilla sessions, PostgreSQL

---

### [03-organizations](./03-organizations) - Multi-Tenant SaaS
**Perfect for:** B2B SaaS applications, team collaboration tools

- Multi-tenant organization support
- Organization membership and roles
- Team invitation system
- Organization-scoped resources
- Tenant data isolation

**Key Technologies:** Organizations plugin, RBAC, PostgreSQL

---

### [04-api-jwt](./04-api-jwt) - Stateless REST API
**Perfect for:** Mobile apps, SPAs, microservices

- Stateless JWT authentication
- API-only mode (no cookies)
- Short-lived access tokens + refresh tokens
- API key generation (Bearer tokens)
- CORS configuration
- Token revocation

**Key Technologies:** JWT plugin, Bearer plugin, CORS

---

## 🚀 Quick Start

### Prerequisites

All examples require:
- **Go 1.21+**
- **PostgreSQL** (or MySQL/SQLite with minor adjustments)
- **Aegis CLI** tool

Install Aegis CLI:
```bash
go install github.com/theinventorylib/aegis/cmd/aegis@latest
```

### Running an Example

1. **Choose an example** and navigate to its directory:
   ```bash
   cd 01-basic-auth
   ```

2. **Read the README** for specific setup instructions

3. **Create database:**
   ```bash
   createdb aegis_example
   ```

4. **Export migrations:**
   ```bash
   aegis export --dialect postgres --output ./migrations
   # For examples with plugins:
   aegis export --dialect postgres --plugins oauth,jwt --output ./migrations
   ```

5. **Run migrations:**
   ```bash
   psql aegis_example < migrations/*.sql
   ```

6. **Install Go dependencies:**
   ```bash
   go mod init example
   go mod tidy
   ```

7. **Run the application:**
   ```bash
   go run main.go
   ```

8. **Visit** http://localhost:8080

---

## 🎯 Choosing the Right Example

| Use Case | Example | Key Features |
|----------|---------|-------------|
| Traditional web app with login | 01-basic-auth | Email/password, sessions, cookies |
| App with social login | 02-oauth-auth | Google/GitHub OAuth, account linking |
| Multi-tenant SaaS platform | 03-organizations | Organizations, teams, roles, isolation |
| Mobile app backend | 04-api-jwt | JWT, stateless, refresh tokens |
| Microservices API | 04-api-jwt | API keys, stateless, CORS |

---

## 🔧 Common Configuration

### Database Connection

All examples use PostgreSQL by default. Connection string format:

```go
db, err := sql.Open("postgres", "postgres://user:password@localhost/dbname?sslmode=disable")
```

For other databases:
- **MySQL:** `"mysql", "user:password@tcp(localhost:3306)/dbname"`
- **SQLite:** `"sqlite3", "./aegis.db"`

### Master Secret

Generate a secure secret for production:

```bash
openssl rand -base64 32
```

Use it in your application:

```go
config.WithSecret([]byte(os.Getenv("MASTER_SECRET")))
```

### Environment Variables

Create a `.env` file:

```bash
DATABASE_URL=postgres://user:password@localhost/dbname?sslmode=disable
MASTER_SECRET=your-generated-secret-here
BASE_URL=http://localhost:8080
```

---

## 📖 Example Structure

Each example includes:

```
example-name/
├── main.go           # Main application code
├── README.md         # Detailed setup and usage guide
├── .env.example      # Example environment configuration (if applicable)
└── migrations/       # Generated database migrations (after running aegis export)
```

---

## 🔒 Security Notes

### Development vs. Production

**Development (examples use these settings):**
- HTTP (not HTTPS)
- `CookieSecure: false`
- Simple secrets
- Relaxed CORS
- Detailed error messages

**Production (you should use):**
- HTTPS only
- `CookieSecure: true`
- Cryptographically random secrets
- Strict CORS policies
- Rate limiting
- Generic error messages
- Audit logging
- Environment-based configuration

### Production Checklist

- [ ] Use HTTPS
- [ ] Generate secure random secrets (`openssl rand -base64 32`)
- [ ] Store secrets in environment variables or secret manager
- [ ] Enable `CookieSecure: true`
- [ ] Configure appropriate CORS origins
- [ ] Enable rate limiting
- [ ] Set up logging and monitoring
- [ ] Use connection pooling for database
- [ ] Implement proper error handling
- [ ] Enable audit logging
- [ ] Set appropriate session/token expiry times
- [ ] Use database migrations in production
- [ ] Set up database backups

---

## 🧪 Testing the Examples

### Using cURL

Each example README includes cURL commands for testing. Basic pattern:

```bash
# Save cookies
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123"}' \
  -c cookies.txt

# Use cookies in subsequent requests
curl http://localhost:8080/api/profile -b cookies.txt
```

### Using Postman

1. Import the base URL: `http://localhost:8080`
2. For session-based auth: Enable "Automatically follow redirects" and "Save cookies"
3. For JWT auth: Set `Authorization: Bearer {token}` header

### Using HTTPie

```bash
# Login and save session
http POST :8080/auth/login email=user@example.com password=SecurePass123 --session=user

# Use saved session
http :8080/api/profile --session=user
```

---

## 🔌 Aegis Plugins Used

| Plugin | Examples | Purpose |
|--------|----------|---------|
| Email/Password (Core) | All | Built-in authentication |
| OAuth | 02-oauth-auth | Social login (Google, GitHub) |
| Organizations | 03-organizations | Multi-tenancy |
| JWT | 04-api-jwt | Stateless tokens |
| Bearer | 04-api-jwt | API keys |

Install additional plugins:
```bash
aegis export --dialect postgres --plugins oauth,jwt,organizations,bearer,admin,emailotp,sms
```

---

## 📚 Learning Path

**Recommended order for learning:**

1. **Start with 01-basic-auth**
   - Understand core concepts
   - Session management
   - Protected routes

2. **Try 02-oauth-auth**
   - Add social login
   - Account linking
   - OAuth flow

3. **Explore 03-organizations**
   - Multi-tenancy patterns
   - Role-based access
   - Team collaboration

4. **Master 04-api-jwt**
   - Stateless authentication
   - Mobile/SPA backends
   - Token management

---

## 🆘 Troubleshooting

### "Failed to connect to database"

- Ensure PostgreSQL is running: `pg_isready`
- Check connection string
- Verify database exists: `psql -l`

### "Table does not exist"

- Run migrations: `psql dbname < migrations/*.sql`
- Verify migrations were exported: `ls migrations/`

### "Session not persisting"

- Check that cookies are enabled
- Verify `CookieSecure` matches your protocol (http vs https)
- For API mode, use JWT instead of sessions

### "CORS errors"

- Enable CORS in your application
- Add your origin to allowed origins
- Set `AllowCredentials` appropriately

### "Invalid or expired token" (JWT)

- Token may have expired (default: 15 minutes)
- Use refresh token to get new access token
- Check system time is correct

---

## 🤝 Contributing

Found an issue with an example? Have a suggestion?

1. Open an issue describing the problem or enhancement
2. Submit a pull request with improvements
3. Share your own examples in the discussions

---

## 📖 Additional Resources

- [Aegis Documentation](https://pkg.go.dev/github.com/theinventorylib/aegis)
- [Project README](../README.md)
- [Architecture Guide](../ARCHITECTURE.md)
- [Security Practices](../SECURITY.md)
- [API Reference](https://pkg.go.dev/github.com/theinventorylib/aegis/core)

---

## 📄 License

These examples are part of the Aegis project and are licensed under the MIT License. See [LICENSE](../LICENSE) for details.

---

## 💬 Get Help

- **GitHub Issues:** [Report bugs or ask questions](https://github.com/theinventorylib/aegis/issues)
- **Discussions:** [Community forum](https://github.com/theinventorylib/aegis/discussions)
- **Documentation:** [pkg.go.dev](https://pkg.go.dev/github.com/theinventorylib/aegis)

Happy coding! 🚀
