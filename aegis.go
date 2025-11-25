// Package aegis is the main authentication framework instance.
package aegis

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"github.com/theinventorylib/aegis/config"
	"github.com/theinventorylib/aegis/core"
	"github.com/theinventorylib/aegis/db"
	"github.com/theinventorylib/aegis/models"
	"github.com/theinventorylib/aegis/plugins"
	"github.com/theinventorylib/aegis/server"
)

// Aegis is the main authentication framework instance.
type Aegis struct {
	config  *config.Config
	db      db.Provider
	router  server.Router
	auth    *core.AuthService
	session *core.SessionService
	plugins []pluginRegistration
	mu      sync.RWMutex // Protects plugins slice for thread-safe access
}

// pluginRegistration holds a plugin with its registration metadata.
type pluginRegistration struct {
	plugin   plugins.Plugin
	priority int // Lower numbers = higher priority (initialized first)
}

// New creates a new Aegis instance with the provided options.
func New(opts ...config.Option) (*Aegis, error) {
	cfg := config.Default()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("Initializing Aegis framework",
			"session_expiry", cfg.SessionExpiry,
			"refresh_expiry", cfg.RefreshExpiry)
	}

	aegis := &Aegis{
		config: cfg,
		db:     cfg.DB,
		router: cfg.Router,
	}

	// Initialize core services
	sessionConfig := &core.SessionConfig{
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
		if cfg.Logger != nil {
			cfg.Logger.Info("Redis session storage enabled",
				"host", cfg.Redis.Host,
				"port", cfg.Redis.Port)
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

	if cfg.Logger != nil {
		cfg.Logger.Info("Core services initialized successfully",
			"argon2_time", cfg.Argon2Time,
			"argon2_memory_kb", cfg.Argon2Memory/1024)
	}

	// Apply custom ID generator if provided
	if cfg.IDGenerator != nil {
		core.SetCustomIDGenerator(cfg.IDGenerator)
		if cfg.Logger != nil {
			cfg.Logger.Debug("Custom ID generator configured")
		}
	}

	return aegis, nil
}

// MountRoutes mounts all authentication routes to the router with the given prefix.
// Plugins are mounted in priority order (lower priority number = mounted first).
func (a *Aegis) MountRoutes(prefix string) {
	// Mount core routes
	server.MountRoutes(a.router, a.auth, a.session, prefix)

	// Get sorted plugins for mounting
	sortedPlugins := a.getSortedPlugins()

	// Mount plugin routes in priority order
	for _, reg := range sortedPlugins {
		reg.plugin.MountRoutes(a.router, prefix)
	}
}

// Use registers a plugin with the Aegis instance with context support and default priority.
// Plugins registered with Use have priority 100 (executed after high-priority plugins).
// This is the canonical method for plugin registration.
//
// The context allows callers to:
//   - Set initialization timeouts
//   - Cancel plugin initialization
//   - Pass context-specific values to plugin Init
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//	err := aegis.Use(ctx, myPlugin)
func (a *Aegis) Use(ctx context.Context, plugin plugins.Plugin) error {
	return a.UseWithPriority(ctx, plugin, 100)
}

// UseWithPriority registers a plugin with an explicit priority.
// Lower priority values are initialized and mounted first.
//
// Priority guidelines:
//   - 0-50: Critical infrastructure plugins (database, logging)
//   - 51-99: High-priority auth plugins (password, OAuth)
//   - 100: Default priority (Use method)
//   - 101-150: Standard plugins (email, SMS)
//   - 151+: Low-priority plugins (admin dashboards, analytics)
//
// Example:
//
//	// Register password plugin with high priority
//	err := aegis.UseWithPriority(ctx, passwordPlugin, 60)
func (a *Aegis) UseWithPriority(ctx context.Context, plugin plugins.Plugin, priority int) error {
	// Log plugin registration attempt if logger is configured
	if a.config.Logger != nil {
		a.config.Logger.Info("Registering plugin",
			"name", plugin.Name(),
			"version", plugin.Version(),
			"priority", priority)
	}

	// Initialize plugin with context
	if err := plugin.Init(ctx, a); err != nil {
		if a.config.Logger != nil {
			a.config.Logger.Error("Plugin initialization failed",
				"name", plugin.Name(),
				"error", err.Error())
		}
		return fmt.Errorf("failed to initialize plugin %s: %w", plugin.Name(), err)
	}

	// Thread-safe plugin registration with priority
	a.mu.Lock()
	a.plugins = append(a.plugins, pluginRegistration{
		plugin:   plugin,
		priority: priority,
	})
	a.mu.Unlock()

	if a.config.Logger != nil {
		a.config.Logger.Info("Plugin registered successfully",
			"name", plugin.Name(),
			"priority", priority)
	}

	return nil
}

// RegisterPlugin is deprecated. Use Use instead.
// This method forwards to Use with context.Background().
//
// Deprecated: Use Use(ctx, plugin) for better context control.
func (a *Aegis) RegisterPlugin(plugin plugins.Plugin) error {
	return a.Use(context.Background(), plugin)
}

// GetDB returns the database provider
func (a *Aegis) GetDB() db.Provider {
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

// AuthMiddleware returns middleware that validates sessions and injects user into context.
// Panics if session service is nil (should never happen if New succeeded).
func (a *Aegis) AuthMiddleware() func(http.Handler) http.Handler {
	if a.session == nil {
		panic("aegis: session service is nil - this should never happen")
	}
	return core.AuthMiddleware(a.session)
}

// RequireAuth returns middleware that requires authentication.
// Panics if session service is nil (should never happen if New succeeded).
func (a *Aegis) RequireAuth() func(http.Handler) http.Handler {
	if a.session == nil {
		panic("aegis: session service is nil - this should never happen")
	}
	return core.RequireAuthMiddleware(a.session)
}

// GetUser returns the authenticated user from the request context
func (a *Aegis) GetUser(ctx context.Context) (*models.User, error) {
	return core.GetUser(ctx)
}

// Authenticated checks if the request context has an authenticated user
func (a *Aegis) Authenticated(ctx context.Context) bool {
	return core.Authenticated(ctx)
}

// GetPlugin retrieves a registered plugin by name.
// Returns (plugin, true) if found, (nil, false) if not found.
// This method is thread-safe.
func (a *Aegis) GetPlugin(name string) (plugins.Plugin, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, reg := range a.plugins {
		if reg.plugin.Name() == name {
			return reg.plugin, true
		}
	}
	return nil, false
}

// GetPlugins returns a copy of all registered plugins.
// Plugins are returned in priority order (lower priority first).
// Returns a copy to prevent external modification.
// This method is thread-safe.
func (a *Aegis) GetPlugins() []plugins.Plugin {
	sortedRegs := a.getSortedPlugins()

	// Extract plugins from registrations
	result := make([]plugins.Plugin, len(sortedRegs))
	for i, reg := range sortedRegs {
		result[i] = reg.plugin
	}
	return result
}

// getSortedPlugins returns plugins sorted by priority (lower priority first).
// Must be called with at least read lock held, or returns a locked copy.
func (a *Aegis) getSortedPlugins() []pluginRegistration {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Create a copy for sorting
	sorted := make([]pluginRegistration, len(a.plugins))
	copy(sorted, a.plugins)

	// Sort by priority (lower numbers first)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].priority < sorted[j].priority
	})

	return sorted
}
