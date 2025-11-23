package plugins

import "sync"

// registry holds all registered plugins
var (
	registry = make(map[string]Plugin)
	mu       sync.RWMutex
)

// Register adds a plugin to the global registry
// This allows plugins to self-register via init() functions
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Name()] = p
}

// Get retrieves a plugin by name from the registry
func Get(name string) (Plugin, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// All returns all registered plugins
func All() []Plugin {
	mu.RLock()
	defer mu.RUnlock()
	plugins := make([]Plugin, 0, len(registry))
	for _, p := range registry {
		plugins = append(plugins, p)
	}
	return plugins
}

// Unregister removes a plugin from the registry
// Useful for testing or dynamic plugin management
func Unregister(name string) {
	mu.Lock()
	defer mu.Unlock()
	delete(registry, name)
}

// Clear removes all plugins from the registry
// Primarily for testing purposes
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Plugin)
}
