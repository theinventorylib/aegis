package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/theinventorylib/aegis"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/plugins/admin"
	"github.com/theinventorylib/aegis/plugins/email"
	"github.com/theinventorylib/aegis/plugins/password"
	"github.com/theinventorylib/aegis/plugins/sms"
	"github.com/theinventorylib/aegis/server"

	// Import PostgreSQL driver
	_ "github.com/lib/pq"
)

// MockEmailProvider implements email.Provider for testing
type MockEmailProvider struct{}

func (m *MockEmailProvider) SendEmail(to, subject, body string) error {
	fmt.Printf("[MockEmail] To: %s, Subject: %s\n", to, subject)
	return nil
}

// SendOTP implements email.Provider
func (m *MockEmailProvider) SendOTP(to, code string) error {
	fmt.Printf("[MockEmail] OTP To: %s, Code: %s\n", to, code)
	return nil
}

// MockSMSProvider implements sms.Provider for testing
type MockSMSProvider struct{}

func (m *MockSMSProvider) SendSMS(to, body string) error {
	fmt.Printf("[MockSMS] To: %s, Body: %s\n", to, body)
	return nil
}

// SendOTP implements sms.Provider
func (m *MockSMSProvider) SendOTP(to, code string) error {
	fmt.Printf("[MockSMS] OTP To: %s, Code: %s\n", to, code)
	return nil
}

func main() {
	// 1. Initialize Database
	// In production, use environment variables
	connString := "postgres://user:password@localhost:5432/aegis_db?sslmode=disable"
	sqlDB, err := sql.Open("postgres", connString)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer sqlDB.Close()

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Create database provider
	database := db.NewSQLProvider(sqlDB, db.PostgreSQL)

	// 2. Initialize Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	aegisRouter := server.NewChiRouter(r)

	// 3. Initialize Plugins

	// Password Plugin
	passwordPlugin := password.New(&password.Config{
		DB:     database,
		UserDB: database,
		Hasher: core.DefaultPasswordHasherConfig(),
	})

	// Email Plugin (with Password support)
	emailPlugin := email.New(&email.Config{
		DB:             database,
		Provider:       &MockEmailProvider{},
		OTPExpiry:      15 * time.Minute,
		PasswordPlugin: passwordPlugin,
	})

	// SMS Plugin (with Password support)
	smsPlugin := sms.New(&sms.Config{
		DB:             database,
		Provider:       &MockSMSProvider{},
		OTPExpiry:      5 * time.Minute,
		OTPLength:      6,
		PasswordPlugin: passwordPlugin,
	})

	// Admin Plugin
	adminPlugin := admin.New(database)

	// 4. Initialize Aegis Core
	auth, err := aegis.New(
		config.WithDB(sqlDB, db.PostgreSQL),
		config.WithRouter(aegisRouter),
		config.WithJWTSecret([]byte("demo-secret-key-do-not-use-in-prod")),
	)
	if err != nil {
		log.Fatal("Failed to initialize Aegis:", err)
	}

	// Register plugins
	if err := auth.Use(passwordPlugin); err != nil {
		log.Fatal("Failed to register password plugin:", err)
	}
	if err := auth.Use(emailPlugin); err != nil {
		log.Fatal("Failed to register email plugin:", err)
	}
	if err := auth.Use(smsPlugin); err != nil {
		log.Fatal("Failed to register sms plugin:", err)
	}
	if err := auth.Use(adminPlugin); err != nil {
		log.Fatal("Failed to register admin plugin:", err)
	}

	// 5. Mount Routes
	// This mounts all registered plugins under the default prefix (usually /auth)
	// or custom prefix if configured.
	auth.MountRoutes("/auth")

	// 6. Protected Routes Example
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware())

		r.Get("/api/me", func(w http.ResponseWriter, r *http.Request) {
			user, err := auth.GetUser(r.Context())
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			w.Write([]byte(fmt.Sprintf("Hello, %s!", user.ID)))
		})
	})

	// 7. Admin Routes Example (Protected by Admin Middleware)
	r.Group(func(r chi.Router) {
		// Note: You usually want to combine AuthMiddleware + AdminMiddleware
		// But the admin plugin's middleware might handle user fetching too.
		// Let's check admin.AdminMiddleware implementation - it calls core.GetUser.
		// So we need AuthMiddleware first to populate the context.
		r.Use(auth.AuthMiddleware())
		r.Use(adminPlugin.AdminMiddleware)

		r.Get("/api/admin/dashboard", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Welcome to the Admin Dashboard"))
		})
	})

	// Start Server
	port := ":3001"
	fmt.Printf("Server running on http://localhost%s\n", port)
	fmt.Println("Try the following endpoints:")
	fmt.Println("  POST   /auth/email/login       (Login with Email + Password)")
	fmt.Println("  POST   /auth/sms/send          (Send SMS OTP)")
	fmt.Println("  POST   /auth/sms/verify        (Verify SMS OTP)")
	fmt.Println("  POST   /auth/password/change   (Change Password)")
	fmt.Println("  GET    /api/admin/users        (List Users - Admin only)")

	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatal(err)
	}
}
