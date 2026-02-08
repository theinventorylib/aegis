// Package aegis provides the main authentication framework instance.
//
// Aegis is a comprehensive authentication and authorization framework for Go
// web applications. It provides:
//
//   - User authentication: Email/password, OAuth, JWT, SMS, email OTP
//   - Session management: Token-based sessions with Redis caching
//   - Password security: Argon2id hashing with OWASP-recommended parameters
//   - Rate limiting: Sliding window rate limiting with brute force protection
//   - Multi-tenancy: Organization-based isolation via plugins
//   - Plugin system: Extensible architecture for custom authentication methods
//   - Audit logging: Comprehensive security event tracking
//   - Database support: PostgreSQL, MySQL, SQLite with migration tools
//
// Architecture Modes:
//
//  1. Web Mode (Default):
//     - Best for: Server-Side Rendered apps (Templ, HTML templates), SPAs on same domain.
//     - Auth: HTTP-Only Cookies (secure, max-age, same-site).
//     - Security: CSRF Protection enabled (requires Master Secret).
//     - Bearer: Disabled by default (enable with config.WithBearerAuth(true)).
//
//  2. API Mode:
//     - Best for: Mobile apps, CLI tools, Service-to-Service, standalone frontends.
//     - Auth: "Authorization: Bearer <token>" header (auto-enabled).
//     - Security: CSRF checks skipped (cookies not used for auth).
//     - Enabled via: config.WithAPIOnlyMode(true).
//
//  3. Dual Mode (Web + API):
//     - Best for: Apps serving both browser clients and API consumers.
//     - Auth: Both cookies and Bearer tokens accepted.
//     - Security: CSRF Protection enabled for cookie-based requests.
//     - Enabled via: config.WithBearerAuth(true) (without APIMode).
//
// Quick Start:
//
//	package main
//
//	import (
//		"context"
//		"database/sql"
//		"net/http"
//
//		"github.com/theinventorylib/aegis"
//		"github.com/theinventorylib/aegis/config"
//		"github.com/go-chi/chi/v5"
//		_ "github.com/lib/pq"
//	)
//
//	func main() {
//		db, _ := sql.Open("postgres", "postgres://localhost/myapp?sslmode=disable")
//		r := chi.NewRouter()
//
//		// Create Aegis instance
//		cfg := config.Default().
//			WithDB(db).
//			WithRouter(r).
//			WithSecret([]byte("your-32-byte-secret-key-here!"))
//		a, _ := aegis.New(context.Background(), cfg)
//
//		// Mount authentication routes
//		a.MountRoutes("/auth")
//
//		// Use authentication middleware
//		r.Use(a.AuthMiddleware())
//
//		// Protected routes
//		r.Group(func(r chi.Router) {
//			r.Use(a.RequireAuth())
//			r.Get("/api/profile", profileHandler)
//		})
//
//		http.ListenAndServe(":8080", r)
//	}
package aegis

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/theinventorylib/aegis/auth"
	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/router"
)

// Aegis is the main entry point for the Aegis authentication framework.
//
// This struct coordinates all authentication subsystems:
//   - Core authentication: User/account/session/verification services
//   - Rate limiting: Request throttling and brute force protection
//   - Plugin system: Extensible authentication methods (OAuth, JWT, etc.)
//   - Middleware: HTTP handlers for authentication and authorization
//
// Aegis is thread-safe and can be shared across goroutines. All plugin
// registration and configuration must be done before serving HTTP requests.
//
// Architecture:
//   - auth: Core services (User, Account, Session, Verification)
//   - router: HTTP router for mounting routes
//   - plugins: Registered plugins (OAuth, JWT, Organizations, etc.)
//   - rateLimiter: Optional rate limiting for public endpoints
//   - loginAttemptTracker: Brute force protection for login endpoints
//
// Example:
//
//	a, err := aegis.New(ctx,
//		config.WithDB(db),
//		config.WithRouter(router),
//		config.WithSecret(secret),
//	)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Register plugins
//	a.Use(ctx, oauth.New(...))
//	a.Use(ctx, jwt.New(...))
//
//	// Mount routes
//	a.MountRoutes("/auth")
type Aegis struct {
	// config holds the framework configuration
	config *config.Config

	// router is the HTTP router for mounting routes
	router router.Router

	// auth provides core authentication services
	auth *core.AuthService

	// rateLimiter provides request rate limiting (optional)
	rateLimiter *core.RateLimiter

	// loginAttemptTracker provides brute force protection (optional)
	loginAttemptTracker *core.LoginAttemptTracker

	// mu protects the plugins map for concurrent access
	mu sync.RWMutex

	// plugins holds all registered plugins with their metadata
	plugins map[string]pluginRegistration
}

