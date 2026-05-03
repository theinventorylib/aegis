// Package main demonstrates basic email/password authentication with Aegis.
//
// This example shows:
//   - Setting up Aegis with email/password authentication
//   - Mounting authentication routes
//   - Protecting routes with authentication middleware
//   - Handling user registration and login
//
// Run this example:
//  1. Set up a PostgreSQL database
//  2. Run migrations: aegis export --dialect postgres --output ./migrations
//  3. Update the database connection string below
//  4. go run main.go
//  5. Visit http://localhost:8080
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/router/routers"
)

func main() {
	// 1. Connect to database
	db, err := sql.Open("postgres", "postgres://user:password@localhost/aegis_basic?sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// 2. Create HTTP router with chi
	mux := chi.NewRouter()
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	r := routers.NewChiRouter(mux)

	// 3. Create Aegis instance
	cfg := config.Default().
		WithDB(db).
		WithRouter(r).
		WithSecret([]byte("your-32-byte-secret-key-here!!!!"))
	a, err := aegis.New(context.Background(), cfg)
	if err != nil {
		log.Fatal("Failed to create Aegis instance:", err)
	}

	// 4. Mount Aegis routes
	// This registers the following endpoints:
	//   - POST /auth/default/signup   - User registration
	//   - POST /auth/default/login    - User login
	//   - POST /auth/default/logout   - User logout
	//   - GET  /auth/default/session  - Get current session (protected)
	a.MountRoutes("/auth")

	// 5. Public routes
	mux.Get("/", homeHandler)
	mux.Get("/login", loginPageHandler)
	mux.Get("/signup", signupPageHandler)

	// 6. Protected routes (require authentication)
	mux.Group(func(r chi.Router) {
		r.Use(a.RequireAuth())
		r.Get("/dashboard", dashboardHandler)
		r.Get("/profile", profileHandler)
	})

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Aegis Basic Auth Example</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        .button { 
            display: inline-block; 
            padding: 12px 24px; 
            margin: 10px 5px; 
            background: #2c5aa0; 
            color: white; 
            text-decoration: none; 
            border-radius: 4px;
        }
        .button:hover { background: #1e4080; }
        .section { margin: 30px 0; padding: 20px; background: #f9f9f9; border-radius: 4px; }
        code { background: #e8e8e8; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>Aegis Basic Authentication Example</h1>
    <p>This example demonstrates basic email/password authentication with Aegis.</p>
    
    <div class="section">
        <h2>Getting Started</h2>
        <a href="/signup" class="button">Sign Up</a>
        <a href="/login" class="button">Login</a>
    </div>
    
    <div class="section">
        <h2>Features</h2>
        <ul>
            <li>Email/password registration</li>
            <li>Secure password hashing (bcrypt)</li>
            <li>Session-based authentication</li>
            <li>Protected routes</li>
            <li>CSRF protection</li>
        </ul>
    </div>
    
    <div class="section">
        <h2>API Endpoints</h2>
        <p><code>POST /auth/default/signup</code> - Register new user</p>
        <p><code>POST /auth/default/login</code> - Login</p>
        <p><code>POST /auth/default/logout</code> - Logout</p>
        <p><code>GET /auth/default/session</code> - Get current session (protected)</p>
    </div>
</body>
</html>
	`))
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Login - Aegis Basic Auth</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 400px; margin: 100px auto; padding: 20px; }
        input { width: 100%; padding: 10px; margin: 10px 0; box-sizing: border-box; }
        button { width: 100%; padding: 12px; background: #2c5aa0; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
        button:hover { background: #1e4080; }
        .error { color: #dc3545; background: #f8d7da; padding: 10px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Login</h1>
    <div id="error" class="error" style="display: none;"></div>
    
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
                const errorDiv = document.getElementById('error');
                errorDiv.textContent = 'Login failed: ' + (error.error || 'Unknown error');
                errorDiv.style.display = 'block';
            }
        });
    </script>
</body>
</html>
	`))
}

func signupPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>Sign Up - Aegis Basic Auth</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 400px; margin: 100px auto; padding: 20px; }
        input { width: 100%; padding: 10px; margin: 10px 0; box-sizing: border-box; }
        button { width: 100%; padding: 12px; background: #2c5aa0; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 16px; }
        button:hover { background: #1e4080; }
        .error { color: #dc3545; background: #f8d7da; padding: 10px; border-radius: 4px; margin: 10px 0; }
    </style>
</head>
<body>
    <h1>Sign Up</h1>
    <div id="error" class="error" style="display: none;"></div>
    
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
                const errorDiv = document.getElementById('error');
                errorDiv.textContent = 'Signup failed: ' + (error.error || 'Unknown error');
                errorDiv.style.display = 'block';
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
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .card { background: #f9f9f9; padding: 20px; margin: 20px 0; border-radius: 4px; }
        button { padding: 10px 20px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background: #c82333; }
    </style>
</head>
<body>
    <h1>Dashboard</h1>
    
    <div class="card">
        <h2>Welcome, %s!</h2>
        <p>Email: %s</p>
        <p>User ID: %s</p>
    </div>
    
    <div class="card">
        <h3>Your Profile</h3>
        <p>This is a protected page. Only authenticated users can see this.</p>
        <a href="/profile">View Full Profile</a>
    </div>
    
    <button onclick="logout()">Logout</button>
    
    <script>
        async function logout() {
            const res = await fetch('/auth/default/logout', {
                method: 'POST',
            });
            
            if (res.ok) {
                window.location.href = '/';
            }
        }
    </script>
</body>
</html>
	`, user.Name, user.Email, user.ID)))
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	user, err := core.GetUser(r.Context())
	if err != nil || user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Profile</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .card { background: #f9f9f9; padding: 20px; margin: 20px 0; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>User Profile</h1>
    
    <div class="card">
        <h2>%s</h2>
        <p><strong>Email:</strong> %s</p>
        <p><strong>User ID:</strong> %s</p>
        <p><strong>Created:</strong> %s</p>
    </div>
    
    <a href="/dashboard">← Back to Dashboard</a>
</body>
</html>
	`, user.Name, user.Email, user.ID, user.CreatedAt.Format("2006-01-02"))))
}
