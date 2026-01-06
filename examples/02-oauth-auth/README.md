# OAuth Authentication Example

This example demonstrates OAuth authentication with Aegis, supporting Google and GitHub login, plus traditional email/password authentication.

## Features

- Social login with Google OAuth
- Social login with GitHub OAuth
- Traditional email/password authentication as fallback
- Automatic account linking (same email = same user)
- Interactive web interface
- Session management

## Prerequisites

- Go 1.21 or higher
- PostgreSQL database
- OAuth credentials from Google and/or GitHub
- Aegis CLI tool (for migrations)

## Setup

### 1. Install Dependencies

```bash
go mod init aegis-oauth-example
go get github.com/theinventorylib/aegis
go get github.com/go-chi/chi/v5
go get github.com/lib/pq
go get github.com/markbates/goth
go get github.com/gorilla/sessions
```

### 2. Create Database

```bash
createdb aegis_oauth
```

### 3. Set Up OAuth Credentials

#### Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select existing one
3. Enable Google+ API
4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client ID"
5. Application type: "Web application"
6. Authorized redirect URIs: `http://localhost:8080/auth/oauth/google/callback`
7. Copy Client ID and Client Secret

#### GitHub OAuth

1. Go to [GitHub Settings](https://github.com/settings/developers)
2. Click "New OAuth App"
3. Application name: "Aegis OAuth Example"
4. Homepage URL: `http://localhost:8080`
5. Authorization callback URL: `http://localhost:8080/auth/oauth/github/callback`
6. Register application
7. Copy Client ID and Client Secret

### 4. Export and Run Migrations

```bash
# Install Aegis CLI
go install github.com/theinventorylib/aegis/cmd/aegis@latest

# Export migrations with OAuth plugin
aegis export --dialect postgres --plugins oauth --output ./migrations

# Run migrations
psql aegis_oauth < migrations/001_aegis_auth_schema.sql
psql aegis_oauth < migrations/002_oauth_schema.sql
```

### 5. Configure Environment Variables

Create a `.env` file or export these variables:

```bash
export DATABASE_URL="postgres://user:password@localhost/aegis_oauth?sslmode=disable"
export MASTER_SECRET="$(openssl rand -base64 32)"
export GOOGLE_CLIENT_ID="your-google-client-id"
export GOOGLE_CLIENT_SECRET="your-google-client-secret"
export GITHUB_CLIENT_ID="your-github-client-id"
export GITHUB_CLIENT_SECRET="your-github-client-secret"
export BASE_URL="http://localhost:8080"
```

Or create a `.env` file:

```env
DATABASE_URL=postgres://user:password@localhost/aegis_oauth?sslmode=disable
MASTER_SECRET=your-generated-secret-here
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-google-client-secret
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
BASE_URL=http://localhost:8080
```

### 6. Run the Application

```bash
# Load environment variables (if using .env file)
export $(cat .env | xargs)

# Run the application
go run main.go
```

Visit http://localhost:8080

## Usage

### Sign In with Google

1. Click "Sign in with Google" on the home page
2. You'll be redirected to Google's consent screen
3. Grant permissions
4. You'll be redirected back and automatically logged in
5. Visit `/dashboard` to see your profile

### Sign In with GitHub

1. Click "Sign in with GitHub" on the home page
2. You'll be redirected to GitHub's authorization page
3. Authorize the application
4. You'll be redirected back and automatically logged in

### Traditional Email/Password

1. Click "Sign Up" to create an account
2. Fill in name, email, and password
3. Click "Login" to sign in with email/password

### Account Linking

If you sign in with Google using `alice@gmail.com` and then sign in with GitHub using the same email, both accounts will be linked to the same user. You can see all linked accounts by querying the database:

```sql
SELECT * FROM auth.accounts WHERE user_id = 'your-user-id';
```

## OAuth Flow Explained

### 1. Initiate OAuth Flow

```
GET /auth/oauth/google
```

- Aegis generates a random state token for CSRF protection
- Stores state in session
- Redirects to Google's authorization URL

### 2. User Grants Permission

User is at Google's consent screen and clicks "Allow"

### 3. Callback Handler

```
GET /auth/oauth/google/callback?code=...&state=...
```

- Aegis validates state token (CSRF protection)
- Exchanges authorization code for access token
- Fetches user profile from Google
- Creates or updates user account
- Creates session
- Sets session cookie
- Redirects to dashboard

### 4. Subsequent Requests

All subsequent requests include the session cookie, authenticating the user automatically.

## API Endpoints

### OAuth Endpoints

- `GET /auth/oauth/:provider` - Initiate OAuth flow (google, github)
- `GET /auth/oauth/:provider/callback` - OAuth callback handler

### Traditional Auth Endpoints

- `POST /auth/signup` - Register with email/password
- `POST /auth/login` - Login with email/password
- `POST /auth/logout` - Logout (revoke session)
- `GET /auth/user` - Get current user

### Protected Endpoints

- `GET /dashboard` - User dashboard (web page)
- `GET /api/profile` - Get user profile (JSON)
- `GET /api/accounts` - List linked authentication providers

## Database Schema

The OAuth plugin adds an `oauth_accounts` table:

```sql
CREATE TABLE auth.oauth_accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_account_id TEXT NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES auth.users(id) ON DELETE CASCADE
);
```

## Security Considerations

1. **HTTPS Required**: In production, always use HTTPS
2. **State Token**: Aegis includes CSRF protection via state parameter
3. **Secure Cookies**: Set `CookieSecure: true` in production
4. **Token Storage**: OAuth tokens are encrypted in the database
5. **Redirect URI Validation**: Only whitelisted redirect URIs are accepted
6. **Session Expiry**: Configure appropriate session timeouts

## Production Checklist

- [ ] Use HTTPS (`CookieSecure: true`)
- [ ] Store secrets in environment variables or secret manager
- [ ] Configure proper OAuth redirect URIs in provider consoles
- [ ] Enable rate limiting
- [ ] Set up monitoring and logging
- [ ] Configure CORS appropriately
- [ ] Use secure session and CSRF secrets
- [ ] Set appropriate token expiration times
- [ ] Validate OAuth scopes match your requirements

## Troubleshooting

### "oauth_failed" Error

- Check that OAuth credentials are correctly set
- Verify redirect URI matches exactly (including http/https)
- Ensure OAuth provider application is not in "test mode"

### Session Not Persisting

- Check that cookies are enabled in browser
- Verify `CookieSecure` matches your protocol (http vs https)
- Check `CookieDomain` setting

### Account Not Linking

- Verify that emails match exactly (case-sensitive)
- Check that email is provided by OAuth provider
- Some providers require additional scopes for email access

## Next Steps

- See example `03-organizations` for multi-tenant support
- See example `04-api-jwt` for stateless JWT authentication
- Check the [Goth documentation](https://github.com/markbates/goth) for more OAuth providers