// pluginRegistration holds a plugin with its registration metadata.
//
// Plugins are initialized and mounted in priority order:
//   - Priority 0-49: System plugins (initialized first)
//   - Priority 50-99: Authentication plugins (OAuth, JWT, etc.)
//   - Priority 100+: Application plugins (default)
//
// Lower priority numbers are initialized and mounted first.
type pluginRegistration struct {
	// plugin is the registered plugin instance
	plugin plugins.Plugin

	// priority determines initialization/mounting order (lower = earlier)
	priority int
}

// New creates a new Aegis authentication framework instance.
//
// This function initializes all core services:
//   - Database connection validation
//   - Password hashing (Argon2id with OWASP parameters)
//   - Session management (with optional Redis caching)
//   - Rate limiting (if enabled)
//   - Login attempt tracking (brute force protection)
//   - Audit logging (security event tracking)
//
// Configuration is provided via functional options (config.With* functions).
// Required options:
//   - config.WithDB(db): Database connection (*sql.DB)
//   - config.WithRouter(router): HTTP router implementing router.Router
//   - config.WithSecret(secret): 32-byte master secret for key derivation
//
// Optional configuration:
//   - config.WithRedis(host, port, password, db): Enable Redis caching
//   - config.WithRateLimiting(): Enable rate limiting
//   - config.WithLogger(logger): Structured logging
//   - config.WithAuditLogger(logger): Security audit logging
//   - config.WithArgon2Time/WithArgon2Memory: Password hashing tuning
//
// Parameters:
//   - ctx: Context for initialization (can be canceled)
//   - opts: Configuration options (config.With* functions)
//
// Returns:
//   - *Aegis: Configured framework instance
//   - error: Configuration validation errors
//
// Example:
//
//	db, _ := sql.Open("postgres", "postgres://localhost/myapp?sslmode=disable")
//	router := chi.NewRouter()
//	secret := []byte("your-32-byte-secret-key-here!")
//
//	a, err := aegis.New(context.Background(),
//		config.WithDB(db),
//		config.WithRouter(router),
//		config.WithSecret(secret),
//		config.WithRedis("localhost", 6379, "", 0),
//		config.WithRateLimiting(),
//	)
//	if err != nil {
//		log.Fatal("Failed to initialize Aegis:", err)
//	}
func New(_ context.Context, cfg *config.Config) (*Aegis, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	aegis := &Aegis{
		config:  cfg,
		router:  cfg.Router,
		plugins: make(map[string]pluginRegistration),
	}

	// Apply ID generation configuration from config
	// This sets up the package-level ID generator that will be used by core.GenerateID()
	if cfg.IDGenerator != nil {
		// Custom generator provided - use it regardless of strategy
		core.SetCustomIDGenerator(cfg.IDGenerator)
		if cfg.Logger != nil {
			cfg.Logger.Info("ID generation configured with custom generator")
		}
	} else if cfg.IDStrategy != "" {
		// Strategy specified without custom generator
		core.SetIDStrategy(cfg.IDStrategy)
		if cfg.Logger != nil {
			cfg.Logger.Info("ID generation strategy configured", "strategy", cfg.IDStrategy)
		}
	}
	// If neither is set, the default (ULID) is already configured

	// Initialize core services
	cfg.Auth.DB = cfg.DB
	authConn := auth.New(cfg.Auth)

	hashConfig := &core.PasswordHasherConfig{
		Argon2Time:      cfg.Argon2Time,
		Argon2Memory:    cfg.Argon2Memory,
		Argon2Threads:   cfg.Argon2Threads,
		Argon2KeyLength: cfg.Argon2KeyLength,
	}

	auditLogger := cfg.AuditLogger
	if auditLogger == nil {
		auditLogger = &core.NoOpAuditLogger{}
	}

	// Initialize sub-services via AuthService
	var loginAttemptTracker *core.LoginAttemptTracker
	var redisClient *redis.Client

	// Initialize Redis client if configured
	if cfg.Redis != nil {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
	}

	// Initialize rate limiting if enabled
	if cfg.RateLimitEnabled {
		rateLimitConfig := cfg.RateLimitConfig
		if rateLimitConfig == nil {
			rateLimitConfig = core.DefaultRateLimitConfig()
		}
		aegis.rateLimiter = core.NewRateLimiter(rateLimitConfig, redisClient, auditLogger)

		loginAttemptConfig := cfg.LoginAttemptConfig
		if loginAttemptConfig == nil {
			loginAttemptConfig = core.DefaultLoginAttemptConfig()
		}
		aegis.loginAttemptTracker = core.NewLoginAttemptTracker(loginAttemptConfig, redisClient)
		loginAttemptTracker = aegis.loginAttemptTracker

		if cfg.Logger != nil {
			cfg.Logger.Info("Rate limiting enabled",
				"requests_per_window", rateLimitConfig.RequestsPerWindow,
				"window_duration", rateLimitConfig.WindowDuration)
		}
	}

	// Create AuthService
	aegis.auth = core.NewAuthService(cfg.CoreAuth, authConn, hashConfig, auditLogger, loginAttemptTracker)

	// Enable Bearer token authentication if configured
	// Bearer auth is auto-enabled in API mode unless explicitly disabled
	if cfg.IsBearerAuthEnabled() {
		aegis.auth.Session.EnableBearerAuth()
		if cfg.Logger != nil {
			cfg.Logger.Info("Bearer token authentication enabled")
		}
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("Core services initialized successfully",
			"argon2_time", cfg.Argon2Time)
	}

	return aegis, nil
}

