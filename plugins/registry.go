package plugins

import (
	"context"
	"sync"

	"github.com/theinventorylib/aegis/core"
)

var (
	registry = make(map[string]Plugin)
	mu       sync.RWMutex
)

// Register adds a plugin to the global registry
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Name()] = p
}

// Get retrieves a plugin by name
func Get(name string) (Plugin, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// List returns all registered plugins
func List() []Plugin {
	mu.RLock()
	defer mu.RUnlock()
	list := make([]Plugin, 0, len(registry))
	for _, p := range registry {
		list = append(list, p)
	}
	return list
}

// Clear removes all plugins from the registry
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Plugin)
}

// =============================================================================
// Plugin Context Helpers
// =============================================================================
// These re-export core context functions for convenient plugin access.
// Plugins can use these instead of importing core directly for context operations.

// PluginData is re-exported from core for plugin convenience
type PluginData = core.PluginData

// RequestMeta is re-exported from core for plugin convenience
type RequestMeta = core.RequestMeta

// GetPluginData retrieves the plugin data store from context.
// For plugin-internal data (not exposed in API responses), use namespaced keys.
func GetPluginData(ctx context.Context) *PluginData {
	return core.GetPluginData(ctx)
}

// GetPluginValue is a convenience function to get a plugin value from context.
func GetPluginValue(ctx context.Context, key string) interface{} {
	return core.GetPluginValue(ctx, key)
}

// SetPluginValue is a convenience function to set a plugin value in context.
// For plugin-internal data (not exposed in API responses), use namespaced keys.
func SetPluginValue(ctx context.Context, key string, value interface{}) {
	core.SetPluginValue(ctx, key, value)
}

// GetRequestMeta retrieves request metadata from context.
func GetRequestMeta(ctx context.Context) *RequestMeta {
	return core.GetRequestMeta(ctx)
}

// GetRequestID retrieves the request ID from context (useful for logging/tracing).
func GetRequestID(ctx context.Context) string {
	return core.GetRequestID(ctx)
}

// GetIPAddress retrieves the client IP from context metadata.
func GetIPAddress(ctx context.Context) string {
	return core.GetIPAddress(ctx)
}

// GetUserAgent retrieves the user agent from context metadata.
func GetUserAgent(ctx context.Context) string {
	return core.GetUserAgent(ctx)
}

// GetUserID retrieves the authenticated user's ID from context.
// Returns empty string if not authenticated.
func GetUserID(ctx context.Context) string {
	return core.GetUserID(ctx)
}

// =============================================================================
// Enriched User Helpers
// =============================================================================

// EnrichedUser is re-exported from core for plugin convenience
type EnrichedUser = core.EnrichedUser

// GetEnrichedUser retrieves the enriched user from context.
// Returns nil if not authenticated.
func GetEnrichedUser(ctx context.Context) *EnrichedUser {
	return core.GetEnrichedUser(ctx)
}

// ExtendUser adds extension data to the enriched user in context.
// Use simple field names - these become top-level fields in JSON responses.
//
// Example:
//
//	plugins.ExtendUser(ctx, "role", "admin")
//	plugins.ExtendUser(ctx, "claims", claims)
//	plugins.ExtendUser(ctx, "organizations", []string{"org1", "org2"})
//
// These produce JSON like: {"id": "...", "email": "...", "role": "admin", "organizations": [...]}
func ExtendUser(ctx context.Context, key string, value interface{}) {
	core.ExtendUser(ctx, key, value)
}

// GetUserExtension retrieves a specific extension from the enriched user.
func GetUserExtension(ctx context.Context, key string) interface{} {
	return core.GetUserExtension(ctx, key)
}

// GetUserExtensionString retrieves a string extension from the enriched user.
func GetUserExtensionString(ctx context.Context, key string) string {
	return core.GetUserExtensionString(ctx, key)
}

// GetUserExtensionBool retrieves a bool extension from the enriched user.
func GetUserExtensionBool(ctx context.Context, key string) bool {
	return core.GetUserExtensionBool(ctx, key)
}

// =============================================================================
// User Enrichment
// =============================================================================

// RunUserEnrichers runs all registered plugins that implement UserEnricher.
// This populates the EnrichedUser with plugin-specific extension fields.
// Called automatically by middleware after authentication.
func RunUserEnrichers(ctx context.Context, user *core.EnrichedUser) error {
	mu.RLock()
	defer mu.RUnlock()

	for _, p := range registry {
		if enricher, ok := p.(UserEnricher); ok {
			if err := enricher.EnrichUser(ctx, user); err != nil {
				// Log error but continue with other enrichers
				// Individual plugin errors shouldn't fail the request
				continue
			}
		}
	}
	return nil
}

// RunUserEnrichersByName runs specific plugins' user enrichers by name.
// Useful when you only want to enrich with specific plugins.
func RunUserEnrichersByName(ctx context.Context, user *core.EnrichedUser, pluginNames ...string) error {
	mu.RLock()
	defer mu.RUnlock()

	for _, name := range pluginNames {
		if p, ok := registry[name]; ok {
			if enricher, ok := p.(UserEnricher); ok {
				if err := enricher.EnrichUser(ctx, user); err != nil {
					continue
				}
			}
		}
	}
	return nil
}
