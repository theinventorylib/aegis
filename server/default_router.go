package server

import "net/http"

// DefaultRouter wraps the standard library http.ServeMux.
type DefaultRouter struct {
	mux *http.ServeMux
}

// NewDefaultRouter creates a new default router using net/http.
func NewDefaultRouter(mux *http.ServeMux) *DefaultRouter {
	return &DefaultRouter{
		mux: mux,
	}
}

// GET registers a GET route.
func (r *DefaultRouter) GET(path string, handler http.HandlerFunc) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			handler(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// POST registers a POST route.
func (r *DefaultRouter) POST(path string, handler http.HandlerFunc) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			handler(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// PUT registers a PUT route.
func (r *DefaultRouter) PUT(path string, handler http.HandlerFunc) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPut {
			handler(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// PATCH registers a PATCH route.
func (r *DefaultRouter) PATCH(path string, handler http.HandlerFunc) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPatch {
			handler(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// DELETE registers a DELETE route.
func (r *DefaultRouter) DELETE(path string, handler http.HandlerFunc) {
	r.mux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			handler(w, req)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
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
