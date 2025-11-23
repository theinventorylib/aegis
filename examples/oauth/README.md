# Aegis OAuth Plugin Example

This example demonstrates how to use the OAuth plugin with Google and GitHub providers.

## Setup

1. **Create OAuth Applications**:

### Google OAuth
- Go to [Google Cloud Console](https://console.cloud.google.com/)
- Create a new project or select existing
- Enable Google+ API
- Go to Credentials > Create Credentials > OAuth 2.0 Client ID
- Add authorized redirect URI: `http://localhost:3000/auth/oauth/google/callback`
- Copy Client ID and Client Secret

### GitHub OAuth
- Go to GitHub Settings > Developer settings > OAuth Apps
- Click "New OAuth App"
- Set callback URL: `http://localhost:3000/auth/oauth/github/callback`
- Copy Client ID and Client Secret

2. **Set environment variables**:

```bash
export GOOGLE_CLIENT_ID="your-google-client-id"
export GOOGLE_CLIENT_SECRET="your-google-client-secret"
export GITHUB_CLIENT_ID="your-github-client-id"
export GITHUB_CLIENT_SECRET="your-github-client-secret"
export SESSION_SECRET="your-32-byte-random-secret"
```

## Usage

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gr1nch3/aegis"
	"github.com/gr1nch3/aegis/config"
	"github.com/gr1nch3/aegis/db"
	"github.com/gr1nch3/aegis/plugins/oauth"
	"github.com/gr1nch3/aegis/server"
)

func main() {
	// Initialize database
	database, err := db.NewPostgresProvider("postgres://user:pass@localhost/dbname")
	if err != nil {
		log.Fatal(err)
	}

	// Create router
	router := server.NewDefaultRouter()

	// Initialize Aegis
	auth, err := aegis.New(
		config.WithPostgres(database),
		config.WithRouter(router),
		config.WithJWTSecret([]byte("your-jwt-secret")),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Mount auth routes
	auth.MountRoutes("/auth")

	// Configure OAuth plugin
	oauthPlugin := oauth.New(&oauth.Config{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleCallbackURL:  "http://localhost:3000/auth/oauth/google/callback",

		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubCallbackURL:  "http://localhost:3000/auth/oauth/github/callback",

		SessionSecret: os.Getenv("SESSION_SECRET"),
	})

	// Register OAuth plugin
	auth.Use(oauthPlugin)

	// Start server
	log.Println("Server starting on :3000...")
	http.ListenAndServe(":3000", router.Handler())
}
```

## Testing OAuth Flow

1. **Start the server**:
```bash
go run main.go
```

2. **Test Google OAuth**:
```bash
# Open in browser:
http://localhost:3000/auth/oauth/google
```

3. **Test GitHub OAuth**:
```bash
# Open in browser:
http://localhost:3000/auth/oauth/github
```

## OAuth Endpoints

- `GET /auth/oauth/google` - Start Google OAuth flow
- `GET /auth/oauth/github` - Start GitHub OAuth flow
- `GET /auth/oauth/google/callback` - Google callback
- `GET /auth/oauth/github/callback` - GitHub callback

## Handling OAuth Response

After successful authentication, the callback will return user data:

```json
{
  "success": true,
  "provider": "google",
  "email": "user@example.com",
  "name": "John Doe",
  "provider_user_id": "123456789"
}
```

You should extend the callback handler to:
1. Check if user exists by provider_user_id
2. Create user if not exists
3. Create Aegis session
4. Redirect to your app