// MountRoutes mounts all authentication routes to the router with the given prefix.
//
// This function registers HTTP handlers for:
//
// Core Routes:
//
//	POST {prefix}/login - Email+password login (if enabled)
//	POST {prefix}/signup - Email+password registration (if enabled)
//	POST {prefix}/logout - Logout and invalidate session
//	GET  {prefix}/user - Get current user data
//	GET  {prefix}/session - Get current session info
//	GET  {prefix}/sessions - List all user sessions
//	POST {prefix}/session/refresh - Refresh session with refresh token
//	DELETE {prefix}/sessions/:id - Revoke specific session
//	DELETE {prefix}/sessions - Revoke all sessions
//
// Plugin Routes:
//
// Each registered plugin can mount additional routes under {prefix}/{plugin-name}:
//   - OAuth: {prefix}/oauth/* - OAuth provider flows (Google, GitHub, etc.)
//   - JWT: {prefix}/jwt/* - JWT token generation and validation
//   - Organizations: {prefix}/organizations/* - Organization management
//   - Admin: {prefix}/admin/* - Administrative user management
//   - EmailOTP: {prefix}/email-otp/* - Email verification codes
//   - SMS: {prefix}/sms/* - SMS verification codes
//
// Plugin Mounting Order:
//
// Plugins are mounted in priority order (lower priority numbers first):
//  1. System plugins (priority 0-49)
//  2. Authentication plugins (priority 50-99)
//  3. Application plugins (priority 100+, default)
//
// This ensures dependencies are initialized correctly (e.g., Organizations
// plugin is mounted before Admin plugin which depends on it).
//
// Parameters:
//   - prefix: URL prefix for all routes (e.g., "/auth", "/api/v1/auth")
//
// Example:
//
//	a := aegis.New(...)
//	a.Use(ctx, oauth.New(...))  // Mounts at /auth/oauth/*
//	a.Use(ctx, jwt.New(...))    // Mounts at /auth/jwt/*
//	a.MountRoutes("/auth")      // Core routes at /auth/*
//
//	// Routes available:
//	// POST /auth/login
//	// POST /auth/signup
//	// POST /auth/logout
//	// GET  /auth/user
//	// POST /auth/oauth/google/login
//	// POST /auth/jwt/token
//	// etc.
func (a *Aegis) MountRoutes(prefix string) {
	// Mount core routes under the "Default" subpath with optional rate limiter
	var coreAuth *core.AuthConfig
	if a.config != nil {
		coreAuth = a.config.CoreAuth
	}
	corePrefix := prefix + "/default"
	router.MountRoutes(a.router, a.auth, coreAuth, corePrefix, a.rateLimiter)

	// Get sorted plugins for mounting
	sortedPlugins := a.GetPlugins()
	for _, p := range sortedPlugins {
		p.MountRoutes(a.router, prefix+"/"+p.Name())
	}
}

