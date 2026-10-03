package aegis

import "strings"

// MountOption customizes where Aegis mounts its routes for a single
// MountRoutes call.
//
// Options are applied in order, so later options win when they target the
// same plugin.
type MountOption func(*mountConfig)

// mountConfig holds the route placement requested by MountOptions.
type mountConfig struct {
	// corePath is the core email/password + session sub-path under the
	// MountRoutes prefix. Only meaningful when corePathSet is true.
	corePath    string
	corePathSet bool
	// pluginPaths maps plugin name -> prefix override.
	pluginPaths map[string]string
}

// WithCorePath sets the sub-path under the MountRoutes prefix where the core
// email/password and session routes are registered.
//
// The value is relative to the prefix passed to MountRoutes unless it starts
// with "/", in which case it is treated as an absolute path. The default is
// "default", which preserves the historical /{prefix}/default/... layout. An
// empty value mounts core routes directly under the prefix.
//
// Example:
//
//	a.MountRoutes("/auth", aegis.WithCorePath("")) // POST /auth/login
func WithCorePath(sub string) MountOption {
	return func(c *mountConfig) {
		c.corePath = sub
		c.corePathSet = true
	}
}

// WithPluginPrefix overrides where the named plugin mounts its routes.
//
// Path semantics:
//   - absolute (leading "/"): used verbatim; "/" is the server root
//   - relative: resolved under the MountRoutes prefix
//   - empty: mounted at the MountRoutes prefix root
//
// Unknown plugin names are ignored (a warning is logged when a logger is
// configured).
//
// Example:
//
//	a.MountRoutes("/auth",
//		aegis.WithPluginPrefix("admin", "/admin"),
//		aegis.WithPluginPrefix("organizations", "/orgs"),
//		aegis.WithPluginPrefix("openapi", "/"),
//	)
func WithPluginPrefix(name, path string) MountOption {
	return func(c *mountConfig) {
		if c.pluginPaths == nil {
			c.pluginPaths = make(map[string]string)
		}
		c.pluginPaths[name] = path
	}
}

// corePrefix resolves the full prefix for core routes.
func (c *mountConfig) corePrefix(prefix string) string {
	if !c.corePathSet {
		return joinMountPath(prefix, "default")
	}
	return resolveMountPath(prefix, c.corePath)
}

// pluginPrefix resolves the full prefix for one plugin.
func (c *mountConfig) pluginPrefix(prefix, name string) string {
	if override, ok := c.pluginPaths[name]; ok {
		return resolveMountPath(prefix, override)
	}
	return joinMountPath(prefix, name)
}

// resolveMountPath resolves an override against the base mount prefix.
// Absolute overrides escape the prefix; relative ones nest under it; "/" and
// "" address the server root and the prefix root respectively. The result is
// normalised: no trailing slash, and the server root is represented as "".
func resolveMountPath(prefix, path string) string {
	if strings.HasPrefix(path, "/") {
		return normalizeMountPath(path)
	}
	if path == "" {
		return normalizeMountPath(prefix)
	}
	return joinMountPath(prefix, path)
}

// joinMountPath concatenates base and sub with exactly one separating slash
// and a leading slash, e.g. ("/auth", "default") -> "/auth/default" and
// ("", "default") -> "/default".
func joinMountPath(base, sub string) string {
	base = strings.TrimRight(base, "/")
	sub = strings.Trim(strings.TrimLeft(sub, "/"), "/")
	switch {
	case sub == "":
		if base == "" {
			return "/"
		}
		return base
	case base == "":
		return "/" + sub
	default:
		return base + "/" + sub
	}
}

// normalizeMountPath trims trailing slashes; an all-slash path collapses to
// "" because the server root is represented without a trailing separator.
func normalizeMountPath(path string) string {
	return strings.TrimRight(path, "/")
}
