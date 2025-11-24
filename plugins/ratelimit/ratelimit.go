// Package ratelimit provides rate limiting functionality.
package ratelimit

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/theinventorylib/aegis"
)

// Plugin implements rate limiting
type Plugin struct {
	limits map[string]*rateLimiter
	mu     sync.RWMutex
}

// rateLimiter tracks rate limit for a specific key
type rateLimiter struct {
	requests int
	window   time.Time
	limit    int
	duration time.Duration
}

// New creates a new rate limit plugin
func New() *Plugin {
	return &Plugin{
		limits: make(map[string]*rateLimiter),
	}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "ratelimit"
}

// Init initializes the rate limit plugin
func (p *Plugin) Init(a *aegis.Aegis) error {
	// Get the router
	router := a.GetRouter()

	// Apply rate limiting middleware to sensitive routes
	router.Use(p.Middleware(100, time.Minute)) // 100 requests per minute

	return nil
}

// Middleware returns a rate limiting middleware
func (p *Plugin) Middleware(limit int, duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP as key
			key := r.RemoteAddr

			// Check if X-Forwarded-For header exists
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				key = forwarded
			}

			// Check rate limit
			if !p.allow(key, limit, duration) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// allow checks if a request should be allowed
func (p *Plugin) allow(key string, limit int, duration time.Duration) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	// Get or create rate limiter for this key
	rl, exists := p.limits[key]
	if !exists {
		rl = &rateLimiter{
			requests: 1,
			window:   now,
			limit:    limit,
			duration: duration,
		}
		p.limits[key] = rl
		return true
	}

	// Check if window has expired
	if now.Sub(rl.window) > rl.duration {
		rl.requests = 1
		rl.window = now
		return true
	}

	// Check if limit exceeded
	if rl.requests >= rl.limit {
		return false
	}

	rl.requests++
	return true
}

// Cleanup removes expired rate limiters
func (p *Plugin) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for key, rl := range p.limits {
		if now.Sub(rl.window) > rl.duration*2 {
			delete(p.limits, key)
		}
	}
}

// StartCleanup starts a goroutine that periodically cleans up expired limiters
func (p *Plugin) StartCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			p.Cleanup()
		}
	}()
}

// GetStatus returns the current status for a key
func (p *Plugin) GetStatus(key string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	rl, exists := p.limits[key]
	if !exists {
		return fmt.Sprintf("No requests from %s", key)
	}

	remaining := rl.limit - rl.requests
	if remaining < 0 {
		remaining = 0
	}

	return fmt.Sprintf("%d/%d requests, %d remaining", rl.requests, rl.limit, remaining)
}
