package server

import (
	"net/http"
	"sync"
)

// DefaultRouter wraps the standard library http.ServeMux with method-aware routing.
type DefaultRouter struct {
	mux      *http.ServeMux
	handlers map[string]map[string]http.HandlerFunc // path -> method -> handler
	mu       sync.RWMutex
}

// NewDefaultRouter creates a new default router using net/http.
func NewDefaultRouter(mux *http.ServeMux) *DefaultRouter {
	return &DefaultRouter{
		mux:      mux,
		handlers: make(map[string]map[string]http.HandlerFunc),
	}
}

// registerHandler registers a handler for a specific method and path.
func (r *DefaultRouter) registerHandler(path, method string, handler http.HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Initialize path handlers map if needed
	if r.handlers[path] == nil {
		r.handlers[path] = make(map[string]http.HandlerFunc)

		// Register a single handler for this path that dispatches based on method
		r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
			r.mu.RLock()
			methodHandlers := r.handlers[path]
			r.mu.RUnlock()

			if h, ok := methodHandlers[req.Method]; ok {
				h(w, req)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})
	}

	// Store the handler for this method
	r.handlers[path][method] = handler
}

// GET registers a GET route.
func (r *DefaultRouter) GET(path string, handler http.HandlerFunc) {
	r.registerHandler(path, http.MethodGet, handler)
}

// POST registers a POST route.
func (r *DefaultRouter) POST(path string, handler http.HandlerFunc) {
	r.registerHandler(path, http.MethodPost, handler)
}

// PUT registers a PUT route.
func (r *DefaultRouter) PUT(path string, handler http.HandlerFunc) {
	r.registerHandler(path, http.MethodPut, handler)
}

// PATCH registers a PATCH route.
func (r *DefaultRouter) PATCH(path string, handler http.HandlerFunc) {
	r.registerHandler(path, http.MethodPatch, handler)
}

// DELETE registers a DELETE route.
func (r *DefaultRouter) DELETE(path string, handler http.HandlerFunc) {
	r.registerHandler(path, http.MethodDelete, handler)
}

// Use registers middleware (applies to all routes).
func (r *DefaultRouter) Use(middleware func(http.Handler) http.Handler) {
	// For the default router, we wrap the entire mux.
	r.mux = middleware(r.mux).(*http.ServeMux)
}

// ServeHTTP implements http.Handler.
func (r *DefaultRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Handler returns the underlying http.Handler (useful for http.ListenAndServe).
func (r *DefaultRouter) Handler() http.Handler {
	return r.mux
}
