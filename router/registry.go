package router

import (
	"net/http"
	"sort"
	"strings"
	"sync"
)

// RouteEntry describes a single (method, path) registration captured by
// the route registry. Owner identifies the logical source of the route
// (e.g. a plugin name, or "core" for framework-level routes).
type RouteEntry struct {
	Method string
	Path   string
	Owner  string
}

// ConflictFunc is invoked whenever the registry observes a second
// registration of the same (method, path) under a different owner. It
// is informational — the registry still records the duplicate entry
// and forwards the registration to the underlying router. Callers wire
// this to their logger of choice.
//
// Passing nil disables conflict reporting.
type ConflictFunc func(method, path, newOwner, prevOwner string)

// Registry records every route mounted through a Recorder. It exists
// for two reasons:
//
//  1. Conflict detection. Most underlying routers (chi, mux, gin) silently
//     accept a duplicate (method, path) registration and either overwrite
//     the previous handler or panic at runtime. The registry surfaces
//     these collisions through the supplied ConflictFunc as soon as the
//     second registration happens, naming both owners.
//
//  2. Introspection. Tools like an OpenAPI generator, an audit dashboard,
//     or a debug endpoint want a complete inventory of the HTTP surface
//     area of the application without re-implementing route tracking.
//
// The Registry has no opinion about routing semantics — it only observes.
type Registry struct {
	mu      sync.RWMutex
	entries []RouteEntry
	// seen maps "<METHOD> <path>" to the first owner that claimed the
	// route, so duplicate detection is a single map lookup.
	seen map[string]string
}

// NewRegistry returns an empty registry. A single registry is typically
// shared across every Recorder created during application startup.
func NewRegistry() *Registry {
	return &Registry{seen: make(map[string]string)}
}

// Routes returns a snapshot of all recorded routes, sorted by path then
// method. The returned slice is detached from the registry; callers may
// mutate it freely.
func (r *Registry) Routes() []RouteEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RouteEntry, len(r.entries))
	copy(out, r.entries)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}

// record adds an entry, invoking onConflict if the (method, path) was
// already claimed by a different owner.
func (r *Registry) record(method, path, owner string, onConflict ConflictFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := method + " " + path
	if prev, ok := r.seen[key]; ok && prev != owner && onConflict != nil {
		onConflict(method, path, owner, prev)
	}
	if _, ok := r.seen[key]; !ok {
		r.seen[key] = owner
	}
	r.entries = append(r.entries, RouteEntry{Method: method, Path: path, Owner: owner})
}

// NewRecorder wraps inner so every route registered through the returned
// Router (and any GroupRouter derived from it) is recorded in reg under
// the given owner. prefix is the absolute path that will be prepended to
// each registration before recording (so an OpenAPI list shows fully
// qualified URLs); pass "" if the caller already passes absolute paths.
//
// onConflict is called for any duplicate (method, path) registered under
// a different owner; pass nil to disable conflict reporting.
func NewRecorder(inner Router, reg *Registry, owner, prefix string, onConflict ConflictFunc) Router {
	return &recordingRouter{inner: inner, reg: reg, owner: owner, prefix: prefix, onConflict: onConflict}
}

// recordingRouter decorates a Router and forwards every registration
// to both the registry and the wrapped router.
type recordingRouter struct {
	inner      Router
	reg        *Registry
	owner      string
	prefix     string
	onConflict ConflictFunc
}

func (r *recordingRouter) GET(path string, h http.HandlerFunc) {
	r.reg.record("GET", JoinPath(r.prefix, path), r.owner, r.onConflict)
	r.inner.GET(path, h)
}
func (r *recordingRouter) POST(path string, h http.HandlerFunc) {
	r.reg.record("POST", JoinPath(r.prefix, path), r.owner, r.onConflict)
	r.inner.POST(path, h)
}
func (r *recordingRouter) PUT(path string, h http.HandlerFunc) {
	r.reg.record("PUT", JoinPath(r.prefix, path), r.owner, r.onConflict)
	r.inner.PUT(path, h)
}
func (r *recordingRouter) PATCH(path string, h http.HandlerFunc) {
	r.reg.record("PATCH", JoinPath(r.prefix, path), r.owner, r.onConflict)
	r.inner.PATCH(path, h)
}
func (r *recordingRouter) DELETE(path string, h http.HandlerFunc) {
	r.reg.record("DELETE", JoinPath(r.prefix, path), r.owner, r.onConflict)
	r.inner.DELETE(path, h)
}
func (r *recordingRouter) Use(mw func(http.Handler) http.Handler) { r.inner.Use(mw) }
func (r *recordingRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.inner.ServeHTTP(w, req)
}
func (r *recordingRouter) Group(path, name string) GroupRouter {
	return &recordingGroup{
		inner:      r.inner.Group(path, name),
		reg:        r.reg,
		owner:      r.owner,
		prefix:     JoinPath(r.prefix, path),
		onConflict: r.onConflict,
	}
}

// recordingGroup mirrors recordingRouter for the GroupRouter interface,
// carrying the accumulated prefix forward into nested groups.
type recordingGroup struct {
	inner      GroupRouter
	reg        *Registry
	owner      string
	prefix     string
	onConflict ConflictFunc
}

func (g *recordingGroup) GET(path string, h http.HandlerFunc) {
	g.reg.record("GET", JoinPath(g.prefix, path), g.owner, g.onConflict)
	g.inner.GET(path, h)
}
func (g *recordingGroup) POST(path string, h http.HandlerFunc) {
	g.reg.record("POST", JoinPath(g.prefix, path), g.owner, g.onConflict)
	g.inner.POST(path, h)
}
func (g *recordingGroup) PUT(path string, h http.HandlerFunc) {
	g.reg.record("PUT", JoinPath(g.prefix, path), g.owner, g.onConflict)
	g.inner.PUT(path, h)
}
func (g *recordingGroup) PATCH(path string, h http.HandlerFunc) {
	g.reg.record("PATCH", JoinPath(g.prefix, path), g.owner, g.onConflict)
	g.inner.PATCH(path, h)
}
func (g *recordingGroup) DELETE(path string, h http.HandlerFunc) {
	g.reg.record("DELETE", JoinPath(g.prefix, path), g.owner, g.onConflict)
	g.inner.DELETE(path, h)
}
func (g *recordingGroup) Use(mw func(http.Handler) http.Handler) { g.inner.Use(mw) }
func (g *recordingGroup) Group(path, name string) GroupRouter {
	return &recordingGroup{
		inner:      g.inner.Group(path, name),
		reg:        g.reg,
		owner:      g.owner,
		prefix:     JoinPath(g.prefix, path),
		onConflict: g.onConflict,
	}
}

// JoinPath concatenates a prefix and a sub-path, normalising the boundary
// between them so e.g. "/auth/" + "/jwt" yields "/auth/jwt" rather than
// "/auth//jwt". An empty input is treated as "" — there is no implicit
// "/" prefix, so callers retain full control over absolute vs relative
// paths.
func JoinPath(prefix, sub string) string {
	switch {
	case prefix == "":
		return sub
	case sub == "":
		return prefix
	}
	left := strings.TrimRight(prefix, "/")
	right := sub
	if !strings.HasPrefix(right, "/") {
		right = "/" + right
	}
	return left + right
}