// Use registers a plugin with the Aegis instance using default priority (100).
//
// Plugins extend Aegis with additional authentication methods and features:
//   - OAuth: Social login (Google, GitHub, Facebook, etc.)
//   - JWT: Token-based authentication
//   - Organizations: Multi-tenant organization support
//   - Admin: Administrative user management dashboard
//   - EmailOTP: Email verification codes
//   - SMS: Phone number verification
//   - Bearer: Bearer token authentication
//   - OpenAPI: Auto-generated API documentation
//
// Plugin initialization:
//  1. Plugin.Init() is called with the Aegis instance
//  2. Plugin can access database, auth services, rate limiter, etc.
//  3. Plugin is registered in the plugin map
//  4. Plugin routes are NOT mounted yet (call MountRoutes after all plugins registered)
//
// Plugin registration must happen BEFORE MountRoutes() is called.
//
// Default priority is 100 (application plugins). For custom priority, use UseWithPriority.
//
// Parameters:
//   - ctx: Context for plugin initialization (can be canceled)
//   - plugin: Plugin instance to register
//
// Returns:
//   - error: Plugin initialization errors
//
// Example:
//
//	a := aegis.New(...)
//
//	// Register OAuth plugin for social login
//	oauthPlugin := oauth.New(oauthConfig, keyManager, dialect)
//	if err := a.Use(ctx, oauthPlugin); err != nil {
//		log.Fatal("Failed to register OAuth plugin:", err)
//	}
//
//	// Register JWT plugin for token authentication
//	jwtPlugin := jwt.New(jwtConfig, keyManager, dialect)
//	if err := a.Use(ctx, jwtPlugin); err != nil {
//		log.Fatal("Failed to register JWT plugin:", err)
//	}
//
//	// Mount all routes (including plugin routes)
//	a.MountRoutes("/auth")
func (a *Aegis) Use(ctx context.Context, plugin plugins.Plugin) error {
	return a.UseWithPriority(ctx, plugin, 100)
}

// UseWithPriority registers a plugin with an explicit initialization priority.
//
// Priority controls the order of plugin initialization and route mounting:
//   - Priority 0-49: System plugins (organizations, core extensions)
//   - Priority 50-99: Authentication plugins (OAuth, JWT, EmailOTP, SMS)
//   - Priority 100+: Application plugins (custom plugins, default)
//
// Lower priority numbers are initialized and mounted first. This allows
// plugins to depend on other plugins (e.g., Admin plugin depends on
// Organizations plugin, so Organizations has lower priority).
//
// Use this when:
//   - Your plugin depends on another plugin
//   - You need to control route mounting order
//   - You're building system-level plugins
//
// Most applications should use Use() with default priority (100).
//
// Parameters:
//   - ctx: Context for plugin initialization
//   - plugin: Plugin instance to register
//   - priority: Initialization/mounting priority (lower = earlier)
//
// Example:
//
//	// Register Organizations plugin first (priority 10)
//	orgPlugin := organizations.New(orgConfig, dialect)
//	a.UseWithPriority(ctx, orgPlugin, 10)
//
//	// Register Admin plugin second (priority 50, depends on Organizations)
//	adminPlugin := admin.New(adminConfig, dialect)
//	a.UseWithPriority(ctx, adminPlugin, 50)
func (a *Aegis) UseWithPriority(ctx context.Context, plugin plugins.Plugin, priority int) error {
	// Log plugin registration attempt if logger is configured
	if a.config != nil && a.config.Logger != nil {
		a.config.Logger.Info("Registering plugin",
			"name", plugin.Name(),
			"priority", priority)
	}

	// Instantiate the multi-generic Aegis interface for the plugin
	// This allow the plugin to use any models while we use defaults internally
	pAegis := &pluginAegisWrapper{Aegis: a}

	// Initialize plugin
	if err := plugin.Init(ctx, pAegis); err != nil {
		return err
	}

	a.mu.Lock()
	a.plugins[plugin.Name()] = pluginRegistration{
		plugin:   plugin,
		priority: priority,
	}
	a.mu.Unlock()

	return nil
}

