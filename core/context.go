package core

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/theinventorylib/aegis/auth"
)

// =============================================================================
// Context Key Types
// =============================================================================

// contextKey is a custom type for context keys to prevent collisions.
//
// Using a custom unexported type ensures that Aegis context keys never conflict
// with user application keys or keys from other libraries. This follows Go best
// practices for context keys as described in the context package documentation.
type contextKey string

// Core context keys - all prefixed with "aegis_" for clarity and debugging.
// These keys are used throughout the framework to pass authentication and
// request metadata through the request lifecycle.
const (
	// userContextKey stores the base auth.User for authenticated requests
	userContextKey contextKey = "aegis_user"

	// enrichedUserKey stores the EnrichedUser with plugin extensions
	enrichedUserKey contextKey = "aegis_enriched_user"

	// sessionContextKey stores the current session for authenticated requests
	sessionContextKey contextKey = "aegis_session"

	// requestIDContextKey stores a unique identifier for this request (for logging/tracing)
	requestIDContextKey contextKey = "aegis_request_id"

	// requestMetaKey stores arbitrary request metadata
	requestMetaKey contextKey = "aegis_request_meta"

	// pathParamFuncKey stores the router's path parameter extraction function
	pathParamFuncKey contextKey = "aegis_path_param_func"

	// pluginDataKey stores plugin-specific data that shouldn't be in EnrichedUser
	pluginDataKey contextKey = "aegis_plugin_data"

	// contextInitializedKey indicates if the Aegis context has been initialized
	contextInitializedKey contextKey = "aegis_initialized"
)

// =============================================================================
// Enriched User - Extended User with Plugin Data
// =============================================================================

// EnrichedUser wraps the core auth.User with plugin-specific extensions.
//
// This is a key extensibility mechanism in Aegis that allows plugins to augment
// the base user model with additional fields without modifying the core schema.
// Plugin data is stored in the Extensions map and automatically merged into
// JSON responses.
//
// Common use cases:
//   - Admin plugin adds: "role", "permissions"
//   - Organizations plugin adds: "organizations", "currentOrg"
//   - JWT plugin adds: "claims", "tokenExp"
//   - Email verification plugin adds: "emailVerified"
//
// Extension keys should be simple field names (not nested paths). The MarshalJSON
// implementation flattens extensions as top-level fields in API responses.
//
// Example plugin usage:
//
//	enriched := core.GetEnrichedUser(ctx)
//	enriched.Set("role", "admin")
//	enriched.Set("emailVerified", true)
//	enriched.Set("organizations", []string{"org1", "org2"})
//
// Example API response:
//
//	{
//	  "id": "01HXYZ...",
//	  "email": "user@example.com",
//	  "name": "John Doe",
//	  "role": "admin",               // From extension
//	  "emailVerified": true,          // From extension
//	  "organizations": ["org1", "org2"] // From extension
//	}
//
// Thread safety: All methods are safe for concurrent use via internal mutex.
type EnrichedUser struct {
	*auth.User

	// Extensions holds additional fields from plugins.
	// Keys are simple field names: "role", "verified", "organizations", etc.
	// These are flattened into the JSON response as top-level fields.
	Extensions map[string]any `json:"-"` // Excluded from default marshal, handled in MarshalJSON

	mu sync.RWMutex
}

// NewEnrichedUser creates an EnrichedUser wrapping a core User.
// The Extensions map is initialized empty, ready for plugins to populate.
func NewEnrichedUser(user *auth.User) *EnrichedUser {
	return &EnrichedUser{
		User:       user,
		Extensions: make(map[string]any),
	}
}

// Set adds or updates an extension field.
//
// This is typically called by plugins during request processing to add their
// data to the user context. Key should be a simple field name that will become
// a top-level field in JSON responses.
//
// Thread-safe for concurrent plugin access.
func (eu *EnrichedUser) Set(key string, value any) {
	eu.mu.Lock()
	defer eu.mu.Unlock()
	if eu.Extensions == nil {
		eu.Extensions = make(map[string]any)
	}
	eu.Extensions[key] = value
}

// Get retrieves an extension value by key.
// Returns nil if the key doesn't exist.
//
// For type-safe access, prefer the typed getters (GetString, GetBool, etc.).
func (eu *EnrichedUser) Get(key string) any {
	eu.mu.RLock()
	defer eu.mu.RUnlock()
	if eu.Extensions == nil {
		return nil
	}
	return eu.Extensions[key]
}

