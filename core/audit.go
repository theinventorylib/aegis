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
	Details map[string]interface{} `json:"details,omitempty"`

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
	LogAuthEvent(ctx context.Context, eventType AuditEventType, userID, ipAddress, userAgent string, success bool, details map[string]interface{}) error
}

// NoOpAuditLogger is a no-op implementation that discards all events.
// Useful for testing or when audit logging is not required.
type NoOpAuditLogger struct{}

// LogEvent implements AuditLogger.
func (n *NoOpAuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	return nil
}

// LogAuthEvent implements AuditLogger.
func (n *NoOpAuditLogger) LogAuthEvent(ctx context.Context, eventType AuditEventType, userID, ipAddress, userAgent string, success bool, details map[string]interface{}) error {
	return nil
}

// LoggerAuditLogger implements AuditLogger using a structured logger interface.
// This adapter allows using any logger (zap, logrus, slog) that implements
// the Info/Error/Debug methods.
type LoggerAuditLogger struct {
	logger interface {
		Info(msg string, keysAndValues ...interface{})
		Error(msg string, keysAndValues ...interface{})
		Debug(msg string, keysAndValues ...interface{})
	}
}

// NewLoggerAuditLogger creates an audit logger that writes to a structured logger.
func NewLoggerAuditLogger(logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}) *LoggerAuditLogger {
	return &LoggerAuditLogger{logger: logger}
}

// LogEvent implements AuditLogger.
func (l *LoggerAuditLogger) LogEvent(ctx context.Context, event *AuditEvent) error {
	if l.logger == nil {
		return nil
	}

	fields := []interface{}{
		"event_type", event.EventType,
		"user_id", event.UserID,
		"ip_address", event.IPAddress,
		"user_agent", event.UserAgent,
		"resource", event.Resource,
		"action", event.Action,
		"success", event.Success,
		"timestamp", event.Timestamp,
	}

	if event.Error != "" {
		fields = append(fields, "error", event.Error)
	}

	if len(event.Details) > 0 {
		for k, v := range event.Details {
			fields = append(fields, k, v)
		}
	}

	l.logger.Info("audit event", fields...)
	return nil
}

// LogAuthEvent implements AuditLogger.
func (l *LoggerAuditLogger) LogAuthEvent(ctx context.Context, eventType AuditEventType, userID, ipAddress, userAgent string, success bool, details map[string]interface{}) error {
	event := &AuditEvent{
		ID:        GenerateID(),
		EventType: eventType,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Details:   details,
		Timestamp: time.Now(),
		Success:   success,
	}

	return l.LogEvent(ctx, event)
}