// GetAuthService returns the core auth service
func (a *Aegis) GetAuthService() *core.AuthService {
	return a.auth
}

// DeriveSecret derives a purpose-specific secret from the master secret.
func (a *Aegis) DeriveSecret(purpose string) []byte {
	return a.config.DeriveSecret(purpose)
}

// GetDB returns the database connection
func (a *Aegis) GetDB() *sql.DB {
	return a.config.DB
}

// ValidateSchemaRequirements validates that the database has the required tables.
func (a *Aegis) ValidateSchemaRequirements(ctx context.Context, requirements []plugins.SchemaRequirement) error {
	// new schema validator
	validator := core.NewSchemaValidator(a.config.DB)
	return validator.ValidateRequirements(ctx, requirements)
}

// AuthMiddleware returns middleware that validates sessions and injects user into context.
//
// This is the PRIMARY middleware for Aegis authentication. It:
//  1. Initializes request context (if not already done)
//  2. Extracts session token from cookie or Authorization header
//  3. Validates the session (database + Redis cache)
//  4. Loads the authenticated user
//  5. Creates EnrichedUser for plugin extensions
//  6. Runs all UserEnricher plugins to populate extension fields
//  7. Injects user and session into request context
//
// After this middleware, handlers can access:
//   - core.GetUser(ctx): Base user data
//   - core.GetEnrichedUser(ctx): User with plugin extensions (role, org, etc.)
//   - core.GetSession(ctx): Current session data
//
// Plugin Extensions:
//
// Plugins implementing UserEnricher automatically extend the user object:
//   - Organizations plugin: Adds organization_id, organization fields
//   - Admin plugin: Adds role, permissions fields
//   - Custom plugins: Add any custom fields
//
// All extension fields are flattened into the top-level JSON response.
//
// Unauthenticated Requests:
//
// This middleware does NOT block unauthenticated requests. It simply skips
// user injection if no valid session is found. Use RequireAuth() middleware
// for routes that MUST be authenticated.
//
// Usage:
//
//	router := chi.NewRouter()
//
//	// Apply to all routes (authentication is optional)
//	router.Use(aegis.AuthMiddleware())
//
//	// Public routes (no auth required)
//	router.Get("/api/public", publicHandler)
//
//	// Protected routes (auth required)
//	router.Group(func(r chi.Router) {
//		r.Use(aegis.RequireAuth())
//		r.Get("/api/profile", profileHandler)
//	})
func (a *Aegis) AuthMiddleware() func(http.Handler) http.Handler {
	if a.auth.Session == nil {
		panic("aegis: session service is nil - this should never happen")
	}

	// Get the base auth middleware from core
	baseMiddleware := core.AuthMiddleware(a.auth.Session)

	return func(next http.Handler) http.Handler {
		return baseMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// If authenticated, run all plugin enrichers
			if enriched := core.GetEnrichedUser(ctx); enriched != nil {
				// Run all registered plugin enrichers
				a.runUserEnrichers(ctx, enriched)
			}

			next.ServeHTTP(w, r)
		}))
	}
}

