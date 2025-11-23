package aegis

import (
	"context"
	"fmt"
	"net/http"

	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// Aegis is the main authentication framework instance
type Aegis struct {
	config  *config.Config
	db      db.DBProvider
	router  server.Router
	auth    *core.AuthService
	session *core.SessionService
	plugins []plugins.Plugin
}

// New creates a new Aegis instance with the provided options
func New(opts ...config.Option) (*Aegis, error) {
	cfg := config.Default()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	aegis := &Aegis{
		config: cfg,
		db:     cfg.DB,
		router: cfg.Router,
	}

	// Initialize core services
	sessionConfig := &core.SessionConfig{
		JWTSecret:     cfg.JWTSecret,
		SessionExpiry: cfg.SessionExpiry,
		RefreshExpiry: cfg.RefreshExpiry,
		CookieSettings: core.CookieSettings{
			Domain:   cfg.CookieDomain,
			Secure:   cfg.CookieSecure,
			HTTPOnly: cfg.CookieHTTPOnly,
			SameSite: cfg.CookieSameSite,
		},
	}

	if cfg.Redis != nil {
		sessionConfig.Redis = &core.RedisConfig{
			Host:     cfg.Redis.Host,
			Port:     cfg.Redis.Port,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		}
	}

	hashConfig := &core.PasswordHasherConfig{
		Argon2Time:      cfg.Argon2Time,
		Argon2Memory:    cfg.Argon2Memory,
		Argon2Threads:   cfg.Argon2Threads,
		Argon2KeyLength: cfg.Argon2KeyLength,
	}

	aegis.session = core.NewSessionService(cfg.DB, sessionConfig)
	aegis.auth = core.NewAuthService(cfg.DB, aegis.session, hashConfig)

	// Apply custom ID generator if provided
	if cfg.IDGenerator != nil {
		core.SetCustomIDGenerator(cfg.IDGenerator)
	}

	return aegis, nil
}

// MountRoutes mounts all authentication routes to the router with the given prefix
func (a *Aegis) MountRoutes(prefix string) {
	server.MountRoutes(a.router, a.auth, a.session, prefix)
}

// Use registers a plugin with the Aegis instance
func (a *Aegis) Use(plugin plugins.Plugin) error {
	ctx := context.Background()

	// Initialize plugin with context
	if err := plugin.Init(ctx, a); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", plugin.Name(), err)
	}

	// Auto-mount plugin routes if router is available
	if a.router != nil {
		plugin.MountRoutes(a.router, "/api")
	}

	a.plugins = append(a.plugins, plugin)
	return nil
}

// RegisterPlugin is an alias for Use for clarity
func (a *Aegis) RegisterPlugin(plugin plugins.Plugin) error {
	return a.Use(plugin)
}

// GetDB returns the database provider
func (a *Aegis) GetDB() db.DBProvider {
	return a.db
}

// GetRouter returns the router
func (a *Aegis) GetRouter() server.Router {
	return a.router
}

// GetConfig returns the configuration
func (a *Aegis) GetConfig() *config.Config {
	return a.config
}

// GetSessionService returns the session service
func (a *Aegis) GetSessionService() *core.SessionService {
	return a.session
}

// AuthMiddleware returns middleware that validates sessions and injects user into context
func (a *Aegis) AuthMiddleware() func(http.Handler) http.Handler {
	return core.AuthMiddleware(a.session)
}

// RequireAuth returns middleware that requires authentication
func (a *Aegis) RequireAuth() func(http.Handler) http.Handler {
	return core.RequireAuthMiddleware(a.session)
}

// GetUser returns the authenticated user from the request context
func (a *Aegis) GetUser(ctx context.Context) (*core.User, error) {
	return core.GetUser(ctx)
}

// Authenticated checks if the request context has an authenticated user
func (a *Aegis) Authenticated(ctx context.Context) bool {
	return core.Authenticated(ctx)
}
