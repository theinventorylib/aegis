// Package main demonstrates OAuth authentication with Aegis.
//
// This example shows:
//   - Setting up Aegis with OAuth providers (Google, GitHub)
//   - Email/password authentication as fallback
//   - Social login flow
//   - Linking multiple authentication providers to one user
//
// Run this example:
//  1. Set up OAuth credentials (see README.md)
//  2. Set up a PostgreSQL database
//  3. Run migrations with oauth plugin
//  4. Update configuration below
//  5. go run main.go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/plugins/oauth"
	oauthTypes "github.com/theinventorylib/aegis/plugins/oauth/types"
	"github.com/theinventorylib/aegis/router/routers"
)

func main() {
	// 1. Load configuration from environment
	dbURL := getEnv("DATABASE_URL", "postgres://user:password@localhost/aegis_oauth?sslmode=disable")
	masterSecret := getEnv("MASTER_SECRET", "your-32-byte-secret-key-here!!!!")

	// OAuth credentials
	googleClientID := getEnv("GOOGLE_CLIENT_ID", "")
	googleClientSecret := getEnv("GOOGLE_CLIENT_SECRET", "")
	githubClientID := getEnv("GITHUB_CLIENT_ID", "")
	githubClientSecret := getEnv("GITHUB_CLIENT_SECRET", "")

	baseURL := getEnv("BASE_URL", "http://localhost:8080")

	// 2. Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	// 3. Configure OAuth providers using Goth
	// Aegis uses Goth for OAuth provider integration
	gothProviders := []goth.Provider{}

	if googleClientID != "" && googleClientSecret != "" {
		gothProviders = append(gothProviders, google.New(
			googleClientID,
			googleClientSecret,
			fmt.Sprintf("%s/auth/oauth/google/callback", baseURL),
			"email", "profile",
		))
		log.Println("✓ Google OAuth enabled")
	}

	if githubClientID != "" && githubClientSecret != "" {
		gothProviders = append(gothProviders, github.New(
			githubClientID,
			githubClientSecret,
			fmt.Sprintf("%s/auth/oauth/github/callback", baseURL),
			"user:email",
		))
		log.Println("✓ GitHub OAuth enabled")
	}

	if len(gothProviders) == 0 {
		log.Println("⚠ No OAuth providers configured. Set GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET or GITHUB_CLIENT_ID/GITHUB_CLIENT_SECRET")
	}

	// 4. Create session store for OAuth state management
	sessionStore := sessions.NewCookieStore([]byte(masterSecret))
	sessionStore.Options.HttpOnly = true
	sessionStore.Options.Secure = false // Set to true in production with HTTPS

	// 5. Create HTTP router with wrapper
	mux := chi.NewRouter()
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	r := routers.NewChiRouter(mux)

	// 6. Create OAuth plugin
	oauthConfig := &oauth.Config{
		CallbackURL: baseURL + "/auth",
		Providers:   []oauthTypes.ProviderConfig{},
	}

	// Configure providers
	if googleClientID != "" && googleClientSecret != "" {
		oauthConfig.Providers = append(oauthConfig.Providers, oauthTypes.ProviderConfig{
			ProviderID:   "google",
			ProviderType: "google",
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			Scopes:       []string{"email", "profile"},
		})
	}

	if githubClientID != "" && githubClientSecret != "" {
		oauthConfig.Providers = append(oauthConfig.Providers, oauthTypes.ProviderConfig{
			ProviderID:   "github",
			ProviderType: "github",
			ClientID:     githubClientID,
			ClientSecret: githubClientSecret,
			Scopes:       []string{"user:email"},
		})
	}

	oauthPlugin := oauth.New(oauthConfig, nil, plugins.DialectPostgres)

	// 7. Create Aegis instance
	cfg := config.Default().WithDB(db).WithRouter(r).WithSecret([]byte(masterSecret))
	a, err := aegis.New(context.Background(),
		cfg,
	)
	if err != nil {
		log.Fatal("Failed to create Aegis instance:", err)
	}

	// Register OAuth plugin
	if err := a.Use(context.Background(), oauthPlugin); err != nil {
		log.Fatal("Failed to register OAuth plugin:", err)
	}

	// 8. Mount Aegis routes
	// OAuth routes are automatically mounted:
	//   - GET  /auth/oauth/:provider          - Initiate OAuth flow
	//   - GET  /auth/oauth/:provider/callback - OAuth callback handler
	//   - POST /auth/default/signup           - Traditional signup
	//   - POST /auth/default/login            - Traditional login
	a.MountRoutes("/auth")

	// 9. Public routes
	mux.Get("/", homeHandler)
	mux.Get("/login", loginPageHandler)
	mux.Get("/signup", signupPageHandler)

	// 10. Protected routes
	mux.Group(func(r chi.Router) {
		r.Use(a.RequireAuth())

		r.Get("/dashboard", dashboardHandler)
		r.Get("/api/profile", profileHandler)
		r.Get("/api/accounts", accountsHandler)
	})

	log.Printf("Server starting on %s\n", baseURL)
	log.Println("OAuth endpoints:")
	if googleClientID != "" {
		log.Println("  - Google: GET /auth/oauth/google")
	}
	if githubClientID != "" {
		log.Println("  - GitHub: GET /auth/oauth/github")
	}
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Aegis OAuth Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 900px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            margin: 10px 5px; 
            background: #2c5aa0; 
            color: white; 
            text-decoration: none; 
            border-radius: 4px;
            border: none;
            cursor: pointer;
            font-size: 16px;
        }
        .button:hover { background: #1e4080; }
        .google { background: #DB4437; }
        .google:hover { background: #C33D2F; }
        .github { background: #333; }
        .github:hover { background: #222; }
        .section { margin: 30px 0; padding: 20px; background: #f9f9f9; border-radius: 4px; }
        code { background: #e8e8e8; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>Aegis OAuth Authentication Example</h1>
    <p>This example demonstrates social login with Google and GitHub, plus traditional email/password authentication.</p>
    
    <div class="section">
        <h2>🔐 Sign In Options</h2>
        <a href="/auth/oauth/google" class="button google">Sign in with Google</a>
        <a href="/auth/oauth/github" class="button github">Sign in with GitHub</a>
        <br>
        <a href="/login" class="button">Sign in with Email</a>
        <a href="/signup" class="button">Sign Up</a>
    </div>
    
    <div class="section">
        <h2>📚 How It Works</h2>
        <h3>OAuth Flow</h3>
        <ol>
            <li>Click "Sign in with Google/GitHub"</li>
            <li>Redirected to provider's authentication page</li>
            <li>Grant permissions</li>
            <li>Redirected back to app with authorization code</li>
            <li>Aegis exchanges code for access token</li>
            <li>User profile is retrieved and account created/linked</li>
            <li>Session is created and user is logged in</li>
        </ol>
        
        <h3>Account Linking</h3>
        <p>If you sign in with multiple providers using the same email address, 
        Aegis automatically links them to the same user account.</p>
    </div>
    
    <div class="section">
        <h2>🔗 API Endpoints</h2>
        <p><code>GET /auth/oauth/:provider</code> - Start OAuth flow (google, github)</p>
        <p><code>GET /auth/oauth/:provider/callback</code> - OAuth callback handler</p>
        <p><code>POST /auth/default/signup</code> - Traditional signup</p>
        <p><code>POST /auth/default/login</code> - Traditional login</p>
        <p><code>GET /auth/default/session</code> - Get current session (protected)</p>
        <p><code>GET /api/accounts</code> - List linked accounts (protected)</p>
    </div>
</body>
</html>
	`))
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	errorMsg := r.URL.Query().Get("error")
	errorHTML := ""
	if errorMsg != "" {
		// Escape user-supplied error message to prevent reflected XSS
		safe := html.EscapeString(errorMsg)
		errorHTML = fmt.Sprintf(`<div style="color: red; padding: 10px; background: #ffebee; border-radius: 4px; margin-bottom: 20px;">%s</div>`, safe)
	}

	w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Login - Aegis OAuth Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 400px; margin: 100px auto; padding: 20px; }
        input { width: 100%%; padding: 10px; margin: 10px 0; box-sizing: border-box; }
        button { width: 100%%; padding: 12px; background: #2c5aa0; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
        button:hover { background: #1e4080; }
        .oauth-buttons { margin: 20px 0; }
        .oauth-button { display: block; width: 100%%; padding: 10px; margin: 10px 0; text-align: center; text-decoration: none; border-radius: 4px; color: white; }
        .google { background: #DB4437; }
        .github { background: #333; }
        .divider { text-align: center; margin: 20px 0; color: #666; }
    </style>
</head>
<body>
    <h1>Login</h1>
    %s
    
    <div class="oauth-buttons">
        <a href="/auth/oauth/google" class="oauth-button google">Continue with Google</a>
        <a href="/auth/oauth/github" class="oauth-button github">Continue with GitHub</a>
    </div>
    
    <div class="divider">OR</div>
    
    <form id="loginForm">
        <input type="email" name="email" placeholder="Email" required>
        <input type="password" name="password" placeholder="Password" required>
        <button type="submit">Login</button>
    </form>
    
    <p style="text-align: center; margin-top: 20px;">
        Don't have an account? <a href="/signup">Sign up</a>
    </p>
    
    <script>
        document.getElementById('loginForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = {
                email: formData.get('email'),
                password: formData.get('password')
            };
            
            const res = await fetch('/auth/default/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            
            if (res.ok) {
                window.location.href = '/dashboard';
            } else {
                const error = await res.json();
                alert('Login failed: ' + (error.error || 'Unknown error'));
            }
        });
    </script>
</body>
</html>
	`, errorHTML)))
}

func signupPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Sign Up - Aegis OAuth Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 400px; margin: 100px auto; padding: 20px; }
        input { width: 100%; padding: 10px; margin: 10px 0; box-sizing: border-box; }
        button { width: 100%; padding: 12px; background: #2c5aa0; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
        button:hover { background: #1e4080; }
        .oauth-buttons { margin: 20px 0; }
        .oauth-button { display: block; width: 100%; padding: 10px; margin: 10px 0; text-align: center; text-decoration: none; border-radius: 4px; color: white; }
        .google { background: #DB4437; }
        .github { background: #333; }
        .divider { text-align: center; margin: 20px 0; color: #666; }
    </style>
</head>
<body>
    <h1>Sign Up</h1>
    
    <div class="oauth-buttons">
        <a href="/auth/oauth/google" class="oauth-button google">Sign up with Google</a>
        <a href="/auth/oauth/github" class="oauth-button github">Sign up with GitHub</a>
    </div>
    
    <div class="divider">OR</div>
    
    <form id="signupForm">
        <input type="text" name="name" placeholder="Full Name" required>
        <input type="email" name="email" placeholder="Email" required>
        <input type="password" name="password" placeholder="Password (min 8 characters)" required minlength="8">
        <button type="submit">Sign Up</button>
    </form>
    
    <p style="text-align: center; margin-top: 20px;">
        Already have an account? <a href="/login">Login</a>
    </p>
    
    <script>
        document.getElementById('signupForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            const formData = new FormData(e.target);
            const data = {
                name: formData.get('name'),
                email: formData.get('email'),
                password: formData.get('password')
            };
            
            const res = await fetch('/auth/default/signup', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            
            if (res.ok) {
                window.location.href = '/dashboard';
            } else {
                const error = await res.json();
                alert('Signup failed: ' + (error.error || 'Unknown error'));
            }
        });
    </script>
</body>
</html>
	`))
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard - Aegis OAuth Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .user-card { background: #f9f9f9; padding: 20px; border-radius: 4px; margin: 20px 0; }
        .button { padding: 10px 20px; background: #2c5aa0; color: white; text-decoration: none; border-radius: 4px; border: none; cursor: pointer; }
        .button:hover { background: #1e4080; }
        .logout { background: #d32f2f; }
        .logout:hover { background: #b71c1c; }
    </style>
</head>
<body>
    <h1>Dashboard</h1>
    
    <div class="user-card">
        <h2>Welcome, %s!</h2>
        <p><strong>Email:</strong> %s</p>
        <p><strong>User ID:</strong> %s</p>
        <p><strong>Member since:</strong> %s</p>
    </div>
    
    <div>
        <a href="/api/accounts" class="button">View Linked Accounts</a>
        <button class="button logout" onclick="logout()">Logout</button>
    </div>
    
    <script>
        async function logout() {
            const res = await fetch('/auth/default/logout', { method: 'POST' });
            if (res.ok) {
                window.location.href = '/';
            }
        }
    </script>
</body>
</html>
	`, user.Name, user.Email, user.ID, user.CreatedAt.Format("January 2, 2006"))))
}

// Response types for consistent API responses

type OAuthProfileResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AccountsInfo struct {
	UserID string `json:"user_id"`
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Data: OAuthProfileResponse{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	})
}

func accountsHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// In a real app, you'd query the accounts table here
	// This is just a demonstration
	core.WriteJSON(w, http.StatusOK, &core.Response{
		Success: true,
		Message: "Query the auth.accounts table to see all linked authentication providers",
		Data: AccountsInfo{
			UserID: user.ID,
		},
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