// runUserEnrichers runs all plugins that implement UserEnricher.
func (a *Aegis) runUserEnrichers(ctx context.Context, user *core.EnrichedUser) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, reg := range a.plugins {
		if enricher, ok := reg.plugin.(plugins.UserEnricher); ok {
			// Errors from individual enrichers shouldn't fail the request
			err := enricher.EnrichUser(ctx, user)
			_ = err
		}
	}
}

// RequireAuth returns middleware that requires authentication.
//
// This middleware BLOCKS unauthenticated requests with 401 Unauthorized.
// Use this for protected routes that require a valid session.
//
// This middleware should be applied AFTER AuthMiddleware() in the middleware chain:
//  1. AuthMiddleware() - Validates session and injects user (optional)
//  2. RequireAuth() - Blocks if no user in context (required)
//
// Response on unauthenticated request:
//
//	401 Unauthorized
//	{
//	  "success": false,
//	  "error": "Authentication required"
//	}
//
// Usage:
//
//	router := chi.NewRouter()
//	router.Use(aegis.AuthMiddleware())
//
//	// Protected group - all routes require authentication
//	router.Group(func(r chi.Router) {
//		r.Use(aegis.RequireAuth())
//
//		r.Get("/api/profile", profileHandler)
//		r.Put("/api/profile", updateProfileHandler)
//		r.Get("/api/settings", settingsHandler)
//	})
//
//	// Per-route protection
//	router.Get("/api/public", publicHandler)  // No auth required
//	router.With(aegis.RequireAuth()).Get("/api/private", privateHandler)  // Auth required
func (a *Aegis) RequireAuth() func(http.Handler) http.Handler {
	if a.auth.Session == nil {
		panic("aegis: session service is nil - this should never happen")
	}
	return core.RequireAuthMiddleware(a.auth.Session)
}

// RateLimitMiddleware returns rate limiting middleware (if enabled).
//
// This middleware implements sliding window rate limiting to prevent:
//   - API abuse (too many requests)
//   - Brute force attacks (password guessing)
//   - DoS attacks (service exhaustion)
//
// Rate limiting is configured via config.WithRateLimiting() during initialization.
// Default limits:
//   - General endpoints: 100 requests per minute
//   - Auth endpoints: 10 requests per minute
//
// Rate limiting can use:
//   - Redis: Distributed rate limiting across multiple servers (recommended)
//   - In-memory: Single-server rate limiting (development/testing)
//
// When a client exceeds the rate limit:
//
//	429 Too Many Requests
//	{
//	  "success": false,
//	  "error": "Rate limit exceeded. Please try again later."
//	}
//
// Returns nil if rate limiting is not enabled (no-op).
//
// Usage:
//
//	// Apply rate limiting to all routes
//	if rateLimitMW := aegis.RateLimitMiddleware(); rateLimitMW != nil {
//		router.Use(rateLimitMW)
//	}
//
//	// Apply rate limiting to specific routes
//	router.With(aegis.RateLimitMiddleware()).Post("/api/expensive-operation", handler)
func (a *Aegis) RateLimitMiddleware() func(http.Handler) http.Handler {
	if a.rateLimiter == nil {
		return nil
	}
	return core.RateLimitMiddleware(a.rateLimiter)
}

// GetRateLimiter returns the rate limiter instance (may be nil if not enabled).
//
// Use this to access rate limiting functionality directly:
//   - Check rate limits programmatically
//   - Implement custom rate limiting logic
//   - Track rate limit status
//
// Returns nil if rate limiting is not enabled via config.WithRateLimiting().
func (a *Aegis) GetRateLimiter() *core.RateLimiter {
	return a.rateLimiter
}

// GetLoginAttemptTracker returns the login attempt tracker (may be nil if not enabled).
//
// The login attempt tracker provides brute force protection by:
//   - Counting failed login attempts per email/username
//   - Locking accounts after too many failures (default: 5 in 15 minutes)
//   - Automatically unlocking accounts after timeout (default: 15 minutes)
//
// Use this to:
//   - Check if an account is locked
//   - Manually unlock accounts (admin functionality)
//   - Customize lockout behavior
//
// Returns nil if rate limiting is not enabled via config.WithRateLimiting().
func (a *Aegis) GetLoginAttemptTracker() *core.LoginAttemptTracker {
	return a.loginAttemptTracker
}

