package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Router is the interface for HTTP routers
type Router interface {
	GET(path string, handler http.HandlerFunc)
	POST(path string, handler http.HandlerFunc)
	PUT(path string, handler http.HandlerFunc)
	PATCH(path string, handler http.HandlerFunc)
	DELETE(path string, handler http.HandlerFunc)
	Use(middleware func(http.Handler) http.Handler)
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// ChiRouter adapts chi.Router to our Router interface
type ChiRouter struct {
	mux chi.Router
}

// NewChiRouter creates a new Chi router adapter
func NewChiRouter(mux chi.Router) *ChiRouter {
	return &ChiRouter{mux: mux}
}

// GET registers a GET route
func (r *ChiRouter) GET(path string, handler http.HandlerFunc) {
	r.mux.Get(path, handler)
}

// POST registers a POST route
func (r *ChiRouter) POST(path string, handler http.HandlerFunc) {
	r.mux.Post(path, handler)
}

// PUT registers a PUT route
func (r *ChiRouter) PUT(path string, handler http.HandlerFunc) {
	r.mux.Put(path, handler)
}

// PATCH registers a PATCH route
func (r *ChiRouter) PATCH(path string, handler http.HandlerFunc) {
	r.mux.Patch(path, handler)
}

// DELETE registers a DELETE route
func (r *ChiRouter) DELETE(path string, handler http.HandlerFunc) {
	r.mux.Delete(path, handler)
}

// Use registers middleware
func (r *ChiRouter) Use(middleware func(http.Handler) http.Handler) {
	r.mux.Use(middleware)
}

// ServeHTTP implements http.Handler
func (r *ChiRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}
