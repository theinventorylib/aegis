package core

import (
	"context"
	"time"
)

// AuditEventType categorizes different types of security and authentication events.
// These types enable filtering, alerting, and compliance reporting.
type AuditEventType string

const (
	// Authentication events track login/logout activity

	// AuditEventLoginSuccess indicates a successful user authentication
	AuditEventLoginSuccess AuditEventType = "login_success"

	// AuditEventLoginFailed indicates a failed authentication attempt
	// (wrong password, non-existent user, account locked, etc.)
	AuditEventLoginFailed AuditEventType = "login_failed"

	// AuditEventLogout indicates an explicit user logout
	AuditEventLogout AuditEventType = "logout"

	// AuditEventSessionRefresh indicates a session was refreshed using a refresh token
	AuditEventSessionRefresh AuditEventType = "session_refresh"

	// AuditEventSessionExpired indicates a session expired due to timeout
	AuditEventSessionExpired AuditEventType = "session_expired"

	// User management events track account lifecycle

	// AuditEventUserCreated indicates a new user account was created
	AuditEventUserCreated AuditEventType = "user_created"

	// AuditEventUserUpdated indicates user data was modified
	AuditEventUserUpdated AuditEventType = "user_updated"

	// AuditEventUserDeleted indicates a user account was deleted
	AuditEventUserDeleted AuditEventType = "user_deleted"

	// AuditEventEmailChanged indicates a user's email address was changed
	AuditEventEmailChanged AuditEventType = "email_changed"

	// Password events track credential changes

	// AuditEventPasswordChanged indicates a user changed their password
	AuditEventPasswordChanged AuditEventType = "password_changed"

	// AuditEventPasswordReset indicates a password was reset via recovery flow
	AuditEventPasswordReset AuditEventType = "password_reset"

	// Security events track threats and anomalies

	// AuditEventRateLimitHit indicates a client exceeded rate limits
	AuditEventRateLimitHit AuditEventType = "rate_limit_hit"

	// AuditEventAccountLocked indicates an account was locked due to failed attempts
	AuditEventAccountLocked AuditEventType = "account_locked"

	// AuditEventSuspiciousActivity indicates anomalous behavior was detected
	AuditEventSuspiciousActivity AuditEventType = "suspicious_activity"
)

// AuditEvent represents a structured security audit log entry.
//
// Audit logs enable:
//   - Security monitoring and threat detection
//   - Compliance reporting (GDPR, SOC 2, HIPAA, etc.)
//   - Forensic investigation after incidents
//   - User activity tracking
//
// Events should be written to durable storage (database, log aggregator)
// for retention and analysis.
type AuditEvent struct {
	// ID is a unique identifier for this event
	ID string `json:"id"`

	// EventType categorizes what happened
	EventType AuditEventType `json:"event_type"`

	// UserID identifies who performed the action (empty if unauthenticated)
	UserID string `json:"user_id,omitempty"`

	// IPAddress of the client that triggered this event
	IPAddress string `json:"ip_address,omitempty"`

	// UserAgent of the client that triggered this event
	UserAgent string `json:"user_agent,omitempty"`

	// Resource identifies what was acted upon (e.g., "user:123", "session:abc")
	Resource string `json:"resource,omitempty"`

	// Action describes what was done (e.g., "create", "update", "delete")
	Action string `json:"action,omitempty"`

	// Details contains additional context specific to this event type
	Details map[string]any `json:"details,omitempty"`

	// Timestamp of when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// Success indicates if the action succeeded
	Success bool `json:"success"`

	// Error contains the error message if Success is false
	Error string `json:"error,omitempty"`
}

// AuditLogger defines the interface for audit event logging.
// Implementations can write to databases, files, log aggregators (Splunk,
// Elasticsearch), or SIEM systems.
type AuditLogger interface {
	// LogEvent records a detailed audit event.
	// Should not block - consider async/buffered implementations for high throughput.
	LogEvent(ctx context.Context, event *AuditEvent) error

	// LogAuthEvent is a convenience method for common authentication events.
	// Creates and logs an AuditEvent with authentication-specific fields.
	// IP address and user agent are automatically extracted from the request
	// context (populated by AegisContextMiddleware).
	LogAuthEvent(ctx context.Context, eventType AuditEventType, userID string, success bool, details map[string]any) error
}

// NoOpAuditLogger is a no-op implementation that discards all events.
// Useful for testing or when audit logging is not required.
type NoOpAuditLogger struct{}

// logAuthEvent writes an audit event, absorbing failures into the
// structured logger instead of failing the operation. Audit-write failures
// must never break auth flows, but they should not vanish silently either.
// Single owner of that policy; use this instead of calling LogAuthEvent
// directly from services.
func logAuthEvent(ctx context.Context, logger Logger, l AuditLogger, event AuditEventType, userID string, success bool, details map[string]any) {
	if err := l.LogAuthEvent(ctx, event, userID, success, details); err != nil {
		if logger != nil {
			logger.Error("audit: failed to write event",
				"event", string(event), "user_id", userID, "error", err)
		}
	}
}

// LogEvent implements AuditLogger.
func (n *NoOpAuditLogger) LogEvent(_ context.Context, _ *AuditEvent) error {
	return nil
}

// LogAuthEvent implements AuditLogger.
func (n *NoOpAuditLogger) LogAuthEvent(_ context.Context, _ AuditEventType, _ string, _ bool, _ map[string]any) error {
	return nil
}