// GetString retrieves a string extension value.
// Returns empty string if the key doesn't exist or value is not a string.
func (eu *EnrichedUser) GetString(key string) string {
	v := eu.Get(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetBool retrieves a bool extension value.
// Returns false if the key doesn't exist or value is not a bool.
func (eu *EnrichedUser) GetBool(key string) bool {
	v := eu.Get(key)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// GetStringSlice retrieves a string slice extension value.
// Returns nil if the key doesn't exist or value is not a []string.
func (eu *EnrichedUser) GetStringSlice(key string) []string {
	v := eu.Get(key)
	if ss, ok := v.([]string); ok {
		return ss
	}
	return nil
}

// GetMap retrieves a map extension value.
// Returns nil if the key doesn't exist or value is not a map.
func (eu *EnrichedUser) GetMap(key string) map[string]any {
	v := eu.Get(key)
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

// Has checks if an extension key exists.
// Returns true even if the value is nil.
func (eu *EnrichedUser) Has(key string) bool {
	eu.mu.RLock()
	defer eu.mu.RUnlock()
	if eu.Extensions == nil {
		return false
	}
	_, ok := eu.Extensions[key]
	return ok
}

// Keys returns all extension keys currently set.
// Useful for debugging or iterating over all extensions.
func (eu *EnrichedUser) Keys() []string {
	eu.mu.RLock()
	defer eu.mu.RUnlock()
	keys := make([]string, 0, len(eu.Extensions))
	for k := range eu.Extensions {
		keys = append(keys, k)
	}
	return keys
}

// MarshalJSON implements json.Marshaler for API responses.
// Extensions are flattened as top-level fields in the JSON output.
func (eu *EnrichedUser) MarshalJSON() ([]byte, error) {
	eu.mu.RLock()
	defer eu.mu.RUnlock()

	// Create a map with core user fields
	result := map[string]any{
		"id":        eu.ID,
		"email":     eu.Email,
		"name":      eu.Name,
		"avatar":    eu.Avatar,
		"disabled":  eu.Disabled,
		"createdAt": eu.CreatedAt,
		"updatedAt": eu.UpdatedAt,
	}

	// Add metadata if present
	if len(eu.Metadata) > 0 {
		result["metadata"] = eu.Metadata
	}

	// Flatten extensions as top-level fields
	for key, value := range eu.Extensions {
		result[key] = value
	}

	return json.Marshal(result)
}

// ToAPIResponse returns a map suitable for JSON API responses.
// Extensions are flattened as top-level fields.
func (eu *EnrichedUser) ToAPIResponse() map[string]any {
	return eu.ToAPIResponseFiltered(nil)
}

// ToAPIResponseFiltered returns a map suitable for JSON API responses,
// optionally filtering extension fields based on the provided config.
// If config is nil, all fields are included.
func (eu *EnrichedUser) ToAPIResponseFiltered(config *UserFieldsConfig) map[string]any {
	eu.mu.RLock()
	defer eu.mu.RUnlock()

	resp := map[string]any{
		"id":        eu.ID,
		"email":     eu.Email,
		"name":      eu.Name,
		"avatar":    eu.Avatar,
		"disabled":  eu.Disabled,
		"createdAt": eu.CreatedAt,
		"updatedAt": eu.UpdatedAt,
	}

	if len(eu.Metadata) > 0 {
		resp["metadata"] = eu.Metadata
	}

	// Build allowed fields set if config specifies fields
	var allowedFields map[string]bool
	if config != nil && len(config.Fields) > 0 {
		allowedFields = make(map[string]bool, len(config.Fields))
		for _, f := range config.Fields {
			allowedFields[f] = true
		}
	}

	// Flatten extensions as top-level fields (filtered if config provided)
	for key, value := range eu.Extensions {
		if allowedFields == nil || allowedFields[key] {
			resp[key] = value
		}
	}

	return resp
}

// sessionToMap converts a session to a map for API responses.
func sessionToMap(session *auth.Session) map[string]any {
	if session == nil {
		return nil
	}

	return map[string]any{
		"id":        session.ID,
		"expiresAt": session.ExpiresAt,
		"createdAt": session.CreatedAt,
		"ipAddress": session.IPAddress,
		"userAgent": session.UserAgent,
	}
}

// =============================================================================
// Session With User Response
// =============================================================================

// SessionWithUser combines session and enriched user data for API responses.
// This is returned by session validation endpoints.
// The user data includes all extension fields flattened.
type SessionWithUser struct {
	Session *auth.Session `json:"session"`
	User    *EnrichedUser `json:"user"`
}

// ToAPIResponse returns a map suitable for JSON API responses.
// Session includes all session fields, user includes all extension fields flattened.
func (swu *SessionWithUser) ToAPIResponse() map[string]any {
	return swu.ToAPIResponseFiltered(nil)
}

// ToAPIResponseFiltered returns a map with optional user field filtering.
// Session data is always fully included; only user extension fields are filtered.
func (swu *SessionWithUser) ToAPIResponseFiltered(config *UserFieldsConfig) map[string]any {
	resp := make(map[string]any)

	if swu.Session != nil {
		resp["session"] = sessionToMap(swu.Session)
	}

	if swu.User != nil {
		resp["user"] = swu.User.ToAPIResponseFiltered(config)
	}

	return resp
}

// =============================================================================
// Request Metadata
// =============================================================================

// RequestMeta contains metadata about the current request.
// This is useful for audit logging, rate limiting, and plugin access.
type RequestMeta struct {
	// RequestID is a unique identifier for this request (for tracing)
	RequestID string
	// IPAddress is the client's IP address
	IPAddress string
	// UserAgent is the client's User-Agent header
	UserAgent string
	// Method is the HTTP method (GET, POST, etc.)
	Method string
	// Path is the request path
	Path string
}

// =============================================================================
// Plugin Data Store
// =============================================================================

// PluginData is a thread-safe store for plugin-specific context data.
// Plugins can store and retrieve their own data without key collisions.
type PluginData struct {
	mu   sync.RWMutex
	data map[string]any
}

// NewPluginData creates a new plugin data store
func NewPluginData() *PluginData {
	return &PluginData{
		data: make(map[string]any),
	}
}

// Set stores a value for a plugin. The key should be namespaced by plugin name.
// Example: pluginData.Set("jwt:token_type", "access")
func (pd *PluginData) Set(key string, value any) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.data[key] = value
}

// Get retrieves a value from the plugin data store.
// Returns nil if the key doesn't exist.
func (pd *PluginData) Get(key string) any {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	return pd.data[key]
}

// GetString retrieves a string value, returning empty string if not found or wrong type.
func (pd *PluginData) GetString(key string) string {
	v := pd.Get(key)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// GetBool retrieves a bool value, returning false if not found or wrong type.
func (pd *PluginData) GetBool(key string) bool {
	v := pd.Get(key)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// Has checks if a key exists in the plugin data store.
func (pd *PluginData) Has(key string) bool {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	_, ok := pd.data[key]
	return ok
}

// Delete removes a key from the plugin data store.
func (pd *PluginData) Delete(key string) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	delete(pd.data, key)
}

// Keys returns all keys in the plugin data store.
func (pd *PluginData) Keys() []string {
	pd.mu.RLock()
	defer pd.mu.RUnlock()
	keys := make([]string, 0, len(pd.data))
	for k := range pd.data {
		keys = append(keys, k)
	}
	return keys
}

// =============================================================================
// Path Parameter Function
// =============================================================================

// PathParamFunc is a function type for extracting path parameters from requests.
// This allows different routers to provide their own implementation.
type PathParamFunc func(r *http.Request, name string) string

// =============================================================================
// Context Setters (With* functions)
// =============================================================================

// WithUser adds a user to the context.
// This is typically called by AuthMiddleware after validating a session.
func WithUser(ctx context.Context, user *auth.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// WithSession adds a session to the context.
// This is called by AuthMiddleware along with WithUser.
func WithSession(ctx context.Context, session *auth.Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// WithRequestID adds a request ID to the context for tracing.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

// WithRequestMeta adds request metadata to the context.
// This includes IP address, user agent, method, and path.
func WithRequestMeta(ctx context.Context, meta *RequestMeta) context.Context {
	return context.WithValue(ctx, requestMetaKey, meta)
}

// WithPathParamFunc adds a path parameter extraction function to the context.
// This is called by router middleware to inject the router-specific implementation.
func WithPathParamFunc(ctx context.Context, fn PathParamFunc) context.Context {
	return context.WithValue(ctx, pathParamFuncKey, fn)
}

// WithPluginData adds a plugin data store to the context.
// This is called once per request to initialize the plugin data store.
func WithPluginData(ctx context.Context, pd *PluginData) context.Context {
	return context.WithValue(ctx, pluginDataKey, pd)
}

// WithEnrichedUser adds an enriched user to the context.
// This is called by AuthMiddleware after creating the EnrichedUser.
func WithEnrichedUser(ctx context.Context, eu *EnrichedUser) context.Context {
	return context.WithValue(ctx, enrichedUserKey, eu)
}

// WithContextInitialized marks the context as initialized by Aegis.
// This is used internally to check if AegisContextMiddleware was called.
func WithContextInitialized(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextInitializedKey, true)
}

// =============================================================================
// Context Getters (Get* functions)
// =============================================================================

// GetUser extracts the user from the context.
// Returns an error if no user is present (not authenticated).
func GetUser(ctx context.Context) (*auth.User, error) {
	user, ok := ctx.Value(userContextKey).(*auth.User)
	if !ok || user == nil {
		return nil, NewAuthError(AuthErrorCodeUnauthorized, "user not found in context")
	}
	return user, nil
}

// GetEnrichedUser extracts the enriched user from context.
// This includes all plugin extensions (admin role, jwt claims, org memberships, etc.)
// Returns nil if no authenticated user or enriched user not set.
func GetEnrichedUser(ctx context.Context) *EnrichedUser {
	eu, ok := ctx.Value(enrichedUserKey).(*EnrichedUser)
	_ = ok
	return eu
}

// MustGetEnrichedUser extracts the enriched user, panicking if not found.
// Use only in handlers where authentication is guaranteed.
func MustGetEnrichedUser(ctx context.Context) *EnrichedUser {
	eu := GetEnrichedUser(ctx)
	if eu == nil {
		panic("MustGetEnrichedUser called without enriched user in context")
	}
	return eu
}

// MustGetUser extracts the user from context, panicking if not found.
// Use this only in handlers where authentication is guaranteed by middleware.
func MustGetUser(ctx context.Context) *auth.User {
	user, err := GetUser(ctx)
	if err != nil {
		panic("MustGetUser called without authenticated user in context")
	}
	return user
}

// GetSession extracts the session from the context.
// Returns nil if no session is present.
// This acts as a per-request cache - once a session is stored in context,
// it can be retrieved without hitting the database or Redis again.
func GetSession(ctx context.Context) *auth.Session {
	session, ok := ctx.Value(sessionContextKey).(*auth.Session)
	_ = ok
	return session
}

// HasSession checks if a session exists in context (already validated for this request).
// Use this to avoid redundant session validation within the same request.
func HasSession(ctx context.Context) bool {
	return GetSession(ctx) != nil
}

// IsContextInitialized checks if AegisContextMiddleware has been run.
// This is used internally to ensure proper middleware chain ordering.
func IsContextInitialized(ctx context.Context) bool {
	initialized, err := ctx.Value(contextInitializedKey).(bool)
	_ = err
	return initialized
}

// GetRequestID extracts the request ID from the context.
// Returns empty string if not set.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDContextKey).(string); ok {
		return id
	}
	return ""
}

// GetRequestMeta extracts the request metadata from the context.
// Returns nil if not set.
func GetRequestMeta(ctx context.Context) *RequestMeta {
	meta, err := ctx.Value(requestMetaKey).(*RequestMeta)
	_ = err
	return meta
}

// GetPluginData extracts the plugin data store from the context.
// Returns nil if not initialized. Plugins should check for nil.
func GetPluginData(ctx context.Context) *PluginData {
	pd, err := ctx.Value(pluginDataKey).(*PluginData)
	_ = err
	return pd
}

// GetPluginValue is a convenience function to get a plugin value directly from context.
// Returns nil if plugin data is not initialized or key doesn't exist.
func GetPluginValue(ctx context.Context, key string) any {
	pd := GetPluginData(ctx)
	if pd == nil {
		return nil
	}
	return pd.Get(key)
}

// SetPluginValue is a convenience function to set a plugin value directly in context.
// Does nothing if plugin data is not initialized.
func SetPluginValue(ctx context.Context, key string, value any) {
	pd := GetPluginData(ctx)
	if pd != nil {
		pd.Set(key, value)
	}
}

// =============================================================================
// Authentication Helpers
// =============================================================================

// Authenticated checks if the context has an authenticated user.
func Authenticated(ctx context.Context) bool {
	user, err := GetUser(ctx)
	if err != nil {
		return false
	}
	return user != nil
}

// GetUserID is a convenience function to get just the user ID from context.
// Returns empty string if not authenticated.
func GetUserID(ctx context.Context) string {
	user, err := GetUser(ctx)
	if err != nil {
		return ""
	}
	return user.ID
}

// =============================================================================
// User Extension Helpers (for plugins to enrich user data)
// =============================================================================

// ExtendUser adds data to the enriched user in context.
// If no enriched user exists, this is a no-op.
// Plugins should call this in their middleware or handlers to add their data.
//
// Example:
//
//	core.ExtendUser(ctx, "admin:role", "admin")
//	core.ExtendUser(ctx, "orgs:memberships", []string{"org1", "org2"})
func ExtendUser(ctx context.Context, key string, value any) {
	eu := GetEnrichedUser(ctx)
	if eu != nil {
		eu.Set(key, value)
	}
}

// GetUserExtension retrieves a specific extension from the enriched user.
// Returns nil if user is not authenticated or extension doesn't exist.
func GetUserExtension(ctx context.Context, key string) any {
	eu := GetEnrichedUser(ctx)
	if eu == nil {
		return nil
	}
	return eu.Get(key)
}

// GetUserExtensionString retrieves a string extension from the enriched user.
func GetUserExtensionString(ctx context.Context, key string) string {
	eu := GetEnrichedUser(ctx)
	if eu == nil {
		return ""
	}
	return eu.GetString(key)
}

// GetUserExtensionBool retrieves a bool extension from the enriched user.
func GetUserExtensionBool(ctx context.Context, key string) bool {
	eu := GetEnrichedUser(ctx)
	if eu == nil {
		return false
	}
	return eu.GetBool(key)
}

// =============================================================================
// Request Helpers
// =============================================================================

// GetPathParam extracts a path parameter from the request using the router's
// path param function stored in context. Falls back to Go 1.22+ PathValue.
func GetPathParam(r *http.Request, name string) string {
	if fn, ok := r.Context().Value(pathParamFuncKey).(PathParamFunc); ok && fn != nil {
		if value := fn(r, name); value != "" {
			return value
		}
	}

	// Fallback to Go 1.22+ standard library
	return r.PathValue(name)
}

// GetSanitizedPathParam extracts and sanitizes a path parameter.
// This is the recommended way to retrieve IDs and other path parameters
// from the URL path.
func GetSanitizedPathParam(r *http.Request, name string) string {
	return SanitizeString(GetPathParam(r, name), nil)
}

// GetIPAddress extracts the IP address from context metadata.
// Falls back to extracting from request if not in context.
func GetIPAddress(ctx context.Context) string {
	if meta := GetRequestMeta(ctx); meta != nil {
		return meta.IPAddress
	}
	return ""
}

// GetUserAgent extracts the user agent from context metadata.
func GetUserAgent(ctx context.Context) string {
	if meta := GetRequestMeta(ctx); meta != nil {
		return meta.UserAgent
	}
	return ""
}

// =============================================================================
// Context Builder (for testing and programmatic use)
// =============================================================================

// AegisContext is a builder for creating Aegis-enriched contexts.
// Useful for testing and programmatic context creation.
type AegisContext struct {
	ctx context.Context
}

// NewAegisContext creates a new context builder from an existing context.
func NewAegisContext(ctx context.Context) *AegisContext {
	return &AegisContext{ctx: ctx}
}

// WithUser adds a user to the context.
func (ac *AegisContext) WithUser(user *auth.User) *AegisContext {
	ac.ctx = WithUser(ac.ctx, user)
	// Also create enriched user
	ac.ctx = WithEnrichedUser(ac.ctx, NewEnrichedUser(user))
	return ac
}

// WithSession adds a session to the context.
func (ac *AegisContext) WithSession(session *auth.Session) *AegisContext {
	ac.ctx = WithSession(ac.ctx, session)
	return ac
}

// WithRequestID adds a request ID to the context.
func (ac *AegisContext) WithRequestID(id string) *AegisContext {
	ac.ctx = WithRequestID(ac.ctx, id)
	return ac
}

// WithRequestMeta adds request metadata to the context.
func (ac *AegisContext) WithRequestMeta(meta *RequestMeta) *AegisContext {
	ac.ctx = WithRequestMeta(ac.ctx, meta)
	return ac
}

// WithPluginData adds plugin data to the context.
func (ac *AegisContext) WithPluginData() *AegisContext {
	ac.ctx = WithPluginData(ac.ctx, NewPluginData())
	return ac
}

// WithExtension adds a user extension to the context.
// Requires WithUser to be called first.
func (ac *AegisContext) WithExtension(key string, value any) *AegisContext {
	ExtendUser(ac.ctx, key, value)
	return ac
}

// Context returns the built context.
func (ac *AegisContext) Context() context.Context {
	return ac.ctx
}