// GetPlugins returns all registered plugins sorted by priority.
//
// Plugins are returned in priority order (lowest priority number first).
// This is the same order used for initialization and route mounting.
//
// Use this to:
//   - List all active plugins
//   - Inspect plugin configuration
//   - Debug plugin initialization order
//
// Returns a copy of the plugin list (modifications don't affect Aegis).
//
// Example:
//
//	for _, plugin := range aegis.GetPlugins() {
//		fmt.Printf("Plugin: %s\n", plugin.Name())
//	}
func (a *Aegis) GetPlugins() []plugins.Plugin {
	a.mu.RLock()
	defer a.mu.RUnlock()

	regs := make([]pluginRegistration, 0, len(a.plugins))
	for _, reg := range a.plugins {
		regs = append(regs, reg)
	}

	// Sort by priority
	sort.Slice(regs, func(i, j int) bool {
		return regs[i].priority < regs[j].priority
	})

	result := make([]plugins.Plugin, 0, len(regs))
	for _, reg := range regs {
		result = append(result, reg.plugin)
	}
	return result
}

// GetPlugin retrieves a registered plugin by name.
//
// Use this to access plugin-specific functionality after registration:
//
// Parameters:
//   - name: Plugin name (from plugin.Name())
//
// Returns:
//   - plugins.Plugin: The plugin instance
//   - bool: true if found, false otherwise
//
// Example:
//
//	if plugin, ok := aegis.GetPlugin("oauth"); ok {
//		// Plugin is registered, use it
//		oauthPlugin := plugin.(*oauth.Plugin)
//		// ...
//	}
func (a *Aegis) GetPlugin(name string) (plugins.Plugin, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	reg, ok := a.plugins[name]
	if ok {
		return reg.plugin, true
	}
	return nil, false
}

// GetPluginTyped retrieves a plugin by name and type-casts it to the expected type.
//
// This is a generic helper that combines GetPlugin() with type assertion.
// Use this when you know the plugin type and want compile-time type safety.
//
// Parameters:
//   - a: Aegis instance
//   - name: Plugin name
//
// Returns:
//   - T: Plugin instance of the expected type
//   - bool: true if found and type matches, false otherwise
//
// Example:
//
//	oauthPlugin, ok := aegis.GetPluginTyped[*oauth.Plugin](aegis, "oauth")
//	if ok {
//		// oauthPlugin is *oauth.Plugin, type-safe!
//		providers := oauthPlugin.GetProviders()
//	}
func GetPluginTyped[T plugins.Plugin](a *Aegis, name string) (T, bool) {
	var zero T
	p, ok := a.GetPlugin(name)
	if !ok {
		return zero, false
	}

	typed, ok := p.(T)
	return typed, ok
}

// pluginAegisWrapper implements the generic plugins.Aegis interface by wrapping a non-generic Aegis instance.
type pluginAegisWrapper struct {
	*Aegis
}

func (w *pluginAegisWrapper) GetAuthService() *core.AuthService {
	// Re-wrap the single-generic AuthService into the multi-generic interface if needed.
	// Actually, the framework's GetAuthService returns *core.AuthService .
	// This wrapper needs to return the multi-parameter version for the interface.
	// Since A, V, S are pinned anyway, this is just a type alias/cast.
	return w.Aegis.GetAuthService()
}

func (w *pluginAegisWrapper) GetLogger() config.Logger {
	return w.config.Logger
}

func (w *pluginAegisWrapper) GetRateLimiter() *core.RateLimiter {
	return w.rateLimiter
}

func (w *pluginAegisWrapper) DB() *sql.DB {
	return w.config.DB
}

func (w *pluginAegisWrapper) GetPlugin(name string) (plugins.Plugin, bool) {
	return w.Aegis.GetPlugin(name)
}
